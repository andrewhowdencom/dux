package trigger

import (
	"context"
	"sync"
	"testing"
)

func TestInMemoryEventBus(t *testing.T) {
	bus := NewInMemoryEventBus()
	ctx := context.Background()

	var wg sync.WaitGroup
	var publishedPrompts []string
	var mu sync.Mutex

	handler := func(ctx context.Context, prompt string) error {
		defer wg.Done()
		mu.Lock()
		publishedPrompts = append(publishedPrompts, prompt)
		mu.Unlock()
		return nil
	}

	bus.Subscribe("topic1", handler)
	bus.Subscribe("topic1", handler)
	bus.Subscribe("topic2", handler)

	wg.Add(2)
	bus.Publish(ctx, "topic1", "hello")
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(publishedPrompts) != 2 {
		t.Errorf("expected 2 prompts, got %d", len(publishedPrompts))
	}
	if publishedPrompts[0] != "hello" || publishedPrompts[1] != "hello" {
		t.Errorf("expected hello, got %v", publishedPrompts)
	}
}
