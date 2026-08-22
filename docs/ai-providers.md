# AI Providers

Ship needs an AI provider to turn your project info and deployment description into a `DeploymentPlan`. You configure exactly one provider per project, in `.ship/config.yaml` (see [Configuration](configuration.md)).

## Supported providers

| `ai.provider` value | Provider |
|---|---|
| `anthropic` or `claude` | Anthropic (Claude) |
| `openai` | OpenAI |
| `gemini` or `google` | Google Gemini |

Set `ai.model` to pin a specific model (e.g. `claude-sonnet-4-6`, `gpt-4o`, `gemini-2.5-flash`); leave it blank to use the provider's default.

You only need an API key for the one provider you pick — Ship never calls more than one provider per deploy.

`ship doctor` calls `Ping` on your configured provider to confirm the key is valid and reachable.

## What every provider is asked to generate

All three providers receive the same system prompt and are expected to return the same JSON shape: a Dockerfile, `docker-compose.yml`, Nginx config, GitHub Actions workflow, list of services, SSL flag, domain, required env vars, human-readable steps, and a summary.

The prompt constrains what comes back, regardless of provider:

- **Services** — only adds a database/cache/queue if you asked for one or it's evident from your dependencies (no unsolicited Kafka, RabbitMQ, or monitoring stacks).
- **Images** — official Docker Hub images only (no Bitnami or other third-party registries), major.minor tags at most, `-alpine`/`-slim` variants preferred.
- **Compose** — Compose v2 syntax, the app service is always named `app` (never the project name), Nginx runs as a container proxying to `app:<port>`.
- **Dockerfiles** — multi-stage builds only for compiled languages (Go, Rust, Java); single-stage for Python/Node/Ruby/PHP; non-root user; health checks; `restart: unless-stopped`.
- **SSL** — only enabled when you provide a real domain, never for a raw IP.
- **CI/CD** — the GitHub Actions workflow triggers on push to `main` and deploys over SSH.

See `internal/ai/provider.go` for the exact prompt if you're debugging a specific generation issue.

## Adding a provider

Providers implement the `ai.Provider` interface (`GenerateDeploymentPlan`, `Ping`, `Name`) in `internal/ai/`. See [CONTRIBUTING.md](../CONTRIBUTING.md) if you want to add one.
