# Getting Started

See the main [README](../README.md#installation) for installation options (Homebrew, install script, or `go install`).

Every Ship command is interactive — there are no flags to memorize. Run a command and answer the prompts.

## `ship init`

Run this once per project, from the project root.

1. **App details** — app name (used as the remote directory and as the `docker-compose` service naming context) and an optional domain. Leave the domain blank to deploy on the server's IP.
2. **Server details** — the VPS host/IP and the SSH user to connect as.
3. **SSH key** — Ship scans `~/.ssh` for `id_ed25519`, `id_rsa`, `id_ecdsa`, and `ship_ed25519`. Pick one, or let Ship generate a new ED25519 key pair at `~/.ssh/ship_ed25519`. If it generates one, it prints the `authorized_keys` line to add on your server.
4. **AI provider** — Anthropic (Claude), OpenAI, or Google Gemini, plus an API key for whichever you pick.

This writes `.ship/config.yaml` in the project directory and adds `.ship/` to your project's `.gitignore` (it contains your API key and SSH key path). See [Configuration](configuration.md) for the full schema.

## `ship connect`

Opens an SSH connection using the config from `ship init` and reports back:

- OS and architecture
- Free disk and memory
- Whether Docker is installed (if missing, install it yourself before running `deploy` — see [FAQ & Troubleshooting](faq-troubleshooting.md))

Use this to verify SSH access before running `deploy`.

## `ship deploy`

1. Detects your project's language, framework, port, and whether it looks like it needs Postgres/Redis/a background worker (see [Architecture](architecture.md)).
2. Asks you to describe the deployment in plain English (e.g. "Deploy with PostgreSQL and Redis, enable HTTPS, deploy on push to main").
3. Sends the project info + your description to your configured AI provider, which returns a full deployment plan: Dockerfile, `docker-compose.yml`, Nginx config, GitHub Actions workflow, required env vars, and a step-by-step summary.
4. Renders the plan and asks you to confirm.
5. If the plan includes a GitHub Actions workflow, saves it locally to `.github/workflows/deploy.yml`.
6. Connects over SSH and: creates `~/ship/<app-name>` on the server, uploads your source, uploads the generated Dockerfile/Compose/Nginx config/`.env` template, brings up the containers with `docker compose up -d --build`, and polls until something is running or 3 minutes pass.
7. Runs an HTTP health check against your domain (or the server IP if you didn't set one) and reports success or failure.

If a deploy fails partway through, `ship deploy` stops at the failing step — check `ship doctor` or `ship logs`.

## `ship doctor`

Runs through: config file validity, SSH connectivity, Docker on the server, whether your AI provider's API key is valid, and (if you set a domain) whether it resolves and responds over HTTP. Good first stop when something isn't working.

## `ship logs`

Streams `docker compose logs -f --tail=100` from `$HOME/ship/<app-name>` on your server (the same directory `ship deploy` uploads to). `Ctrl+C` to stop.

## `ship config`

View your current config (API key and SSH key path are masked) and edit AI provider/key, app settings, or server settings without re-running the whole `init` flow.
