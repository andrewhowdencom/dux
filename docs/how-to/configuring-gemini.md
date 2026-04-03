# How-To: Configuring Gemini Provider

Dux has native support for Google's Gemini API via the official `google.golang.org/genai` SDK. This allows you to communicate directly with Google AI Studio using your API key.

## Adding the Provider

Open your core configuration file (e.g., `~/.config/dux/config.yaml` or `$XDG_CONFIG_HOME/dux/config.yaml`). Append a `gemini` provider to the `providers` list under the `llm` section.

```yaml
llm:
  default_provider: "gemini-api"

  providers:
    # A standard Gemini API deployment targeting Google AI Studio
    - id: "gemini-api"
      type: "gemini"
      config:
        api_key: "AIzaSy..." # Place your own Google AI Studio API key here!
        
        # Optional: override the default target model parameter.
        # This will default to `gemini-3-flash-preview` if left blank.
        model: "gemini-2.5-pro"
```

### Supported Configuration Keys

When specifying a Gemini block (i.e. `type: "gemini"`), you can map the following fields within the inner `config` object:

*   `api_key` (string, required): Your valid Google AI Studio API Key.
*   `model` (string, optional): The target Gemini model name (e.g., `gemini-1.5-pro`, `gemini-2.5-flash`, `gemini-3-flash-preview`).

## Using the Provider in the CLI

Once your configuration is saved, start Dux using the `chat` subcommand. Ensure your `--provider` matches the configured `id` you specified above. 

```bash
# Explicitly use the Gemini router
dux chat --provider="gemini-api"
```

Dux will print:
`Starting interactive chat using provider: gemini-api (gemini). Press Ctrl+D/Ctrl+C to exit.`

Enjoy chatting with Google's state-of-the-art models!
