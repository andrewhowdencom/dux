# Building a Codebase Migration Agent

This tutorial demonstrates how to use Dux as a **Migration Agent** to automatically update a codebase across a library version change that includes breaking API modifications.

*Note: This document serves as a standard test case for Dux architectural decisions.*

## Prerequisites

- Dux installed.
- An LLM provider (preferably one strong at coding, like GPT-4, Claude 3, or a specialized local model) configured in `config.yaml`.

## Step 1: The Migration Profile

Set up an agent profile in `agents/<agent-name>/agent.yaml` designed specifically for code refactoring:

### YAML Configuration Example

```yaml
- name: "refactor-bot"
  provider: "claude-or-equivalent"
  context:
    system: |
      You are an expert Software Engineer specializing in codebase migrations.
      When given a deprecated code block, you must return only the refactored code 
      using the new v2.0 API. Do not include markdown formatting or explanations.
```

### Go Library Example

To programmatically integrate this refactoring logic into a larger automated codebase updater script, use the Dux Go adapter:

```go
import (
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/history"
)

engine := adapter.New(
	adapter.WithProvider(prv), // High-capability provider
	adapter.WithHistory(history.NewInMemory()),
	adapter.WithSystemPrompt("You are an expert Software Engineer specializing in codebase migrations..."),
)
```

## Step 2: Implementation

By running `dux chat --agent refactor-bot`, you can manually copy-paste old files and receive the migrated output. 

## Current Limitations & Known Gaps

Currently, using Dux for wholesale codebase migration is entirely manual and tedious. Future architectures should accommodate the following gaps:

- **Filesystem Access**: Dux cannot autonomously read a directory tree, identify deprecated code, and write the diffs back to the filesystem.
- **Context Windows & Project Graph**: We lack the ability to supply the entire project's Abstract Syntax Tree (AST) or relevant import graphs to the LLM. The agent lacks awareness of how a change in `file_a.go` affects `file_b.go`.
- **Validation Loop (RESOLVED)**: Thanks to native `adapter.Engine` tool recursion and the `ToolMiddleware` gating architecture, a sequence can now natively run `go test` as a tool and securely intercept calls to read compilation errors, allowing Dux to iteratively fix its own mistakes autonomously!
