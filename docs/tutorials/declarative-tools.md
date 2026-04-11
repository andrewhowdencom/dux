# Tutorial: Creating a Declarative Binary Tool

In this tutorial, you will learn how to enable an AI agent to execute local CLI programs (like `git` or `kubectl`) securely without writing any Go code. By the end, we will have created a reusable "Toolset" that safely exposes Git capabilities.

## Introduction

Agents usually interact with the terminal through a generic `bash` tool. While powerful, this gives the agent unbounded access and exposes the system to potential shell injection vulnerabilities if the AI hallucinates bad arguments.

By defining "Declarative Binary Tools", you strictly bound the capabilities. You define the executable (`git`), the fixed arguments (`push`), and precisely what dynamic inputs the LLM is allowed to provide (`branch`).

## Step 1: Defining the Tool Schema

Open your global `dux` configuration file (e.g. `~/.config/dux/config.yaml`).

We will define a new tool called `git_push`. Add the following YAML:

```yaml
tools:
  - name: git_push
    binary:
      executable: git
      args: ["push", "origin", "{branch}"]
      inputs:
        branch:
          type: string
          description: "The name of the git branch to push"
          required: true
```

### What happens here?
- **`executable`**: We instruct `dux` to run the `git` binary.
- **`inputs`**: We inform the LLM that this tool requires exactly one string, the `branch`.
- **`args`**: The specific POSIX argument array sent to the binary. The `{branch}` placeholder will be substituted at runtime with whatever the LLM determined the branch was. 

Because we bypassed the shell entirely, a malicious branch name of `main; rm -rf /` is inherently treated as literal text by Git, avoiding destructive system commands!

## Step 2: Adding Human-in-the-Loop Supervision

Pushing to `main` is destructive. We can write a specific policy using Common Expression Language (CEL) to ensure a human approves any pushes to `main`.

Update your configuration:

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
          description: "The name of the git branch to push"
          required: true
```

If the LLM issues a push to `main`, `dux` will suspend execution and prompt the user. For a branch like `feature/update`, execution happens automatically.

## Step 3: Grouping Tools into a "Toolset"

As you add more Git commands (like `git_status`, `git_commit`), configuring them individually becomes tedious. You can group them via hierarchical nesting inside the `tools` array to create a Toolset.

```yaml
tools:
  - name: git_operations
    tools:
      - name: git_status
        binary:
          executable: git
          args: ["status"]
      - name: git_push
        requirements:
          supervision: "args.branch == 'main'"
        binary:
          # ... (as defined above)
```

## Step 4: Activating the Toolset on an Agent

Finally, configure an Agent to use this Toolset. Open your agent's configuration file (e.g. `agents/coding_agent/agent.yaml`):

```yaml
name: coding_agent
provider: openai
context:
  tools:
    - name: stdlib
    - name: git_operations
```

When you interact with `coding_agent`, `dux` will dynamically flatten the `git_operations` configuration and provision `git_status` and `git_push` securely!
