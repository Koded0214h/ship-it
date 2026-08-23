package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

func TestDetectLanguageAndFramework(t *testing.T) {
	cases := []struct {
		name          string
		files         map[string]string
		wantLanguage  string
		wantFramework string
	}{
		{"go plain", map[string]string{"go.mod": "module x\n"}, "Go", "Go"},
		{"go gin", map[string]string{"go.mod": "require github.com/gin-gonic/gin v1.9.0\n"}, "Go", "Gin"},
		{"go echo", map[string]string{"go.mod": "require github.com/labstack/echo v4\n"}, "Go", "Echo"},
		{"go fiber", map[string]string{"go.mod": "require github.com/gofiber/fiber v2\n"}, "Go", "Fiber"},
		{"go chi", map[string]string{"go.mod": "require github.com/go-chi/chi v5\n"}, "Go", "Chi"},

		{"node plain", map[string]string{"package.json": `{"dependencies":{}}`}, "Node.js", "Node.js"},
		{"node next", map[string]string{"package.json": `{"dependencies":{"next":"14.0.0"}}`}, "Node.js", "Next.js"},
		{"node express", map[string]string{"package.json": `{"dependencies":{"express":"4.0.0"}}`}, "Node.js", "Express"},
		{"node fastify", map[string]string{"package.json": `{"dependencies":{"fastify":"4.0.0"}}`}, "Node.js", "Fastify"},
		{"node nest", map[string]string{"package.json": `{"dependencies":{"@nestjs/core":"10.0.0"}}`}, "Node.js", "NestJS"},
		{"node nuxt", map[string]string{"package.json": `{"dependencies":{"nuxt":"3.0.0"}}`}, "Node.js", "Nuxt"},

		{"python plain", map[string]string{"requirements.txt": "requests==2.0\n"}, "Python", "Python"},
		{"python django", map[string]string{"requirements.txt": "django==5.0\n"}, "Python", "Django"},
		{"python fastapi", map[string]string{"requirements.txt": "fastapi==0.1\n"}, "Python", "FastAPI"},
		{"python flask", map[string]string{"requirements.txt": "flask==3.0\n"}, "Python", "Flask"},
		{"python pyproject", map[string]string{"pyproject.toml": "fastapi = \"^0.1\"\n"}, "Python", "FastAPI"},

		{"ruby plain", map[string]string{"Gemfile": "source 'https://rubygems.org'\n"}, "Ruby", "Ruby"},
		{"ruby rails", map[string]string{"Gemfile": "gem 'rails'\n"}, "Ruby", "Rails"},
		{"ruby sinatra", map[string]string{"Gemfile": "gem 'sinatra'\n"}, "Ruby", "Sinatra"},

		{"java maven", map[string]string{"pom.xml": "<project></project>"}, "Java", "Spring Boot"},
		{"java gradle", map[string]string{"build.gradle": "plugins {}"}, "Java", "Spring Boot"},

		{"rust plain", map[string]string{"Cargo.toml": "[package]\nname=\"x\"\n"}, "Rust", "Rust"},
		{"rust actix", map[string]string{"Cargo.toml": "actix-web = \"4\"\n"}, "Rust", "Actix"},
		{"rust axum", map[string]string{"Cargo.toml": "axum = \"0.7\"\n"}, "Rust", "Axum"},

		{"php plain", map[string]string{"composer.json": `{"require":{}}`}, "PHP", "PHP"},
		{"php laravel", map[string]string{"composer.json": `{"require":{"laravel/framework":"^11"}}`}, "PHP", "Laravel"},
		{"php symfony", map[string]string{"composer.json": `{"require":{"symfony/framework-bundle":"^7"}}`}, "PHP", "Symfony"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range c.files {
				writeFile(t, dir, name, content)
			}
			info, err := Detect(dir)
			if err != nil {
				t.Fatalf("Detect: %v", err)
			}
			if info.Language != c.wantLanguage {
				t.Errorf("Language: got %q, want %q", info.Language, c.wantLanguage)
			}
			if info.Framework != c.wantFramework {
				t.Errorf("Framework: got %q, want %q", info.Framework, c.wantFramework)
			}
		})
	}
}

func TestDetectUnknownWithoutStartScriptErrors(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "README.md", "hello\n")

	if _, err := Detect(dir); err == nil {
		t.Fatal("expected an error for an unrecognized project with no start.sh, got nil")
	}
}

func TestDetectUnknownWithStartScriptFallsBack(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "start.sh", "#!/bin/sh\necho hi\n")

	info, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect with start.sh present should not error: %v", err)
	}
	if info.Language != "Unknown" {
		t.Fatalf("expected Language=Unknown, got %q", info.Language)
	}
	if info.Framework != "Generic" {
		t.Fatalf("expected Framework=Generic, got %q", info.Framework)
	}
}

func TestDetectEnvFileExists(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module x\n")
	writeFile(t, dir, ".env", "FOO=bar\n")

	info, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !info.EnvFileExists {
		t.Fatal("expected EnvFileExists=true")
	}
}

func TestDetectServicesFromDockerCompose(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module x\n")
	writeFile(t, dir, "docker-compose.yml", "services:\n  db:\n    image: postgres:16\n  cache:\n    image: redis:7\n")

	info, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !info.HasDatabase {
		t.Error("expected HasDatabase=true from docker-compose.yml")
	}
	if !info.HasRedis {
		t.Error("expected HasRedis=true from docker-compose.yml")
	}
}

func TestDetectServicesFromPackageFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "psycopg2==2.9\ncelery==5.3\n")

	info, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !info.HasDatabase {
		t.Error("expected HasDatabase=true from psycopg2 in requirements.txt")
	}
	if !info.HasWorker {
		t.Error("expected HasWorker=true from celery in requirements.txt")
	}
}

func TestDetectServicesFromEnvFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module x\n")
	writeFile(t, dir, ".env", "DATABASE_URL=postgres://localhost/db\n")

	info, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !info.HasDatabase {
		t.Error("expected HasDatabase=true from DATABASE_URL in .env")
	}
}

func TestDefaultPortAssignedWhenUnset(t *testing.T) {
	cases := []struct {
		framework string
		wantPort  int
	}{
		{"Django", 8000},
		{"FastAPI", 8000},
		{"Flask", 8000},
		{"Rails", 3000},
		{"Express", 3000},
		{"Next.js", 3000},
		{"Gin", 8080},
		{"Laravel", 8000},
		{"SomethingElse", 8080},
	}
	for _, c := range cases {
		t.Run(c.framework, func(t *testing.T) {
			if got := defaultPort(c.framework); got != c.wantPort {
				t.Errorf("defaultPort(%q) = %d, want %d", c.framework, got, c.wantPort)
			}
		})
	}
}

func TestPackageFilesCollected(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module x\n")
	writeFile(t, dir, "package.json", "{}")

	info, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(info.PackageFiles) != 2 {
		t.Fatalf("expected 2 package files collected, got %v", info.PackageFiles)
	}
}
