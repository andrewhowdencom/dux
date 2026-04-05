# How-To: Configuring Agents

Dux allows you to construct and switch between predefined "Agents". An Agent is simply a predefined **system prompt** bound to a specific **LLM provider**.

By separating agent configurations from core application configuration, you can easily maintain distinct personas or specialized task runners without cluttering your core setup.

## The Agents Configuration Directory

By default, `dux` will look for an agents configuration directory located at `$XDG_CONFIG_HOME/dux/agents/` (usually `~/.config/dux/agents/`). You can explicitly specify the directory location using the `-a` or `--agents-dir` global flags.

Within this directory, each agent should be placed in its own folder. The folder name acts as the conceptual unit for the agent. Inside each folder, you must define the agent using an `agent.yaml` file (`~/.config/dux/agents/<agent-name>/agent.yaml`).

### Example Configuration

An `agent.yaml` file uses a YAML object describing the agent. You can start by checking the `examples/dux/agents/` folder available in the root repository. Below is a sample configuration:

```yaml
name: "qa"
  provider: "ollama-local"
  context:
    system: |
      You are a specialized Question & Answer agent.
      Respond strictly to the prompt with complete, accurate answers.
      Always incorporate the injected enrichers as part of your source truth context.
    enrichers:
      - type: "time"
      - type: "os"
    tools:
      - name: "bash"
        enabled: true
        requirements:
          supervision: true
      - name: "file_read"
        enabled: true
      - name: "file_write"
        enabled: true
        requirements:
          supervision: true
      - name: "file_patch"
        enabled: true
        requirements:
          supervision: true
      - name: "file_list"
        enabled: true

- name: "writer"
  provider: "openai"
  context:
    system: |
      You are a technical writer conforming to the Diátaxis documentation framework.

- name: "researcher"
  provider: "openai"
  context:
    tools:
      - name: "filesystem"
        enabled: true
        mcp:
          command: "npx"
          args: ["-y", "@modelcontextprotocol/server-filesystem", "/home/user/docs"]
      - name: "weather"
        enabled: true
        mcp:
          url: "http://localhost:3000/sse"
          headers:
            "Authorization": "Bearer token123"
```

### Agent Fields

*   `name` (string): The identifier you will use in the CLI.
*   `provider` (string): The LLM Provider ID from your core configuration (e.g., `config.yaml` `llm.providers` array).
*   `context` (object): Options for defining dynamic and static inputs.
    *   `system` (string): The initial prompt injected seamlessly at the start of your chat instance.
    *   `enrichers` (array): A list of dynamic context injection tools (e.g. `type: "time"` or `type: "os"`).
    *   `tools` (array): A list of local or remote MCP tools to bind to the agent context.
        *   `name` (string): Identifier for the tool or MCP server.
        *   `enabled` (bool): Whether the tool is active.
        *   `mcp` (object): Options for an external Model Context Protocol server.
            *   `command` (string): Command to execute a local server in `stdio` mode (e.g., `npx`).
            *   `args` (array): Arguments passed to the `command` (e.g., `["-y", "@modelcontextprotocol/server-filesystem", "/src"]`).
            *   `env` (map): Arbitrary key/value pairs for local subprocess environment variables.
            *   `url` (string): Absolute URL endpoint targeting an `sse` event stream. (If provided, takes precedence over `command`).
            *   `headers` (map): Arbitrary key/value HTTP headers (e.g., Authorization) sent to remote `sse` servers.

## Using an Agent in the CLI

To invoke a specific agent, pass the `--agent` flag to the `chat` subcommand:

```bash
dux chat --agent devops
```

> **Note:** The `--agent` flag is mutually exclusive with the `--provider` flag. If you specify an agent, `dux` will internally fallback to the provider configured in the agent's YAML definition!
