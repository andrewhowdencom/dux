package web

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/internal/ui"
	"github.com/andrewhowdencom/dux/internal/ui/web/frontend"
	"github.com/andrewhowdencom/dux/pkg/llm"
	pkgui "github.com/andrewhowdencom/dux/pkg/ui"
	"github.com/google/uuid"
)

// Streamer interface abstracts the physical Engine for testing
type Streamer interface {
	Stream(ctx context.Context, inputMessage llm.Message) (<-chan llm.Message, error)
}

type EngineFactory func(ctx context.Context, agentName string, providerID string, agentsFilePath string, hitl llm.HITLHandler, unsafeAllTools bool) (Streamer, *config.InstanceConfig, func(), error)

type Server struct {
	agentsDir     string
	agentName     string
	providerID    string
	hitl          *WebHITL
	engineFactory EngineFactory
	sessionKey    []byte
}

// NewMux creates a new HTTP serve mux for the UI.
func NewMux(agentsDir string, agentName string, providerID string) *http.ServeMux {
	key, err := getOrCreateSessionKey()
	if err != nil {
		slog.Error("failed to load persisten session key, generating volatile key", "err", err)
		key = make([]byte, 32)
		_, _ = io.ReadFull(rand.Reader, key)
	}

	mux := http.NewServeMux()
	srv := &Server{
		agentsDir:  agentsDir,
		agentName:  agentName,
		providerID: providerID,
		hitl:       NewWebHITL(),
		engineFactory: func(ctx context.Context, agentName, providerID, agentsFilePath string, hitl llm.HITLHandler, unsafeAllTools bool) (Streamer, *config.InstanceConfig, func(), error) {
			return ui.NewEngine(ctx, agentName, providerID, agentsFilePath, hitl, unsafeAllTools)
		},
		sessionKey: key,
	}

	mux.HandleFunc("/api/session", srv.handleSession)
	mux.HandleFunc("/api/chat", srv.handleChat)
	mux.HandleFunc("/api/chat/approve", srv.handleApprove)

	// Mount the frontend fs
	mux.Handle("/", http.FileServer(http.FS(frontend.FS)))

	return mux
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
		Prompt string `json:"prompt"`
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
	path := s.agentsDir
	engine, _, cleanup, err := s.engineFactory(ctx, s.agentName, s.providerID, path, s.hitl, false)
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

	view := &WebView{
		encoder: encoder,
		rc:      rc,
	}

	session := &pkgui.ChatSession{
		ID:      sessionID,
		Engine:  engine,
		View:    view,
	}

	if err := session.StreamQuery(ctx, payload.Prompt); err != nil {
		slog.Error("error during chat engine stream", "err", err)
	}
	slog.Debug("chat engine stream completed successfully")
}

// WebView implements the different UI Extension features.
type WebView struct {
	encoder *json.Encoder
	rc      *http.ResponseController
}

func (v *WebView) RenderTextChunk(chunk string) {
	if err := v.encoder.Encode(map[string]any{"type": "text", "content": chunk}); err != nil {
		slog.Error("ENCODE ERROR", "err", err)
	}
}

func (v *WebView) RenderError(err error) {
	if err := v.encoder.Encode(map[string]any{"type": "error", "error": err.Error()}); err != nil {
		slog.Error("ENCODE ERROR", "err", err)
	}
}

func (v *WebView) PromptHITL(req *llm.ToolRequestPart) {
	err := v.encoder.Encode(map[string]any{
		"type":    "hitl_request",
		"call_id": req.ToolID,
		"tool":    req.Name,
		"args":    req.Args,
	})
	if err != nil {
		slog.Error("ENCODE ERROR", "err", err)
	}
}

func (v *WebView) Flush() {
	if err := v.rc.Flush(); err != nil {
		slog.Error("FLUSH ERROR", "err", err)
	}
}

func (v *WebView) RenderThinkingChunk(chunk string) {
	if err := v.encoder.Encode(map[string]any{"type": "thinking", "content": chunk}); err != nil {
		slog.Error("ENCODE ERROR", "err", err)
	}
}

func (v *WebView) RenderToolIntent(toolName string, args any) {
	// The Web frontend currently handles the hitl_request separately.
}

func (v *WebView) RenderToolResult(toolName string, result any, isError bool) {
	err := v.encoder.Encode(map[string]any{
		"type":     "tool_result",
		"tool":     toolName,
		"result":   fmt.Sprintf("%v", result),
		"is_error": isError,
	})
	if err != nil {
		slog.Error("ENCODE ERROR", "err", err)
	}
}

func (v *WebView) RenderTelemetry(telemetry llm.TelemetryPart) {
	err := v.encoder.Encode(map[string]any{
		"type":             "telemetry",
		"input_tokens":     telemetry.InputTokens,
		"output_tokens":    telemetry.OutputTokens,
		"reasoning_tokens": telemetry.ReasoningTokens,
		"duration_secs":    telemetry.Duration.Seconds(),
	})
	if err != nil {
		slog.Error("ENCODE ERROR", "err", err)
	}
}

func (v *WebView) OnCommand(cmd string, args []string) {
	if cmd == "/new" {
		err := v.encoder.Encode(map[string]any{
			"type":    "command",
			"command": cmd,
			"args":    args,
		})
		if err != nil {
			slog.Error("ENCODE ERROR", "err", err)
		}
	}
}
