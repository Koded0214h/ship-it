# Configuration

Ship stores per-project config at `.ship/config.yaml`, created by `ship init` and editable via `ship config`. Every command that talks to your server or AI provider walks up from the current directory looking for `.ship/config.yaml`, so you can run `ship deploy` etc. from a subdirectory of your project too.

`.ship/` is added to your `.gitignore` automatically — it contains your AI API key and SSH key path, so it should never be committed.

## Schema

```yaml
app:
  name: my-api          # required — used as the remote directory name
  domain: ""             # optional — leave blank to deploy on the server's IP
  port: 8080              # the port your app listens on

server:
  host: ""                # required — IP or hostname of your VPS
  user: ""                # required — SSH user
  key_path: ""             # path to your private SSH key
  port: 22                  # SSH port, defaults to 22 if unset/zero

ai:
  provider: ""             # required — "anthropic", "openai", or "gemini"
  api_key: ""               # required
  model: ""                  # optional — leave blank to use the provider's default model
```

## Validation

Before deploying, Ship checks that `app.name`, `server.host`, `server.user`, `ai.provider`, and `ai.api_key` are all set. If any are missing, `ship deploy` refuses to run and tells you to re-run `ship init`.

## AI provider values

`ai.provider` accepts:

- `anthropic` (or `claude`) — Claude models
- `openai` — OpenAI models
- `gemini` (or `google`) — Google Gemini models

See [AI Providers](ai-providers.md) for model defaults and what each provider is asked to generate.

## Editing config

`ship config` shows your current settings (API key and SSH key path are masked) and lets you update one section at a time — AI, app, or server — without repeating the full `ship init` flow.
