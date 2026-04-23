# Reference: Tools

Tools allow agents to perform actions outside of their core generative capabilities, such as interacting with the file system, running commands, or accessing contextual metadata.

This document serves as a reference for all built-in tools available natively within `dux`. Tools are organized by **bundle** (namespace). In your agent configuration you can enable an entire bundle at once instead of listing each tool individually.

## Tool Bundles Overview

| Bundle | Description | Tools |
|--------|-------------|-------|
| `stdlib` | General-purpose utilities (time, math, encoding, etc.) | `get_current_time`, `get_current_date`, `timer`, `stopwatch`, `sleep`, `evaluate_math`, `generate_uuid`, `generate_random_number`, `base64_encode`, `base64_decode`, `url_encode`, `url_decode` |
| `filesystem` | Read and mutate files and directories | `file_read`, `file_write`, `file_patch`, `file_list`, `file_search` |
| `bash` | Execute arbitrary shell commands | `bash` |
| `workspace_plans` | Create and manage architectural plans in the session workspace | `plan_create`, `plan_read`, `plan_update`, `plan_list`, `plan_approve` |
| `librarian` | Access global orchestrator memory | `read_working_memory` |
| `semantic` | Read and write to the semantic knowledge graph | `semantic_write_triple`, `semantic_write_statement`, `semantic_read`, `semantic_search`, `semantic_delete`, `semantic_validate`, `semantic_create_relationship`, `semantic_traverse_graph` |

---

## `stdlib`

General-purpose utilities that are safe for unbounded usage.

| Tool | Description | Supervision |
|------|-------------|-------------|
| `get_current_time` | Returns the current system time in RFC3339 format. | — |
| `get_current_date` | Returns the current system date in YYYY-MM-DD format. | — |
| `timer` | Sets a blocking timer for a given number of seconds. | — |
| `stopwatch` | Starts, stops, or checks the status of a named stopwatch. | — |
| `sleep` | Pauses execution for a specific number of milliseconds. | — |
| `evaluate_math` | Safely evaluates a mathematical expression string. | — |
| `generate_uuid` | Generates a UUID v4 string. | — |
| `generate_random_number` | Produces a random integer between a min (inclusive) and max (exclusive). | — |
| `base64_encode` | Base64 encodes a string. | — |
| `base64_decode` | Base64 decodes a string. | — |
| `url_encode` | URL encodes a string for query parameters. | — |
| `url_decode` | URL decodes a previously escaped string. | — |

### `get_current_time`

**Parameters:**
```json
{"type":"object","properties":{}}
```

### `get_current_date`

**Parameters:**
```json
{"type":"object","properties":{}}
```

### `timer`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "duration_seconds": {
      "type": "integer",
      "description": "The duration of the timer in seconds."
    }
  },
  "required": ["duration_seconds"]
}
```

### `stopwatch`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "action": {
      "type": "string",
      "enum": ["start", "stop", "status"],
      "description": "The action to perform on the stopwatch."
    },
    "name": {
      "type": "string",
      "description": "The name of the stopwatch."
    }
  },
  "required": ["action", "name"]
}
```

### `sleep`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "milliseconds": {
      "type": "integer",
      "description": "The amount of time in milliseconds to halt execution. Do not exceed 10000 (10s) without good reason."
    }
  },
  "required": ["milliseconds"]
}
```

### `evaluate_math`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "expression": {
      "type": "string",
      "description": "The mathematical expression to evaluate string (e.g. 3 * 4)"
    }
  },
  "required": ["expression"]
}
```

### `generate_uuid`

**Parameters:**
```json
{"type":"object","properties":{}}
```

### `generate_random_number`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "min": {
      "type": "integer",
      "description": "The minimum value, inclusive."
    },
    "max": {
      "type": "integer",
      "description": "The maximum value, exclusive. Must be higher than min."
    }
  },
  "required": ["min", "max"]
}
```

### `base64_encode`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "text": {
      "type": "string"
    }
  },
  "required": ["text"]
}
```

### `base64_decode`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "encoded": {
      "type": "string"
    }
  },
  "required": ["encoded"]
}
```

### `url_encode`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "text": {
      "type": "string"
    }
  },
  "required": ["text"]
}
```

### `url_decode`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "encoded": {
      "type": "string"
    }
  },
  "required": ["encoded"]
}
```

---

## `filesystem`

Read and mutate files and directories.

| Tool | Description | Supervision |
|------|-------------|-------------|
| `file_read` | Reads the contents of a file (up to 800 lines, with pagination). | — |
| `file_write` | Creates or completely overwrites a file. | Recommended |
| `file_patch` | Edits an existing file by replacing an exact snippet of text. | Recommended |
| `file_list` | Lists files and directories at a path (skips hidden by default). | — |
| `file_search` | Searches for text or regex in a file or directory recursively. | — |

### `file_read`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute or relative path to the file to read."
    },
    "start_line": {
      "type": "integer",
      "description": "The line number to start reading from (1-indexed). Inclusive."
    },
    "end_line": {
      "type": "integer",
      "description": "The line number to end reading at (1-indexed). Inclusive."
    }
  },
  "required": ["path"]
}
```

