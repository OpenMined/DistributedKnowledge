# LLM Configuration

This directory contains configuration files for various LLM providers that can be used with the application.

## Model Configuration

The application supports multiple LLM providers through a flexible configuration system. You can configure which provider to use by creating a JSON file with the following structure:

```json
{
  "provider": "openai",
  "api_key": "your-api-key-here",
  "model": "gpt-3.5-turbo",
  "base_url": "https://api.openai.com/v1",
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 1000
  },
  "headers": {
    "custom-header": "value"
  }
}
```

### Required Fields

- `provider`: The LLM provider to use. Currently supported values are:
  - `openai` - OpenAI API (ChatGPT, GPT-4, etc.)
  - `anthropic` - Anthropic API (Claude)
  - `ollama` - Ollama for local models

- `model`: The specific model to use from the provider.

### Optional Fields

- `api_key`: API key for authentication (required for cloud providers).
- `base_url`: Override the default API endpoint URL.
- `parameters`: Provider-specific parameters like temperature and max_tokens.
- `headers`: Custom HTTP headers to include with API requests.

## Example Configurations

This directory includes example configuration files for different providers:

- `model_config.json` - OpenAI configuration
- `model_config_anthropic.json` - Anthropic (Claude) configuration
- `model_config_ollama.json` - Ollama (local models) configuration

## Usage

To use a specific configuration, provide the path to the configuration file via the `--model-config` flag when starting the application:

```bash
./dk --model-config=./config/model_config.json
```

You can create multiple configuration files and switch between them as needed.