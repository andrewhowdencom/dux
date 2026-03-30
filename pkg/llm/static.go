package llm

import "context"

// StaticEngine implements Engine and immediately returns a predefined
// slice of Message objects, completely ignoring the inputMessage.
// This is primarily useful for testing and providing canned responses.
type StaticEngine struct {
	responses []Message
}

// NewStaticEngine creates a new StaticEngine yielding the provided responses in order.
func NewStaticEngine(responses ...Message) *StaticEngine {
	return &StaticEngine{
		responses: responses,
	}
}

// Stream spins up an unbuffered channel and immediately pushes all predefined responses to it,
// simulating a fast LLM generation cycle.
func (s *StaticEngine) Stream(ctx context.Context, inputMessage Message) (<-chan Message, error) {
	out := make(chan Message)

	go func() {
		defer close(out)
		for _, r := range s.responses {
			select {
			case <-ctx.Done():
				return
			case out <- r:
				// Push successful
			}
		}
	}()

	return out, nil
}
