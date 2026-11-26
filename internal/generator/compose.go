package generator

import (
	"fmt"
	"strings"

	"github.com/Koded0214h/ship/internal/detector"
)

func DockerCompose(info *detector.ProjectInfo, appName string, port int, ssl bool) string {
	var sb strings.Builder

	sb.WriteString("services:\n")
	sb.WriteString(appService(info, appName))
	sb.WriteString(nginxService(ssl))

	if info.HasDatabase {
		sb.WriteString(postgresService())
	}
	if info.HasRedis {
		sb.WriteString(redisService())
	}
	if info.HasWorker && workerCommand(info) != "" {
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

// The app publishes no port on the server: nginx reaches it over the Compose
// network, so apps cannot collide on host ports.
func appService(info *detector.ProjectInfo, appName string) string {
	depends := dependsOn(info)
	return fmt.Sprintf(`  app:
    build: .
    container_name: %s
    restart: unless-stopped
    env_file:
      - .env
    networks:
      - app%s
`, appName, depends)
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
      - .env
    networks:
      - app
    depends_on:
      - app
`, appName, workerCmd)
}

func workerCommand(info *detector.ProjectInfo) string {
	switch info.Framework {
	case "Django":
		return fmt.Sprintf(`["python", "-m", "celery", "-A", "%s", "worker", "--loglevel=info"]`, djangoProject(info.RootDir))
	case "Rails":
		return `["bundle", "exec", "sidekiq"]`
	}
	// Ship does not know how to start a worker for other stacks, so it adds none.
	if info.Language == "" || info.Language == "Node.js" {
		return `["node", "worker.js"]`
	}
	return ""
}
