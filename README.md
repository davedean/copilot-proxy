# Copilot Chat Proxy

A simple proxy that lets you use GitHub Copilot Chat with OpenAI-compatible tools.

GitHub Copilot Chat follows the OpenAI API standard, but its authentication uses a unique token mechanism that most applications can't handle directly. This proxy manages authentication for you so that you can use it with compatible tools such as [codex](https://github.com/openai/codex).

## Features

- Authenticate with GitHub using device flow
- Persistent token storage (no need to re-authenticate each time)
- token-only mode for scripted access to tokens
- Compatible with OpenAI API compatible tools

## Usage

### Web Interface

- Run the proxy: `go run .`
- Go to <http://127.0.0.1:8080> and log in with GitHub
- You will get a GitHub token as an API key

### Token Only

If you've authenticated before, you can use token-only mode to quickly get your token:

```bash
go run . -token-only
```

This will display your token and exit without starting the web server.

### Using with OpenAI Codex

```bash
export COPILOT_API_KEY="${copilot-proxy -token-only}"
export COPILOT_BASE_URL="http://127.0.0.1:8080"
codex --provider copilot
```

### Using with Aider

```bash
export OPENAI_API_KEY="${copilot-proxy -token-only}"
export OPENAI_BASE_URL="http://127.0.0.1:8080"
aider -m openai/gpt-4o
```

All models available will be prefixed with openai, like so:

```
tested:
openai/gpt-4o

broken:
openai/claude-3.5-sonnet
openai/gpt-4.1
openai/gemini-2.0-flash-001
```

## Command-line Options

| Flag          | Description                | Default               |
|---------------|----------------------------|-----------------------|
| `-listen`     | Address to listen on       | `127.0.0.1:8080`      |
| `-token-file` | Path to token storage file | `copilot_tokens.json` |
| `-token-only` | Display token and exit     | `false`               |

### Examples

```bash
# Listen on a different port
go run . -listen 127.0.0.1:8923

# Use a custom token file
go run . -token-file ~/.copilot/tokens.json

# Get token without starting web server
go run . -token-only
```

**Don't want to run it yourself?**

I have it hosted on <https://cope.duti.dev>. (Just replace <http://127.0.0.1:8080> in the instructions with that URL)
