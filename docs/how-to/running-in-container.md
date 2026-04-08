# How-To: Running Agents in Containers

This guide walks you through the process of packaging the `dux` CLI, your predefined agent configurations, and any local Model Context Protocol (MCP) servers into a single, hermetically sealed OCI container. 

This pattern is ideal for deploying standalone, task-specific agents where `dux` acts as PID 1 to manage the lifecycle and interaction with tools isolated within the container's filesystem.

## Prerequisites

*   Docker or Podman installed on your host machine.
*   A fundamental understanding of [Agent Configuration](./configuring-agents.md) and [MCP Configuration](./configuring-mcp.md).

## Step 1: Prepare the Configuration

First, construct the directory structure containing the configuration that `dux` will use inside the container. 

```bash
mkdir -p my-agent-bundle/config/agents/demo
```

### The Agent Profile
Create an `agent.yaml` in `my-agent-bundle/config/agents/demo/`:

```yaml
name: "demo"
provider: "openai" # Map to the provider defined in your main config
context:
  system: |
    You are a hermetic agent running within a container.
    Use your available tools to perform operations strictly within your isolated environment.
  tools:
    - name: "filesystem"
      enabled: true
      mcp:
        command: "npx"
        args: ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]
```

### The Core Configuration
Create a master configuration file at `my-agent-bundle/config/config.yaml` to set up your LLM provider credentials. It is highly recommended to use environment variable expansion for secrets so they are not baked into the image.

```yaml
llm:
  providers:
    - id: "openai"
      type: "openai"
      base_url: "https://api.openai.com/v1"
      token: "${OPENAI_API_KEY}" # Expanded securely at runtime
```

## Step 2: Write the Dockerfile

In your project root, create a `Dockerfile`. We will use a multi-stage build to compile `dux` from source, then construct a minimal runtime image containing the dependencies required by your MCP servers.

```dockerfile
# Stage 1: Build Dux
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY . .
RUN go build -o /bin/dux .

# Stage 2: Construct the Agent Runtime
FROM alpine:latest

# Install dependencies needed by your MCP tools.
# For example, if your MCP server relies on Node.js / npx:
RUN apk add --no-cache nodejs npm

# Setup an isolated working directory for the agent's operations
RUN mkdir -p /workspace

# Copy the constructed Dux binary
COPY --from=builder /bin/dux /usr/local/bin/dux

# Copy the configuration bundle into the container
COPY ./my-agent-bundle/config /etc/dux

# Tell Dux to read its configuration from /etc/dux
ENV XDG_CONFIG_HOME=/etc

# Execution: Run Dux as the primary entrypoint
# The `run` command will execute background triggers and interactive components
ENTRYPOINT ["dux"]
CMD ["run", "demo"]
```

## Step 3: Build the Container Image

Build the OCI image using Docker or Podman from the root of the repository:

```bash
docker build -t hermetic-dux-agent -f Dockerfile .
```

## Step 4: Run the Container

When you run the container, `dux` will execute in isolation. To provide context to the filesystem server, you can supply volume mounts where appropriate. Ensure you pass in the required environment variables (like API keys) at runtime.

### Interactive Mode
If your agent triggers include `chat`, you can run the container attached to your terminal's standard input:

```bash
docker run -it \
  -e OPENAI_API_KEY="sk-..." \
  -v $(pwd)/data:/workspace \
  hermetic-dux-agent chat --agent demo
```

### Background / Daemon Mode
If your agent is purely a background worker (e.g., triggered by schedule or event loop), run it detached:

```bash
docker run -d \
  -e OPENAI_API_KEY="sk-..." \
  hermetic-dux-agent run demo
```

## Why run inside a Container?

Because the container tightly isolates the environment, your MCP tools can run with full access to the container filesystem without risking the host machine. If your agent is given tools capable of executing arbitrary code or bash commands, the impact is constrained strictly to the ephemeral container sandbox, providing an excellent security boundary for autonomous workloads.
