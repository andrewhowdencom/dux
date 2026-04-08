package adapter

import (
	"context"
	"sort"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/enrich"
	"github.com/andrewhowdencom/dux/pkg/llm/provider"
)

// Engine orchestrates the convergence loop between the LLM provider,
// tools, and conversation history.
type Engine struct {
	injectors    []llm.Injector
	provider     provider.Generator
	middlewares  []llm.ToolMiddleware
}

// Option configures the Engine via the functional options pattern.
type Option func(*Engine)

func WithInjector(i llm.Injector) Option {
	return func(e *Engine) {
		e.injectors = append(e.injectors, i)
	}
}

// WithWorkingMemory sets the engine's working memory backend.
func WithWorkingMemory(h llm.Injector) Option {
	return WithInjector(h)
}

// WithProvider sets the core LLM inference provider.
func WithProvider(p provider.Generator) Option {
	return func(e *Engine) {
		e.provider = p
	}
}

// WithSystemPrompt sets an overarching system prompt injected dynamically at stream time.
func WithSystemPrompt(prompt string) Option {
	return WithInjector(enrich.NewPrompt(prompt))
}

// WithEnrichers sets the dynamic context enrichers to be evaluated before streaming.
func WithEnrichers(enrichers []llm.Injector) Option {
	return func(e *Engine) {
		e.injectors = append(e.injectors, enrichers...)
	}
}

// WithResolver adds a dynamic tool resolution strategy.
func WithResolver(r llm.ToolProvider) Option {
	return WithInjector(r)
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

	// Since we no longer explicitly track history here, history persistence is up to the caller
	// or must be explicitly captured. For now, we seed the recursion loop with the initial message.

	go func() {
		defer close(out)

		q := llm.InjectQuery{
			Text:      inputMessage.Text(),
		}

		// Initial recursive loop trigger (no tool results yet)
		e.recursiveStream(ctx, q, inputMessage, out, nil)
	}()

	return out, nil
}

// recursiveStream fetches history, injects definitions, streams from the provider, handles tools, and restarts if necessary.
func (e *Engine) recursiveStream(ctx context.Context, q llm.InjectQuery, initialInput llm.Message, out chan<- llm.Message, pendingResults []llm.ToolResultPart) {
	if len(pendingResults) > 0 {
		q.PendingToolResults = pendingResults
	} else {
		q.PendingToolResults = nil
	}

	msgs, err := e.buildPromptMessages(ctx, q, initialInput)
	if err != nil {
		sessID, _ := llm.SessionIDFromContext(ctx) // fallback for error sending
		e.sendError(ctx, out, err, sessID)
		return
	}

	if e.provider == nil {
		return
	}

	sessionID, err := llm.SessionIDFromContext(ctx)
	if err != nil {
		e.sendError(ctx, out, err, "")
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

	// We might need an explicit way for history to capture these!
	// For now, if an injector implements workmem.WorkingMemory, we can trigger append.
	// But history should be isolated. This will need a callback or specific WorkingMemory injector interface.
	for _, inj := range e.injectors {
		if appendable, ok := inj.(interface{
			Append(ctx context.Context, sessionID string, msg llm.Message) error
		}); ok && len(accumulatedParts) > 0 {
			bundledMsg := llm.Message{
				SessionID: sessionID,
				Identity:  llm.Identity{Role: "assistant"},
				Parts:     accumulatedParts,
			}
			if err := appendable.Append(ctx, sessionID, bundledMsg); err != nil {
				e.sendError(ctx, out, err, sessionID)
				return
			}
		}
	}

	// Executed after the provider closes its stream slice
	if len(pendingCalls) > 0 {
		var results []llm.ToolResultPart
		for _, tc := range pendingCalls {
			result := e.executeToolWithMiddleware(ctx, tc)
			results = append(results, result)
		}

		// ToolResult parts need to be appended to history as well!
		toolMsg := llm.Message{
			SessionID: sessionID,
			Identity:  llm.Identity{Role: "tool"},
		}
		for _, pr := range results {
			toolMsg.Parts = append(toolMsg.Parts, pr)
		}
		e.safeSend(ctx, out, toolMsg)

		for _, inj := range e.injectors {
			if appendable, ok := inj.(interface{
				Append(ctx context.Context, sessionID string, msg llm.Message) error
			}); ok {
				if err := appendable.Append(ctx, sessionID, toolMsg); err != nil {
					e.sendError(ctx, out, err, sessionID)
					return
				}
			}
		}

		// Re-enter the loop with the executed results
		e.recursiveStream(ctx, q, initialInput, out, results)
	}
}

// buildPromptMessages pieces together system prompts, enrichers, tool schemas, and history.
func (e *Engine) buildPromptMessages(ctx context.Context, q llm.InjectQuery, initialInput llm.Message) ([]llm.Message, error) {
	var msgs []llm.Message
	
	// Ensure initial user input hits history appending before returning prompt messages!
	// Only do this on the very first turn! (len(q.PendingToolResults) == 0)
	if len(q.PendingToolResults) == 0 {
		sessionID, err := llm.SessionIDFromContext(ctx)
		if err != nil {
			return nil, err
		}
		for _, inj := range e.injectors {
			if appendable, ok := inj.(interface{
				Append(ctx context.Context, sessionID string, msg llm.Message) error
			}); ok {
				if err := appendable.Append(ctx, sessionID, initialInput); err != nil {
					return nil, err
				}
			}
		}
	}

	for _, inj := range e.injectors {
		injMsgs, err := inj.Inject(ctx, q)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, injMsgs...)
	}

	// Then, sort by Volatility Ascending (Static=0 first, High=10 last)
	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].Volatility < msgs[j].Volatility
	})

	return msgs, nil
}

func (e *Engine) executeToolWithMiddleware(ctx context.Context, req llm.ToolRequestPart) llm.ToolResultPart {
	resultPart := llm.ToolResultPart{
		ToolID: req.ToolID,
		Name:   req.Name,
	}

	var targetTool llm.Tool
	var namespace string
	// Search through ToolProviders
	for _, inj := range e.injectors {
		if tp, ok := inj.(llm.ToolProvider); ok {
			if t, found := tp.GetTool(req.Name); found {
				targetTool = t
				namespace = tp.Namespace()
				break
			}
		}
	}

	if targetTool == nil {
		resultPart.Result = "Error: Tool not found"
		resultPart.IsError = true
		return resultPart
	}

	evalCtx := context.WithValue(ctx, llm.ContextKeyNamespace, namespace)

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

	res, err := executionFunc(evalCtx)
	if err != nil {
		resultPart.Result = err.Error()
		resultPart.IsError = true
	} else {
		resultPart.Result = res
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
