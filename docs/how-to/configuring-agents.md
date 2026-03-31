# How-To: Configuring Agents

Dux allows you to construct and switch between predefined "Agents". An Agent is simply a predefined **system prompt** bound to a specific **LLM provider**.

By separating agent configurations from core application configuration, you can easily maintain distinct personas or specialized task runners without cluttering your core setup.

## The Agents Specification File

By default, `dux` will look for an agents specification file located at `$XDG_CONFIG_HOME/dux/agents.yaml` (usually `~/.config/dux/agents.yaml`). You can explicitly specify the file location using the `-a` or `--agents-file` global flags.

### Example Configuration

An agents file uses a YAML array of agent definitions. You can start by checking the `agents.example.yaml` file available in the root repository. Below is a sample configuration:

```yaml
- name: "devops"
  provider: "ollama-local"
  system_prompt: |
    You are an expert DevOps engineer. Always output production-ready configurations
    and explain the security implications of your advice.

- name: "writer"
  provider: "openai"
  system_prompt: |
    You are a technical writer conforming to the Diátaxis documentation framework.
```

### Agent Fields

*   `name` (string): The identifier you will use in the CLI.
*   `provider` (string): The LLM Provider ID from your core configuration (e.g., `config.yaml` `llm.providers` array).
*   `system_prompt` (string): The initial prompt injected seamlessly at the start of your chat instance to guide the LLM's behavior.

## Using an Agent in the CLI

To invoke a specific agent, pass the `--agent` flag to the `chat` subcommand:

```bash
dux chat --agent devops
```

> **Note:** The `--agent` flag is mutually exclusive with the `--provider` flag. If you specify an agent, `dux` will internally fallback to the provider configured in the agent's YAML definition!
