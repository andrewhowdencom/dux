package stdlib

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// Base64EncodeTool
type Base64EncodeTool struct{}
func NewBase64Encode() llm.Tool { return &Base64EncodeTool{} }
func (t *Base64EncodeTool) Name() string { return "base64_encode" }
func (t *Base64EncodeTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Base64 encodes a string.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
	}
}
func (t *Base64EncodeTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	text, ok := args["text"].(string)
	if !ok {
		return nil, fmt.Errorf("missing string text parameter")
	}
	return map[string]string{"encoded": base64.StdEncoding.EncodeToString([]byte(text))}, nil
}

// Base64DecodeTool
type Base64DecodeTool struct{}
func NewBase64Decode() llm.Tool { return &Base64DecodeTool{} }
func (t *Base64DecodeTool) Name() string { return "base64_decode" }
func (t *Base64DecodeTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Base64 decodes a string.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"encoded":{"type":"string"}},"required":["encoded"]}`),
	}
}
func (t *Base64DecodeTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	encoded, ok := args["encoded"].(string)
	if !ok {
		return nil, fmt.Errorf("missing string encoded parameter")
	}
	dec, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	return map[string]string{"text": string(dec)}, nil
}

// URLEncodeTool
type URLEncodeTool struct{}
func NewURLEncode() llm.Tool { return &URLEncodeTool{} }
func (t *URLEncodeTool) Name() string { return "url_encode" }
func (t *URLEncodeTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "URL encodes a string safely for query parameters.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
	}
}
func (t *URLEncodeTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	text, ok := args["text"].(string)
	if !ok {
		return nil, fmt.Errorf("missing string text parameter")
	}
	return map[string]string{"encoded": url.QueryEscape(text)}, nil
}

// URLDecodeTool
type URLDecodeTool struct{}
func NewURLDecode() llm.Tool { return &URLDecodeTool{} }
func (t *URLDecodeTool) Name() string { return "url_decode" }
func (t *URLDecodeTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "URL decodes a previously escaped string.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"encoded":{"type":"string"}},"required":["encoded"]}`),
	}
}
func (t *URLDecodeTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	encoded, ok := args["encoded"].(string)
	if !ok {
		return nil, fmt.Errorf("missing string encoded parameter")
	}
	dec, err := url.QueryUnescape(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode url text: %w", err)
	}
	return map[string]string{"text": dec}, nil
}
