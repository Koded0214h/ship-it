package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kodedlabs/ship/internal/detector"
)

func Dockerfile(info *detector.ProjectInfo) string {
	switch info.Language {
	case "Python":
		return pythonDockerfile(info)
	case "Node.js":
		return nodeDockerfile(info)
	case "Go":
		return goDockerfile(info)
	case "Ruby":
		return rubyDockerfile(info)
	case "Java":
		return javaDockerfile(info)
	case "Rust":
		return rustDockerfile(info)
	case "PHP":
		return phpDockerfile(info)
	default:
		return genericDockerfile(info)
	}
}

func pythonDockerfile(info *detector.ProjectInfo) string {
	manager := "pip"
	installCmd := "pip install --no-cache-dir -r requirements.txt"
	if fileExists(info.RootDir, "Pipfile") {
		manager = "pipenv"
		installCmd = "pipenv install --system --deploy"
	} else if fileExists(info.RootDir, "pyproject.toml") {
		manager = "poetry"
		installCmd = "pip install --no-cache-dir ."
	}
	_ = manager

	startCmd := pythonStartCmd(info)

	return fmt.Sprintf(`FROM python:3.12-slim

WORKDIR /app

RUN addgroup --system app && adduser --system --group app

COPY requirements*.txt pyproject.toml Pipfile* ./
RUN %s

COPY . .
RUN chown -R app:app /app

USER app

EXPOSE %d

%s
`, installCmd, info.Port, startCmd)
}

func pythonStartCmd(info *detector.ProjectInfo) string {
	switch info.Framework {
	case "Django":
		return fmt.Sprintf(`CMD ["gunicorn", "--bind", "0.0.0.0:%d", "--workers", "4", "config.wsgi:application"]`, info.Port)
	case "FastAPI":
		return fmt.Sprintf(`CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "%d"]`, info.Port)
	case "Flask":
		return fmt.Sprintf(`CMD ["gunicorn", "--bind", "0.0.0.0:%d", "app:app"]`, info.Port)
	default:
		return `CMD ["python", "main.py"]`
	}
}

func nodeDockerfile(info *detector.ProjectInfo) string {
	packageManager := "npm"
	if fileExists(info.RootDir, "pnpm-lock.yaml") {
		packageManager = "pnpm"
	} else if fileExists(info.RootDir, "yarn.lock") {
		packageManager = "yarn"
	}

	installCmd := packageManager + " install --production"
	buildCmd := ""
	if info.Framework == "Next.js" || info.Framework == "Nuxt" {
		buildCmd = fmt.Sprintf("\nRUN %s run build", packageManager)
		installCmd = packageManager + " install"
	}

	startCmd := fmt.Sprintf(`CMD ["%s", "start"]`, packageManager)
	if info.Framework == "Express" || info.Framework == "Fastify" || info.Framework == "NestJS" {
		startCmd = `CMD ["node", "dist/main.js"]`
		if fileExists(info.RootDir, "index.js") {
			startCmd = `CMD ["node", "index.js"]`
		}
	}

	return fmt.Sprintf(`FROM node:20-alpine AS builder

WORKDIR /app
COPY package*.json yarn.lock* pnpm-lock.yaml* ./
RUN %s
COPY . .%s

FROM node:20-alpine
WORKDIR /app
RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /app .
RUN chown -R app:app /app

USER app
EXPOSE %d

%s
`, installCmd, buildCmd, info.Port, startCmd)
}

func goDockerfile(info *detector.ProjectInfo) string {
	return fmt.Sprintf(`FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /app/server .
RUN chown -R app:app /app

USER app
EXPOSE %d

CMD ["./server"]
`, info.Port)
}

func rubyDockerfile(info *detector.ProjectInfo) string {
	startCmd := `CMD ["bundle", "exec", "puma", "-C", "config/puma.rb"]`
	if info.Framework == "Sinatra" {
		startCmd = `CMD ["bundle", "exec", "ruby", "app.rb"]`
	}
	return fmt.Sprintf(`FROM ruby:3.3-slim

WORKDIR /app
RUN apt-get update && apt-get install -y build-essential libpq-dev && rm -rf /var/lib/apt/lists/*

RUN addgroup --system app && adduser --system --group app

COPY Gemfile Gemfile.lock ./
RUN bundle install --without development test

COPY . .
RUN chown -R app:app /app

USER app
EXPOSE %d

%s
`, info.Port, startCmd)
}

func javaDockerfile(info *detector.ProjectInfo) string {
	buildTool := "mvn"
	buildCmd := "mvn package -DskipTests"
	jarFile := "target/*.jar"
	if fileExists(info.RootDir, "build.gradle") {
		buildTool = "gradle"
		buildCmd = "./gradlew bootJar"
		jarFile = "build/libs/*.jar"
	}
	_ = buildTool

	return fmt.Sprintf(`FROM eclipse-temurin:21-jdk-alpine AS builder

WORKDIR /app
COPY . .
RUN %s

FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /app/%s app.jar
RUN chown -R app:app /app

USER app
EXPOSE %d

ENTRYPOINT ["java", "-jar", "app.jar"]
`, buildCmd, jarFile, info.Port)
}

func rustDockerfile(info *detector.ProjectInfo) string {
	return fmt.Sprintf(`FROM rust:1.82-alpine AS builder

RUN apk add --no-cache musl-dev
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs && cargo build --release && rm src/main.rs

COPY src ./src
RUN cargo build --release

FROM alpine:3.20
RUN apk --no-cache add ca-certificates

WORKDIR /app
RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /app/target/release/app .
RUN chown -R app:app /app

USER app
EXPOSE %d

CMD ["./app"]
`, info.Port)
}

func phpDockerfile(info *detector.ProjectInfo) string {
	return fmt.Sprintf(`FROM php:8.3-fpm-alpine

RUN apk add --no-cache nginx supervisor && \
    docker-php-ext-install pdo pdo_pgsql opcache

WORKDIR /var/www/html
COPY composer.json composer.lock ./
RUN php -r "copy('https://getcomposer.org/installer', 'composer-setup.php');" && \
    php composer-setup.php --install-dir=/usr/local/bin --filename=composer && \
    composer install --no-dev --optimize-autoloader

COPY . .

EXPOSE %d
CMD ["php-fpm"]
`, info.Port)
}

func genericDockerfile(info *detector.ProjectInfo) string {
	return fmt.Sprintf(`FROM ubuntu:24.04

WORKDIR /app
COPY . .

EXPOSE %d

CMD ["./start.sh"]
`, info.Port)
}

func fileExists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}
