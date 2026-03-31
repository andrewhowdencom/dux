# Running Locally

Whether you are configuring `dux` from scratch, extending its execution pipeline, or building out its internal APIs, you can quickly build the application and target your local configurations.

## Compiling Dux

Dux is written in Go. Running the build is straightforward given you have the proper Go toolchain installed on your OS. The project root contains a `Taskfile.yml` file automating key operations like linting and unit testing.

1.  Start by cloning the codebase if you haven't yet:
    ```bash
    git clone https://github.com/andrewhowdencom/dux.git
    cd dux
    ```

2.  Run the tests or build your binary manually. We utilize the `task` CLI runner:
    ```bash
    # Optionally: validate the build process, linting, and smoke tests
    task validate

    # Alternatively, compile directly
    task build
    ```

This produces a `dux` binary directly in your root.

## Setting Up Configuration

Before chatting with your CLI or targeting an HTTP host, you need a dynamic Viper configuration file that maps to LLM providers. By default, Dux searches via standard `XDG_CONFIG_HOME` directories, defaulting directly to `~/.config/dux/config.yaml`.

```bash
# Make the target directory
mkdir -p ~/.config/dux/

# Copy the example yaml included in the repository
cp config.example.yaml ~/.config/dux/config.yaml
```

You can view the specific [LiteLLM Integration Tutorial](../tutorials/litellm.md) to understand mapping OpenAI proxy configurations, or simply leave the `ollama-local` fallback if you have the Ollama daemon running.

## Running the Application

### The CLI Chat Repl
Using `./dux chat` creates a synchronous REPL shell tied sequentially to the recursive `adapter` loops mapping API returns over the console output without any asynchronous rendering limits.

```bash
./dux chat --provider="ollama-local"
```

### The HTTP Server
If you want to use the API routes remotely, or ensure health checks pass before deploying, you can use the built HTTP listener.

```bash
./dux http serve
```

This starts a server on `:8080` returning standard health status payloads: `http://localhost:8080/healthz`.
