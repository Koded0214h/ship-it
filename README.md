# Ship 🚀

> Deploy your applications to your own VPS using simple, natural language.

📚 [Docs](docs/README.md) · [Contributing](CONTRIBUTING.md) · [Security](SECURITY.md)

Ship is an AI-powered CLI that eliminates the complexity of configuring deployments, Docker, reverse proxies, SSL, and CI/CD pipelines.

Instead of spending hours writing Dockerfiles, GitHub Actions workflows, Nginx configurations, and deployment scripts, simply describe what you want.

```bash
ship deploy

> Deploy my Django app with PostgreSQL and Redis.
> Use Docker.
> Deploy automatically when I push to main.
> Enable HTTPS.
```

Ship generates the deployment plan, configures your server, and gets your application online—all while you keep full ownership of your infrastructure.

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
* 🔒 HTTPS & SSL configuration
* 🚀 GitHub Actions CI/CD generation
* 🔍 Automatic project detection
* 🖥️ SSH-based VPS deployments
* 📋 Deployment summaries
* 🛠️ Health checks & diagnostics
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
brew install kodedlabs/tap/ship
```

**Install script** (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/kodedlabs/ship/main/install.sh | sh
```

**Go install**

```bash
go install github.com/kodedlabs/ship@latest
```

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

## Supported Frameworks (MVP)

Backend

* Framework agnostic — Ship detects your stack and generates configs to match, rather than requiring a specific framework.

Infrastructure

* Docker
* Docker Compose
* PostgreSQL
* Redis
* GitHub Actions
* Ubuntu VPS

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

### MVP

* [ ] Project detection
* [ ] SSH connections
* [ ] Docker generation
* [ ] Docker Compose generation
* [ ] GitHub Actions generation
* [ ] Nginx configuration
* [ ] HTTPS setup
* [ ] Deployment execution

### v0.2

* [ ] Rollbacks
* [ ] Deployment history
* [ ] Interactive deployment plans
* [ ] Environment variable management

### v0.3

* [ ] Monitoring
* [ ] Automatic backups
* [ ] Multi-server deployments
* [ ] Background workers
* [ ] Cron jobs

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
* [glamour](https://github.com/charmbracelet/glamour) — Markdown rendering