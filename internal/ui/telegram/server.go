package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/adrg/xdg"
	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/internal/ui"
	"github.com/andrewhowdencom/dux/pkg/llm"
	pkgui "github.com/andrewhowdencom/dux/pkg/ui"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Streamer interface {
	Stream(ctx context.Context, inputMessage llm.Message) (<-chan llm.Message, error)
}

type EngineFactory func(ctx context.Context, agentName string, providerID string, agentsFilePath string, hitl llm.HITLHandler, unsafeAllTools bool) (Streamer, *config.InstanceConfig, func(), error)

type Session struct {
	ChatID      int64
	Engine      Streamer
	CleanupOpts []func()
	Queue       chan tgbotapi.Update
	HITL        *TelegramHITL
	Ctx         context.Context
	Cancel      context.CancelFunc
}

type Server struct {
	bot        *tgbotapi.BotAPI
	whitelist  map[int64]bool
	agentsFile string

	sessionsMutex sync.RWMutex
	sessions      map[int64]*Session

	engineFactory EngineFactory
	cfg           Config
}

type Config struct {
	Token          string
	WebhookURL     string
	WebhookAddress string
	AllowedUsers   []int64
	AgentsFile     string
	AgentName      string
	ProviderID     string
	UnsafeAllTools bool
}

func NewServer(cfg Config) (*Server, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot api: %w", err)
	}

	wl := make(map[int64]bool)
	for _, id := range cfg.AllowedUsers {
		wl[id] = true
	}

	return &Server{
		bot:        bot,
		whitelist:  wl,
		agentsFile: cfg.AgentsFile,
		sessions:   make(map[int64]*Session),
		engineFactory: func(ctx context.Context, agentName, providerID, agentsFilePath string, hitl llm.HITLHandler, unsafeAllTools bool) (Streamer, *config.InstanceConfig, func(), error) {
			return ui.NewEngine(ctx, agentName, providerID, agentsFilePath, hitl, unsafeAllTools)
		},
		cfg: cfg,
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	slog.Info("Starting Telegram Bot", "user", s.bot.Self.UserName)

	var updates tgbotapi.UpdatesChannel

	// Note: We leave Webhook implementation for the handler below, or if webhooks were pre-configured
	// For simplicity in CLI binding, typical webhook usage with tgbotapi starts an http listener
	// and creates a webhook URL. If webhook url is empty, we fall back to polling.
	
	// This function starts polling by default, or you can route HTTP webhooks to the updates channel externally.
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates = s.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			s.bot.StopReceivingUpdates()
			return ctx.Err()
		case update := <-updates:
			s.processUpdate(update)
		}
	}
}

// StartWebhook starts the webhook server
func (s *Server) StartWebhook(ctx context.Context, webhookURL, address string) error {
	wh, _ := tgbotapi.NewWebhook(webhookURL)
	_, err := s.bot.Request(wh)
	if err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	info, err := s.bot.GetWebhookInfo()
	if err != nil {
		return fmt.Errorf("failed to get webhook info: %w", err)
	}

	if info.LastErrorDate != 0 {
		slog.Warn("Telegram callback failed previously", "error", info.LastErrorMessage)
	}

	updates := s.bot.ListenForWebhook("/")
	
	go func() {
		if err := http.ListenAndServe(address, nil); err != nil {
			slog.Error("Webhook HTTP Server failed", "error", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updates:
			s.processUpdate(update)
		}
	}
}

func (s *Server) processUpdate(update tgbotapi.Update) {
	var chatID int64

	if update.Message != nil {
		chatID = update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		chatID = update.CallbackQuery.Message.Chat.ID
	} else {
		return // Ignore other updates for now
	}

	if len(s.whitelist) > 0 && !s.whitelist[chatID] {
		slog.Warn("Received update from unauthorized user", "chat_id", chatID)
		if update.Message != nil {
			msg := tgbotapi.NewMessage(chatID, "Unauthorized.")
			_, _ = s.bot.Send(msg)
		}
		return
	}

	sess := s.getSession(chatID)

	if update.CallbackQuery != nil {
		go func() {
			err := sess.HITL.Resolve(update.CallbackQuery)
			if err != nil {
				slog.Error("failed to resolve callback query", "error", err)
			}
		}()
		return
	}

	select {
	case sess.Queue <- update:
		// Queued successfully
	default:
		slog.Warn("Session queue full", "chat_id", chatID)
		if update.Message != nil {
			msg := tgbotapi.NewMessage(chatID, "Too many messages queued. Please wait.")
			_, _ = s.bot.Send(msg)
		}
	}
}

func (s *Server) getSession(chatID int64) *Session {
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	if sess, ok := s.sessions[chatID]; ok {
		return sess
	}

	hitl := NewTelegramHITL(s.bot, chatID)
	baseCtx, cancel := context.WithCancel(context.Background())
	ctx := llm.WithSessionID(baseCtx, fmt.Sprintf("telegram-%d", chatID))
	sess := &Session{
		ChatID: chatID,
		Queue:  make(chan tgbotapi.Update, 10),
		HITL:   hitl,
		Ctx:    ctx,
		Cancel: cancel,
	}
	s.sessions[chatID] = sess

	go s.sessionWorker(sess)
	return sess
}

func (s *Server) sessionWorker(sess *Session) {
	for {
		select {
		case <-sess.Ctx.Done():
			return
		case update := <-sess.Queue:
			if update.Message != nil && update.Message.Text != "" {
				s.handleMessage(sess, update.Message)
			}
		}
	}
}

func (s *Server) handleMessage(sess *Session, msg *tgbotapi.Message) {
	if sess.Engine == nil {
		path := s.agentsFile
		if path == "" {
			p, _ := xdg.ConfigFile("dux/agents.yaml")
			path = p
		}
		// Fallback to default provider/agent rules: just pass empty strings and let config system load defaults
		eng, _, cleanup, err := s.engineFactory(sess.Ctx, s.cfg.AgentName, s.cfg.ProviderID, path, sess.HITL, s.cfg.UnsafeAllTools)
		if err != nil {
			slog.Error("failed to initialize telegram engine", "error", err)
			out := tgbotapi.NewMessage(sess.ChatID, fmt.Sprintf("Initialization error: %v", err))
			_, _ = s.bot.Send(out)
			return
		}
		sess.Engine = eng
		sess.CleanupOpts = append(sess.CleanupOpts, cleanup)
	}

	st := NewStreamTracker(s.bot, sess.ChatID)
	stopWorker := st.StartWorker(sess.Ctx)
	defer stopWorker()

	session := &pkgui.ChatSession{
		ID:      fmt.Sprintf("telegram-%d", sess.ChatID),
		Engine:  sess.Engine,
		View:    st,
	}

	if err := session.StreamQuery(sess.Ctx, msg.Text); err != nil {
		out := tgbotapi.NewMessage(sess.ChatID, fmt.Sprintf("Error during stream execution: %v", err))
		_, _ = s.bot.Send(out)
	}
	st.Flush()
}
