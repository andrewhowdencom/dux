# Integrating with LiteLLM

This tutorial focuses on abstracting your model requests securely using the popular [LiteLLM](https://docs.litellm.ai/) proxy. LiteLLM is ideal for standardizing interactions across 100+ LLMs into an elegant, generic OpenAI compatible API format.

By using Dux with LiteLLM, you map all your API requests through a single entry point which removes the requirement of hard-coding provider specifics into your Dux environment.

## 1. Setting up the LiteLLM Proxy

You will need the python package installed locally. Use tools like `pipx` or a python virtual environment to install LiteLLM.

```bash
pipx install litellm
```

### Starting the proxy

To run the proxy, just point it at the underlying provider you've chosen. Let's assume you want to use Anthropic's `claude-3-opus-20240229`. 

First, ensure your API key is in your environment shell:
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

Then, boot the proxy server:
```bash
litellm --model anthropic/claude-3-opus-20240229
```

This will run the generic OpenAI compatible endpoint locally on:
`http://0.0.0.0:4000`

## 2. Configuring Dux

Once your proxy is online and healthy, we need to instruct Dux to route API requests to that endpoint. We can do so by mapping an `openai` provider block in `config.yaml`.

Locate your configuration file (typically `~/.config/dux/config.yaml` or `$XDG_CONFIG_HOME/dux/config.yaml`). Modify or append the following provider list:

```yaml
llm:
  default_provider: "litellm-proxy"

  providers:
    # A standard generic OpenAI wrapper mapping to the local LiteLLM Proxy
    - id: "litellm-proxy"
      type: "openai"
      config:
        base_url: "http://0.0.0.0:4000/v1"
        model: "anthropic/claude-3-opus-20240229"
        api_key: "any" # LiteLLM proxy doesn't require a real OpenAI key by default
```

!!! note
    Since Dux uses standard OpenAI abstractions when the `type: "openai"` is provided, you still need to append `/v1` to the localhost port for the endpoints to resolve correctly.

## 3. Talking to the CLI

With the config saved, start Dux! Use the `chat` command via CLI. Since we set the `default_provider`, running `dux chat` alone is enough. If you want to be explicit, use the `--provider` parameter:

```bash
# Explicitly use the LiteLLM router
dux chat --provider="litellm-proxy"
```

Dux will print:
`Starting interactive chat using provider: litellm-proxy (openai). Press Ctrl+D/Ctrl+C to exit.`

You are now talking directly to your local proxy abstraction. All chat history handling, recursion loops, and provider tool abstractions mapping are entirely delegated under the Dux layer while LiteLLM seamlessly brokers the LLM execution under the hood!
