package working

import (
	"github.com/andrewhowdencom/dux/pkg/llm"
)

// WorkingMemory encapsulates context-window management for LLM sessions.
type WorkingMemory interface {
	llm.History
}
