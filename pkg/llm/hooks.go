package llm

import (
	"context"
	"time"
)

// BeforeStartRequest carries context for the BeforeStart hook.
type BeforeStartRequest struct {
	SessionID  string
	InitialMsg Message
	Mode       string // workflow mode, if applicable
}

// BeforeGenerateRequest carries context for the BeforeGenerate hook.
// Hooks are executed serially in registration order. Each hook may read,
// append, reorder, or filter CurrentMessages.
type BeforeGenerateRequest struct {
	SessionID       string
	CurrentMessages []Message // accumulated from prior hooks
	PendingResults  []ToolResultPart
	Mode            string
}

// BeforeToolRequest carries context for the BeforeTool hook.
type BeforeToolRequest struct {
	SessionID string
	Mode      string
	ToolCall  ToolRequestPart
	CallIndex int // 0-based index of this tool call within the current session
}

// AfterToolRequest carries context for the AfterTool hook.
type AfterToolRequest struct {
	SessionID string
	Mode      string
	ToolCall  ToolRequestPart
	Result    ToolResultPart
	Duration  time.Duration
	Error     error // non-nil if the tool execution itself failed
}

// ToolExecutionRecord captures a single tool call and its outcome for AfterComplete.
type ToolExecutionRecord struct {
	ToolCall ToolRequestPart
	Result   ToolResultPart
	Duration time.Duration
	Error    error
}

// AfterCompleteRequest carries context for the AfterComplete hook.
type AfterCompleteRequest struct {
	SessionID    string
	Mode         string
	FinalMessage Message
	ToolHistory  []ToolExecutionRecord
	Transition   *TransitionSignalPart
}

// BeforeStartHook fires once per Stream() call, before any LLM interaction.
type BeforeStartHook func(ctx context.Context, req BeforeStartRequest) error

// BeforeGenerateHook fires before each LLM call (including recursive turns).
// Hooks execute serially and may mutate CurrentMessages.
// req is passed by pointer so mutations are visible to subsequent hooks.
type BeforeGenerateHook func(ctx context.Context, req *BeforeGenerateRequest) error

// BeforeToolHook fires before each tool execution. Returning an error blocks
// the tool; the engine converts the error into a synthetic ToolResultPart{IsError:true}
// and feeds it back to the LLM.
type BeforeToolHook func(ctx context.Context, req BeforeToolRequest) error

// AfterToolHook fires after each tool execution, regardless of success or failure.
type AfterToolHook func(ctx context.Context, req AfterToolRequest) error

// AfterCompleteHook fires when the recursive convergence loop terminates
// (no more pending tool calls).
type AfterCompleteHook func(ctx context.Context, req AfterCompleteRequest) error
