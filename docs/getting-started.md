# Getting Started

> Alpha: use a disposable or dedicated server.

See the main [README](../README.md#installation) for installation options.

## Before you start

You need a Linux VPS (Ubuntu is the target) with:

- Docker Engine and Compose v2 ([install guide](https://docs.docker.com/engine/install/)). Ship does not install them.
- An SSH user that can run `docker` without `sudo`.
- `bash` and `flock` (present on Ubuntu).
- Port 80 free and reachable, or another port set as `app.public_port`.

## `ship init`

Run this once per project, from the project root.

1. **App details**: app name and an optional domain.
2. **Server details**: an IP, hostname or `~/.ssh/config` alias, and the SSH user. For an alias, Ship offers to copy its host, user, port and key. Jump hosts and proxy commands are not supported.
3. **SSH key**: pick one, type a path, or let Ship generate `~/.ssh/ship_ed25519`. An existing key is never overwritten.
4. **AI provider** (optional): Anthropic, OpenAI or Google Gemini with an API key, or skip.

This writes `.ship/config.yaml` and adds `.ship/` to `.gitignore`. See [Configuration](configuration.md).

## `ship connect`

Tests SSH and reports OS, architecture, disk, memory, Docker and Compose.

The first time, Ship shows the server's host key fingerprint and asks whether to trust it. Compare it with your provider's console. Trusted keys are saved to `~/.ssh/known_hosts`; a changed key is always refused.

## `ship deploy`

1. Detects language, framework, port, and whether Postgres, Redis or a worker is likely needed.
2. Builds a plan from the built-in templates, or from your AI provider if configured (`--no-ai` skips it). If the provider fails or returns an unusable plan, Ship asks whether to use the template instead.
3. If you have a Dockerfile, asks whether to keep it.
4. Prints the files and steps, then asks for confirmation.
5. On the server: checks requirements, takes a lock, uploads source, writes files, builds images, and only then replaces containers.
6. Checks that the app container stays up and that port 80 answers.

| Flag | Effect |
|---|---|
| `--no-ai` | Use the built-in templates |
| `--dry-run` | Show the plan; connect to nothing and change nothing. With AI configured, the plan request is still sent to the provider |
| `--yes`, `-y` | Ask nothing: keep your Dockerfile, use `app.port`, stop if the AI plan fails. Run `ship connect` once first, since unknown servers are still refused. Protected keys must be in ssh-agent |
| `--describe "..."` | The description for the AI provider (required with `--yes` when AI is configured) |

A failed deploy exits non-zero. If the **build** fails, running containers are not changed. If the new containers fail **after** starting, there is no automatic rollback yet.

HTTPS and CI/CD generation are not supported yet.

## `ship doctor`

Checks config, SSH, Docker and Compose, free disk, the app's containers, your AI key (if set), whether your domain's DNS points at the server, and whether the app answers from your machine. It only reads from the server, and exits non-zero if a check fails.

## `ship logs` and `ship status`

`ship logs` streams the last 100 lines and follows; `Ctrl+C` stops. Options: `--tail N`, `--service app`, `--no-follow`, and `--build` for the last image build. `ship status` lists the app's containers.

## `ship config`

View and edit settings without re-running `init`.
