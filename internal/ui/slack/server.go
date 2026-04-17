package slack

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/internal/ui"
	"github.com/andrewhowdencom/dux/pkg/llm"
	pkgui "github.com/andrewhowdencom/dux/pkg/ui"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type Streamer interface {
	Stream(ctx context.Context, inputMessage llm.Message) (<-chan llm.Message, error)
}

type EngineFactory func(ctx context.Context, agentName string, providerID string, agentsFilePath string, hitl llm.HITLHandler, unsafeAllTools bool) (Streamer, *config.InstanceConfig, func(), error)

type Config struct {
	AppToken       string
	BotToken       string
	SigningSecret  string
	WebhookURL     string
	WebhookAddress string
	ReplyMode      string
	AgentsFile     string
	AgentName      string
	ProviderID     string
	UnsafeAllTools bool
}

type Session struct {
	ID          string // Usually thread_ts or channel ID
	ChannelID   string
	ThreadTS    string
	Engine      Streamer
	CleanupOpts []func()
	Queue       chan string // messages
	HITL        *SlackHITL
	Ctx         context.Context
	Cancel      context.CancelFunc
}

type Server struct {
	api            *slack.Client
	socketClient   *socketmode.Client
	sessionsMutex  sync.RWMutex
	sessions       map[string]*Session
	engineFactory  EngineFactory
	cfg            Config
	botID          string
	toolDisplayCfg config.ToolDisplayConfig
}

func NewServer(cfg Config) (*Server, error) {
	var api *slack.Client
	var socketClient *socketmode.Client

	if cfg.AppToken != "" {
		// Socket Mode
		if !strings.HasPrefix(cfg.AppToken, "xapp-") {
			return nil, fmt.Errorf("app_token must start with xapp-")
		}
		api = slack.New(
			cfg.BotToken,
			slack.OptionDebug(false),
			slack.OptionAppLevelToken(cfg.AppToken),
		)
		socketClient = socketmode.New(
			api,
			socketmode.OptionDebug(false),
		)
	} else {
		// Webhook Mode
		api = slack.New(cfg.BotToken, slack.OptionDebug(false))
	}

	return &Server{
		api:          api,
		socketClient: socketClient,
		sessions:     make(map[string]*Session),
		cfg:          cfg,
		engineFactory: func(ctx context.Context, agentName, providerID, agentsFilePath string, hitl llm.HITLHandler, unsafeAllTools bool) (Streamer, *config.InstanceConfig, func(), error) {
			engine, cfg, _, cleanup, err := ui.NewEngine(ctx, agentName, providerID, agentsFilePath, hitl, unsafeAllTools)
			return engine, cfg, cleanup, err
		},
		toolDisplayCfg: config.LoadToolDisplayConfig(),
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	res, err := s.api.AuthTest()
	if err != nil {
		return fmt.Errorf("failed to auth test slack bot: %w", err)
	}
	s.botID = res.UserID
	slog.Info("Successfully authenticated Slack Bot", "bot_id", s.botID, "user", res.User)

	if s.socketClient != nil {
		return s.startSocketMode(ctx)
	}
	return s.startWebhook(ctx)
}

func (s *Server) startSocketMode(ctx context.Context) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-s.socketClient.Events:
				switch evt.Type {
				case socketmode.EventTypeEventsAPI:
					_ = s.socketClient.Ack(*evt.Request)
					eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
					if !ok {
						continue
					}
					s.handleEventAPI(eventsAPIEvent)

				case socketmode.EventTypeSlashCommand:
					_ = s.socketClient.Ack(*evt.Request)
					cmd, ok := evt.Data.(slack.SlashCommand)
					if !ok {
						continue
					}
					s.handleSlashCommand(cmd)

				case socketmode.EventTypeInteractive:
					_ = s.socketClient.Ack(*evt.Request)
					interaction, ok := evt.Data.(slack.InteractionCallback)
					if !ok {
						continue
					}
					s.handleInteractive(interaction)
				}
			}
		}
	}()

	slog.Info("Starting Slack Bot in Socket Mode")
	if err := s.socketClient.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return nil
}

// TODO: Start webhook
func (s *Server) startWebhook(ctx context.Context) error {
	// Simple mux for webhook testing
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.webhookHandler)

	addr := s.cfg.WebhookAddress
	if addr == "" {
		addr = ":8443"
	}
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Slack Webhook Server failed", "error", err)
		}
	}()
	slog.Info("Starting Slack Bot in Webhook Mode", "address", addr)

	<-ctx.Done()
	return srv.Shutdown(context.Background())
}

func (s *Server) webhookHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: proper signature verification if signing secret provided
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleEventAPI(evt slackevents.EventsAPIEvent) {
	if evt.Type == slackevents.URLVerification {
		// Need proper returning of challenge if webhook, but for socket mode handled automatically by library usually
		return
	}
	if evt.Type == slackevents.CallbackEvent {
		innerEvent := evt.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			s.routeMessage(ev.Channel, ev.ThreadTimeStamp, ev.TimeStamp, ev.User, ev.Text, true)
		case *slackevents.MessageEvent:
			if ev.User == s.botID || ev.BotID != "" {
				return // Don't reply to self or other bots
			}
			isMentioned := strings.Contains(ev.Text, fmt.Sprintf("<@%s>", s.botID))
			isDM := strings.HasPrefix(ev.Channel, "D")
			shouldReply := isDM || (s.cfg.ReplyMode == "always") || isMentioned
			if shouldReply {
				s.routeMessage(ev.Channel, ev.ThreadTimeStamp, ev.TimeStamp, ev.User, ev.Text, isMentioned)
			}
		}
	}
}

