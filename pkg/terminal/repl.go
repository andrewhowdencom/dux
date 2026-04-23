package terminal

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/ui"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/muesli/reflow/wordwrap"
)

var (
	userStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("35")).Bold(true)
	assistantStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("41")).Bold(true)
	thinkingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	toolStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("215"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

type terminalConfig struct {
	toolConfigs map[string]ui.ToolDisplayConfig
	defaultIcon string
}

type TerminalOption func(*terminalConfig)

func WithToolDisplayConfig(configs map[string]ui.ToolDisplayConfig, defaultIcon string) TerminalOption {
	return func(cfg *terminalConfig) {
		cfg.toolConfigs = configs
		cfg.defaultIcon = defaultIcon
	}
}

type chatBlock struct {
	kind    string // "text", "thinking", "tool"
	content string
}

type chatMessage struct {
	role      string
	name      string
	blocks    []*chatBlock
	telemetry *llm.TelemetryPart
}

type uiModel struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	engine     llm.Engine
	modelName  string
	theme      string
	agentName  string
	sessionID  string

	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model
	hitl     *BubbleTeaHITL

	pendingToolPrompt *ToolApprovalRequestMsg

	renderer *glamour.TermRenderer
	messages []chatMessage
	err      error

	program       *tea.Program
	isStreaming   bool
	quit          bool
	termFormatter *TerminalFormatter
}

type uiEventMsg struct {
	Type      string
	Content   string
	Name      string
	Args      any
	Result    any
	IsError   bool
	Telemetry llm.TelemetryPart
	Err       error
	Req       *llm.ToolRequestPart
}

func (m *uiModel) RenderTextChunk(chunk string) {
	m.program.Send(uiEventMsg{Type: "text", Content: chunk})
}

func (m *uiModel) RenderError(err error) {
	m.program.Send(uiEventMsg{Type: "error", Err: err})
}

func (m *uiModel) PromptHITL(req *llm.ToolRequestPart) {
	// HITL is already handled dynamically through the BubbleTeaHITL hook channels.
}

func (m *uiModel) Flush() {
	m.program.Send(uiEventMsg{Type: "flush"})
}

func (m *uiModel) RenderThinkingChunk(chunk string) {
	m.program.Send(uiEventMsg{Type: "thinking", Content: chunk})
}

func (m *uiModel) RenderToolIntent(toolName string, args any) {
	argsMap, _ := args.(map[string]interface{})
	formatted := m.termFormatter.FormatToolCall(toolName, argsMap)
	m.program.Send(uiEventMsg{Type: "tool_req", Name: toolName, Args: args, Content: formatted})
}

func (m *uiModel) RenderToolResult(toolName string, result any, isError bool) {
	formatted := m.termFormatter.FormatToolResult(toolName, result, isError)
	m.program.Send(uiEventMsg{Type: "tool_res", Name: toolName, Result: result, IsError: isError, Content: formatted})
}

func (m *uiModel) RenderTelemetry(telemetry llm.TelemetryPart) {
	m.program.Send(uiEventMsg{Type: "telemetry", Telemetry: telemetry})
}

func (m *uiModel) OnCommand(cmd string, args []string) {
	if cmd == "/new" {
		m.program.Send(uiEventMsg{Type: "session_reset"})
	}
}

func waitForHITL(ch chan ToolApprovalRequestMsg) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		return <-ch
	}
}

