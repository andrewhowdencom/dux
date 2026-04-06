package trigger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInteractiveStart(t *testing.T) {
	var wasCalled bool
	startFn := func(ctx context.Context) error {
		wasCalled = true
		return nil
	}

	trigger := NewInteractive(startFn)
	assert.Equal(t, CategoryInteractive, trigger.Category())

	err := trigger.Start(context.Background())
	assert.NoError(t, err)
	assert.True(t, wasCalled)
}

func TestInteractiveStart_NilFunc(t *testing.T) {
	trigger := NewInteractive(nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		_ = trigger.Start(ctx)
		close(done)
	}()

	cancel()
	<-done // Should unblock when context cancels
}
