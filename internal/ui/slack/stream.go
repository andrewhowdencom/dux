package slack

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/slack-go/slack"
)

type StreamTracker struct {
	api       *slack.Client
	channelID string
	threadTS  string
	agentName string
	messageTS string

	mu              sync.Mutex
	text            string
	pendingUpdate   bool
	updateThrottler *time.Ticker

	OnReset func()
}

func NewStreamTracker(api *slack.Client, channelID, threadTS, agentName string) *StreamTracker {
	return &StreamTracker{
		api:       api,
		channelID: channelID,
		threadTS:  threadTS,
		agentName: agentName,
	}
}

// StartWorker initializes the ticker and starts pushing updates to Slack in the background.
func (st *StreamTracker) StartWorker(ctx context.Context) func() {
	st.updateThrottler = time.NewTicker(3 * time.Second) // Important: Slack rate limits are typically 1 API call per second

	opts := []slack.MsgOption{
		slack.MsgOptionText("...", false),
	}
	if st.threadTS != "" {
		opts = append(opts, slack.MsgOptionTS(st.threadTS))
	}
	if st.agentName != "" {
		opts = append(opts, slack.MsgOptionUsername(st.agentName))
	}

	if _, sentTS, err := st.api.PostMessage(st.channelID, opts...); err == nil {
		st.messageTS = sentTS
	} else {
		slog.Error("failed to send initial stream message to slack", "err", err)
	}

	done := make(chan struct{})

	go func() {
		defer st.updateThrottler.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				// Final tick handled in Flush
				return
			case <-st.updateThrottler.C:
				st.tick()
			}
		}
	}()

	return func() {
		close(done)
	}
}

func (st *StreamTracker) RenderTextChunk(chunk string) {
	st.mu.Lock()
	st.text += chunk
	st.pendingUpdate = true
	st.mu.Unlock()
}

func (st *StreamTracker) RenderError(err error) {
	st.mu.Lock()
	st.text += fmt.Sprintf("\n\n❌ Error: %v", err)
	st.pendingUpdate = true
	st.mu.Unlock()
}

func (st *StreamTracker) PromptHITL(req *llm.ToolRequestPart) {
	st.Flush()

	st.mu.Lock()
	st.messageTS = ""
	st.text = ""
	st.pendingUpdate = false
	st.mu.Unlock()
}

func (st *StreamTracker) OnCommand(cmd string, args []string) {
	if cmd == "/new" {
		if st.OnReset != nil {
			st.OnReset()
		}
		st.RenderTextChunk("Started a new conversation session.")
		st.Flush()
	}
}

func (st *StreamTracker) Flush() {
	st.tick()
}

func (st *StreamTracker) tick() {
	st.mu.Lock()
	if !st.pendingUpdate && st.text != "" {
		st.mu.Unlock()
		return
	}

	currentText := st.text
	st.pendingUpdate = false
	st.mu.Unlock()

	opts := []slack.MsgOption{
		slack.MsgOptionText(currentText, false),
	}
	if st.threadTS != "" {
		opts = append(opts, slack.MsgOptionTS(st.threadTS))
	}
	if st.agentName != "" {
		opts = append(opts, slack.MsgOptionUsername(st.agentName))
	}

	if st.messageTS != "" {
		_, _, _, err := st.api.UpdateMessage(st.channelID, st.messageTS, opts...)
		if err != nil {
			slog.Debug("failed to update stream message to slack (usually ok to ignore due to rate limits)", "err", err)
			st.mu.Lock()
			st.pendingUpdate = true
			st.mu.Unlock()
		}
	} else if len(currentText) > 0 && currentText != "..." {
		// Just send a new message if we missed the first one
		if _, sentTS, err := st.api.PostMessage(st.channelID, opts...); err == nil {
			st.messageTS = sentTS
		}
	}
}
