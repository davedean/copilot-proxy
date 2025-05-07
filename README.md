# Copilot Chat Proxy

A simple proxy that lets you use GitHub Copilot Chat with OpenAI-compatible tools.

GitHub Copilot Chat follows the OpenAI API standard, but its authentication uses a unique token mechanism that most applications can't handle directly. This proxy manages authentication for you so that you can use it with compatible tools such as [codex](https://github.com/openai/codex).

## Features

- Authenticate with GitHub using device flow
- Persistent token storage (no need to re-authenticate each time)
- CLI-only mode for quick access to tokens
- Compatible with OpenAI API tools

## Usage

### Web Interface

- Run the proxy: `go run .`
- Go to <http://127.0.0.1:8080> and log in with GitHub
- You will get a GitHub token as an API key

### CLI Mode

If you've authenticated before, you can use CLI-only mode to quickly get your token:

```bash
go run . -cli-only
```

This will display your token and exit without starting the web server.

### Using with OpenAI Codex

```bash
export COPILOT_API_KEY="<your token>"
export COPILOT_BASE_URL="http://127.0.0.1:8080"
codex --provider copilot
```

## Command-line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-listen` | Address to listen on | `127.0.0.1:8080` |
| `-token-file` | Path to token storage file | `copilot_tokens.json` |
| `-cli-only` | Run in CLI-only mode (display token and exit) | `false` |

### Examples

```bash
# Listen on a different port
go run . -listen 127.0.0.1:8923

# Use a custom token file
go run . -token-file ~/.copilot/tokens.json

# Get token without starting web server
go run . -cli-only
```

**Don't want to run it yourself?**

I have it hosted on <https://cope.duti.dev>. (Just replace <http://127.0.0.1:8080> in the instructions with that URL)