func newUIModel(ctx context.Context, sessionID string, initialMessages []llm.Message, engine llm.Engine, modelName, theme, agentName string, hitl *BubbleTeaHITL, opts ...TerminalOption) *uiModel {
	name := agentName
	if name == "" {
		name = "Dux"
	}

	ta := textarea.New()
	ta.Placeholder = fmt.Sprintf("Ask %s a question...", name)
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 10000
	ta.SetHeight(3)

	vp := viewport.New(80, 20)
	vp.SetContent(fmt.Sprintf("Welcome to %s Chat! Type a message and press Enter.", name))

	if theme == "auto" || theme == "" {
		theme = "dark"
	}

	// Create a glamour renderer
	rend, err := glamour.NewTermRenderer(
		glamour.WithStylePath(theme),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		// Fallback for when the file isn't found or standard style fails
		rend, _ = glamour.NewTermRenderer(glamour.WithStandardStyle("dark"), glamour.WithWordWrap(80))
	}

	// Initialize Spinner
	s := spinner.New()
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Apply options
	cfg := &terminalConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Initialize formatter with config
	baseFormatter := ui.NewDefaultToolFormatter(cfg.toolConfigs, cfg.defaultIcon)
	termFormatter := NewTerminalFormatter(baseFormatter, 80)

	uiMsgs := make([]chatMessage, 0, len(initialMessages))
	for _, msg := range initialMessages {
		if msg.Volatility <= llm.VolatilityMedium {
			continue // System prompts usually shouldn't clutter the UI chat
		}
		cm := chatMessage{
			role: msg.Identity.Role,
			name: msg.Identity.Name,
		}
		for _, rawP := range msg.Parts {
			switch p := rawP.(type) {
			case llm.TextPart:
				if len(cm.blocks) > 0 && cm.blocks[len(cm.blocks)-1].kind == "text" {
					cm.blocks[len(cm.blocks)-1].content += string(p)
				} else {
					cm.blocks = append(cm.blocks, &chatBlock{kind: "text", content: string(p)})
				}
			case llm.ReasoningPart:
				if len(cm.blocks) > 0 && cm.blocks[len(cm.blocks)-1].kind == "thinking" {
					cm.blocks[len(cm.blocks)-1].content += string(p)
				} else {
					cm.blocks = append(cm.blocks, &chatBlock{kind: "thinking", content: string(p)})
				}
			case llm.ToolRequestPart:
				cm.blocks = append(cm.blocks, &chatBlock{kind: "tool", content: fmt.Sprintf("Tool Call: %s(%v)", p.Name, p.Args)})
			case llm.ToolResultPart:
				resStr := fmt.Sprintf("%v", p.Result)
				if len(resStr) > 500 {
					resStr = resStr[:500] + " ... (truncated)"
				}
				if p.IsError {
					cm.blocks = append(cm.blocks, &chatBlock{kind: "tool", content: fmt.Sprintf("↳ Error (%s): %s", p.Name, resStr)})
				} else {
					cm.blocks = append(cm.blocks, &chatBlock{kind: "tool", content: fmt.Sprintf("↳ Result (%s): %s", p.Name, strings.ReplaceAll(resStr, "\n", "\\n"))})
				}
			case llm.TelemetryPart:
				if cm.telemetry == nil {
					cm.telemetry = &llm.TelemetryPart{}
				}
				cm.telemetry.InputTokens += p.InputTokens
				cm.telemetry.OutputTokens += p.OutputTokens
				cm.telemetry.ReasoningTokens += p.ReasoningTokens
				cm.telemetry.Duration += p.Duration
			}
		}
		uiMsgs = append(uiMsgs, cm)
	}

	m := &uiModel{
		ctx:           ctx,
		engine:        engine,
		modelName:     modelName,
		theme:         theme,
		agentName:     name,
		textarea:      ta,
		viewport:      vp,
		spinner:       s,
		renderer:      rend,
		messages:      uiMsgs,
		sessionID:     sessionID,
		hitl:          hitl,
		termFormatter: termFormatter,
	}

	if len(uiMsgs) > 0 {
		m.updateViewport()
	}

	return m
}

func (m *uiModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, textarea.Blink, m.spinner.Tick)
	if m.hitl != nil {
		cmds = append(cmds, waitForHITL(m.hitl.RequestCh))
	}
	return tea.Batch(cmds...)
}

