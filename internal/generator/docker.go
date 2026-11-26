package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Koded0214h/ship/internal/detector"
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

// --- Python

func pythonDockerfile(info *detector.ProjectInfo) string {
	// The installer is chosen by what the project actually has.
	install := "COPY requirements*.txt ./\nRUN pip install --no-cache-dir -r requirements.txt"
	manifest := "requirements.txt"
	switch {
	case fileExists(info.RootDir, "requirements.txt"):
	case fileExists(info.RootDir, "Pipfile"):
		manifest = "Pipfile"
		lock := ""
		if !fileExists(info.RootDir, "Pipfile.lock") {
			lock = "pipenv lock && " // installing into the system needs one
		}
		install = "COPY Pipfile* ./\nRUN pip install --no-cache-dir pipenv && " + lock + "pipenv install --system --deploy"
	case fileExists(info.RootDir, "pyproject.toml"):
		manifest = "pyproject.toml"
		// The package itself is installed, so its source must be there first.
		install = "COPY . .\nRUN pip install --no-cache-dir ."
	}

	// Add the server named in CMD when the project does not depend on it.
	server := map[string]string{"Django": "gunicorn", "Flask": "gunicorn", "FastAPI": "uvicorn"}[info.Framework]
	if server != "" && !strings.Contains(strings.ToLower(readProjectFile(info.RootDir, manifest)), server) {
		install += "\nRUN pip install --no-cache-dir " + server
	}

	return fmt.Sprintf(`FROM python:3.12-slim

ENV PYTHONUNBUFFERED=1 PORT=%d
WORKDIR /app

RUN addgroup --system app && adduser --system --group --home /home/app app

%s

COPY . .
RUN chown -R app:app /app

USER app

EXPOSE %d

%s
`, info.Port, install, info.Port, pythonStartCmd(info))
}

func pythonStartCmd(info *detector.ProjectInfo) string {
	switch info.Framework {
	case "Django":
		return fmt.Sprintf(`CMD ["gunicorn", "--bind", "0.0.0.0:%d", "--workers", "4", "%s.wsgi:application"]`, info.Port, djangoProject(info.RootDir))
	case "FastAPI":
		return fmt.Sprintf(`CMD ["uvicorn", "%s", "--host", "0.0.0.0", "--port", "%d"]`, pythonApp(info.RootDir, "FastAPI", "main:app"), info.Port)
	case "Flask":
		return fmt.Sprintf(`CMD ["gunicorn", "--bind", "0.0.0.0:%d", "%s"]`, info.Port, pythonApp(info.RootDir, "Flask", "app:app"))
	default:
		if !fileExists(info.RootDir, "main.py") && fileExists(info.RootDir, "app.py") {
			return `CMD ["python", "app.py"]`
		}
		return `CMD ["python", "main.py"]`
	}
}

// pythonApp finds "module:variable" where the application object is created.
func pythonApp(root, class, fallback string) string {
	assignment := regexp.MustCompile(`(?m)^(\w+)\s*(?::\s*\w+\s*)?=\s*` + class + `\(`)
	for _, file := range []string{"main.py", "app.py", "wsgi.py", "app/main.py", "app/__init__.py", "src/main.py"} {
		if m := assignment.FindStringSubmatch(readProjectFile(root, file)); m != nil {
			module := strings.TrimSuffix(strings.TrimSuffix(file, ".py"), "/__init__")
			return strings.ReplaceAll(module, "/", ".") + ":" + m[1]
		}
	}
	return fallback
}

var identifier = regexp.MustCompile(`^[A-Za-z_]\w*$`)

// djangoProject is the package holding wsgi.py.
func djangoProject(root string) string {
	if root != "" {
		matches, _ := filepath.Glob(filepath.Join(root, "*", "wsgi.py"))
		for _, m := range matches {
			if name := filepath.Base(filepath.Dir(m)); identifier.MatchString(name) {
				return name
			}
		}
	}
	return "config"
}

// --- Node.js

type packageJSON struct {
	Main    string            `json:"main"`
	Scripts map[string]string `json:"scripts"`
}

// safeFile is a relative script path that can be written into a Dockerfile.
var safeFile = regexp.MustCompile(`^(\./)?[A-Za-z0-9_][A-Za-z0-9_./-]*$`)

