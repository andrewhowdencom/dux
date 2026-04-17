package terminal

import (
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/ui"
	"github.com/charmbracelet/lipgloss"
)

type TerminalFormatter struct {
	baseFormatter ui.ToolFormatter
	width         int
}

func NewTerminalFormatter(base ui.ToolFormatter, width int) *TerminalFormatter {
	return &TerminalFormatter{
		baseFormatter: base,
		width:         width,
	}
}

func (f *TerminalFormatter) FormatToolCall(toolName string, args map[string]interface{}) string {
	icon := f.baseFormatter.GetIcon(toolName)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("215")).
		Padding(0, 1)

	header := fmt.Sprintf("%s TOOL: %s", icon, toolName)

	if !f.baseFormatter.ShouldShowArgs(toolName) {
		return border.Render(header)
	}

	content := fmt.Sprintf("Args: %v", args)
	return border.Render(header + "\n" + content)
}

func (f *TerminalFormatter) FormatToolResult(toolName string, result interface{}, isError bool) string {
	icon := f.baseFormatter.GetIcon(toolName)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("41")).
		Padding(0, 1)

	status := "✅"
	if isError {
		status = "❌"
	}
	header := fmt.Sprintf("%s RESULT: %s %s", icon, toolName, status)

	if !f.baseFormatter.ShouldShowResult(toolName) {
		return border.Render(header)
	}

	resStr := fmt.Sprintf("%v", result)
	return border.Render(header + "\n" + resStr)
}
