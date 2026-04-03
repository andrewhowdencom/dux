package adapter

import (
	"context"
	"log/slog"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/enrich"
	"github.com/andrewhowdencom/dux/pkg/llm/history"
	"github.com/andrewhowdencom/dux/pkg/llm/provider"
)

// Engine orchestrates the convergence loop between the LLM provider,
// tools, and conversation history.
type Engine struct {
	history      history.History
	provider     provider.ChatGenerator
	systemPrompt string
	enrichers    []enrich.Enricher
	resolvers    []llm.ToolResolver
	middlewares  []llm.ToolMiddleware
}

// Option configures the Engine via the functional options pattern.
type Option func(*Engine)

// WithHistory sets the engine's history backend.
func WithHistory(h history.History) Option {
	return func(e *Engine) {
		e.history = h
	}
}

// WithProvider sets the core LLM inference provider.
func WithProvider(p provider.ChatGenerator) Option {
	return func(e *Engine) {
		e.provider = p
	}
}

// WithSystemPrompt sets an overarching system prompt injected dynamically at stream time.
func WithSystemPrompt(prompt string) Option {
	return func(e *Engine) {
		e.systemPrompt = prompt
	}
}

// WithEnrichers sets the dynamic context enrichers to be evaluated before streaming.
func WithEnrichers(enrichers []enrich.Enricher) Option {
	return func(e *Engine) {
		e.enrichers = enrichers
	}
}

// WithResolver adds a dynamic tool resolution strategy.
func WithResolver(r llm.ToolResolver) Option {
	return func(e *Engine) {
		e.resolvers = append(e.resolvers, r)
	}
}

// WithToolMiddleware inserts an interceptor into the tool execution chain.
func WithToolMiddleware(mw llm.ToolMiddleware) Option {
	return func(e *Engine) {
		e.middlewares = append(e.middlewares, mw)
	}
}

