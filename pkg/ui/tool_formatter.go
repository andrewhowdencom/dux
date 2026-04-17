package ui

import (
	"fmt"
)

// ToolFormatter handles display formatting for tool calls and results
type ToolFormatter interface {
	// FormatToolCall formats the tool call display (arguments, status)
	FormatToolCall(toolName string, args map[string]interface{}) string

	// FormatToolResult formats the tool result display
	FormatToolResult(toolName string, result interface{}, isError bool) string

	// GetIcon returns the icon for a tool (emoji or unicode)
	GetIcon(toolName string) string

	// ShouldShowArgs returns whether arguments should be displayed
	ShouldShowArgs(toolName string) bool

	// ShouldShowResult returns whether result should be displayed
	ShouldShowResult(toolName string) bool
}

// ToolDisplayConfig holds per-tool display configuration
type ToolDisplayConfig struct {
	Icon         string
	HideArgs     bool
	HideResult   bool
	MaxResultLen int
}

// DefaultToolFormatter provides sensible defaults
type DefaultToolFormatter struct {
	toolConfigs map[string]ToolDisplayConfig
	defaultIcon string
}

// NewDefaultToolFormatter creates a new formatter with the given configs
func NewDefaultToolFormatter(configs map[string]ToolDisplayConfig, defaultIcon string) *DefaultToolFormatter {
	if defaultIcon == "" {
		defaultIcon = "🔧"
	}
	return &DefaultToolFormatter{
		toolConfigs: configs,
		defaultIcon: defaultIcon,
	}
}

// GetIcon returns the configured icon or default
func (f *DefaultToolFormatter) GetIcon(toolName string) string {
	if cfg, ok := f.toolConfigs[toolName]; ok && cfg.Icon != "" {
		return cfg.Icon
	}
	return f.defaultIcon
}

// ShouldShowArgs checks config for the tool
func (f *DefaultToolFormatter) ShouldShowArgs(toolName string) bool {
	if cfg, ok := f.toolConfigs[toolName]; ok {
		return !cfg.HideArgs
	}
	return true
}

// ShouldShowResult checks config for the tool
func (f *DefaultToolFormatter) ShouldShowResult(toolName string) bool {
	if cfg, ok := f.toolConfigs[toolName]; ok {
		return !cfg.HideResult
	}
	return true
}

// FormatToolCall produces formatted tool call text
func (f *DefaultToolFormatter) FormatToolCall(toolName string, args map[string]interface{}) string {
	icon := f.GetIcon(toolName)
	if !f.ShouldShowArgs(toolName) {
		return fmt.Sprintf("%s Tool: %s", icon, toolName)
	}
	return fmt.Sprintf("%s Tool: %s\nArgs: %v", icon, toolName, args)
}

// FormatToolResult produces formatted tool result text
func (f *DefaultToolFormatter) FormatToolResult(toolName string, result interface{}, isError bool) string {
	icon := f.GetIcon(toolName)
	if !f.ShouldShowResult(toolName) {
		if isError {
			return fmt.Sprintf("%s %s: Error", icon, toolName)
		}
		return fmt.Sprintf("%s %s: Success", icon, toolName)
	}

	status := "✅"
	if isError {
		status = "❌"
	}

	resStr := fmt.Sprintf("%v", result)
	maxLen := 500
	if cfg, ok := f.toolConfigs[toolName]; ok && cfg.MaxResultLen > 0 {
		maxLen = cfg.MaxResultLen
	}
	if len(resStr) > maxLen {
		resStr = resStr[:maxLen] + "..."
	}

	return fmt.Sprintf("%s %s Result: %s\n%s", icon, toolName, status, resStr)
}
