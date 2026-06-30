# Ship 🚀

> Deploy your applications to your own VPS using simple, natural language.

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

Coming soon.

```bash
brew install ship
```

or

```bash
npm install -g ship-cli
```

or

```bash
curl -fsSL https://shipcli.dev/install.sh | bash
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

* Backend is meant to be framework agnostic, so it doesnt matter

Frontend

* we only host servers, backend you get..
* we'll have a frontend for cool landing page and setup or docs..?

Infrastructure

* Docker
* Docker Compose
* PostgreSQL
* and any other thats needed
* Redis
* GitHub Actions
* Ubuntu VPS

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

---

## License

MIT


## for colors and text stuff
cobra               → CLI framework
lipgloss            → Colors & styling
bubbles             → Interactive components
bubbletea           → Terminal UI (optional)
huh                 → Beautiful prompts/forms
glamour             → Render Markdown beautifully