package llm

import (
	"context"
	"fmt"
	"time"
)

// NewTimeoutMiddleware wraps an execution loop with a customizable timeout.
// If the underlying tool execution exceeds the specified timeout or the
// explicit mapping inside timeouts, it is forcefully unblocked and an error is returned.
func NewTimeoutMiddleware(timeouts map[string]time.Duration, defaultTimeout time.Duration) ToolMiddleware {
	return func(ctx context.Context, req ToolRequestPart, next func(ctx context.Context) (interface{}, error)) (interface{}, error) {
		timeout, exists := timeouts[req.Name]
		if !exists {
			timeout = defaultTimeout
		}

		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		resChan := make(chan struct {
			res interface{}
			err error
		}, 1)

		go func() {
			res, err := next(ctx)
			resChan <- struct {
				res interface{}
				err error
			}{res, err}
		}()

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("tool execution timed out after %s", timeout)
		case out := <-resChan:
			return out.res, out.err
		}
	}
}
