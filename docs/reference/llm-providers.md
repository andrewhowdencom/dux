# Reference: LLM Providers

Dux supports multiple LLM providers through a unified interface. Providers are configured in your `config.yaml` file and referenced by their `id` in agents or CLI commands.

## Provider Configuration

Providers are defined under the `llm.providers` section of your configuration:

```yaml
llm:
  default_provider: "provider-id"
  
  providers:
    - id: "provider-id"
      type: "provider-type"
      config:
        # Provider-specific configuration keys
```

Each provider block requires:
- **`id`** (string): Unique identifier used to reference this provider in agents or CLI commands
- **`type`** (string): The provider implementation type (see table below)
- **`config`** (map): Provider-specific configuration parameters

## Supported Provider Types

| Provider Type | Description | Use Case |
|--------------|-------------|----------|
| `openai` | OpenAI API or compatible endpoints | Production LLM inference |
| `gemini` | Google Gemini API | Production LLM inference |
| `ollama` | Local Ollama instances | Local development/testing |
| `static` | Mock provider with canned responses | Testing/development |

## OpenAI-Compatible Providers

The `openai` provider type supports any API endpoint that implements the OpenAI Chat Completions API specification. This includes:

- **OpenAI** - Official OpenAI API
- **OpenRouter** - Multi-provider routing platform
- **Fireworks AI** - Fast inference platform for open source models
- **LiteLLM** - Local proxy for 100+ LLMs
- **Any OpenAI-compatible endpoint**

### Generic Configuration

All OpenAI-compatible providers share the same configuration structure:

```yaml
llm:
  providers:
    - id: "my-provider"
      type: "openai"
      config:
        base_url: "https://api.example.com/v1"  # API endpoint
        api_key: "your-api-key"                  # API authentication key
        model: "model-name"                      # Target model identifier
```

### Provider-Specific Examples

#### OpenRouter

[OpenRouter](https://openrouter.ai/) provides unified access to 300+ AI models through a single API with intelligent routing, fallbacks, and cost optimization.

```yaml
llm:
  default_provider: "openrouter"
  
  providers:
    - id: "openrouter"
      type: "openai"
      config:
        base_url: "https://openrouter.ai/api/v1"
        api_key: "sk-or-v1-..."
        model: "google/gemini-3-flash-preview"
```

**Features:**
- Access to 300+ models from multiple providers
- Automatic model routing and fallbacks
- Web search plugins
- Response healing for malformed JSON
- Usage tracking and cost analytics

**Limitations:**
- Requires API key from OpenRouter
- Advanced features (plugins, model routing, service tiers) require additional request parameters not exposed through basic Dux configuration
- Model availability and pricing vary by provider

**Model Selection:** Models use the format `provider/model-name` (e.g., `anthropic/claude-sonnet-4.5`, `openai/gpt-4o`, `google/gemini-3-flash-preview`).

#### Fireworks AI

[Fireworks AI](https://fireworks.ai/) offers fast inference and fine-tuning for open source models with dedicated GPU deployments.

```yaml
llm:
  default_provider: "fireworks"
  
  providers:
    - id: "fireworks"
      type: "openai"
      config:
        base_url: "https://api.fireworks.ai/inference/v1"
        api_key: "your-fireworks-api-key"
        model: "accounts/fireworks/models/llama-v3p1-8b-instruct"
```

**Features:**
- Fast inference on dedicated GPUs
- Full OpenAI API compatibility
- Tool calling support
- Structured outputs
- Embeddings support
- Vision model support

**Model Naming:**
Fireworks uses the format `accounts/fireworks/models/<model-name>` for all models. Browse available models at [fireworks.ai/models](https://fireworks.ai/models).

**Limitations:**
- Requires API key from Fireworks AI
- Advanced features (batch inference, fine-tuning, dedicated deployments) not exposed through basic Dux configuration
- Model availability depends on Fireworks AI catalog

#### LiteLLM

[LiteLLM](https://docs.litellm.ai/) is a local proxy that standardizes interactions across 100+ LLMs into an OpenAI-compatible API.

```yaml
llm:
  default_provider: "litellm"
  
  providers:
    - id: "litellm"
      type: "openai"
      config:
        base_url: "http://localhost:4000/v1"
        api_key: "any"
        model: "anthropic/claude-3-opus-20240229"
```

**Features:**
- Local proxy for 100+ LLM providers
- Standardized OpenAI-compatible API
- No external API key required (uses underlying provider's key)
- Ideal for testing and development

**Limitations:**
- Requires running LiteLLM proxy locally
- Performance depends on underlying provider
- Advanced LiteLLM features (load balancing, caching) not exposed through Dux

### Supported Features

All OpenAI-compatible providers support:

- ✅ Streaming responses (SSE)
- ✅ Tool/function calling
- ✅ Structured outputs (JSON mode)
- ✅ System prompts
- ✅ Temperature and max tokens control
- ✅ Embeddings
- ✅ Model listing

## Other Provider Types

### Gemini

Dux has native support for Google's Gemini API via the official `google.golang.org/genai` SDK.

See [Configuring Gemini](../how-to/configuring-gemini.md) for detailed setup instructions.

**Basic Configuration:**
```yaml
llm:
  providers:
    - id: "gemini"
      type: "gemini"
      config:
        api_key: "AIzaSy..."
        model: "gemini-3-flash-preview"  # Optional, defaults to gemini-3-flash-preview
```

### Ollama

Ollama provides local LLM inference, ideal for development and testing without external API dependencies.

**Basic Configuration:**
```yaml
llm:
  providers:
    - id: "ollama-local"
      type: "ollama"
      config:
        address: "http://localhost:11434"  # Optional, defaults to localhost:11434
        model: "llama3"                     # Optional, defaults to llama3
        num_ctx: 8192                       # Optional, context window size
```

**Features:**
- Local inference, no API key required
- Supports tool calling
- Supports structured outputs
- Supports system prompts
- Full privacy (data stays local)

**Limitations:**
- Requires Ollama daemon running locally
- Image input not supported
- Performance depends on local hardware

### Static

The static provider returns canned responses for testing and development.

**Basic Configuration:**
```yaml
llm:
  providers:
    - id: "static-fallback"
      type: "static"
      config:
        text: "This is a canned response for testing."
```

**Features:**
- Deterministic responses for testing
- No external dependencies
- Supports embeddings (returns mock vectors)
- Supports model listing (returns `["static-model"]`)

**Limitations:**
- No actual LLM inference
- All capabilities disabled
- Only useful for testing/development

## Provider Capabilities Matrix

Different providers support different features. Use this matrix to understand what's available:

| Capability | OpenAI | OpenRouter | Fireworks | Gemini | Ollama | Static |
|-----------|--------|------------|-----------|--------|--------|--------|
| System Prompts | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Tool Calling | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Image Input | ✅ | ✅* | ✅* | ✅ | ❌ | ❌ |
| Structured Output | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Embeddings | ✅ | ✅ | ✅ | ✅ | ✅ | ✅** |
| Model Listing | ✅ | ✅ | ✅ | ✅ | ✅ | ✅** |

*Depends on model support
**Mock implementations for testing

## Related Documentation

- [OpenAI-Compatible Providers Tutorial](../tutorials/openai-compatible.md) - Step-by-step setup for OpenAI, OpenRouter, Fireworks AI, and LiteLLM
- [Configuring Gemini](../how-to/configuring-gemini.md) - Detailed Gemini provider setup
- [Configuring Agents](../how-to/configuring-agents.md) - How to use providers in agent definitions
- [CLI Reference](./cli.md) - Command-line interface documentation
