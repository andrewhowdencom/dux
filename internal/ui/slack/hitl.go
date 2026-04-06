package slack

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/slack-go/slack"
)

type SlackHITL struct {
	api       *slack.Client
	channelID string
	threadTS  string
	agentName string

	mu         sync.Mutex
	pendingMap map[string]chan bool
}

func NewSlackHITL(api *slack.Client, channelID, threadTS, agentName string) *SlackHITL {
	return &SlackHITL{
		api:        api,
		channelID:  channelID,
		threadTS:   threadTS,
		agentName:  agentName,
		pendingMap: make(map[string]chan bool),
	}
}

func generateShortID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *SlackHITL) ApproveTool(ctx context.Context, req llm.ToolRequestPart) (bool, error) {
	slog.Info("ApproveTool requested in Slack", "call_id", req.ToolID, "tool", req.Name)

	argsStr := fmt.Sprintf("%v", req.Args)

	text := fmt.Sprintf("🔐 *Tool Approval Required*\n\n*Tool:* `%s`\n*Args:* ```%s```", req.Name, argsStr)

	shortID := generateShortID()

	// Build block kit message
	headerText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	headerSection := slack.NewSectionBlock(headerText, nil, nil)

	approveBtnTxt := slack.NewTextBlockObject("plain_text", "Approve", false, false)
	approveBtn := slack.NewButtonBlockElement("a_"+shortID, "approve", approveBtnTxt)
	approveBtn.Style = slack.StylePrimary

	denyBtnTxt := slack.NewTextBlockObject("plain_text", "Deny", false, false)
	denyBtn := slack.NewButtonBlockElement("d_"+shortID, "deny", denyBtnTxt)
	denyBtn.Style = slack.StyleDanger

	actionBlock := slack.NewActionBlock("", approveBtn, denyBtn)

	opts := []slack.MsgOption{
		slack.MsgOptionBlocks(headerSection, actionBlock),
	}
	if h.threadTS != "" {
		opts = append(opts, slack.MsgOptionTS(h.threadTS))
	}
	if h.agentName != "" {
		opts = append(opts, slack.MsgOptionUsername(h.agentName))
	}

	_, _, err := h.api.PostMessage(h.channelID, opts...)
	if err != nil {
		slog.Error("failed to send hitl message to slack", "error", err)
		return false, err
	}

	resultChan := make(chan bool, 1)
	h.mu.Lock()
	h.pendingMap[shortID] = resultChan
	h.mu.Unlock()

	var result bool
	select {
	case <-ctx.Done():
		result = false
	case res := <-resultChan:
		result = res
	}

	h.mu.Lock()
	delete(h.pendingMap, shortID)
	h.mu.Unlock()

	return result, nil
}

func (h *SlackHITL) Resolve(interaction slack.InteractionCallback) error {
	if len(interaction.ActionCallback.BlockActions) == 0 {
		return fmt.Errorf("no block actions in interaction")
	}

	action := interaction.ActionCallback.BlockActions[0]
	actionID := action.ActionID

	var callID string
	var approved bool

	if strings.HasPrefix(actionID, "a_") {
		callID = strings.TrimPrefix(actionID, "a_")
		approved = true
	} else if strings.HasPrefix(actionID, "d_") {
		callID = strings.TrimPrefix(actionID, "d_")
		approved = false
	} else {
		return fmt.Errorf("unknown actionID data: %s", actionID)
	}

	h.mu.Lock()
	ch, ok := h.pendingMap[callID]
	h.mu.Unlock()

	if !ok {
		return fmt.Errorf("call_id %s not pending", callID)
	}

	ch <- approved

	status := "Denied ❌"
	if approved {
		status = "Approved ✅"
	}

	// Update original message to remove buttons and show status
	// Slack replaces the message
	blocks := interaction.Message.Blocks.BlockSet
	if len(blocks) > 0 {
		// Just take the first block and append status
		statusTxt := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("\n\n*Status:* %s", status), false, false)
		statusSection := slack.NewSectionBlock(statusTxt, nil, nil)
		
		opts := []slack.MsgOption{
			slack.MsgOptionBlocks(blocks[0], statusSection),
		}
		if h.agentName != "" {
			opts = append(opts, slack.MsgOptionUsername(h.agentName))
		}
		
		_, _, _, err := h.api.UpdateMessage(interaction.Channel.ID, interaction.Message.Timestamp, opts...)
		if err != nil {
			slog.Error("Failed to update HITL message in Slack", "err", err)
		}
	}

	return nil
}
