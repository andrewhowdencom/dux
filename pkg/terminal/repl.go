package terminal

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	userStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("35")).Bold(true)
	assistantStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("41")).Bold(true)
	thinkingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	toolStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("215"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

type chatMessage struct {
	role      string
	name      string
	content   string
	thinking  string
	toolCalls []string
}

type uiModel struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	engine     llm.Engine
	modelName  string
	theme      string

	viewport viewport.Model
	textarea textarea.Model
	hitl     *BubbleTeaHITL

	pendingToolPrompt *ToolApprovalRequestMsg

	renderer *glamour.TermRenderer
	messages []chatMessage
	err      error

	streamCh    <-chan llm.Message
	isStreaming bool
	quit        bool
}

type streamMsg struct {
	msg llm.Message
	err error
}

func waitForNextChunk(ch <-chan llm.Message) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		p, ok := <-ch
		if !ok {
			return streamMsg{err: io.EOF}
		}
		return streamMsg{msg: p}
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

func newUIModel(ctx context.Context, engine llm.Engine, modelName, theme string, hitl *BubbleTeaHITL) uiModel {
	ta := textarea.New()
	ta.Placeholder = "Ask Dux a question..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 10000
	ta.SetHeight(3)

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to Dux Chat! Type a message and press Enter.")

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

	return uiModel{
		ctx:       ctx,
		engine:    engine,
		modelName: modelName,
		theme:     theme,
		textarea:  ta,
		viewport:  vp,
		renderer: rend,
		messages: []chatMessage{},
		hitl:      hitl,
	}
}

func (m uiModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, textarea.Blink)
	if m.hitl != nil {
		cmds = append(cmds, waitForHITL(m.hitl.RequestCh))
	}
	return tea.Batch(cmds...)
}

func (m uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		cmds  []tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, tiCmd, vpCmd)

	switch msg := msg.(type) {
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
				role:    "user",
				content: v,
			})
			m.textarea.Reset()
			m.updateViewport()

			// Create stream context
			streamCtx, cancel := context.WithCancel(m.ctx)
			m.cancelFunc = cancel

			// Send to LLM
			llmMsg := llm.Message{
				SessionID: "cli-session",
				Identity:  llm.Identity{Role: "user"},
				Parts:     []llm.Part{llm.TextPart(v)},
			}

			streamCh, err := m.engine.Stream(streamCtx, llmMsg)
			if err != nil {
				m.err = err
				m.updateViewport()
				break
			}

			m.streamCh = streamCh
			m.isStreaming = true

			// Append empty assistant message to accumulate chunks
			m.messages = append(m.messages, chatMessage{
				role: "assistant", // Default role initially
			})

			cmds = append(cmds, waitForNextChunk(m.streamCh))
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

	case streamMsg:
		if msg.err == io.EOF {
			m.isStreaming = false
			m.cancelFunc = nil
			m.updateViewport()
			break
		}
		if msg.err != nil {
			m.err = msg.err
			m.isStreaming = false
			m.cancelFunc = nil
			m.updateViewport()
			break
		}

		lastIdx := len(m.messages) - 1
		for _, p := range msg.msg.Parts {
			switch part := p.(type) {
			case llm.TextPart:
				m.messages[lastIdx].content += string(part)
			case llm.ReasoningPart:
				m.messages[lastIdx].thinking += string(part)
			case llm.ToolRequestPart:
				m.messages[lastIdx].toolCalls = append(m.messages[lastIdx].toolCalls, fmt.Sprintf("Tool Call: %s(%v)", part.Name, part.Args))
			case llm.ToolResultPart:
				resStr := fmt.Sprintf("%v", part.Result) // Format carefully
				if len(resStr) > 500 {
					resStr = resStr[:500] + " ... (truncated)"
				}
				if part.IsError {
					m.messages[lastIdx].toolCalls = append(m.messages[lastIdx].toolCalls, fmt.Sprintf("↳ Error (%s): %s", part.Name, resStr))
				} else {
					m.messages[lastIdx].toolCalls = append(m.messages[lastIdx].toolCalls, fmt.Sprintf("↳ Result (%s): %s", part.Name, strings.ReplaceAll(resStr, "\n", "\\n")))
				}
			case llm.ToolDefinitionPart:
				m.messages[lastIdx].toolCalls = append(m.messages[lastIdx].toolCalls, fmt.Sprintf("Schema: %s", part.Name))
			}
		}

		m.updateViewport()
		m.viewport.GotoBottom() // Auto-scroll to bottom while streaming
		cmds = append(cmds, waitForNextChunk(m.streamCh))

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
			roleTitle = assistantStyle.Render(fmt.Sprintf("Dux (%s)", m.modelName))
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

		if msg.thinking != "" {
			b.WriteString(thinkingStyle.Render("Thinking:\n" + msg.thinking))
			b.WriteString("\n\n")
		}

		if len(msg.toolCalls) > 0 {
			b.WriteString(toolStyle.Render(strings.Join(msg.toolCalls, "\n")))
			b.WriteString("\n\n")
		}

		if msg.content != "" {
			var formatStr string
			if m.renderer != nil {
				out, err := m.renderer.Render(msg.content)
				if err == nil {
					formatStr = out
				} else {
					formatStr = msg.content // Fallback
				}
			} else {
				formatStr = msg.content
			}
			b.WriteString(strings.TrimSpace(formatStr))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	m.viewport.SetContent(strings.TrimSpace(b.String()))
}

func (m uiModel) View() string {
	if m.quit {
		return "Chat session ended.\n"
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
func StartREPL(ctx context.Context, engine llm.Engine, modelName, theme string, hitl *BubbleTeaHITL, in io.Reader, out io.Writer) error {
	p := tea.NewProgram(
		newUIModel(ctx, engine, modelName, theme, hitl),
		tea.WithAltScreen(),
		tea.WithInput(in),
		tea.WithOutput(out),
	)

	_, err := p.Run()
	return err
}
