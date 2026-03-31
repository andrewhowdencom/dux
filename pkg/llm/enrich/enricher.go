package enrich

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// Enricher defines dynamic context to be injected into the LLM conversation stream.
type Enricher interface {
	// Type returns the identifier for this enricher.
	Type() string
	// Enrich computes the dynamic text to inject based on the configured type.
	Enrich(ctx context.Context) (string, error)
}

// timeEnricher provides the current system time dynamically.
type timeEnricher struct{}

func (e *timeEnricher) Type() string { return "time" }
func (e *timeEnricher) Enrich(ctx context.Context) (string, error) {
	return fmt.Sprintf("<enrichment type=\"time\">\nCurrent Time: %s\n</enrichment>", time.Now().Format(time.RFC3339)), nil
}

// osEnricher provides the basic operating system information dynamically.
type osEnricher struct{}

func (e *osEnricher) Type() string { return "os" }
func (e *osEnricher) Enrich(ctx context.Context) (string, error) {
	return fmt.Sprintf("<enrichment type=\"os\">\nOperating System: %s\nArchitecture: %s\n</enrichment>", runtime.GOOS, runtime.GOARCH), nil
}

// promptEnricher provides statically configured prompt text.
type promptEnricher struct {
	text string
}

func (e *promptEnricher) Type() string { return "prompt" }
func (e *promptEnricher) Enrich(ctx context.Context) (string, error) {
	return fmt.Sprintf("<enrichment type=\"prompt\">\n%s\n</enrichment>", e.text), nil
}

// guardRailEnricher provides statically configured guard rail instructions.
type guardRailEnricher struct {
	text string
}

func (e *guardRailEnricher) Type() string { return "guard_rail" }
func (e *guardRailEnricher) Enrich(ctx context.Context) (string, error) {
	return fmt.Sprintf("<enrichment type=\"guard_rail\">\n%s\n</enrichment>", e.text), nil
}
