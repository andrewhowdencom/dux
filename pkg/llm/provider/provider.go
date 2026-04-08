package provider

import (
	"context"
	"errors"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

var (
	ErrUnsupportedFeature  = errors.New("feature not supported by this target model/provider")
	ErrAuthentication      = errors.New("invalid API credentials")
)

type Capabilities struct {
	SupportsSystemPrompt     bool
	SupportsToolCalling      bool
	SupportsImages           bool
	SupportsStructuredOutput bool
	MaxContextWindow         int
}

type GenerateConfig struct {
	Temperature *float32
	MaxTokens   *int
	JSONSchema  []byte
}

// GenerateOption allows functional modification of a generation request.
type GenerateOption func(*GenerateConfig)

// WithTemperature overrides the provider's default temperature for a single request.
func WithTemperature(t float32) GenerateOption {
	return func(c *GenerateConfig) {
		c.Temperature = &t
	}
}

// WithJSONSchema forces the provider into structured output mode.
func WithJSONSchema(schema []byte) GenerateOption {
	return func(c *GenerateConfig) {
		c.JSONSchema = schema
	}
}

// ModelLister defines standard operations to retrieve available generation models.
type ModelLister interface {
	ListModels(ctx context.Context) ([]string, error)
}

// Generator defines text generation and tool invocation mechanics for standard chat flows.
type Generator interface {
	GenerateStream(ctx context.Context, messages []llm.Message, opts ...GenerateOption) (<-chan llm.Part, error)
}

// Embedder identifies providers capable of generating high-density vector representations.
type Embedder interface {
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

// Provider represents a uniform LLM client.
type Provider interface {
	Generator
	Embedder
	ModelLister

	Capabilities() Capabilities
}
