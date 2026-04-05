package web

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/adrg/xdg"
	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/internal/ui"
	"github.com/andrewhowdencom/dux/internal/ui/web/frontend"
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/google/uuid"
)

// Streamer interface abstracts the physical Engine for testing
type Streamer interface {
	Stream(ctx context.Context, inputMessage llm.Message) (<-chan llm.Message, error)
}

type EngineFactory func(ctx context.Context, agentName string, providerID string, agentsFilePath string, hitl llm.HITLHandler, unsafeAllTools bool) (Streamer, *config.InstanceConfig, func(), error)

type Server struct {
	agentsFile    string
	hitl          *WebHITL
	engineFactory EngineFactory
	sessionKey    []byte
}

// NewMux creates a new HTTP serve mux for the UI.
func NewMux(agentsFile string) *http.ServeMux {
	key, err := getOrCreateSessionKey()
	if err != nil {
		slog.Error("failed to load persisten session key, generating volatile key", "err", err)
		key = make([]byte, 32)
		_, _ = io.ReadFull(rand.Reader, key)
	}

	mux := http.NewServeMux()
	srv := &Server{
		agentsFile: agentsFile,
		hitl:       NewWebHITL(),
		engineFactory: func(ctx context.Context, agentName, providerID, agentsFilePath string, hitl llm.HITLHandler, unsafeAllTools bool) (Streamer, *config.InstanceConfig, func(), error) {
			return ui.NewEngine(ctx, agentName, providerID, agentsFilePath, hitl, unsafeAllTools)
		},
		sessionKey: key,
	}

	mux.HandleFunc("/api/agents", srv.handleAgents)
	mux.HandleFunc("/api/session", srv.handleSession)
	mux.HandleFunc("/api/chat", srv.handleChat)
	mux.HandleFunc("/api/chat/approve", srv.handleApprove)

	// Mount the frontend fs
	mux.Handle("/", http.FileServer(http.FS(frontend.FS)))

	return mux
}

// handleAgents returns a list of available agents
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := s.agentsFile
	if path == "" {
		p, err := xdg.ConfigFile("dux/agents.yaml")
		if err != nil {
			slog.Error("failed to get xdg config file", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		path = p
	}

	agents, err := config.LoadAgents(path)
	if err != nil {
		slog.Error("failed to load agents", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	providers, err := config.LoadLLMProviders()
	if err != nil {
		slog.Error("failed to load LLM providers", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"Agents":    agents,
		"Providers": providers,
	})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID := uuid.New().String()
	enc, err := encryptSessionID(s.sessionKey, sessionID)
	if err != nil {
		slog.Error("failed to encrypt session ID", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "dux_session",
		Value:    enc,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		CallID  string `json:"call_id"`
		Approve bool   `json:"approve"`
	}

	cookie, err := r.Cookie("dux_session")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	_, err = decryptSessionID(s.sessionKey, cookie.Value)
	if err != nil {
		slog.Info("invalid dux_session cookie provided in approval req", "err", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.hitl.Resolve(payload.CallID, payload.Approve); err != nil {
		slog.Error("failed to resolve hitl", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Agent     string `json:"agent"`
		Provider  string `json:"provider"`
		Prompt    string `json:"prompt"`
	}

	cookie, err := r.Cookie("dux_session")
	if err != nil {
		http.Error(w, "unauthorized: missing session", http.StatusUnauthorized)
		return
	}
	sessionID, err := decryptSessionID(s.sessionKey, cookie.Value)
	if err != nil {
		slog.Info("invalid dux_session cookie during chat stream", "error", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		slog.Error("invalid request body", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx := llm.WithSessionID(r.Context(), sessionID)

	// Create engine for this ephemeral request
	path := s.agentsFile
	if path == "" {
		p, _ := xdg.ConfigFile("dux/agents.yaml")
		path = p
	}
	engine, _, cleanup, err := s.engineFactory(ctx, payload.Agent, payload.Provider, path, s.hitl, false)
	if err != nil {
		slog.Error("failed to initialize engine", "error", err)
		http.Error(w, fmt.Sprintf("failed to initialize engine: %v", err), http.StatusInternalServerError)
		return
	}
	defer cleanup()

	// Set headers for streaming text/event-stream or ndjson
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)

	encoder := json.NewEncoder(w)
	streamChan := make(chan llm.Message)
	errChan := make(chan error, 1)

	go func() {
		msg := llm.Message{
			SessionID: sessionID,
			Identity:  llm.Identity{Role: "user"},
			Parts: []llm.Part{
				llm.TextPart(payload.Prompt),
			},
		}

		out, err := engine.Stream(ctx, msg)
		if err != nil {
			errChan <- err
			return
		}
		for m := range out {
			streamChan <- m
		}
		close(streamChan)
		close(errChan)
	}()

	for msg := range streamChan {
		slog.Info("received msg from streamChan", "parts_count", len(msg.Parts))
		if len(msg.Parts) == 0 {
			continue
		}
		part := msg.Parts[0]
		slog.Debug("streaming part", "type", fmt.Sprintf("%T", part))
		switch p := part.(type) {
		case llm.TextPart:
			err := encoder.Encode(map[string]any{"type": "text", "content": string(p)})
			if err != nil { slog.Error("ENCODE ERROR", "err", err) }
		case llm.ReasoningPart:
			err := encoder.Encode(map[string]any{"type": "thinking", "content": string(p)})
			if err != nil { slog.Error("ENCODE ERROR", "err", err) }
		case llm.ToolRequestPart:
			err := encoder.Encode(map[string]any{
				"type":    "hitl_request",
				"call_id": p.ToolID,
				"tool":    p.Name,
				"args":    p.Args,
			})
			if err != nil { slog.Error("ENCODE ERROR", "err", err) }
		case llm.ToolResultPart:
			err := encoder.Encode(map[string]any{
				"type":     "tool_result",
				"tool":     p.Name,
				"result":   fmt.Sprintf("%v", p.Result),
				"is_error": p.IsError,
			})
			if err != nil { slog.Error("ENCODE ERROR", "err", err) }
		case llm.TelemetryPart:
			err := encoder.Encode(map[string]any{
				"type":             "telemetry",
				"input_tokens":     p.InputTokens,
				"output_tokens":    p.OutputTokens,
				"reasoning_tokens": p.ReasoningTokens,
				"duration_secs":    p.Duration.Seconds(),
			})
			if err != nil { slog.Error("ENCODE ERROR", "err", err) }
		default:
			slog.Info("received unknown part type", "type", fmt.Sprintf("%T", p))
		}
		if err := rc.Flush(); err != nil { slog.Error("FLUSH ERROR", "err", err) }
	}

	if err := <-errChan; err != nil {
		slog.Error("error during chat engine stream", "err", err)
		_ = encoder.Encode(map[string]any{"type": "error", "error": err.Error()})
		if err := rc.Flush(); err != nil { slog.Error("FLUSH ERROR", "err", err) }
	}
	slog.Debug("chat engine stream completed successfully")
}
