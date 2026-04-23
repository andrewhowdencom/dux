package adapter

import (
	"context"
	"sort"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/enrich"
	"github.com/andrewhowdencom/dux/pkg/llm/provider"
	"github.com/andrewhowdencom/dux/pkg/memory/working"
)

// Engine orchestrates the convergence loop between the LLM provider,
// tools, and conversation history.
type Engine struct {
	provider   provider.Generator
	history    llm.History
	resolvers  []llm.ToolProvider

	beforeStart    []llm.BeforeStartHook
	beforeGenerate []llm.BeforeGenerateHook
	beforeTool     []llm.BeforeToolHook
	afterTool      []llm.AfterToolHook
	afterComplete  []llm.AfterCompleteHook
}

// Option configures the Engine via the functional options pattern.
type Option func(*Engine)

// WithProvider sets the core LLM inference provider.
func WithProvider(p provider.Generator) Option {
	return func(e *Engine) {
		e.provider = p
	}
}

// WithHistory sets the engine's history backend.
func WithHistory(h llm.History) Option {
	return func(e *Engine) {
		e.history = h
	}
}

// WithWorkingMemory sets the engine's working memory backend.
// It automatically registers a BeforeGenerate hook to inject conversation history.
func WithWorkingMemory(mem working.WorkingMemory) Option {
	return func(e *Engine) {
		e.history = mem
		e.beforeGenerate = append(e.beforeGenerate, working.NewHistoryHook(mem))
	}
}

// WithSystemPrompt sets an overarching system prompt injected dynamically at stream time.
func WithSystemPrompt(prompt string) Option {
	return func(e *Engine) {
		e.beforeGenerate = append(e.beforeGenerate, enrich.NewPrompt(prompt))
	}
}

// WithEnrichers sets the dynamic context enrichers to be evaluated before streaming.
func WithEnrichers(enrichers []llm.BeforeGenerateHook) Option {
	return func(e *Engine) {
		e.beforeGenerate = append(e.beforeGenerate, enrichers...)
	}
}

// WithResolver adds a dynamic tool resolution strategy.
func WithResolver(r llm.ToolProvider) Option {
	return func(e *Engine) {
		e.resolvers = append(e.resolvers, r)
	}
}

// WithBeforeStart registers a hook that fires before the session starts.
func WithBeforeStart(h llm.BeforeStartHook) Option {
	return func(e *Engine) {
		e.beforeStart = append(e.beforeStart, h)
	}
}

// WithBeforeGenerate registers a hook that fires before each LLM call.
func WithBeforeGenerate(h llm.BeforeGenerateHook) Option {
	return func(e *Engine) {
		e.beforeGenerate = append(e.beforeGenerate, h)
	}
}

// WithBeforeTool registers a hook that fires before each tool execution.
func WithBeforeTool(h llm.BeforeToolHook) Option {
	return func(e *Engine) {
		e.beforeTool = append(e.beforeTool, h)
	}
}

// WithAfterTool registers a hook that fires after each tool execution.
func WithAfterTool(h llm.AfterToolHook) Option {
	return func(e *Engine) {
		e.afterTool = append(e.afterTool, h)
	}
}

