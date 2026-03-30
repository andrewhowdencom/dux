package terminal

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

// StartREPL begins a synchronous interactive loop wrapping the engine stream.
func StartREPL(ctx context.Context, engine llm.Engine, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)

	for {
		_, _ = fmt.Fprintf(out, "\n%s[User]>%s ", colorCyan, colorReset)
		if !scanner.Scan() {
			break
		}
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		msg := llm.Message{
			SessionID: "cli-session",
			Identity:  llm.Identity{Role: "user"},
			Parts:     []llm.Part{llm.TextPart(text)},
		}

		streamCh, err := engine.Stream(ctx, msg)
		if err != nil {
			_, _ = fmt.Fprintf(out, "\n%s[System Error]%s %v\n", colorYellow, colorReset, err)
			continue
		}

		for outMsg := range streamCh {
			var prefix, color string
			switch outMsg.Identity.Role {
			case "assistant":
				color = colorGreen
				prefix = "[Dux]"
			case "tool", "system":
				color = colorYellow
				prefix = "[" + strings.ToUpper(outMsg.Identity.Role[:1]) + outMsg.Identity.Role[1:]
				if outMsg.Identity.Name != "" {
					prefix += ":" + outMsg.Identity.Name
				}
				prefix += "]"
			default:
				color = colorReset
				prefix = "[" + outMsg.Identity.Role + "]"
			}

			for _, part := range outMsg.Parts {
				switch p := part.(type) {
				case llm.TextPart:
					_, _ = fmt.Fprintf(out, "%s%s%s %s\n", color, prefix, colorReset, string(p))
				case llm.ToolRequestPart:
					_, _ = fmt.Fprintf(out, "%s%s%s Requesting tool '%s' with args: %v\n", color, prefix, colorReset, p.Name, p.Args)
				case llm.ToolDefinitionPart:
					_, _ = fmt.Fprintf(out, "%s%s%s Provided tool definition: %s\n", color, prefix, colorReset, p.Name)
				default:
					_, _ = fmt.Fprintf(out, "%s%s%s [Unknown part type: %T]\n", color, prefix, colorReset, p)
				}
			}
		}
	}
	return scanner.Err()
}
