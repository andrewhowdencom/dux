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
	reasoning       string
	tools           []string
	telemetry       string
	pendingUpdate   bool
	updateThrottler *time.Ticker
}

func NewStreamTracker(bot *tgbotapi.BotAPI, chatID int64) *StreamTracker {
	return &StreamTracker{
		bot:    bot,
		chatID: chatID,
	}
}

func (st *StreamTracker) ProcessStream(ctx context.Context, stream <-chan llm.Message) {
	st.updateThrottler = time.NewTicker(1500 * time.Millisecond)
	defer st.updateThrottler.Stop()

	// Initial placeholder message
	msg := tgbotapi.NewMessage(st.chatID, "...")
	if sent, err := st.bot.Send(msg); err == nil {
		st.messageID = sent.MessageID
	} else {
		slog.Error("failed to send initial stream message", "err", err)
	}

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				// Final tick
				st.tick()
				return
			case <-st.updateThrottler.C:
				st.tick()
			}
		}
	}()

	for msg := range stream {
		if len(msg.Parts) == 0 {
			continue
		}
		
		st.mu.Lock()
		for _, part := range msg.Parts {
			switch p := part.(type) {
			case llm.TextPart:
				st.text += string(p)
				st.pendingUpdate = true
			case llm.ReasoningPart:
				st.reasoning += string(p)
				st.pendingUpdate = true
			case llm.ToolRequestPart:
				st.tools = append(st.tools, fmt.Sprintf("🔧 <b>Calling Tool %s</b>\n<pre>%v</pre>", p.Name, html.EscapeString(fmt.Sprintf("%v", p.Args))))
				st.pendingUpdate = true
			case llm.ToolResultPart:
				status := "✅"
				if p.IsError {
					status = "❌"
				}
				st.tools = append(st.tools, fmt.Sprintf("%s <b>Result from %s</b>\n<pre>%v</pre>", status, p.Name, html.EscapeString(fmt.Sprintf("%v", p.Result))))
				st.pendingUpdate = true
			case llm.TelemetryPart:
				st.telemetry = fmt.Sprintf("\n\n📊 <b>Telemetry:</b>\nInput: %d | Output: %d | Reasoning: %d | Dur: %s", 
					p.InputTokens, p.OutputTokens, p.ReasoningTokens, p.Duration)
				st.pendingUpdate = true
			}
		}
		st.mu.Unlock()
	}

	close(done)
}

func (st *StreamTracker) tick() {
	st.mu.Lock()
	if !st.pendingUpdate {
		st.mu.Unlock()
		return
	}
	
	var b strings.Builder
	
	if st.reasoning != "" {
		b.WriteString("🤔 <b>Thinking...</b>\n")
		b.WriteString(fmt.Sprintf("<i>%s</i>\n\n", html.EscapeString(st.reasoning)))
	}
	
	if st.text != "" {
		b.WriteString(html.EscapeString(st.text))
	}
	
	if len(st.tools) > 0 {
		b.WriteString("\n\n")
		b.WriteString(strings.Join(st.tools, "\n\n"))
	}
	
	if st.telemetry != "" {
		b.WriteString(st.telemetry)
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