// WithAfterComplete registers a hook that fires when the session completes.
func WithAfterComplete(h llm.AfterCompleteHook) Option {
	return func(e *Engine) {
		e.afterComplete = append(e.afterComplete, h)
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

// UpdateOptions allows hot-swapping configuration elements (like working memory
// and active tools) securely between mode transitions.
func (e *Engine) UpdateOptions(opts ...Option) {
	for _, opt := range opts {
		opt(e)
	}
}

// Stream executes the recursive convergence loop natively incorporating tools and hooks.
func (e *Engine) Stream(ctx context.Context, inputMessage llm.Message) (<-chan llm.Message, error) {
	out := make(chan llm.Message)

	go func() {
		defer close(out)

		sessionID, _ := llm.SessionIDFromContext(ctx)

		// BeforeStart: allow hooks to validate or enrich the session
		if len(e.beforeStart) > 0 {
			for _, h := range e.beforeStart {
				if err := h(ctx, llm.BeforeStartRequest{
					SessionID:  sessionID,
					InitialMsg: inputMessage,
				}); err != nil {
					e.sendError(ctx, out, err, sessionID)
					return
				}
			}
		}

		// Seed the initial user message into history
		if e.history != nil {
			if err := e.history.Append(ctx, sessionID, inputMessage); err != nil {
				e.sendError(ctx, out, err, sessionID)
				return
			}
		}

		// Initial recursive loop trigger (no tool results yet)
		e.recursiveStream(ctx, sessionID, inputMessage, out, nil, nil)
	}()

	return out, nil
}

// recursiveStream fetches history, injects definitions, streams from the provider, handles tools, and restarts if necessary.
func (e *Engine) recursiveStream(ctx context.Context, sessionID string, initialInput llm.Message, out chan<- llm.Message, pendingResults []llm.ToolResultPart, toolHistory []llm.ToolExecutionRecord) {
	// Build prompt messages via BeforeGenerate hooks
	msgs, err := e.buildPromptMessages(ctx, sessionID, pendingResults)
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
	var accumulatedParts []llm.Part

	// Stream all parts to the client synchronously, but accumulate for history
	for part := range partStream {
		msg := llm.Message{
			SessionID: sessionID,
			Identity:  llm.Identity{Role: "assistant"},
			Parts:     []llm.Part{part},
		}

		// Accumulate for history
		accumulatedParts = append(accumulatedParts, part)

		e.safeSend(ctx, out, msg)

		if tr, ok := part.(llm.ToolRequestPart); ok {
			pendingCalls = append(pendingCalls, tr)
		}
	}

	// Append assistant message to history
	if e.history != nil && len(accumulatedParts) > 0 {
		bundledMsg := llm.Message{
			SessionID: sessionID,
			Identity:  llm.Identity{Role: "assistant"},
			Parts:     accumulatedParts,
		}
		if err := e.history.Append(ctx, sessionID, bundledMsg); err != nil {
			e.sendError(ctx, out, err, sessionID)
			return
		}
	}

	// Executed after the provider closes its stream slice
	if len(pendingCalls) == 0 {
		// No more tool calls — session is complete
		finalMsg := llm.Message{
			SessionID: sessionID,
			Identity:  llm.Identity{Role: "assistant"},
			Parts:     accumulatedParts,
		}
		if len(e.afterComplete) > 0 {
			for _, h := range e.afterComplete {
				if err := h(ctx, llm.AfterCompleteRequest{
					SessionID:    sessionID,
					FinalMessage: finalMsg,
					ToolHistory:  toolHistory,
				}); err != nil {
					e.sendError(ctx, out, err, sessionID)
					return
				}
			}
		}
		return
	}

	var results []llm.ToolResultPart
	var transition *llm.TransitionSignalPart

	for callIndex, tc := range pendingCalls {
		resPart := e.executeTool(ctx, sessionID, tc, callIndex)
		if trans, ok := resPart.(llm.TransitionSignalPart); ok {
			transition = &trans

			// Crucial: The LLM Provider expects a graph constraint where every ToolRequest is followed by a ToolResult.
			// Even though we are breaking the runloop to hot-swap the engine, we MUST inject a pseudo-result
			// bridging the old state so the new Engine does not instantly drop the 'corrupted' history matrix.
			results = append(results, llm.ToolResultPart{
				ToolID: tc.ToolID,
				Name:   tc.Name,
				Result: "State Machine Transition successfully hooked.",
			})
		} else if tr, ok := resPart.(llm.ToolResultPart); ok {
			results = append(results, tr)
		}
	}

	// Build tool message for history
	toolMsg := llm.Message{
		SessionID: sessionID,
		Identity:  llm.Identity{Role: "tool"},
	}
	for _, pr := range results {
		toolMsg.Parts = append(toolMsg.Parts, pr)
	}

	if e.history != nil {
		if err := e.history.Append(ctx, sessionID, toolMsg); err != nil {
			e.sendError(ctx, out, err, sessionID)
			return
		}
	}

	if transition != nil {
		transMsg := llm.Message{
			SessionID: sessionID,
			Identity:  llm.Identity{Role: "system"},
			Parts:     []llm.Part{*transition},
		}
		e.safeSend(ctx, out, transMsg)
		return // gracefully break the recursion loop
	}

	e.safeSend(ctx, out, toolMsg)

	// Update tool history and recurse
	newHistory := toolHistory
	for i, tc := range pendingCalls {
		newHistory = append(newHistory, llm.ToolExecutionRecord{
			ToolCall: tc,
			Result:   results[i],
		})
	}

	e.recursiveStream(ctx, sessionID, initialInput, out, results, newHistory)
}

// buildPromptMessages runs BeforeGenerate hooks serially, then injects tool
// definitions, and finally sorts by Volatility.
func (e *Engine) buildPromptMessages(ctx context.Context, sessionID string, pendingResults []llm.ToolResultPart) ([]llm.Message, error) {
	req := llm.BeforeGenerateRequest{
		SessionID:      sessionID,
		PendingResults: pendingResults,
	}

	// Run BeforeGenerate hooks serially; each may mutate CurrentMessages
	for _, h := range e.beforeGenerate {
		if err := h(ctx, &req); err != nil {
			return nil, err
		}
	}

	// Inject tool definitions from resolvers
	if len(e.resolvers) > 0 {
		var toolParts []llm.Part
		for _, r := range e.resolvers {
			// This is a simplified approach: we need to iterate over all tools in each resolver.
			// However, ToolProvider only has GetTool(name). We need a way to list tools.
			// For now, we'll use a helper that collects definitions.
			defs := collectToolDefinitions(r)
			for _, def := range defs {
				toolParts = append(toolParts, def)
			}
		}
		if len(toolParts) > 0 {
			req.CurrentMessages = append(req.CurrentMessages, llm.Message{
				Identity:   llm.Identity{Role: "system"},
				Parts:      toolParts,
				Volatility: llm.VolatilityHigh,
			})
		}
	}

	// Sort by Volatility Ascending (Static=0 first, High=10 last)
	sort.SliceStable(req.CurrentMessages, func(i, j int) bool {
		return req.CurrentMessages[i].Volatility < req.CurrentMessages[j].Volatility
	})

	return req.CurrentMessages, nil
}

// collectToolDefinitions gathers all tool definitions from a ToolProvider.
func collectToolDefinitions(r llm.ToolProvider) []llm.ToolDefinitionPart {
	var defs []llm.ToolDefinitionPart
	for _, t := range r.Tools() {
		defs = append(defs, t.Definition())
	}
	return defs
}

func (e *Engine) executeTool(ctx context.Context, sessionID string, req llm.ToolRequestPart, callIndex int) llm.Part {
	resultPart := llm.ToolResultPart{
		ToolID: req.ToolID,
		Name:   req.Name,
	}

	// BeforeTool hooks
	if len(e.beforeTool) > 0 {
		for _, h := range e.beforeTool {
			if err := h(ctx, llm.BeforeToolRequest{
				SessionID: sessionID,
				ToolCall:  req,
				CallIndex: callIndex,
			}); err != nil {
				resultPart.Result = err.Error()
				resultPart.IsError = true

				// AfterTool: notify observers of the blocked tool
				if len(e.afterTool) > 0 {
					for _, ah := range e.afterTool {
						_ = ah(ctx, llm.AfterToolRequest{
							SessionID: sessionID,
							ToolCall:  req,
							Result:    resultPart,
							Error:     err,
						})
					}
				}
				return resultPart
			}
		}
	}

	var targetTool llm.Tool
	var namespace string
	for _, r := range e.resolvers {
		if t, found := r.GetTool(req.Name); found {
			targetTool = t
			namespace = r.Namespace()
			break
		}
	}

	if targetTool == nil {
		resultPart.Result = "Error: Tool not found"
		resultPart.IsError = true

		if len(e.afterTool) > 0 {
			for _, h := range e.afterTool {
				_ = h(ctx, llm.AfterToolRequest{
					SessionID: sessionID,
					ToolCall:  req,
					Result:    resultPart,
				})
			}
		}
		return resultPart
	}

	evalCtx := context.WithValue(ctx, llm.ContextKeyNamespace, namespace)

	start := time.Now()
	res, err := targetTool.Execute(evalCtx, req.Args)
	duration := time.Since(start)

	if err != nil {
		resultPart.Result = err.Error()
		resultPart.IsError = true
	} else {
		if transitionObj, ok := res.(llm.TransitionSignalPart); ok {
			// AfterTool for transitions
			if len(e.afterTool) > 0 {
				for _, h := range e.afterTool {
					_ = h(ctx, llm.AfterToolRequest{
						SessionID: sessionID,
						ToolCall:  req,
						Result:    resultPart,
						Duration:  duration,
					})
				}
			}
			return transitionObj
		}
		if transitionPtr, ok := res.(*llm.TransitionSignalPart); ok {
			if len(e.afterTool) > 0 {
				for _, h := range e.afterTool {
					_ = h(ctx, llm.AfterToolRequest{
						SessionID: sessionID,
						ToolCall:  req,
						Result:    resultPart,
						Duration:  duration,
					})
				}
			}
			return *transitionPtr
		}
		resultPart.Result = res
	}

	// AfterTool hooks
	if len(e.afterTool) > 0 {
		for _, h := range e.afterTool {
			_ = h(ctx, llm.AfterToolRequest{
				SessionID: sessionID,
				ToolCall:  req,
				Result:    resultPart,
				Duration:  duration,
				Error:     err,
			})
		}
	}

	return resultPart
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
