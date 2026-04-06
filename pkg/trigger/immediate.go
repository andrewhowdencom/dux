package trigger

import "context"

// Immediate triggers a one-shot execution.
type Immediate struct {
	prompt  string
	handler Handler
}

// NewImmediate creates an Immediate trigger.
func NewImmediate(prompt string, handler Handler) *Immediate {
	return &Immediate{
		prompt:  prompt,
		handler: handler,
	}
}

func (i *Immediate) Category() Category {
	return CategoryNonInteractive
}

func (i *Immediate) Start(ctx context.Context) error {
	return i.handler(ctx, i.prompt)
}
