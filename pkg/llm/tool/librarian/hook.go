package librarian

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
)

// NewAfterCompleteHook returns an AfterComplete hook that persists a session
// summary as a StatementFact to semantic memory.  This makes session
// conclusions searchable via the semantic graph without requiring explicit
// tool calls during the conversation.
func NewAfterCompleteHook(service *semantic.Service) llm.AfterCompleteHook {
	return func(ctx context.Context, req llm.AfterCompleteRequest) error {
		if service == nil {
			return nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Session %s summary:\n", req.SessionID))

		// Summarise tool history.
		if len(req.ToolHistory) > 0 {
			sb.WriteString("\nTool Executions:\n")
			for _, rec := range req.ToolHistory {
				status := "success"
				if rec.Result.IsError {
					status = "error"
				}
				sb.WriteString(fmt.Sprintf("- %s [%s]: %v\n", rec.ToolCall.Name, status, rec.Result.Result))
			}
		}

		// Capture the final assistant message.
		for _, p := range req.FinalMessage.Parts {
			if text, ok := p.(llm.TextPart); ok {
				sb.WriteString(fmt.Sprintf("\nFinal Response:\n%s\n", string(text)))
			}
		}

		fact := semantic.StatementFact{
			ID:        fmt.Sprintf("session_%s_%d", req.SessionID, time.Now().Unix()),
			Statement: sb.String(),
			Sources: []semantic.Source{
				{
					URI:         fmt.Sprintf("dux://sessions/%s", req.SessionID),
					RetrievedAt: time.Now().UTC(),
				},
			},
			Tags: []string{"session-summary", "auto-extracted"},
			Metadata: semantic.FactMetadata{
				CreatedAt:    time.Now().UTC(),
				ValidatedAt:  time.Now().UTC(),
				LastAccessed: time.Now().UTC(),
				AccessCount:  1,
			},
		}

		return service.WriteStatement(ctx, fact)
	}
}