### `file_write`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute or relative path to the file to write to."
    },
    "content": {
      "type": "string",
      "description": "The exact string content to write to the file."
    }
  },
  "required": ["path", "content"]
}
```

### `file_patch`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute or relative path to the file to edit."
    },
    "original_snippet": {
      "type": "string",
      "description": "The exact text to find and replace. Must match the target file perfectly."
    },
    "replacement_snippet": {
      "type": "string",
      "description": "The new text that will replace original_snippet."
    }
  },
  "required": ["path", "original_snippet", "replacement_snippet"]
}
```

### `file_list`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute or relative path to the directory to list."
    },
    "include_hidden": {
      "type": "boolean",
      "description": "If true, hidden directories and files (like .git or .env) will be traversed and listed. Defaults to false."
    }
  },
  "required": ["path"]
}
```

### `file_search`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "Absolute or relative path to the directory or file to search in."
    },
    "query": {
      "type": "string",
      "description": "The exact string or regular expression pattern to search for."
    },
    "is_regex": {
      "type": "boolean",
      "description": "If true, treats the query as a regular expression. Defaults to false (exact string match)."
    },
    "include_hidden": {
      "type": "boolean",
      "description": "If true, hidden directories and files (like .git or .env) will be searched. Defaults to false."
    }
  },
  "required": ["path", "query"]
}
```

---

## `bash`

Execute arbitrary shell commands.

| Tool | Description | Supervision |
|------|-------------|-------------|
| `bash` | Executes an arbitrary bash command and returns stdout, stderr, and exit code. | Recommended |

### `bash`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "command": {
      "type": "string",
      "description": "The bash command to execute."
    }
  },
  "required": ["command"]
}
```

---

## `workspace_plans`

Create and manage architectural plans in the session workspace.

| Tool | Description | Supervision |
|------|-------------|-------------|
| `plan_create` | Create a new persistent architectural plan document. | — |
| `plan_read` | Read an existing plan from the session workspace. | — |
| `plan_update` | Overwrite an existing plan. | — |
| `plan_list` | List all existing plans in the session workspace. | — |
| `plan_approve` | Mark a plan as approved after user review. | Recommended |

### `plan_create`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "title": {
      "type": "string"
    },
    "content": {
      "type": "string",
      "description": "The full markdown content of the plan."
    }
  },
  "required": ["title", "content"]
}
```

### `plan_read`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "plan_id": {
      "type": "string"
    }
  },
  "required": ["plan_id"]
}
```

### `plan_update`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "plan_id": {
      "type": "string"
    },
    "content": {
      "type": "string",
      "description": "The complete markdown content to save."
    }
  },
  "required": ["plan_id", "content"]
}
```

### `plan_list`

**Parameters:**
```json
{"type":"object","properties":{}}
```

