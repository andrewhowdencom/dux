---
title: Getting Started with Dux
description: A complete guide to setting up and running your first Dux agent using the declarative YAML configuration.
---

# Getting Started with Dux

Welcome to Dux! This tutorial will walk you through setting up the Dux framework from scratch, configuring your first autonomous agent via YAML, and taking advantage of persistent Semantic Memory for long-term recall.

## 1. Installation

Ensure you have a modern Go environment installed (Go 1.22+ recommended).

Clone the repository and build the binary:

```bash
git clone https://github.com/andrewhowdencom/dux.git
cd dux
go build -o dux main.go
```

## 2. Setting Up Configuration Paths

Dux utilizes declarative YAML configuration to define both its server architecture (`config.yaml`) and the behavior of distinct agents (`agents.yaml` or conf.d directories).

Create a local configuration directory to get started natively:

```bash
mkdir -p ~/.config/dux/agents
```

## 3. Creating the Application Configuration

The primary configuration governs the unified `ui` serving layer (e.g., exposing Web or Telegram interfaces) and dictates where those user interfaces direct tool calls. 

Create heavily commented `~/.config/dux/config.yaml`:

```yaml
# config.yaml (Main Application Config)
database:
  driver: "sqlite3"
  dsn: "file:/home/youruser/.config/dux/dux.db?cache=shared&mode=rwc"

ui:
  - type: terminal
    id: local-cli
    agent_id: "my-first-agent"
    provider:
      type: "ollama"
      id: "local-ollama"
      config:
        model: "llama3.1:latest"
```

## 4. Configuring Your First Agent

Agents represent isolated LLM personalities equipped with varying capabilities. We configure them externally so they are easy to mix, match, and modify. 

Create `~/.config/dux/agents/my-first-agent.yaml`:

```yaml
# my-first-agent.yaml
name: "My First Agent"
id: "my-first-agent"
instructions: >
  You are an incredibly helpful AI assistant operating autonomously on a user's system.
  You have access to a semantic memory store that allows you to safely persist and recall facts.

tools:
  - name: "semantic"
    enabled: true
    requirements:
      # Memory operations are safe autonomously 
      supervision: false
  - name: "bash"
    enabled: true
    timeout_seconds: 30
```

### About Semantic Memory Tools

By enabling the `semantic` tool bundle, your agent immediately gains access to a full suite of read, write, search, and delete functions interacting with the built-in Entity-Attribute-Value (EAV) store!
- **Zero-Touch Migrations**: As soon as Dux resolves the `semantic` bundle under the hood, the internal SQLite driver intelligently provisions and manages migrations (creating indices and `semmem_facts` tables) automatically.
- **Zero-Touch Execution**: In the YAML configuration above, we set `supervision: false`. Instead of prompting you incessantly through a Human-In-The-Loop system, the agent will gracefully write and recall variables completely autonomously in the background! You could also restrict writes dynamically to require approval via Common Expression Language (CEL) policies.

## 5. Running the Engine

With your YAML structure established, you're ready to serve multiple models securely:

```bash
# Set your config locations if you aren't running from the CWD
export DUX_CONFIG_PATH="$HOME/.config/dux/config.yaml"
export DUX_AGENTS_PATH="$HOME/.config/dux/agents"

# Launch DUX
./dux serve
```

Because you launched utilizing the `terminal` UI mapped directly to `my-first-agent`, Dux will instantly drop you into an interactive session.

## 6. Testing Semantic Memory

Try establishing persistent facts conversationally! 

**You:** "I prefer my outputs formatted exclusively as Markdown without trailing greetings."
**Dux:** *(Will autonomously utilize `semantic_write` in the background with `entity: "user"`, `attribute: "preferences"`, `value: "markdown only"`)* 

Even after you exit and restart your terminal process, the Dux engine will automatically rebuild your SQLite instances and the LLM will effortlessly be able to pull and search your previous state!