func nodeDockerfile(info *detector.ProjectInfo) string {
	var pkg packageJSON
	_ = json.Unmarshal([]byte(readProjectFile(info.RootDir, "package.json")), &pkg)

	manager, setup := "npm", ""
	if fileExists(info.RootDir, "pnpm-lock.yaml") {
		manager, setup = "pnpm", "RUN npm install -g pnpm@9\n"
	} else if fileExists(info.RootDir, "yarn.lock") {
		manager = "yarn"
	}

	build := pkg.Scripts["build"] != "" || info.Framework == "Next.js" || info.Framework == "Nuxt"

	install := manager + " install"
	if manager == "npm" && fileExists(info.RootDir, "package-lock.json") {
		install = "npm ci"
	}
	if !build { // building needs the development dependencies
		if manager == "pnpm" {
			install += " --prod"
		} else {
			install += " --production"
		}
	}
	buildStep := ""
	if build {
		buildStep = fmt.Sprintf("\nRUN %s run build", manager)
	}

	return fmt.Sprintf(`FROM node:20-alpine AS builder

WORKDIR /app
%sCOPY package*.json yarn.lock* pnpm-lock.yaml* ./
RUN %s
COPY . .%s

FROM node:20-alpine
ENV NODE_ENV=production PORT=%d
WORKDIR /app
%sRUN addgroup -S app && adduser -S app -G app

COPY --from=builder /app .
RUN chown -R app:app /app

USER app
EXPOSE %d

%s
`, setup, install, buildStep, info.Port, setup, info.Port, nodeStartCmd(info, pkg, manager, build))
}

func nodeStartCmd(info *detector.ProjectInfo, pkg packageJSON, manager string, build bool) string {
	switch {
	case pkg.Scripts["start:prod"] != "":
		return fmt.Sprintf(`CMD ["%s", "run", "start:prod"]`, manager)
	case pkg.Scripts["start"] != "":
		return fmt.Sprintf(`CMD ["%s", "start"]`, manager)
	}
	for _, file := range []string{pkg.Main, "index.js", "server.js", "app.js"} {
		if safeFile.MatchString(file) && fileExists(info.RootDir, file) {
			return fmt.Sprintf(`CMD ["node", "%s"]`, path.Clean(file))
		}
	}
	switch info.Framework {
	case "Express", "Fastify", "NestJS":
		return `CMD ["node", "dist/main.js"]`
	}
	if build && info.Framework == "" {
		return `CMD ["node", "dist/main.js"]`
	}
	return fmt.Sprintf(`CMD ["%s", "start"]`, manager)
}

// --- Go

func goDockerfile(info *detector.ProjectInfo) string {
	return fmt.Sprintf(`FROM golang:%s-alpine AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server %s

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /out/server .
RUN chown -R app:app /app

ENV PORT=%d
USER app
EXPOSE %d

CMD ["./server"]
`, goVersion(info.RootDir), goPackage(info.RootDir), info.Port, info.Port)
}

var goDirective = regexp.MustCompile(`(?m)^go\s+1\.(\d+)`)

// goVersion is the toolchain go.mod asks for, and at least 1.23. The official
// images do not download newer toolchains by themselves.
func goVersion(root string) string {
	minor := 23
	if m := goDirective.FindStringSubmatch(readProjectFile(root, "go.mod")); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil && n > minor {
			minor = n
		}
	}
	return "1." + strconv.Itoa(minor)
}

var goMain = regexp.MustCompile(`(?m)^package main\b`)

func hasGoMain(dir string) bool {
	files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err == nil && !strings.HasSuffix(f, "_test.go") && goMain.Match(data) {
			return true
		}
	}
	return false
}

// goPackage finds the main package: the root, or one under cmd/.
func goPackage(root string) string {
	if root == "" || hasGoMain(root) {
		return "."
	}
	entries, _ := os.ReadDir(filepath.Join(root, "cmd"))
	var mains []string
	for _, e := range entries {
		if e.IsDir() && safeFile.MatchString(e.Name()) && hasGoMain(filepath.Join(root, "cmd", e.Name())) {
			mains = append(mains, e.Name())
		}
	}
	for _, preferred := range []string{"server", "api", "web", "app"} {
		if slices.Contains(mains, preferred) {
			return "./cmd/" + preferred
		}
	}
	if len(mains) > 0 {
		return "./cmd/" + mains[0]
	}
	return "."
}

// --- Ruby

func rubyDockerfile(info *detector.ProjectInfo) string {
	packages := "build-essential"
	if regexp.MustCompile(`gem ['"]pg['"]`).MatchString(readProjectFile(info.RootDir, "Gemfile")) {
		packages += " libpq-dev"
	}
	startCmd := `CMD ["bundle", "exec", "puma", "-C", "config/puma.rb"]`
	if info.Framework == "Sinatra" {
		startCmd = `CMD ["bundle", "exec", "ruby", "app.rb"]`
	}
	return fmt.Sprintf(`FROM ruby:3.3-slim

ENV APP_ENV=production RACK_ENV=production RAILS_ENV=production PORT=%d
WORKDIR /app
RUN apt-get update && apt-get install -y %s && rm -rf /var/lib/apt/lists/*

RUN groupadd --system app && useradd --system --gid app --create-home app

COPY Gemfile Gemfile.lock* ./
RUN bundle config set --global without 'development test' && bundle install

COPY . .
RUN chown -R app:app /app

USER app
EXPOSE %d

%s
`, info.Port, packages, info.Port, startCmd)
}

