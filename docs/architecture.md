# Architecture

## Flow

```
Your project directory
        │
        ▼
internal/detector          → language, framework, port, DB/Redis/worker hints
        │
        ▼
internal/generator         → built-in templates
   or internal/ai          → or a plan from Anthropic/OpenAI/Gemini
        │
        ▼
internal/deploy            → validates the plan, then over SSH: preflight,
                              lock, upload, write files, build, start,
                              verify, health check
        │
        ▼
internal/ssh               → host key checks, commands, upload
        │
        ▼
Your VPS
```

## Packages

- **`internal/detector`** — reads the project directory (no execution, just file presence/content checks) to fill in a `ProjectInfo`: language, framework, listening port, and whether a database/Redis/worker is likely needed, based on `docker-compose.yml`, dependency manifests (`package.json`, `requirements.txt`, `go.mod`, etc.), and `.env` files.

- **`internal/ai`** — the `Provider` interface and the `DeploymentPlan` type (Dockerfile, `docker-compose.yml`, services, domain, env vars, summary). See [AI Providers](ai-providers.md).

- **`internal/generator`** — templates for Dockerfiles, Docker Compose and Nginx. Used for the whole plan in no-AI mode. The Nginx config always comes from here, in both modes.

- **`internal/deploy`** — `ValidatePlan` rejects plans that reach into the host or publish extra ports. `Executor` runs the deploy through the small `Remote` interface, so it can be tested with a fake. Nothing running is touched until the build succeeds, and it never stops containers outside the app's Compose project.

- **`internal/ssh`** — verifies host keys against `known_hosts`, runs cancellable commands, and uploads source without secrets. `internal/ssh/sshtest` is an in-process server for tests.

- **`internal/sshkey`** — finds keys in `~/.ssh` or generates a new ED25519 pair without overwriting.

- **`internal/config`** — reads, writes and validates `.ship/config.yaml`.

- **`cmd/`** — Cobra commands (`init`, `connect`, `deploy`, `doctor`, `logs`, `config`), mostly interactive via `huh`.

## Where credentials go

Your SSH private key never leaves your machine. Your AI API key is sent only to the provider you configured. The AI request contains your description plus the detected language, framework, port, service flags, whether an env file exists, and manifest file names. It does not contain source code, file contents or local paths.
