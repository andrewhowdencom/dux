# Tutorial: OpenAI-Compatible Providers

This tutorial shows you how to use any OpenAI-compatible API provider with Dux. The `openai` provider type works with OpenAI, OpenRouter, Fireworks AI, LiteLLM, and any endpoint implementing the OpenAI Chat Completions API specification.

By using an OpenAI-compatible provider, you can easily switch between different LLM services without changing your Dux configuration structure.

## 1. Understanding OpenAI-Compatible Providers

Dux uses the `type: "openai"` provider to communicate with any service that implements the OpenAI API specification. This includes:

- **OpenAI** - Official OpenAI API (api.openai.com)
- **OpenRouter** - Multi-provider routing platform (openrouter.ai)
- **Fireworks AI** - Fast inference platform (fireworks.ai)
- **LiteLLM** - Local proxy for 100+ LLMs (docs.litellm.ai)
- **Any OpenAI-compatible endpoint** - Self-hosted or other providers

All these providers share the same configuration structure - you just change the `base_url`, `api_key`, and `model` parameters.

For a complete reference of supported providers and their capabilities, see [LLM Providers Reference](../reference/llm-providers.md).

## 2. Setting Up a Provider

### Option A: Using OpenRouter

OpenRouter provides access to 300+ models through a single API key.

1. Get an API key from [openrouter.ai](https://openrouter.ai/)
2. Configure Dux:

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

### Option B: Using Fireworks AI

Fireworks AI offers fast inference on dedicated GPUs.

1. Get an API key from [fireworks.ai](https://fireworks.ai/)
2. Configure Dux:

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

### Option C: Using LiteLLM Proxy

LiteLLM is a local proxy that standardizes access to 100+ LLM providers.

1. Install LiteLLM:
```bash
pipx install litellm
```

2. Start the proxy with your chosen model:
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
litellm --model anthropic/claude-3-opus-20240229
```

This runs a local OpenAI-compatible endpoint at `http://0.0.0.0:4000`

3. Configure Dux:
```yaml
llm:
  default_provider: "litellm-proxy"
  
  providers:
    - id: "litellm-proxy"
      type: "openai"
      config:
        base_url: "http://0.0.0.0:4000/v1"
        model: "anthropic/claude-3-opus-20240229"
        api_key: "any"
```

!!! note
    Since Dux uses standard OpenAI abstractions when `type: "openai"` is provided, you need to append `/v1` to the localhost port for the endpoints to resolve correctly.

### Option D: Using OpenAI Directly

1. Get an API key from [platform.openai.com](https://platform.openai.com/)
2. Configure Dux:

```yaml
llm:
  default_provider: "openai"
  
  providers:
    - id: "openai"
      type: "openai"
      config:
        base_url: "https://api.openai.com/v1"
        api_key: "sk-..."
        model: "gpt-4o"
```

## 3. Configuring Dux

Locate your configuration file (typically `~/.config/dux/config.yaml` or `$XDG_CONFIG_HOME/dux/config.yaml`). Add or modify the `llm.providers` section with your chosen provider configuration from the examples above.

The key fields are:
- **`base_url`** - The API endpoint URL (must end with `/v1` for proper routing)
- **`api_key`** - Your API authentication key
- **`model`** - The target model identifier (format varies by provider)

## 4. Using the Provider

With the config saved, start Dux using the `chat` command:

```bash
# Use the default provider
dux chat

# Or explicitly specify a provider
dux chat --provider="openrouter"
```

Dux will print:
`Starting interactive chat using provider: openrouter (openai). Press Ctrl+D/Ctrl+C to exit.`

You are now connected to your chosen LLM provider. All chat history handling, recursion loops, and tool abstractions are managed by Dux while the provider handles LLM execution.

## 5. Switching Providers

One of the key benefits of using OpenAI-compatible providers is easy switching. To try a different provider:

1. Add another provider block to your config:

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
    
    - id: "fireworks"
      type: "openai"
      config:
        base_url: "https://api.fireworks.ai/inference/v1"
        api_key: "your-fireworks-api-key"
        model: "accounts/fireworks/models/llama-v3p1-8b-instruct"
```

2. Switch between them using the `--provider` flag:
```bash
dux chat --provider="fireworks"
```

## Next Steps

- Explore different models and providers to find the best fit for your use case
- Check out [LLM Providers Reference](../reference/llm-providers.md) for detailed provider capabilities
- Learn how to [configure agents](../how-to/configuring-agents.md) with specific providers
- Set up [custom tools](../how-to/custom-tools.md) to extend agent capabilities