func (m *uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		cmds  []tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, tiCmd, vpCmd)

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc, tea.KeyCtrlD:
			m.quit = true
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			return m, tea.Quit

		case tea.KeyEnter:
			if m.pendingToolPrompt != nil {
				v := strings.TrimSpace(strings.ToLower(m.textarea.Value()))
				if v == "y" || v == "yes" || v == "" {
					m.pendingToolPrompt.ReplyCh <- true
				} else {
					m.pendingToolPrompt.ReplyCh <- false
				}
				m.pendingToolPrompt = nil
				m.textarea.Reset()
				m.updateViewport()
				return m, waitForHITL(m.hitl.RequestCh)
			}

			if m.isStreaming {
				break
			}
			v := strings.TrimSpace(m.textarea.Value())
			if v == "" {
				break
			}

			// Add user message
			m.messages = append(m.messages, chatMessage{
				role:   "user",
				blocks: []*chatBlock{{kind: "text", content: v}},
			})
			m.textarea.Reset()
			m.updateViewport()

			// Send to LLM using unified Session orchestrator
			go func(input string) {
				s := &ui.ChatSession{
					ID:     m.sessionID,
					Engine: m.engine,
					View:   m,
				}
				err := s.StreamQuery(m.ctx, input)
				m.program.Send(uiEventMsg{Type: "done", Err: err})
			}(v)

			m.isStreaming = true

			// Append empty assistant message to accumulate chunks
			m.messages = append(m.messages, chatMessage{
				role: "assistant",
			})
		}

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - m.textarea.Height() - 2
		if m.renderer != nil {
			var err error
			m.renderer, err = glamour.NewTermRenderer(
				glamour.WithStylePath(m.theme),
				glamour.WithWordWrap(msg.Width-4),
			)
			if err != nil {
				m.renderer, _ = glamour.NewTermRenderer(
					glamour.WithStandardStyle("dark"),
					glamour.WithWordWrap(msg.Width-4),
				)
			}
		}
		m.textarea.SetWidth(msg.Width)
		m.updateViewport()

	case uiEventMsg:
		lastIdx := len(m.messages) - 1

		switch msg.Type {
		case "done":
			m.isStreaming = false
			m.cancelFunc = nil
			if msg.Err != nil && msg.Err != io.EOF {
				m.err = msg.Err
			}
			m.updateViewport()
		case "error":
			m.err = msg.Err
			m.isStreaming = false
			m.updateViewport()
		case "text":
			lastMsg := &m.messages[lastIdx]
			if len(lastMsg.blocks) > 0 && lastMsg.blocks[len(lastMsg.blocks)-1].kind == "text" {
				lastMsg.blocks[len(lastMsg.blocks)-1].content += msg.Content
			} else {
				lastMsg.blocks = append(lastMsg.blocks, &chatBlock{kind: "text", content: msg.Content})
			}
		case "thinking":
			lastMsg := &m.messages[lastIdx]
			if len(lastMsg.blocks) > 0 && lastMsg.blocks[len(lastMsg.blocks)-1].kind == "thinking" {
				lastMsg.blocks[len(lastMsg.blocks)-1].content += msg.Content
			} else {
				lastMsg.blocks = append(lastMsg.blocks, &chatBlock{kind: "thinking", content: msg.Content})
			}
		case "tool_req":
			lastMsg := &m.messages[lastIdx]
			lastMsg.blocks = append(lastMsg.blocks, &chatBlock{kind: "tool", content: msg.Content})
		case "tool_res":
			lastMsg := &m.messages[lastIdx]
			lastMsg.blocks = append(lastMsg.blocks, &chatBlock{kind: "tool", content: msg.Content})
		case "telemetry":
			if m.messages[lastIdx].telemetry == nil {
				m.messages[lastIdx].telemetry = &llm.TelemetryPart{
					InputTokens:     msg.Telemetry.InputTokens,
					OutputTokens:    msg.Telemetry.OutputTokens,
					ReasoningTokens: msg.Telemetry.ReasoningTokens,
					Duration:        msg.Telemetry.Duration,
				}
			} else {
				m.messages[lastIdx].telemetry.InputTokens += msg.Telemetry.InputTokens
				m.messages[lastIdx].telemetry.OutputTokens += msg.Telemetry.OutputTokens
				m.messages[lastIdx].telemetry.ReasoningTokens += msg.Telemetry.ReasoningTokens
				m.messages[lastIdx].telemetry.Duration += msg.Telemetry.Duration
			}
		case "session_reset":
			m.sessionID = uuid.New().String()
			m.messages = []chatMessage{
				{role: "assistant", blocks: []*chatBlock{{kind: "text", content: "Started a new conversation session."}}},
			}
			m.textarea.Reset()
			m.updateViewport()
		case "flush":
			m.updateViewport()
			m.viewport.GotoBottom() // Auto-scroll
		}

	case ToolApprovalRequestMsg:
		m.pendingToolPrompt = &msg
	}

	return m, tea.Batch(cmds...)
}

