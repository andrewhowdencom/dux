package static_test

import (
	"context"
	"testing"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/static"
)

func TestStaticProvider(t *testing.T) {
	prv, err := static.New(nil)
	if err != nil {
		t.Fatalf("unexpected error creating static provider: %v", err)
	}

	out, err := prv.GenerateStream(context.Background(), []llm.Message{})
	if err != nil {
		t.Fatalf("unexpected error starting stream: %v", err)
	}

	var parts []llm.Part
	for p := range out {
		parts = append(parts, p)
	}

	if len(parts) != 1 {
		t.Fatalf("expected exactly 1 part emitted by default static, got %d", len(parts))
	}

	if _, ok := parts[0].(llm.TextPart); !ok {
		t.Errorf("expected text part yielded, got %T", parts[0])
	}
}
