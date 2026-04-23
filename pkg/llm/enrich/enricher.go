package enrich

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// NewTime returns a BeforeGenerateHook that injects the current system time.
func NewTime() llm.BeforeGenerateHook {
	return func(ctx context.Context, req *llm.BeforeGenerateRequest) error {
		text := fmt.Sprintf("<enrichment type=\"time\">\nCurrent Time: %s\n</enrichment>", time.Now().Format(time.RFC3339))
		req.CurrentMessages = append(req.CurrentMessages, llm.Message{
			Identity:   llm.Identity{Role: "system"},
			Parts:      []llm.Part{llm.TextPart(text)},
			Volatility: llm.VolatilityHigh,
		})
		return nil
	}
}

// NewOS returns a BeforeGenerateHook that injects OS/architecture information.
func NewOS() llm.BeforeGenerateHook {
	return func(ctx context.Context, req *llm.BeforeGenerateRequest) error {
		text := fmt.Sprintf("<enrichment type=\"os\">\nOperating System: %s\nArchitecture: %s\n</enrichment>", runtime.GOOS, runtime.GOARCH)
		req.CurrentMessages = append(req.CurrentMessages, llm.Message{
			Identity:   llm.Identity{Role: "system"},
			Parts:      []llm.Part{llm.TextPart(text)},
			Volatility: llm.VolatilityStatic,
		})
		return nil
	}
}

// NewPrompt returns a BeforeGenerateHook that injects a static prompt enrichment.
func NewPrompt(text string) llm.BeforeGenerateHook {
	return func(ctx context.Context, req *llm.BeforeGenerateRequest) error {
		text := fmt.Sprintf("<enrichment type=\"prompt\">\n%s\n</enrichment>", text)
		req.CurrentMessages = append(req.CurrentMessages, llm.Message{
			Identity:   llm.Identity{Role: "system"},
			Parts:      []llm.Part{llm.TextPart(text)},
			Volatility: llm.VolatilityStatic,
		})
		return nil
	}
}

// NewGuardRail returns a BeforeGenerateHook that injects guard-rail instructions.
func NewGuardRail(text string) llm.BeforeGenerateHook {
	return func(ctx context.Context, req *llm.BeforeGenerateRequest) error {
		text := fmt.Sprintf("<enrichment type=\"guard_rail\">\n%s\n</enrichment>", text)
		req.CurrentMessages = append(req.CurrentMessages, llm.Message{
			Identity:   llm.Identity{Role: "system"},
			Parts:      []llm.Part{llm.TextPart(text)},
			Volatility: llm.VolatilityStatic,
		})
		return nil
	}
}
