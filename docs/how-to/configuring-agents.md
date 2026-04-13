# How-To: Configuring Agents

Dux allows you to construct and switch between predefined "Agents". An Agent is simply a predefined **system prompt** bound to a specific **LLM provider**.

By separating agent configurations from core application configuration, you can easily maintain distinct personas or specialized task runners without cluttering your core setup.

## The Agents Configuration Directory

By default, `dux` will look for an agents configuration directory located at `$XDG_CONFIG_HOME/dux/agents/` (usually `~/.config/dux/agents/`). You can explicitly specify the directory location using the `-a` or `--agents-dir` global flags.

Within this directory, each agent should be placed in its own folder. The folder name acts as the conceptual unit for the agent. Inside each folder, you must define the agent using an `agent.yaml` file (`~/.config/dux/agents/<agent-name>/agent.yaml`).

### Example Configuration

An `agent.yaml` file uses a YAML object describing the agent. You can start by checking the `examples/dux/agents/` folder available in the root repository. Below is a sample configuration:

```yaml
name: "technical-author"
provider: "ollama-local"
workflow:
  default_mode: "orchestrator"
  modes:
    - name: "orchestrator"
      context:
        system: |
          You are the lead Orchestrator. Route work to researcher and writer sub-agents.
        tools:
          - name: "stdlib"
            enabled: true
      transitions:
        - to: "researcher"
          description: "Delegate to researcher to gather data."
        - to: "writer"
          description: "Delegate to writer to draft the guide."

    - name: "researcher"
      context:
        system: |
          You are a specialized research agent. Gather data and respond completely.
        enrichers:
          - type: "time"
          - type: "os"
        tools:
          - name: "stdlib"
            enabled: true
            requirements:
              supervision: false
          - name: "filesystem"
            enabled: true
            mcp:
              command: "npx"
              args: ["-y", "@modelcontextprotocol/server-filesystem", "/home/user/docs"]
          - name: "librarian"
            enabled: true
      transitions:
        - to: "orchestrator"
          description: "Return to orchestrator with gathered research."

    - name: "writer"
      provider: "openai" # Override the base agent provider with a more expensive model
      context:
        system: |
          You are a technical writer conforming to the Diátaxis documentation framework. 
          Use the 'read_working_memory' tool to recall the exact research data provided.
        tools:
          - name: "librarian"
            enabled: true
      transitions:
        - to: "orchestrator"
          description: "Return to orchestrator with finished draft."

triggers:
  - type: chat
  - type: schedule
    config:
      cron: "@every 5m"
      topic: "qa_health"
      prompt: "Run health diagnostic"
```

### Agent Fields

*   `name` (string): The identifier you will use in the CLI.
*   `provider` (string): The fallback LLM Provider ID (e.g., `ollama-local`).
*   `workflow` (object): Defines the state machine of modes for context routing.
    *   `default_mode` (string): The mode the agent instantiates into initially (e.g., `researcher`).
    *   `modes` (array): A list of available state machine nodes.
        *   `name` (string): The mode identifier.
        *   `provider` (string, *optional*): Override the LLM provider for this specific mode.
        *   `context` (object): The strict focus limitations injected during this phase of execution.
            *   `system` (string): The overarching intent or persona limitation for this mode.
            *   `enrichers` (array): A list of dynamic context injection tools (e.g. time, os).
            *   `tools` (array): The exact tools the LLM has access to during this mode.
                *   `name` (string): Identifier for the tool or MCP server.
                *   `enabled` (bool): Whether the tool is active.
                *   `requirements` (object): Specify CEL-based `supervision` policies.
                *   `mcp` (object): Options for an external Model Context Protocol server.
        *   `transitions` (array): A list of mapped exit parameters for this mode.
            *   `to` (string): The target mode to switch to.
            *   `description` (string): Injected as a dynamic tool prompt telling the LLM when to use this transition.
*   `triggers` (array): A list of execution paradigms the agent should bind to when launched via `dux run`.
    *   `type` (string): The trigger class (e.g., `chat`, `schedule`, `event`, `timer`).
    *   `config` (map): Arbitrary configuration payload for the specific trigger.

## Interacting with Agents

**Run Background & Interactive Triggers:**
To spin up all triggers configured for a specific agent (like background schedules alongside interactive chat), use:

```bash
dux run coder
```

**Immediate One-Shots (Stdin):**
To submit a raw snippet to an agent in a background context without invoking Bubbletea REPL, pipe into `invoke`:

```bash
echo "Check system status" | dux invoke coder
```

**Single Chat Session:**
To strictly invoke an interactive `chat` session explicitly:

```bash
dux chat --agent coder
```

> **Note:** The `--agent` flag in chat limits execution to just the REPL context. Use `dux run` for the full multi-modal Trigger experience.