func (s *Server) handleSlashCommand(cmd slack.SlashCommand) {
	// A slash command generates a new invisible message in the chat
	// Typically /new to reset
	// Route pseudo-message
	s.routeMessage(cmd.ChannelID, "", "", cmd.UserID, cmd.Command+" "+cmd.Text, true)
}

func (s *Server) handleInteractive(interaction slack.InteractionCallback) {
	if interaction.Type == slack.InteractionTypeBlockActions {
		sessID := interaction.Channel.ID
		if interaction.Message.ThreadTimestamp != "" {
			sessID = interaction.Channel.ID + "-" + interaction.Message.ThreadTimestamp
		}

		s.sessionsMutex.RLock()
		sess, ok := s.sessions[sessID]
		s.sessionsMutex.RUnlock()
		if ok {
			go func() {
				err := sess.HITL.Resolve(interaction)
				if err != nil {
					slog.Error("failed to resolve callback query", "error", err)
				}
			}()
		}
	}
}

func (s *Server) routeMessage(channelID, threadTS, msgTS, userID, text string, isMentioned bool) {
	// Cleanup mention from text
	text = strings.ReplaceAll(text, fmt.Sprintf("<@%s>", s.botID), "")
	text = strings.TrimSpace(text)

	if threadTS == "" {
		threadTS = msgTS // Start a new thread or use the message TS as thread root natively
	}

	sessID := channelID + "-" + threadTS

	sess := s.getSession(sessID, channelID, threadTS)

	select {
	case sess.Queue <- text:
	default:
		slog.Warn("Session queue full", "session_id", sessID)
		_, _, _ = s.api.PostMessage(channelID, slack.MsgOptionText("Too many messages queued. Please wait.", false), slack.MsgOptionTS(threadTS))
	}
}

func (s *Server) getSession(sessID, channelID, threadTS string) *Session {
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	if sess, ok := s.sessions[sessID]; ok {
		return sess
	}

	hitl := NewSlackHITL(s.api, channelID, threadTS, s.cfg.AgentName)
	baseCtx, cancel := context.WithCancel(context.Background())
	ctx := llm.WithSessionID(baseCtx, "slack-"+sessID)

	sess := &Session{
		ID:        sessID,
		ChannelID: channelID,
		ThreadTS:  threadTS,
		Queue:     make(chan string, 10),
		HITL:      hitl,
		Ctx:       ctx,
		Cancel:    cancel,
	}
	s.sessions[sessID] = sess

	go s.sessionWorker(sess)
	return sess
}

func (s *Server) sessionWorker(sess *Session) {
	for {
		select {
		case <-sess.Ctx.Done():
			return
		case text := <-sess.Queue:
			if text != "" {
				s.handleMessage(sess, text)
			}
		}
	}
}

func (s *Server) handleMessage(sess *Session, text string) {
	if sess.Engine == nil {
		path := s.cfg.AgentsFile
		eng, _, cleanup, err := s.engineFactory(sess.Ctx, s.cfg.AgentName, s.cfg.ProviderID, path, sess.HITL, s.cfg.UnsafeAllTools)
		if err != nil {
			slog.Error("failed to initialize slack engine", "error", err)
			_, _, _ = s.api.PostMessage(sess.ChannelID, slack.MsgOptionText(fmt.Sprintf("Initialization error: %v", err), false), slack.MsgOptionTS(sess.ThreadTS))
			return
		}
		sess.Engine = eng
		sess.CleanupOpts = append(sess.CleanupOpts, cleanup)
	}

	toolConfigs := make(map[string]pkgui.ToolDisplayConfig)
	for name, cfg := range s.toolDisplayCfg.Tools {
		toolConfigs[name] = pkgui.ToolDisplayConfig{
			Icon:         cfg.Icon,
			HideArgs:     cfg.HideArgs,
			HideResult:   cfg.HideResult,
			MaxResultLen: cfg.MaxResultLen,
		}
	}
	formatter := pkgui.NewDefaultToolFormatter(toolConfigs, s.toolDisplayCfg.DefaultIcon)

	st := NewStreamTracker(s.api, sess.ChannelID, sess.ThreadTS, s.cfg.AgentName, WithToolFormatter(formatter))
	st.OnReset = func() {
		s.sessionsMutex.Lock()
		delete(s.sessions, sess.ID)
		s.sessionsMutex.Unlock()
		sess.Cancel()
	}

	stopWorker := st.StartWorker(sess.Ctx)
	defer stopWorker()

	session := &pkgui.ChatSession{
		ID:     "slack-" + sess.ID,
		Engine: sess.Engine,
		View:   st,
	}

	if err := session.StreamQuery(sess.Ctx, text); err != nil {
		_, _, _ = s.api.PostMessage(sess.ChannelID, slack.MsgOptionText(fmt.Sprintf("Error during stream execution: %v", err), false), slack.MsgOptionTS(sess.ThreadTS))
	}
	st.Flush()
}
