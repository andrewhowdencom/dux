package working

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// NewHistoryHook returns a BeforeGenerateHook that reads the session history
// from the given History store and appends it to CurrentMessages.
func NewHistoryHook(mem llm.History) llm.BeforeGenerateHook {
	return func(ctx context.Context, req *llm.BeforeGenerateRequest) error {
		msgs, err := mem.Read(ctx, req.SessionID)
		if err != nil {
			return err
		}
		req.CurrentMessages = append(req.CurrentMessages, msgs...)
		return nil
	}
}
