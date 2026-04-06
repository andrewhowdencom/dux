# Reference: Tools

Tools allow agents to perform actions outside of their core generative capabilities, such as interacting with the file system, running commands, or accessing contextual metadata. 

This document serves as a reference for all built-in tools available natively within `dux`.

## Available Tools

### System & Environment

#### `bash`
Executes an arbitrary bash command and returns its standard output and standard error.
- **Constraints**: Should generally be configured with `supervision: true` to require human-in-the-loop approval before mutating systemic state.

#### `time`
Returns the current local and UTC times.
- **Constraints**: Safe for unbounded usage.

### File System Operations

#### `file_list`
Lists files and directories at a specified path.
- **Behavior**: Traverses directory trees up to a maximum item limit (default 1000) to prevent token exhaustion. By default, hidden directories (prefixed with `.`) are ignored unless `include_hidden` is set to true.

#### `file_read`
Reads the contents of a specified file.
- **Behavior**: Reads up to 800 lines by default. Supports `start_line` and `end_line` parameters for pagination on large files. Features built-in binary detection and will return an error if a non-text format is encountered. 

#### `file_write`
Creates a new file or completely overwrites an existing file with the provided content.
- **Behavior**: Automatically creates intermediate parent directories (`mkdir -p` behavior) if they do not exist.
- **Constraints**: Typically configured with `supervision: true`.

#### `file_patch`
Edits an existing file by replacing a specific snippet of text.
- **Behavior**: The `original_snippet` must exactly match the text in the file. Cross-platform robustness normalizes both file content and prompt inputs (`\r\n` to `\n`) before replacement to prevent line-ending match failures. If the snippet count does not equal exactly 1, the patch fails to prevent ambiguous or destructive partial modifications.
- **Constraints**: Typically configured with `supervision: true`.

## Tool Bundles & Supervision (CEL)

With `dux` agents configuration, tools natively group together into broader **namespaces** or **bundles** (`stdlib`, `filesystem`, `semantic`). Instead of enabling `file_read` and `file_write` individually, simply specify `name: "filesystem"` in your agent tools configuration. 

You can control execution privileges per tool bundle securely using [Common Expression Language (CEL)](https://github.com/google/cel-spec).

### Example Configuration
```yaml
tools:
  - name: "filesystem"
    enabled: true
    requirements:
      # Evaluates to true/false using `tool_name` or even tool `args`
      supervision: "tool_name == 'file_write' || tool_name == 'file_patch'"
```

By default:
- Unmapped tools or invalid CEL expressions fail-secure to `supervision: true`.
- Evaluated runtime policies provide safe granular automation over destructive processes.
