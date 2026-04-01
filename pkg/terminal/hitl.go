package terminal

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// ToolApprovalRequestMsg is dispatched to the BubbleTea event loop when
// a tool needs interactive approval.
type ToolApprovalRequestMsg struct {
	Req     llm.ToolRequestPart
	ReplyCh chan bool
}

// BubbleTeaHITL implements llm.HITLHandler and passes requests out
// via an unbuffered channel to the UI loop.
type BubbleTeaHITL struct {
	RequestCh chan ToolApprovalRequestMsg
}

// NewBubbleTeaHITL creates a new interactive HITL hook.
func NewBubbleTeaHITL() *BubbleTeaHITL {
	return &BubbleTeaHITL{
		RequestCh: make(chan ToolApprovalRequestMsg),
	}
}

// ApproveTool blocks the engine thread and proxies the approval request
// to BubbleTea for interactive input.
func (h *BubbleTeaHITL) ApproveTool(ctx context.Context, req llm.ToolRequestPart) (bool, error) {
	reply := make(chan bool)
	msg := ToolApprovalRequestMsg{
		Req:     req,
		ReplyCh: reply,
	}

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case h.RequestCh <- msg:
	}

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case res := <-reply:
		return res, nil
	}
}