### `plan_approve`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "plan_id": {
      "type": "string",
      "description": "The ID of the plan to approve."
    }
  },
  "required": ["plan_id"]
}
```

---

## `librarian`

Access global orchestrator memory to discover missing context.

| Tool | Description | Supervision |
|------|-------------|-------------|
| `read_working_memory` | Read the overarching conversation history from the global Orchestrator's working memory. | — |

### `read_working_memory`

**Parameters:**
```json
{"type":"object","properties":{}}
```

---

## `semantic`

Read and write to the semantic knowledge graph (long-term memory).

| Tool | Description | Supervision |
|------|-------------|-------------|
| `semantic_write_triple` | Save a structured fact (entity-attribute-value triple) to long-term memory. | — |
| `semantic_write_statement` | Save a freeform statement to long-term memory. | — |
| `semantic_read` | Read a specific fact by ID from long-term memory. | — |
| `semantic_search` | Search long-term memory for facts matching criteria. | — |
| `semantic_delete` | Delete a fact from long-term memory by ID. | Recommended |
| `semantic_validate` | Mark a fact as validated, updating its validated_at timestamp. | — |
| `semantic_create_relationship` | Create a relationship edge between two entities in the knowledge graph. | — |
| `semantic_traverse_graph` | Traverse the knowledge graph from a starting entity to find related facts. | — |

### `semantic_write_triple`

**Parameters:**
```json
{
  "type": "object",
  "required": ["entity", "attribute", "value"],
  "properties": {
    "entity": {
      "type": "string",
      "description": "The subject of the fact"
    },
    "attribute": {
      "type": "string",
      "description": "The property being set"
    },
    "value": {
      "type": "string",
      "description": "The value to remember"
    },
    "tags": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Optional tags"
    },
    "source_uri": {
      "type": "string",
      "description": "URI of the source"
    },
    "constraints": {
      "type": "object",
      "additionalProperties": {
        "type": "string"
      },
      "description": "Optional key-value constraints for fact relevance"
    }
  }
}
```

### `semantic_write_statement`

**Parameters:**
```json
{
  "type": "object",
  "required": ["statement"],
  "properties": {
    "statement": {
      "type": "string",
      "description": "The statement to remember"
    },
    "tags": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Optional tags"
    },
    "source_uri": {
      "type": "string",
      "description": "URI of the source"
    },
    "constraints": {
      "type": "object",
      "additionalProperties": {
        "type": "string"
      },
      "description": "Optional key-value constraints for fact relevance"
    }
  }
}
```

### `semantic_read`

**Parameters:**
```json
{
  "type": "object",
  "required": ["id"],
  "properties": {
    "id": {
      "type": "string",
      "description": "The fact ID to read"
    }
  }
}
```

### `semantic_search`

**Parameters:**
```json
{
  "type": "object",
  "properties": {
    "type": {
      "type": "string",
      "description": "Filter by fact type (triple or statement)"
    },
    "entity": {
      "type": "string",
      "description": "Filter by entity"
    },
    "attribute": {
      "type": "string",
      "description": "Filter by attribute"
    },
    "value": {
      "type": "string",
      "description": "Filter by value"
    },
    "tag": {
      "type": "string",
      "description": "Filter by tag"
    },
    "statement": {
      "type": "string",
      "description": "Search in statements"
    },
    "constraints": {
      "type": "object",
      "description": "Filter by fact constraints"
    },
    "limit": {
      "type": "integer",
      "description": "Max results"
    },
    "sort_by": {
      "type": "string",
      "description": "Sort field"
    },
    "sort_order": {
      "type": "string",
      "description": "Sort order (asc or desc)"
    }
  }
}
```

### `semantic_delete`

**Parameters:**
```json
{
  "type": "object",
  "required": ["id"],
  "properties": {
    "id": {
      "type": "string",
      "description": "The fact ID to delete"
    }
  }
}
```

### `semantic_validate`

**Parameters:**
```json
{
  "type": "object",
  "required": ["id"],
  "properties": {
    "id": {
      "type": "string",
      "description": "The fact ID to validate"
    }
  }
}
```

### `semantic_create_relationship`

**Parameters:**
```json
{
  "type": "object",
  "required": ["subject", "predicate", "object"],
  "properties": {
    "subject": {
      "type": "string",
      "description": "The source entity (e.g., 'person:john-doe')"
    },
    "predicate": {
      "type": "string",
      "description": "The relationship type (e.g., 'has_condition', 'works_at')"
    },
    "object": {
      "type": "string",
      "description": "The target entity (e.g., 'condition:pvcs')"
    }
  }
}
```

### `semantic_traverse_graph`

**Parameters:**
```json
{
  "type": "object",
  "required": ["start_entity"],
  "properties": {
    "start_entity": {
      "type": "string",
      "description": "The entity to start traversal from (e.g., 'person:john-doe')"
    },
    "predicates": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Optional filter for relationship types to follow"
    },
    "max_depth": {
      "type": "integer",
      "description": "Maximum traversal depth (default: 3)"
    },
    "max_results": {
      "type": "integer",
      "description": "Maximum number of nodes to return (default: 50)"
    }
  }
}
```

---

## Declarative Binary Tools

`dux` allows defining custom CLI tools declaratively via configuration, allowing agents to execute local binaries securely without going through a standard Unix shell.

These tools are defined in the globally available `tools` configuration block (or directly on an agent) using the `binary` mapping.

**Configuration Schema:**
- `executable` (string): The literal name or absolute path of the binary (e.g. `git`, `docker`, `/usr/bin/kubectl`).
- `args` ([]string): An exact array of arguments passed to the executable. Supports `{key}` substitutions matching the defined `inputs` (e.g. `["push", "origin", "{branch}"]`).
- `inputs` (map): A dictionary mapping input parameter names to their JSON Schema definitions:
  - `type` (string): The data type, e.g. `"string"`.
  - `description` (string): Context provided to the LLM about what this parameter controls.
  - `required` (boolean): Whether the LLM must provide this argument.

**Example Definition:**
```yaml
tools:
  - name: my_docker_build
    requirements:
      supervision: "args.tag == 'latest'"
    binary:
      executable: docker
      args: ["build", "-t", "{tag}", "."]
      inputs:
        tag:
          type: string
          description: "Image tag to build"
          required: true
```

---

## Tool Bundles & Supervision (CEL)

With `dux` agents configuration, tools natively group together into broader **namespaces** or **bundles** (`stdlib`, `filesystem`, `semantic`). Instead of enabling `file_read` and `file_write` individually, simply specify `name: "filesystem"` in your agent tools configuration.

You can control execution privileges per tool bundle securely using [Common Expression Language (CEL)](https://github.com/google/cel-spec).

### Example Configuration
```yaml
tools:
  - name: "filesystem"
    enabled: true
    requirements:
      # Evaluates to true/false using tool_name or even tool args
      supervision: "tool_name == 'file_write' || tool_name == 'file_patch'"
```

By default:
- Unmapped tools or invalid CEL expressions fail-secure to `supervision: true`.
- Evaluated runtime policies provide safe granular automation over destructive processes.
