# How-To: Configuring MCP Servers

Model Context Protocol (MCP) servers allow you to seamlessly extend an agent's capabilities by providing it with external tools, data sources, and functionalities over a standardized interface.

Dux supports connecting to both local MCP servers via `stdio` and remote MCP servers via Server-Sent Events (`sse`).

## The MCP Configuration Block

MCPs are registered as **tools** under an agent's `context.tools` configuration inside your `agent.yaml` file (usually located in `~/.config/dux/agents/<agent-name>/agent.yaml`).

To configure a tool as an MCP server, you provide an `mcp` object within the tool definition.

### Configuring a Local MCP Server (Stdio)

Local MCP servers are executed as a subprocess by Dux. Dux communicates with the server over standard input (`stdin`) and standard output (`stdout`).

**Via YAML (CLI):**
```yaml
name: "researcher"
provider: "openai"
context:
  tools:
    - name: "filesystem"
      enabled: true
      mcp:
        command: "npx"
        args: ["-y", "@modelcontextprotocol/server-filesystem", "/home/user/docs"]
        env:
          DEBUG: "mcp:*"
```

**Via Go Library:**
```go
mcpClient, err := client.NewStdioMCPClient(
	"npx",
	[]string{"DEBUG=mcp:*"},
	"-y", "@modelcontextprotocol/server-filesystem", "/home/user/docs",
)
// ... call mcpClient.Start() and wrap in tool.NewMCPResolver(ctx, "filesystem", mcpClient)
```

**Fields for Stdio Connections:**

*   `command` (string): The executable command to run. If this is a script like `npx` or `python`, ensure it's available in your system's PATH.
*   `args` (array): A list of arguments to pass to the `command`.
*   `env` (map, *optional*): Arbitrary key/value pairs for setting local environment variables inside the subprocess.

### Configuring a Remote MCP Server (HTTP / SSE)

Remote MCP servers are hosted on a network and communicate over HTTP. Dux supports two remote transport protocols: `streamable_http` (the default if a URL is provided) and `sse` (Server-Sent Events).

Depending on what the remote server supports, you can explicitly configure the `transport` field to ensure compatibility.

**Via YAML (CLI):**
```yaml
name: "researcher"
provider: "openai"
context:
  tools:
    - name: "weather-service"
      enabled: true
      mcp:
        transport: "sse"
        url: "http://localhost:3000/sse"
        headers:
          "Authorization": "Bearer token123"
```

**Via Go Library (SSE):**
```go
mcpClient, err := client.NewSSEMCPClient(
	"http://localhost:3000/sse",
	transport.WithHeaders(map[string]string{"Authorization": "Bearer token123"}),
)
// ... call mcpClient.Start() and wrap in tool.NewMCPResolver(ctx, "weather-service", mcpClient)
```

**Via Go Library (Streamable HTTP):**
```go
tport, err := transport.NewStreamableHTTP("http://localhost:3000/mcp")
if err == nil {
	mcpClient := client.NewClient(tport)
	// ... call mcpClient.Start() and wrap in tool.NewMCPResolver(ctx, "my-service", mcpClient)
}
```

**Fields for Remote Connections:**

*   `url` (string): The absolute URL endpoint targeting the remote server. If `url` is provided without a `transport`, Dux defaults to `streamable_http`.
*   `transport` (string, *optional*): Set explicitly to `"sse"`, `"streamable_http"`, or `"stdio"`. If not set, Dux auto-detects based on the presence of `command` or `url`.
*   `headers` (map, *optional*): Arbitrary key/value HTTP headers (like `Authorization` or custom API keys) sent to the remote server during connection.

## Common Examples

### Connecting to a Python-based MCP Server

If you have a local python tool built using the MCP SDK:

```yaml
- name: "python-math-tools"
  enabled: true
  mcp:
    command: "python"
    args: ["-m", "my_mcp_math_server"]
```

### Using npx to Run a Typescript Server

Using `npx` with the `-y` flag is a great way to load and run public community servers without installing them globally first. 

Because tools like GitHub grant broad access to external systems, it's highly recommended to secure them using Common Expression Language (CEL) policies. The following example automatically approves safe read operations while requiring human-in-the-loop (HITL) supervision for destructive ones.

```yaml
- name: "github-mcp"
  enabled: true
  requirements:
    # Evaluate to true (requires human approval) unless the tool is performing a read operation
    supervision: "!(tool_name.startsWith('get_') || tool_name.startsWith('list_') || tool_name.startsWith('search_'));"
  mcp:
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: "ghp_xxxxxxxxxxxxxxxxxxxx"
```

> **Note:** `tool_name` is dynamically injected into the CEL execution context, representing the internal name of the specific MCP capability that the LLM is attempting to execute.

## Troubleshooting

1.  **Transport Type Conflict:** Dux auto-detects the transport based on the fields provided. If you provide a `url` without specifying `transport`, it defaults to `streamable_http`. If you provide `command`, it uses `stdio`. You can override this auto-detection by explicitly setting the `transport` field to `"sse"`, `"streamable_http"`, or `"stdio"`.
2.  **Environment Variables:** By default, local Stdio MCPs do *not* inherit your shell's environment variables unless explicitly injected using the `env` map, or through core configuration.
3.  **Command Path:** Ensure local binaries specified in `command` (like `node`, `python`, `npx`) are correctly resolved via the system `$PATH` where Dux is being executed.


