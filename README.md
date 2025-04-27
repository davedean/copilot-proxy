# Copilot Chat Proxy

A simple proxy that lets you use GitHub Copilot Chat with OpenAI-compatible tools.

GitHub Copilot Chat follows the OpenAI API standard, but its authentication uses a unique token mechanism that most applications can't handle directly. This proxy manages authentication for you so that you can use it with compatible tools such as [codex](https://github.com/openai/codex).

Usage:

- Go to <http://127.0.0.1:8080> and log in with GitHub
- You will get a GitHub token as an API key

To use with OpenAI Codex

```bash
export COPILOT_API_KEY="<your token>"
export COPILOT_BASE_URL="http://127.0.0.1:8080"
codex --provider copilot
```

### Running

`go run .`
or
`go run . -listen 127.0.0.1:8923`

**Don't want to run it yourself?**

I have it hosted on <https://cope.duti.dev>. (Just replace <http://127.0.0.1:8080> in the instructions with that URL)
