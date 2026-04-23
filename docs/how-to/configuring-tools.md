# How-To: Configuring Tools for Agents

Agents in `dux` are only as capable as the tools they are permitted to invoke. Configuring tools is the process of selecting which capabilities—built-in bundles, custom binaries, or remote MCP servers—are exposed to a given agent mode, and under what supervision constraints. This guide walks through the practical steps of wiring tools into your agent configuration files.

## The Two-Level Configuration Model

`dux` separates tool *definition* from tool *activation*. Global definitions live in your central `config.yaml` (or equivalent global configuration), where you can declare custom binary tools and reusable toolsets. Activation happens inside each agent's `agent.yaml`, where you explicitly opt-in to the tools and bundles that a specific mode may use. This separation ensures that dangerous capabilities can be defined once but granted selectively, following the principle of least privilege.

When the engine initializes a mode, it begins by loading all globally defined tools, then overlays whatever the agent mode specifies in its `context.tools` array. The resulting flattened map determines the active resolver set for that session. If a tool appears in both the global and agent contexts, the agent-level configuration takes precedence for properties like `enabled` status and supervision rules.

## Enabling Built-in Tool Bundles

Built-in capabilities are grouped into **bundles**—logical namespaces like `stdlib`, `filesystem`, `bash`, `workspace_plans`, `librarian`, and `semantic`. Rather than enumerating individual tools such as `file_read` or `file_write`, you reference the bundle name, and `dux` automatically provisions the entire suite.

To grant an agent mode access to general utilities and filesystem operations, add the following to the mode's `context` block:

```yaml
context:
  tools:
    - name: "stdlib"
      enabled: true
    - name: "filesystem"
      enabled: true
```

By default, the `stdlib` bundle is implicitly injected into the global tool pool with supervision disabled, meaning all agents receive safe computational utilities unless you explicitly override or disable them.

## Configuring Supervision with CEL

Not all tools are equally safe to run unattended. `dux` uses Common Expression Language (CEL) to evaluate whether a specific tool invocation requires human approval. You can attach a `requirements.supervision` expression to any bundle or individual tool configuration. If the expression evaluates to `true`, the execution is paused pending user confirmation; if `false`, it proceeds automatically.

A typical pattern is to allow read-only filesystem access autonomously while flagging writes for review:

```yaml
context:
  tools:
    - name: "filesystem"
      enabled: true
      requirements:
        supervision: "tool_name == 'file_write' || tool_name == 'file_patch'"
```

If you omit the supervision requirement entirely, `dux` applies safe defaults: most read-only operations (such as `file_read` or `get_current_time`) execute without prompting, while destructive or system-level tools (like `bash` or `file_write`) default to supervised. You can disable all supervision unconditionally—useful for trusted background agents—by setting `supervision: false`, though this should be done with caution.

## Declarative Binary Tools

When you need the agent to execute a local CLI program, you can define a **declarative binary tool** in the global `tools` array. These definitions bypass the shell entirely, eliminating injection risks by treating LLM-provided arguments as literal values substituted into a fixed argument array.

Consider exposing a constrained `git push` operation. In your global `config.yaml`:

```yaml
tools:
  - name: git_push
    requirements:
      supervision: "args.branch == 'main'"
    binary:
      executable: git
      args: ["push", "origin", "{branch}"]
      inputs:
        branch:
          type: string
          description: "The git branch to push"
          required: true
```

The `{branch}` placeholder is replaced at runtime with the LLM's supplied input. Because there is no shell interpolation, a malicious input such as `main; rm -rf /` is passed literally to Git and rejected safely.

Once defined globally, reference the tool by name inside any agent mode:

```yaml
context:
  tools:
    - name: git_push
      enabled: true
```

## Integrating MCP Servers

Remote tools provided via the Model Context Protocol (MCP) can be attached directly to a mode without global pre-registration. This is useful for ephemeral or agent-specific integrations, such as a filesystem server scoped to a particular project directory.

```yaml
context:
  tools:
    - name: project_files
      enabled: true
      mcp:
        command: "npx"
        args: ["-y", "@modelcontextprotocol/server-filesystem", "/home/user/project"]
```

MCP tools appear alongside native tools in the LLM's function-calling context. Supervision rules apply to them just as they do to built-in bundles.

## Nesting Tools into Reusable Toolsets

As your agent fleet grows, repeating the same three or four tool definitions across every agent becomes unmaintainable. You can define a **toolset**—a parent tool block containing nested tools—in the global configuration, then reference the parent name in agents.

In `config.yaml`:

```yaml
tools:
  - name: infrastructure
    tools:
      - name: kubectl_get_pods
        binary:
          executable: kubectl
          args: ["get", "pods", "-n", "{namespace}"]
          inputs:
            namespace:
              type: string
              description: "Kubernetes namespace"
              required: true
      - name: docker_status
        binary:
          executable: docker
          args: ["ps"]
```

Then, in an agent mode:

```yaml
context:
  tools:
    - name: infrastructure
      enabled: true
```

During initialization, `dux` recursively flattens the `infrastructure` node and registers each leaf tool individually, while still respecting any supervision policies defined at either the parent or child level.

## Complete Example

The following `agent.yaml` illustrates a coding agent with graduated privileges across workflow modes. The planning mode receives only safe utilities, the execution mode gains filesystem and bash access with selective autonomy, and the review mode can run tests unsupervised but must approve all other shell commands.

```yaml
name: coder
provider: ollama-local
workflow:
  default_mode: aide
  modes:
    - name: aide
      context:
        system: |
          You are Aide, an expert software engineer. Gather requirements, then delegate.
      transitions:
        - to: planning
          type: delegation
        - to: execution
          type: delegation

    - name: planning
      context:
        system: |
          You are a planning specialist. Create and revise architectural plans.
        tools:
          - name: workspace_plans
            enabled: true

    - name: execution
      context:
        system: |
          You are an execution specialist. Implement the approved plan.
        tools:
          - name: stdlib
            enabled: true
            requirements:
              supervision: false
          - name: filesystem
            enabled: true
            requirements:
              supervision: false
          - name: bash
            enabled: true
            requirements:
              supervision: "!(args.command.startsWith('go test') || args.command.startsWith('make test'))"

    - name: review
      context:
        system: |
          You are a code reviewer. Verify implementation quality.
        tools:
          - name: bash
            enabled: true
            requirements:
              supervision: "!(args.command.startsWith('go test') || args.command.startsWith('make test'))"
      transitions:
        - type: return
```

Notice how the same `bash` bundle is reused across modes with different CEL policies. This demonstrates the power of the two-level model: the bundle is defined once by the system, but its operational constraints are contextual to each mode's risk profile.
