# CLI Reference

Dux provides a compact but powerful Command Line Interface for testing and running your configured LLMs. 

The entry point of the application is the `dux` binary itself. All subcommands inherit globally available configuration options:

### Global Flags
| Flag          | Description                                                                              | Default                        |
| ------------- | ---------------------------------------------------------------------------------------- | ------------------------------ |
| `--config`    | Explicit configuration file path override (falls back to `$XDG_CONFIG_HOME/dux/config.yaml`). | empty string                  |
| `--log-level` | Modifies the standard `slog` output level (e.g. `debug`, `info`, `warn`).                 | `info`                         |

---

## `dux chat`
Starts a local REPL session allowing you to chat with a mapped LLM provider from your terminal asynchronously.

### Usage
```bash
./dux chat [flags]
```

### Flags
| Flag         | Description                                                      | Default |
| ------------ | ---------------------------------------------------------------- | ------- |
| `--provider` | Explicitly specify a provider `id` block defined in `config.yaml`. | none   |

**Example**
```bash
# Chat through an ollama configuration using verbose debug logs
./dux chat --provider="ollama-local" --log-level="debug"
```

---

## `dux serve`
Bootstraps all configured user interfaces (Web, Telegram, etc) based on the `ui` definitions in your `config.yaml`.

### Usage
```bash
./dux serve [type] [flags]
```

By default it spans all configured endpoints concurrently.

### Flags
| Flag         | Description                                                            | Default |
| ------------ | ---------------------------------------------------------------------- | ------- |
| `<type>`     | Optional positional arg (`web`, `telegram`) to filter endpoints        | none    |
| `--agent`    | Filter which configured instances to start by targeting this bound agent | none    |
| `--provider` | Filter by configured provider block                                    | none    |

**Example**
```bash
# Start all UIs
./dux serve

# Only start configured telegram setups targeted at the "pirate" agent
./dux serve telegram --agent="pirate"
```
