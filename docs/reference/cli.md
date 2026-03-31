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

## `dux http serve`
Bootstraps a local listener running health endpoints for monitoring. This listens on `:8080`.

### Usage
```bash
./dux http serve [flags]
```

This starts the application and blocks your terminal while logging requests. The server automatically routes `GET: /healthz`.

**Example**
```bash
# Run the application headless
./dux http serve
```