// New creates a new Engine configured with the provided options.
func New(opts ...Option) *Engine {
	e := &Engine{}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Stream executes the recursive convergence loop natively incorporating tools and middleware constraints.
func (e *Engine) Stream(ctx context.Context, inputMessage llm.Message) (<-chan llm.Message, error) {
	out := make(chan llm.Message)

	// Append initial user input to history (Only once!)
	if e.history != nil {
		if err := e.history.Append(ctx, inputMessage.SessionID, inputMessage); err != nil {
			return nil, err
		}
	}

	go func() {
		defer close(out)

		tools, err := e.resolveTools(ctx)
		if err != nil {
			e.sendError(ctx, out, err, inputMessage.SessionID)
			return
		}

		// Initial recursive loop trigger (no tool results yet)
		e.recursiveStream(ctx, inputMessage.SessionID, out, tools, nil)
	}()

	return out, nil
}

// recursiveStream fetches history, injects definitions, streams from the provider, handles tools, and restarts if necessary.
func (e *Engine) recursiveStream(ctx context.Context, sessionID string, out chan<- llm.Message, tools []llm.Tool, pendingResults []llm.ToolResultPart) {
	// If we were called recursively with tool results, write them to history before querying the model again
	if len(pendingResults) > 0 {
		var parts []llm.Part
		for _, pr := range pendingResults {
			parts = append(parts, pr)
		}
		toolMsg := llm.Message{
			SessionID: sessionID,
			Identity:  llm.Identity{Role: "tool"},
			Parts:     parts,
		}
		if e.history != nil {
			if err := e.history.Append(ctx, sessionID, toolMsg); err != nil {
				e.sendError(ctx, out, err, sessionID)
				return
			}
		}
		e.safeSend(ctx, out, toolMsg) 
	}

	msgs, err := e.buildPromptMessages(ctx, sessionID, tools)
	if err != nil {
		e.sendError(ctx, out, err, sessionID)
		return
	}

	if e.provider == nil {
		return
	}

	partStream, err := e.provider.GenerateStream(ctx, msgs)
	if err != nil {
		e.sendError(ctx, out, err, sessionID)
		return
	}

	var pendingCalls []llm.ToolRequestPart

	// Stream all parts to the client and history
	for part := range partStream {
		msg := llm.Message{
			SessionID: sessionID,
			Identity:  llm.Identity{Role: "assistant"},
			Parts:     []llm.Part{part},
		}
		if e.history != nil {
			if err := e.history.Append(ctx, msg.SessionID, msg); err != nil {
				e.sendError(ctx, out, err, msg.SessionID)
				return
			}
		}
		e.safeSend(ctx, out, msg)

		if tr, ok := part.(llm.ToolRequestPart); ok {
			pendingCalls = append(pendingCalls, tr)
		}
	}

	// Executed after the provider closes its stream slice
	if len(pendingCalls) > 0 {
		var results []llm.ToolResultPart
		for _, tc := range pendingCalls {
			result := e.executeToolWithMiddleware(ctx, tc, tools)
			results = append(results, result)
		}

		// Re-enter the loop with the executed results
		e.recursiveStream(ctx, sessionID, out, tools, results)
	}
}

// buildPromptMessages pieces together system prompts, enrichers, tool schemas, and history.
func (e *Engine) buildPromptMessages(ctx context.Context, sessionID string, tools []llm.Tool) ([]llm.Message, error) {
	var msgs []llm.Message
	if e.history != nil {
		var err error
		msgs, err = e.history.GetMessages(ctx, sessionID)
		if err != nil {
			return nil, err
		}
	}

	var systemParts []llm.Part
	if e.systemPrompt != "" {
		systemParts = append(systemParts, llm.TextPart(e.systemPrompt))
	}

	if len(e.enrichers) > 0 {
		var enrichmentData string
		for _, en := range e.enrichers {
			res, err := en.Enrich(ctx)
			if err != nil {
				slog.Debug("failed to evaluate enricher", "type", en.Type(), "error", err)
				continue
			}
			if res != "" {
				enrichmentData += res + "\n"
			}
		}
		if enrichmentData != "" {
			systemParts = append(systemParts, llm.TextPart(enrichmentData))
		}
	}

	// Always inject definitions so providers know what tools are available downstream
	hasToolDefs := false
	for _, m := range msgs {
		for _, part := range m.Parts {
			if part.Type() == llm.TypeToolDefinition {
				hasToolDefs = true
				break
			}
		}
	}
	if !hasToolDefs {
		for _, t := range tools {
			systemParts = append(systemParts, t.Definition())
		}
	}

	if len(systemParts) > 0 {
		systemMsg := llm.Message{
			SessionID: sessionID,
			Identity:  llm.Identity{Role: "system"},
			Parts:     systemParts,
		}
		msgs = append([]llm.Message{systemMsg}, msgs...)
	}

	return msgs, nil
}

func (e *Engine) executeToolWithMiddleware(ctx context.Context, req llm.ToolRequestPart, tools []llm.Tool) llm.ToolResultPart {
	resultPart := llm.ToolResultPart{
		ToolID: req.ToolID,
		Name:   req.Name,
	}

	var targetTool llm.Tool
	for _, t := range tools {
		if t.Name() == req.Name {
			targetTool = t
			break
		}
	}

	if targetTool == nil {
		resultPart.Result = "Error: Tool not found"
		resultPart.IsError = true
		return resultPart
	}

	// Build the middleware execution chain backwards
	executionFunc := func(c context.Context) (interface{}, error) {
		return targetTool.Execute(c, req.Args)
	}

	for i := len(e.middlewares) - 1; i >= 0; i-- {
		mw := e.middlewares[i]
		next := executionFunc
		executionFunc = func(c context.Context) (interface{}, error) {
			return mw(c, req, next)
		}
	}

	res, err := executionFunc(ctx)
	if err != nil {
		resultPart.Result = err.Error()
		resultPart.IsError = true
	} else {
		resultPart.Result = res
	}

	return resultPart
}

func (e *Engine) resolveTools(ctx context.Context) ([]llm.Tool, error) {
	var allTools []llm.Tool
	for _, r := range e.resolvers {
		tools, err := r.Resolve(ctx)
		if err != nil {
			return nil, err
		}
		allTools = append(allTools, tools...)
	}
	return allTools, nil
}


func (e *Engine) sendError(ctx context.Context, out chan<- llm.Message, err error, sessionID string) {
	msg := llm.Message{
		SessionID: sessionID,
		Identity:  llm.Identity{Role: "system"},
		Parts:     []llm.Part{llm.TextPart("Error: " + err.Error())},
	}
	e.safeSend(ctx, out, msg)
}

func (e *Engine) safeSend(ctx context.Context, out chan<- llm.Message, msg llm.Message) {
	select {
	case <-ctx.Done():
	case out <- msg:
	}
}
