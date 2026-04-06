package trigger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImmediateStart(t *testing.T) {
	ctx := context.Background()

	var receivedPrompt string
	handler := func(ctx context.Context, prompt string) error {
		receivedPrompt = prompt
		return nil
	}

	immediate := NewImmediate("test prompt", handler)
	assert.Equal(t, CategoryNonInteractive, immediate.Category())

	err := immediate.Start(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test prompt", receivedPrompt)
}
