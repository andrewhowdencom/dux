package telegram

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/andrewhowdencom/dux/pkg/llm"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramHITL struct {
	bot    *tgbotapi.BotAPI
	chatID int64

	mu         sync.Mutex
	pendingMap map[string]chan bool
}

func NewTelegramHITL(bot *tgbotapi.BotAPI, chatID int64) *TelegramHITL {
	return &TelegramHITL{
		bot:        bot,
		chatID:     chatID,
		pendingMap: make(map[string]chan bool),
	}
}

func generateShortID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *TelegramHITL) ApproveTool(ctx context.Context, req llm.ToolRequestPart) (bool, error) {
	slog.Info("ApproveTool requested in Telegram", "call_id", req.ToolID, "tool", req.Name)

	argsStr := fmt.Sprintf("%v", req.Args)

	text := fmt.Sprintf("🔐 Tool Approval Required\n\nTool: %s\nArgs: %s", req.Name, argsStr)

	msg := tgbotapi.NewMessage(h.chatID, text)

	shortID := generateShortID()

	btnApprove := tgbotapi.NewInlineKeyboardButtonData("Approve", "a_"+shortID)
	btnDeny := tgbotapi.NewInlineKeyboardButtonData("Deny", "d_"+shortID)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btnApprove, btnDeny),
	)

	_, err := h.bot.Send(msg)
	if err != nil {
		slog.Error("failed to send hitl message", "error", err)
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

func (h *TelegramHITL) Resolve(query *tgbotapi.CallbackQuery) error {
	data := query.Data

	var callID string
	var approved bool

	if strings.HasPrefix(data, "a_") {
		callID = strings.TrimPrefix(data, "a_")
		approved = true
	} else if strings.HasPrefix(data, "d_") {
		callID = strings.TrimPrefix(data, "d_")
		approved = false
	} else {
		return fmt.Errorf("unknown callback data: %s", data)
	}

	h.mu.Lock()
	ch, ok := h.pendingMap[callID]
	h.mu.Unlock()

	if !ok {
		text := "This request has already been resolved or expired."
		_, _ = h.bot.Request(tgbotapi.NewCallback(query.ID, text))
		return fmt.Errorf("call_id %s not pending", callID)
	}

	ch <- approved

	// Update the original message to remove buttons
	status := "Denied ❌"
	if approved {
		status = "Approved ✅"
	}

	newText := fmt.Sprintf("%s\n\nStatus: %s", query.Message.Text, status)
	edit := tgbotapi.NewEditMessageText(h.chatID, query.Message.MessageID, newText)
	_, _ = h.bot.Send(edit)

	// Acknowledge the callback
	_, _ = h.bot.Request(tgbotapi.NewCallback(query.ID, status))

	return nil
}
