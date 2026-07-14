package generator

import (
	"fmt"
	"strings"

	"github.com/kodedlabs/ship/internal/detector"
)

func DockerCompose(info *detector.ProjectInfo, appName string, port int, ssl bool) string {
	var sb strings.Builder

	sb.WriteString("services:\n")
	sb.WriteString(appService(info, appName, port))
	sb.WriteString(nginxService(ssl))

	if info.HasDatabase {
		sb.WriteString(postgresService())
	}
	if info.HasRedis {
		sb.WriteString(redisService())
	}
	if info.HasWorker {
		sb.WriteString(workerService(info, appName))
	}

	if info.HasDatabase || info.HasRedis {
		sb.WriteString("\nvolumes:\n")
		if info.HasDatabase {
			sb.WriteString("  postgres_data:\n")
		}
		if info.HasRedis {
			sb.WriteString("  redis_data:\n")
		}
	}

	sb.WriteString("\nnetworks:\n  app:\n    driver: bridge\n")

	return sb.String()
}

func appService(info *detector.ProjectInfo, appName string, port int) string {
	depends := dependsOn(info)
	return fmt.Sprintf(`  app:
    build: .
    container_name: %s
    restart: unless-stopped
    ports:
      - "127.0.0.1:%d:%d"
    env_file:
      - path: .env
        required: false
    networks:
      - app%s
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:%d/"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
`, appName, port, port, depends, port)
}

func nginxService(ssl bool) string {
	ports := `      - "80:80"`
	if ssl {
		ports += `
      - "443:443"`
	}
	return fmt.Sprintf(`
  nginx:
    image: nginx:alpine
    container_name: nginx
    restart: unless-stopped
    ports:
%s
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    networks:
      - app
    depends_on:
      app:
        condition: service_started
`, ports)
}

func dependsOn(info *detector.ProjectInfo) string {
	var deps []string
	if info.HasDatabase {
		deps = append(deps, "postgres")
	}
	if info.HasRedis {
		deps = append(deps, "redis")
	}
	if len(deps) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n    depends_on:\n")
	for _, d := range deps {
		sb.WriteString(fmt.Sprintf("      %s:\n        condition: service_healthy\n", d))
	}
	return sb.String()
}

func postgresService() string {
	return `
  postgres:
    image: postgres:16-alpine
    container_name: postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-app}
      POSTGRES_USER: ${POSTGRES_USER:-app}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - app
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-app}"]
      interval: 10s
      timeout: 5s
      retries: 5
`
}

func redisService() string {
	return `
  redis:
    image: redis:7-alpine
    container_name: redis
    restart: unless-stopped
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    networks:
      - app
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
`
}

func workerService(info *detector.ProjectInfo, appName string) string {
	workerCmd := workerCommand(info)
	return fmt.Sprintf(`
  worker:
    build: .
    container_name: %s-worker
    restart: unless-stopped
    command: %s
    env_file:
      - path: .env
        required: false
    networks:
      - app
    depends_on:
      - app
`, appName, workerCmd)
}

func workerCommand(info *detector.ProjectInfo) string {
	switch info.Framework {
	case "Django":
		return `["python", "-m", "celery", "-A", "config", "worker", "--loglevel=info"]`
	case "Rails":
		return `["bundle", "exec", "sidekiq"]`
	default:
		return `["node", "worker.js"]`
	}
}
