package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Koded0214h/ship/internal/detector"
)

func TestDockerfilePerLanguage(t *testing.T) {
	cases := []struct {
		name string
		info *detector.ProjectInfo
		want []string
	}{
		{"python default", &detector.ProjectInfo{Language: "Python", Port: 8000}, []string{"FROM python:3.12-slim", `CMD ["python", "main.py"]`, "USER app", "EXPOSE 8000"}},
		{"python fastapi", &detector.ProjectInfo{Language: "Python", Framework: "FastAPI", Port: 8000}, []string{`CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]`}},
		{"python django", &detector.ProjectInfo{Language: "Python", Framework: "Django", Port: 8000}, []string{"gunicorn", "config.wsgi:application"}},
		{"node default", &detector.ProjectInfo{Language: "Node.js", Port: 3000}, []string{"FROM node:20-alpine AS builder", "npm install --production", "USER app", "EXPOSE 3000"}},
		{"node express", &detector.ProjectInfo{Language: "Node.js", Framework: "Express", Port: 3000}, []string{`CMD ["node", "dist/main.js"]`}},
		{"node next builds", &detector.ProjectInfo{Language: "Node.js", Framework: "Next.js", Port: 3000}, []string{"npm run build"}},
		{"go", &detector.ProjectInfo{Language: "Go", Port: 8080}, []string{"FROM golang:1.23-alpine AS builder", "CGO_ENABLED=0", "USER app", "EXPOSE 8080"}},
		{"ruby default", &detector.ProjectInfo{Language: "Ruby", Port: 3000}, []string{"FROM ruby:3.3-slim", `CMD ["bundle", "exec", "puma", "-C", "config/puma.rb"]`}},
		{"ruby sinatra", &detector.ProjectInfo{Language: "Ruby", Framework: "Sinatra", Port: 3000}, []string{`CMD ["bundle", "exec", "ruby", "app.rb"]`}},
		{"java", &detector.ProjectInfo{Language: "Java", Port: 8080}, []string{"eclipse-temurin", "mvn package -DskipTests", "app.jar"}},
		{"rust", &detector.ProjectInfo{Language: "Rust", Port: 8080}, []string{"FROM rust:1.82-alpine AS builder", "cargo build --release"}},
		{"php", &detector.ProjectInfo{Language: "PHP", Port: 8000}, []string{"FROM php:8.3-fpm-alpine", `CMD ["php-fpm"]`}},
		{"unknown falls back to generic", &detector.ProjectInfo{Language: "Unknown", Port: 8080}, []string{"FROM ubuntu:24.04", `CMD ["./start.sh"]`}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.info.RootDir == "" {
				c.info.RootDir = t.TempDir()
			}
			got := Dockerfile(c.info)
			for _, want := range c.want {
				if !contains(got, want) {
					t.Errorf("Dockerfile output missing %q:\n%s", want, got)
				}
			}
		})
	}
}

func TestDockerfileNonRootUser(t *testing.T) {
	// Every supported language except PHP/generic drops privileges before
	// running the app — a baseline security expectation worth locking in.
	languages := []string{"Python", "Node.js", "Go", "Ruby", "Java", "Rust"}
	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			info := &detector.ProjectInfo{Language: lang, Port: 8080, RootDir: t.TempDir()}
			got := Dockerfile(info)
			if !contains(got, "USER app") {
				t.Errorf("%s Dockerfile does not switch to a non-root user:\n%s", lang, got)
			}
		})
	}
}

func TestPythonDockerfilePicksPoetryForPyproject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[tool.poetry]\n"), 0644); err != nil {
		t.Fatalf("writing pyproject.toml: %v", err)
	}
	info := &detector.ProjectInfo{Language: "Python", Port: 8000, RootDir: dir}
	got := Dockerfile(info)
	if !contains(got, "pip install --no-cache-dir .") {
		t.Errorf("expected pyproject.toml install path, got:\n%s", got)
	}
}

func TestNodeDockerfilePicksPnpmLockfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0644); err != nil {
		t.Fatalf("writing pnpm-lock.yaml: %v", err)
	}
	info := &detector.ProjectInfo{Language: "Node.js", Port: 3000, RootDir: dir}
	got := Dockerfile(info)
	if !contains(got, "pnpm install") {
		t.Errorf("expected pnpm install command, got:\n%s", got)
	}
}

func TestJavaDockerfilePicksGradleForBuildGradle(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "build.gradle"), []byte(""), 0644); err != nil {
		t.Fatalf("writing build.gradle: %v", err)
	}
	info := &detector.ProjectInfo{Language: "Java", Port: 8080, RootDir: dir}
	got := Dockerfile(info)
	if !contains(got, "./gradlew bootJar") {
		t.Errorf("expected gradle build command, got:\n%s", got)
	}
	if !contains(got, "build/libs/*.jar") {
		t.Errorf("expected gradle jar path, got:\n%s", got)
	}
}
