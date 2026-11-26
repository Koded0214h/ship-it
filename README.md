# Ship 🚀

> Deploy your applications to your own VPS using simple, natural language.

📚 [Docs](docs/README.md) · [Contributing](CONTRIBUTING.md) · [Security](SECURITY.md)

> ⚠️ **Alpha software. Use a disposable or dedicated server, not one you care about.**
> The released `v0.1.0` binary stops and removes **every** container on the server during a deploy, uploads your local `.env` over the server's, does not verify SSH host keys, and can report success when the build failed. These are being fixed on `main` and are not yet released.

Ship is a CLI that deploys your application to your own VPS with Docker Compose and Nginx. It can use an AI provider to plan the deployment, or built-in templates with no AI at all.

Instead of spending hours writing Dockerfiles, Nginx configurations, and deployment scripts, simply describe what you want.

```bash
ship deploy

> Deploy my Django app with PostgreSQL and Redis.
> Use Docker.
```

Ship generates the deployment plan, shows you the files, and builds and starts your application on a server that already has Docker—all while you keep full ownership of your infrastructure.

**Not supported yet:** HTTPS/SSL, CI/CD generation, installing Docker for you, rollbacks, and more than one app per server.

---

## Why Ship?

Modern deployment is unnecessarily complicated.

A typical deployment often requires configuring:

* Docker
* Docker Compose
* Reverse Proxy (Nginx/Caddy)
* HTTPS & SSL
* GitHub Actions
* SSH
* Firewall rules
* Environment variables
* System services
* Health checks
* Deployment scripts

For many developers, this means hours of setup before writing a single line of application code.

Ship aims to make deployment conversational.

---

## Features

* 🤖 AI-assisted deployment planning
* 🐳 Automatic Docker & Docker Compose generation
* 🔍 Automatic project detection
* 🖥️ SSH-based VPS deployments
* 📋 Deployment summaries
* 🛠️ Health checks & diagnostics
* 🔒 HTTPS (planned)
* 🚀 CI/CD generation (planned)
* 🔄 Rollback support (planned)
* 📈 Monitoring integrations (planned)

---

## How it Works

```text
Your Project
      │
      ▼
Ship CLI
      │
      ├── Detects your framework
      ├── Analyzes your project
      ├── Generates a deployment plan
      ▼
AI Provider
      │
      ▼
Structured Deployment Plan
      │
      ▼
Ship CLI
      │
      ▼
Your VPS
```

Your SSH credentials never leave your machine.

The AI helps create the deployment plan, while the CLI performs all server operations locally.

---

## Installation

**Homebrew** (macOS/Linux)

```bash
brew install Koded0214h/tap/ship
```

**Install script** (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/Koded0214h/ship/main/install.sh | sh
```

**Go install**

```bash
go install github.com/Koded0214h/ship@latest
```

> `go install` fails with `v0.1.0` (module path mismatch) and will work from the next release. Homebrew and the install script currently provide `v0.1.0`, which requires an AI key.

---

## Quick Start

Initialize Ship inside your project.

```bash
ship init
```

Connect your server.

```bash
ship connect
```

Deploy your application.

```bash
ship deploy
```

Need help?

```bash
ship doctor
```

View deployment logs.

```bash
ship logs
```

---

## Deploying without an AI key

An AI provider is optional (on `main`; not in `v0.1.0`). If you don't have (or don't want to use) an API key, choose **"Skip — deploy without AI"** during `ship init`, or clear the provider later by selecting **"None — deploy without AI"** in `ship config`. With no key configured, Ship falls back to built-in templates.

The no-key path generates a Dockerfile, `docker-compose.yml` and Nginx config from your detected framework and dependencies instead of a written description. It asks for your domain (optional) and deploys over HTTP. You can also pass `--no-ai` to skip a configured provider for one run, and if the provider fails Ship offers to continue with the templates.

---

## Supported Frameworks (MVP)

Backend

* Ship detects your stack and generates configs to match. The apps in [`examples/`](examples/) are built and requested over HTTP in CI: Node (Express, pnpm, TypeScript build), Python (Flask, FastAPI, Django, Pipenv, pyproject), Go, Rust, Java (Maven), Ruby (Sinatra) and PHP.
* Other layouts and frameworks (for example Next.js, NestJS, Rails, Gradle) are detected but not tested. You can always keep your own Dockerfile.

Infrastructure

* Docker
* Docker Compose
* PostgreSQL
* Redis
* Ubuntu VPS (Docker and Compose v2 already installed)

Ship deploys server-side applications to a VPS you own; it doesn't host anything itself. See [Roadmap](#roadmap) for what's planned next.

---

## Philosophy

Ship is not another hosting provider.

It helps developers deploy applications to servers they already own.

You control:

* Your VPS
* Your SSH keys
* Your cloud provider
* Your AI provider

Ship simply removes the repetitive DevOps work.

---

## Roadmap

### MVP — shipped

* [x] Project detection
* [x] SSH connections
* [x] Docker generation
* [x] Docker Compose generation
* [x] Nginx configuration
* [x] Deployment execution

### v0.2

* [ ] [Automatic HTTPS](https://github.com/Koded0214h/ship/issues/16)
* [ ] Working CI/CD generation
* [ ] [Rollbacks](https://github.com/Koded0214h/ship/issues/2)
* [ ] [Deployment history](https://github.com/Koded0214h/ship/issues/3)
* [ ] [Interactive deployment plan editing](https://github.com/Koded0214h/ship/issues/4)
* [ ] [Environment variable management](https://github.com/Koded0214h/ship/issues/5)

### v0.3

* [ ] [Monitoring integrations](https://github.com/Koded0214h/ship/issues/6)
* [ ] [Automatic backups](https://github.com/Koded0214h/ship/issues/7)
* [ ] [Multi-server deployments](https://github.com/Koded0214h/ship/issues/8)
* [ ] [Background workers & cron jobs](https://github.com/Koded0214h/ship/issues/9)
* [ ] [Plugin system](https://github.com/Koded0214h/ship/issues/10)

See the [open issues](https://github.com/Koded0214h/ship/issues) for the full backlog, including smaller bugs and cleanup tasks.

---

## Security

Ship does **not** send your SSH keys or server credentials to any external service.

Server operations are executed locally through the CLI.

Only project metadata required to generate deployment plans may be sent to your configured AI provider.

---

## Contributing

Contributions, ideas, and bug reports are always welcome.

If you've ever thought:

> "Deploying this should be easier."

...you're exactly who Ship is for.

See [CONTRIBUTING.md](CONTRIBUTING.md) for dev setup and how to submit changes, and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for community guidelines. Security issues should go through [SECURITY.md](SECURITY.md) instead of a public issue.

---

## License

MIT — see [LICENSE](LICENSE).

---

## Built With

* [cobra](https://github.com/spf13/cobra) — CLI framework
* [lipgloss](https://github.com/charmbracelet/lipgloss) — colors & styling
* [bubbles](https://github.com/charmbracelet/bubbles) — interactive components
* [bubbletea](https://github.com/charmbracelet/bubbletea) — terminal UI
* [huh](https://github.com/charmbracelet/huh) — prompts & forms