// --- Java

func javaDockerfile(info *detector.ProjectInfo) string {
	image := "maven:3-eclipse-temurin-21-alpine"
	buildCmd := "mvn package -DskipTests"
	jars := "target/*.jar"
	if fileExists(info.RootDir, "build.gradle") || fileExists(info.RootDir, "build.gradle.kts") {
		// Gradle projects are built with their own wrapper.
		image = "eclipse-temurin:21-jdk-alpine"
		buildCmd = "chmod +x gradlew 2>/dev/null; ./gradlew bootJar"
		jars = "build/libs/*.jar"
	}

	return fmt.Sprintf(`FROM %s AS builder

WORKDIR /app
COPY . .
RUN %s
RUN cp "$(ls %s | grep -v -e '-plain' -e '-sources' -e '-javadoc' | head -n 1)" /app.jar

FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /app.jar app.jar
RUN chown -R app:app /app

ENV PORT=%d SERVER_PORT=%d
USER app
EXPOSE %d

ENTRYPOINT ["java", "-jar", "app.jar"]
`, image, buildCmd, jars, info.Port, info.Port, info.Port)
}

// --- Rust

var (
	cargoSection = regexp.MustCompile(`(?m)^\s*\[+\s*([\w.-]+)\s*\]+`)
	cargoName    = regexp.MustCompile(`(?m)^\s*name\s*=\s*"([A-Za-z0-9_-]+)"`)
)

// rustBinary is the name of the executable cargo produces.
func rustBinary(root string) string {
	manifest := readProjectFile(root, "Cargo.toml")
	sections := cargoSection.FindAllStringSubmatchIndex(manifest, -1)
	names := map[string]string{}
	for i, s := range sections {
		end := len(manifest)
		if i+1 < len(sections) {
			end = sections[i+1][0]
		}
		section := manifest[s[2]:s[3]]
		if m := cargoName.FindStringSubmatch(manifest[s[1]:end]); m != nil && names[section] == "" {
			names[section] = m[1]
		}
	}
	for _, section := range []string{"bin", "package"} {
		if names[section] != "" {
			return names[section]
		}
	}
	return "app"
}

func rustDockerfile(info *detector.ProjectInfo) string {
	// Cache mounts keep compiled dependencies between deploys.
	return fmt.Sprintf(`FROM rust:1-alpine AS builder

RUN apk add --no-cache musl-dev
WORKDIR /app
COPY . .
RUN --mount=type=cache,target=/usr/local/cargo/registry --mount=type=cache,target=/app/target \
    cargo build --release && cp target/release/%s /server

FROM alpine:3.20
RUN apk --no-cache add ca-certificates

WORKDIR /app
RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /server ./app
RUN chown -R app:app /app

ENV PORT=%d
USER app
EXPOSE %d

CMD ["./app"]
`, rustBinary(info.RootDir), info.Port, info.Port)
}

// --- PHP

// phpDockerfile uses Apache, which speaks HTTP; php-fpm alone cannot sit behind
// an HTTP reverse proxy.
func phpDockerfile(info *detector.ProjectInfo) string {
	docroot := "/var/www/html"
	if fileExists(info.RootDir, "public/index.php") {
		docroot += "/public"
	}
	return fmt.Sprintf(`FROM php:8.3-apache

RUN apt-get update && apt-get install -y libpq-dev unzip git && rm -rf /var/lib/apt/lists/* && \
    docker-php-ext-install pdo_pgsql pdo_mysql opcache && a2enmod rewrite
RUN sed -ri 's/^Listen 80$/Listen %d/' /etc/apache2/ports.conf && \
    printf '<VirtualHost *:%d>\n  DocumentRoot %s\n  <Directory %s>\n    AllowOverride All\n    Require all granted\n  </Directory>\n  ErrorLog /dev/stderr\n  CustomLog /dev/stdout combined\n</VirtualHost>\n' > /etc/apache2/sites-available/000-default.conf

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer
WORKDIR /var/www/html
COPY . .
RUN if [ -f composer.json ]; then COMPOSER_ALLOW_SUPERUSER=1 composer install --no-dev --no-interaction --optimize-autoloader; fi
RUN chown -R www-data:www-data /var/www/html

EXPOSE %d
CMD ["apache2-foreground"]
`, info.Port, info.Port, docroot, docroot, info.Port)
}

// --- Other

func genericDockerfile(info *detector.ProjectInfo) string {
	return fmt.Sprintf(`FROM ubuntu:24.04

WORKDIR /app
COPY . .
RUN chmod +x start.sh

ENV PORT=%d
EXPOSE %d

CMD ["./start.sh"]
`, info.Port, info.Port)
}

func fileExists(dir, name string) bool {
	if dir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

func readProjectFile(dir, name string) string {
	if dir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return string(data)
}