func (m *uiModel) updateViewport() {
	var b strings.Builder

	for _, msg := range m.messages {
		roleTitle := ""
		switch msg.role {
		case "user":
			roleTitle = userStyle.Render("User")
		case "assistant":
			roleTitle = assistantStyle.Render(fmt.Sprintf("%s (%s)", m.agentName, m.modelName))
		default:
			title := msg.role
			if len(title) > 0 {
				title = strings.ToUpper(title[:1]) + strings.ToLower(title[1:])
			}
			roleTitle = toolStyle.Render(title)
		}

		if msg.name != "" {
			roleTitle += " (" + toolStyle.Render(msg.name) + ")"
		}

		b.WriteString(fmt.Sprintf("%s:\n", roleTitle))

		for _, block := range msg.blocks {
			switch block.kind {
			case "thinking":
				wrappedThinking := wordwrap.String("Thinking:\n"+block.content, m.viewport.Width)
				b.WriteString(thinkingStyle.Render(wrappedThinking))
				b.WriteString("\n\n")
			case "tool":
				b.WriteString(toolStyle.Render(block.content))
				b.WriteString("\n\n")
			case "text":
				if block.content != "" {
					var formatStr string
					if m.renderer != nil {
						out, err := m.renderer.Render(block.content)
						if err == nil {
							formatStr = out
						} else {
							formatStr = block.content // Fallback
						}
					} else {
						formatStr = block.content
					}
					b.WriteString(strings.TrimSpace(formatStr))
					b.WriteString("\n\n")
				}
			}
		}

		if msg.telemetry != nil {
			var res string
			if msg.telemetry.ReasoningTokens > 0 {
				res = fmt.Sprintf(" (including %d reasoning)", msg.telemetry.ReasoningTokens)
			}
			telemetryStr := fmt.Sprintf("⚡ %.1fs | Tokens: %d in, %d out%s",
				msg.telemetry.Duration.Seconds(),
				msg.telemetry.InputTokens,
				msg.telemetry.OutputTokens,
				res,
			)
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(telemetryStr))
			b.WriteString("\n\n")
		}

		b.WriteString("\n")
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	m.viewport.SetContent(strings.TrimSpace(b.String()))
}

func (m *uiModel) View() string {
	if m.quit {
		return "Chat session ended.\n"
	}

	if m.isStreaming {
		m.textarea.Prompt = m.spinner.View() + " "
		m.textarea.Placeholder = fmt.Sprintf("%s is thinking...", m.agentName)
	} else {
		m.textarea.Prompt = "┃ "
		m.textarea.Placeholder = fmt.Sprintf("Ask %s a question...", m.agentName)
	}

	var inputView string
	if m.pendingToolPrompt != nil {
		inputView = lipgloss.NewStyle().Foreground(lipgloss.Color("215")).Render(
			fmt.Sprintf("Approve tool '%s' execution? [Y/n]: ", m.pendingToolPrompt.Req.Name),
		) + "\n" + m.textarea.View()
	} else {
		inputView = m.textarea.View()
	}

	return fmt.Sprintf(
		"%s\n%s\n%s",
		m.viewport.View(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("─", m.viewport.Width)),
		inputView,
	)
}

// StartREPL begins a synchronous interactive loop wrapping the engine stream.
func StartREPL(ctx context.Context, sessionID string, initialMessages []llm.Message, engine llm.Engine, modelName, theme, agentName string, hitl *BubbleTeaHITL, in io.Reader, out io.Writer, opts ...TerminalOption) error {
	m := newUIModel(ctx, sessionID, initialMessages, engine, modelName, theme, agentName, hitl, opts...)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithInput(in),
		tea.WithOutput(out),
	)
	m.program = p

	_, err := p.Run()
	return err
}
