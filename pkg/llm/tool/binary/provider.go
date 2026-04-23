package binary

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/pkg/llm"
)

type binaryProvider struct {
	tool llm.Tool
}

func (p *binaryProvider) Namespace() string {
	return p.tool.Name()
}

func (p *binaryProvider) GetTool(name string) (llm.Tool, bool) {
	if name == p.tool.Name() {
		return p.tool, true
	}
	return nil, false
}

func (p *binaryProvider) Tools() []llm.Tool {
	return []llm.Tool{p.tool}
}

type binaryTool struct {
	name        string
	description string
	executable  string
	args        []string
	inputs      map[string]config.ToolInput
}

func (t *binaryTool) Name() string {
	return t.name
}

func (t *binaryTool) Definition() llm.ToolDefinitionPart {
	props := make(map[string]interface{})
	var required []string

	for name, input := range t.inputs {
		props[name] = map[string]string{
			"type":        input.Type,
			"description": input.Description,
		}
		if input.Required {
			required = append(required, name)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": props,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	// We must supply empty properties if none are defined
	if len(props) == 0 {
		schema["properties"] = map[string]interface{}{}
	}

	b, _ := json.Marshal(schema)

	return llm.ToolDefinitionPart{
		Name:        t.name,
		Description: t.description,
		Parameters:  b,
	}
}

func (t *binaryTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var finalArgs []string

	for _, arg := range t.args {
		resolvedArg := arg
		// Perform very simple {key} replacements.
		for k, v := range args {
			strVal := fmt.Sprintf("%v", v)
			resolvedArg = strings.ReplaceAll(resolvedArg, "{"+k+"}", strVal)
		}
		finalArgs = append(finalArgs, resolvedArg)
	}

	cmd := exec.CommandContext(ctx, t.executable, finalArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %v\noutput: %s", err, string(out))
	}

	return string(out), nil
}

// NewProvider converts a config.BinaryTool configuration into an llm.ToolProvider.
func NewProvider(name string, t *config.BinaryTool) llm.ToolProvider {
	description := t.Description
	if description == "" {
		description = fmt.Sprintf("Executes the %s binary", t.Executable)
	}
	bt := &binaryTool{
		name:        name,
		description: description,
		executable:  t.Executable,
		args:        t.Args,
		inputs:      t.Inputs,
	}
	return &binaryProvider{
		tool: bt,
	}
}
