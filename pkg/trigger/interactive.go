package trigger

import "context"

// Interactive signifies that the agent should be launched within a stateful session layer (e.g., REPL).
type Interactive struct {
	// StartFn is typically the entrypoint for starting the REPL
	StartFn func(ctx context.Context) error
}

// NewInteractive creates an Interactive trigger.
func NewInteractive(startFn func(ctx context.Context) error) *Interactive {
	return &Interactive{
		StartFn: startFn,
	}
}

func (i *Interactive) Category() Category {
	return CategoryInteractive
}

func (i *Interactive) Start(ctx context.Context) error {
	if i.StartFn != nil {
		return i.StartFn(ctx)
	}
	<-ctx.Done()
	return nil
}
