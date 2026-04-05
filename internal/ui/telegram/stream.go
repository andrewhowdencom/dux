package telegram

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type StreamTracker struct {
	bot       *tgbotapi.BotAPI
	chatID    int64
	messageID int

	mu              sync.Mutex
	text            string
	pendingUpdate   bool
	updateThrottler *time.Ticker

	OnReset func()
}

func NewStreamTracker(bot *tgbotapi.BotAPI, chatID int64) *StreamTracker {
	return &StreamTracker{
		bot:    bot,
		chatID: chatID,
	}
}

// StartWorker initializes the ticker and starts pushing updates to telegram in the background.
func (st *StreamTracker) StartWorker(ctx context.Context) func() {
	st.updateThrottler = time.NewTicker(1500 * time.Millisecond)

	// Initial placeholder message
	msg := tgbotapi.NewMessage(st.chatID, "...")
	if sent, err := st.bot.Send(msg); err == nil {
		st.messageID = sent.MessageID
	} else {
		slog.Error("failed to send initial stream message", "err", err)
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
	st.messageID = 0
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
	if !st.pendingUpdate {
		st.mu.Unlock()
		return
	}
	
	var b strings.Builder
	
	if st.text != "" {
		b.WriteString(html.EscapeString(st.text))
	}

	st.pendingUpdate = false
	currentText := b.String()
	st.mu.Unlock()

	if st.messageID != 0 {
		if len(currentText) > 4000 {
			currentText = currentText[:4000] + "\n...[truncated]"
		}
		
		edit := tgbotapi.NewEditMessageText(st.chatID, st.messageID, currentText)
		edit.ParseMode = tgbotapi.ModeHTML
		_, err := st.bot.Send(edit)
		if err != nil {
			slog.Debug("failed to update stream message (usually ok to ignore)", "err", err)
			st.mu.Lock()
			st.pendingUpdate = true
			st.mu.Unlock()
		}
	} else if len(currentText) > 0 && currentText != "..." {
		// Just send a new message if we missed the first one
		msg := tgbotapi.NewMessage(st.chatID, currentText)
		msg.ParseMode = tgbotapi.ModeHTML
		if sent, err := st.bot.Send(msg); err == nil {
			st.messageID = sent.MessageID
		}
	}
}
