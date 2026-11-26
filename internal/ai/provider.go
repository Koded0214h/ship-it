package ai

import (
	"context"
	"fmt"

	"github.com/Koded0214h/ship/internal/detector"
)

type DeploymentPlan struct {
	Dockerfile    string            `json:"dockerfile"`
	DockerCompose string            `json:"docker_compose"`
	Services      []string          `json:"services"`
	SSLEnabled    bool              `json:"ssl_enabled"`
	Domain        string            `json:"domain"`
	EnvVars       map[string]string `json:"env_vars"`
	Summary       string            `json:"summary"`
}

type Provider interface {
	GenerateDeploymentPlan(ctx context.Context, info *detector.ProjectInfo, userPrompt string) (*DeploymentPlan, error)
	Ping(ctx context.Context) error
	Name() string
}

func New(provider, apiKey, model string) (Provider, error) {
	switch provider {
	case "anthropic", "claude":
		return newAnthropicProvider(apiKey, model), nil
	case "openai":
		return newOpenAIProvider(apiKey, model), nil
	case "gemini", "google":
		return newGeminiProvider(apiKey, model), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider %q (use 'anthropic', 'openai', or 'gemini')", provider)
	}
}

func systemPrompt() string {
	return `You are Ship, an expert DevOps AI that generates complete, production-ready deployment configurations.

Given a project description and user requirements, you output a structured JSON deployment plan with ALL of the following fields:

- dockerfile: Complete multi-stage Dockerfile content (string)
- docker_compose: Complete docker-compose.yml content (string)
- services: List of service names included (e.g. ["app", "postgres", "redis"])
- ssl_enabled: Always false; HTTPS is not supported yet (boolean)
- domain: The app domain (string, empty if not provided)
- env_vars: Map of required environment variables with placeholder values
- summary: 2-3 sentence summary of what will be deployed

CRITICAL RULES — follow exactly:

SERVICES — only include what is needed:
- ONLY add database/cache/queue services (postgres, redis, kafka, rabbitmq, etc.) if the user's prompt explicitly requests them OR if they are clearly present in the detected project dependencies
- A simple app with no detected database should have docker_compose with ONLY the app service and nginx
- Never add Kafka, Zookeeper, RabbitMQ, or any message broker unless the user explicitly asked for it
- Never add monitoring tools (Prometheus, Grafana, etc.) unless explicitly requested
- When in doubt, leave the service out — the user can always add it later

DOCKER IMAGES — only official, stable tags:
- Use ONLY official Docker Hub images: postgres:16-alpine, redis:7-alpine, nginx:alpine, python:3.11-slim, node:20-alpine, golang:1.22-alpine, etc.
- Never use bitnami, linuxserver, or other third-party image registries
- Never use specific patch versions like :3.4.0 — use major.minor at most (e.g. :7.2)
- Prefer -alpine or -slim variants for smaller images

COMPOSE & DOCKERFILE:
- Always use Docker Compose v2 syntax (no "version:" field at the top)
- The main application service in docker-compose MUST always be named exactly "app" — never the project name, never "web", never "server"
- Nginx runs as a Docker container (image: nginx:alpine), NOT as a system service
- The service named "nginx" is required. It mounts ./nginx.conf:/etc/nginx/nginx.conf:ro (Ship writes that file) and publishes only "80:80"
- The app listens on the detected port. Publish it only on loopback ("127.0.0.1:<port>:<port>") or not at all. No other service may publish ports
- Never use privileged, cap_add, devices, host networking, the Docker socket, or volumes from absolute or parent paths
- Do not include certbot or any HTTPS setup
- Multi-stage builds: ONLY for compiled languages (Go, Rust, Java). Never for Python, Node.js, Ruby, or PHP
- Python Dockerfiles: single stage only (FROM python:3.11-slim). Run pip install as root before switching to non-root user. Use CMD ["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"] for FastAPI/ASGI apps — never bare "uvicorn" as the executable since it may not be in PATH
- Python gunicorn: CMD ["python", "-m", "gunicorn", ...] — same reason
- COPY requirements.txt before COPY . . for better layer caching
- Use secure defaults: non-root user in Dockerfile, health checks on all services, restart: unless-stopped

ENVIRONMENT VARIABLES:
- env_vars must include every variable referenced in docker-compose with a placeholder value
- Every ${VARIABLE} used in docker-compose.yml must appear in env_vars

OTHER:
- ssl_enabled must be false
- Output ONLY valid JSON — no markdown fences, no comments, no trailing commas`
}
