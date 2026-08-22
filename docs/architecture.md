# Architecture

## Flow

```
Your project directory
        │
        ▼
internal/detector          → language, framework, port, DB/Redis/worker hints
        │
        ▼
internal/ai                → sends project info + your description to
                              Anthropic/OpenAI/Gemini, gets back a
                              DeploymentPlan (Dockerfile, docker-compose.yml,
                              nginx config, GitHub Actions workflow,
                              env vars, steps, summary)
        │
        ▼
internal/deploy (Executor) → runs the plan over SSH:
                              create app dir → upload source → upload
                              generated files → start containers →
                              health check
        │
        ▼
internal/ssh                → SSH/SCP primitives used by the executor
        │
        ▼
Your VPS
```

## Packages

- **`internal/detector`** — reads the project directory (no execution, just file presence/content checks) to fill in a `ProjectInfo`: language, framework, listening port, and whether a database/Redis/worker is likely needed, based on `docker-compose.yml`, dependency manifests (`package.json`, `requirements.txt`, `go.mod`, etc.), and `.env` files.

- **`internal/ai`** — defines the `Provider` interface (`GenerateDeploymentPlan`, `Ping`, `Name`) and `New(provider, apiKey, model)`, which returns an Anthropic, OpenAI, or Gemini client. All three send the same system prompt and are expected to return the same `DeploymentPlan` JSON shape. See [AI Providers](ai-providers.md) for the constraints the prompt enforces on generated output.

- **`internal/generator`** — Go template-based generators for Dockerfiles (per-language), Docker Compose, Nginx configs, and a GitHub Actions workflow. In the current deploy flow, only `generator.NginxConfig` is actually used — as a fallback that overrides the AI-generated Nginx config when there's no domain/SSL, since the AI often generates an HTTP→HTTPS redirect that breaks plain-IP access. The Dockerfile/Compose/GitHub Actions generators exist but aren't currently wired into `ship deploy` — the AI-generated versions are used instead.

- **`internal/deploy`** — the `Executor` that turns a `DeploymentPlan` into a running deployment: creates the remote app directory, uploads your source (via `internal/ssh`'s tar+gzip transfer, skipping `.git`, `node_modules`, `vendor`, and similar), writes the Dockerfile/Compose/Nginx/`.env` files, runs `docker compose up -d --build`, polls for a running container (up to 3 minutes), then does an HTTP health check.

- **`internal/ssh`** — thin wrapper over `golang.org/x/crypto/ssh`: `Exec`/`ExecStream` for commands, `WriteFile`/`WriteFileSudo` for writing remote files via stdin, `UploadDir` for tar-based directory transfer, and `GetServerInfo` for the diagnostics `ship connect`/`ship doctor` show.

- **`internal/sshkey`** — finds existing keys in `~/.ssh` (`id_ed25519`, `id_rsa`, `id_ecdsa`, `ship_ed25519`) or generates a new ED25519 pair at `~/.ssh/ship_ed25519` during `ship init`.

- **`internal/config`** — reads/writes `.ship/config.yaml`, walking up parent directories to find it, and validates required fields before a deploy.

- **`cmd/`** — Cobra commands (`init`, `connect`, `deploy`, `doctor`, `logs`, `config`). All interactive via `huh` prompts — no CLI flags.

## Where credentials go

Your SSH private key never leaves your machine — it's read locally and used to sign the SSH handshake. Your AI API key is read from `.ship/config.yaml` and sent only to the provider you configured, as part of the `GenerateDeploymentPlan` and `Ping` calls. Project metadata (detected language/framework/port and your free-text deployment description) is what gets sent to the AI provider — not your source code, except for filenames/dependency manifests already summarized into `ProjectInfo`.
