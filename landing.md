# Ship Landing Page

## Navigation

```
Ship

Docs
GitHub
Roadmap

[Install]
```

---

# Hero

### Headline

## Deploy to your VPS using plain English.

### Subtitle

Stop spending hours configuring Docker, Nginx, SSL, and CI/CD.

Describe your deployment. Ship generates the infrastructure and deploys it to your own server.

```
[ Install Ship ]
[ View on GitHub ]
```

---

## Interactive Terminal

```
$ ship deploy

What would you like to deploy?

> A Django app with PostgreSQL and Redis.
> Use Docker.
> Deploy whenever I push to main.
> Enable HTTPS.

✓ Analyzing repository...

✓ Django detected
✓ PostgreSQL detected
✓ Docker Compose generated
✓ GitHub Actions generated
✓ Nginx configured
✓ SSL enabled

🚀 Deployment complete.

https://example.com
```

---

# Why Ship?

Most deployment tools do one of two things:

• Hide your infrastructure.
• Expect you to become a DevOps engineer.

Ship does neither.

You own your VPS.

Ship simply automates the repetitive parts.

---

# Features

## Automatic Project Detection

Ship understands your repository.

No templates.

No boilerplate.

---

## AI Deployment Planning

Describe your infrastructure in plain English.

Ship creates a deployment plan before executing anything.

---

## Own Your Infrastructure

Deploy to your existing VPS.

No vendor lock-in.

---

## CI/CD Included

Generate GitHub Actions automatically.

Deploy on every push.

---

## Secure by Design

SSH keys never leave your machine.

Deployment happens locally through the CLI.

---

# How It Works

```
Your Project

      │

Ship CLI

      │

AI Planner

      │

Deployment Plan

      │

SSH

      │

Your VPS
```

---

# One Command

```bash
ship init
ship connect
ship deploy
```

That's it.

---

# Roadmap

✅ MVP

* Project detection
* Docker generation
* GitHub Actions
* Docker Compose
* SSH deployments

🚧 Coming Soon

* Rollbacks
* Monitoring
* Multi-server deployments
* Automatic backups
* Plugin system

---

# Open Source

Ship is open source.

Contributions, ideas, and feedback are always welcome.

[ GitHub ]

---

# FAQ

### Does Ship host my app?

No.

Ship deploys to your own VPS.

---

### Does Ship require Docker?

No.

It can generate Docker configurations for you.

---

### Does Ship store my SSH keys?

Never.

Your credentials stay on your machine.

---

### Can I use my own AI provider?

Yes.

Bring your own Gemini, NVIDIA, OpenAI, Anthropic, or compatible API key.

---

# Footer

MIT License

Built with Go, Cobra, and ❤️ by KodedlABS who were tired of configuring deployments.
