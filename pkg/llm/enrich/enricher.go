package enrich

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// timeEnricher provides the current system time dynamically.
type timeEnricher struct{}

func NewTime() llm.Injector { return &timeEnricher{} }

func (e *timeEnricher) Inject(ctx context.Context, q llm.InjectQuery) ([]llm.Message, error) {
	text := fmt.Sprintf("<enrichment type=\"time\">\nCurrent Time: %s\n</enrichment>", time.Now().Format(time.RFC3339))
	return []llm.Message{{
		Identity:   llm.Identity{Role: "system"},
		Parts:      []llm.Part{llm.TextPart(text)},
		Volatility: llm.VolatilityHigh,
	}}, nil
}

// osEnricher provides the basic operating system information dynamically.
type osEnricher struct{}

func NewOS() llm.Injector { return &osEnricher{} }

func (e *osEnricher) Inject(ctx context.Context, q llm.InjectQuery) ([]llm.Message, error) {
	text := fmt.Sprintf("<enrichment type=\"os\">\nOperating System: %s\nArchitecture: %s\n</enrichment>", runtime.GOOS, runtime.GOARCH)
	return []llm.Message{{
		Identity:   llm.Identity{Role: "system"},
		Parts:      []llm.Part{llm.TextPart(text)},
		Volatility: llm.VolatilityStatic,
	}}, nil
}

// promptEnricher provides statically configured prompt text.
type promptEnricher struct {
	text string
}

func NewPrompt(text string) llm.Injector { return &promptEnricher{text: text} }

func (e *promptEnricher) Inject(ctx context.Context, q llm.InjectQuery) ([]llm.Message, error) {
	text := fmt.Sprintf("<enrichment type=\"prompt\">\n%s\n</enrichment>", e.text)
	return []llm.Message{{
		Identity:   llm.Identity{Role: "system"},
		Parts:      []llm.Part{llm.TextPart(text)},
		Volatility: llm.VolatilityStatic,
	}}, nil
}

// guardRailEnricher provides statically configured guard rail instructions.
type guardRailEnricher struct {
	text string
}

func NewGuardRail(text string) llm.Injector { return &guardRailEnricher{text: text} }

func (e *guardRailEnricher) Inject(ctx context.Context, q llm.InjectQuery) ([]llm.Message, error) {
	text := fmt.Sprintf("<enrichment type=\"guard_rail\">\n%s\n</enrichment>", e.text)
	return []llm.Message{{
		Identity:   llm.Identity{Role: "system"},
		Parts:      []llm.Part{llm.TextPart(text)},
		Volatility: llm.VolatilityStatic,
	}}, nil
}
