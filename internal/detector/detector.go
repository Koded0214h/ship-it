package detector

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type ProjectInfo struct {
	Language      string
	Framework     string
	HasDatabase   bool
	HasRedis      bool
	HasWorker     bool
	Port          int
	EnvFileExists bool
	PackageFiles  []string
	RootDir       string
}

func Detect(dir string) (*ProjectInfo, error) {
	info := &ProjectInfo{RootDir: dir}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		name := f.Name()
		switch name {
		case ".env", ".env.example", ".env.sample":
			info.EnvFileExists = true
		}
		if isPackageFile(name) {
			info.PackageFiles = append(info.PackageFiles, name)
		}
	}

	detectLanguageAndFramework(info, dir)
	detectServices(info, dir)

	if info.Port == 0 {
		info.Port = defaultPort(info.Framework)
	}

	return info, nil
}

func detectLanguageAndFramework(info *ProjectInfo, dir string) {
	if exists(dir, "go.mod") {
		info.Language = "Go"
		info.Framework = detectGoFramework(dir)
		return
	}
	if exists(dir, "package.json") {
		info.Language = "Node.js"
		info.Framework = detectNodeFramework(dir)
		return
	}
	if exists(dir, "requirements.txt") || exists(dir, "Pipfile") || exists(dir, "pyproject.toml") {
		info.Language = "Python"
		info.Framework = detectPythonFramework(dir)
		return
	}
	if exists(dir, "Gemfile") {
		info.Language = "Ruby"
		info.Framework = detectRubyFramework(dir)
		return
	}
	if exists(dir, "pom.xml") || exists(dir, "build.gradle") {
		info.Language = "Java"
		info.Framework = "Spring Boot"
		return
	}
	if exists(dir, "Cargo.toml") {
		info.Language = "Rust"
		info.Framework = detectRustFramework(dir)
		return
	}
	if exists(dir, "composer.json") {
		info.Language = "PHP"
		info.Framework = detectPHPFramework(dir)
		return
	}
	info.Language = "Unknown"
	info.Framework = "Generic"
}

func detectGoFramework(dir string) string {
	content := readFile(filepath.Join(dir, "go.mod"))
	switch {
	case strings.Contains(content, "gin-gonic/gin"):
		return "Gin"
	case strings.Contains(content, "labstack/echo"):
		return "Echo"
	case strings.Contains(content, "gofiber/fiber"):
		return "Fiber"
	case strings.Contains(content, "go-chi/chi"):
		return "Chi"
	default:
		return "Go"
	}
}

func detectNodeFramework(dir string) string {
	content := readFile(filepath.Join(dir, "package.json"))
	switch {
	case strings.Contains(content, `"next"`):
		return "Next.js"
	case strings.Contains(content, `"express"`):
		return "Express"
	case strings.Contains(content, `"fastify"`):
		return "Fastify"
	case strings.Contains(content, `"@nestjs/core"`):
		return "NestJS"
	case strings.Contains(content, `"nuxt"`):
		return "Nuxt"
	default:
		return "Node.js"
	}
}

func detectPythonFramework(dir string) string {
	for _, file := range []string{"requirements.txt", "Pipfile", "pyproject.toml"} {
		content := readFile(filepath.Join(dir, file))
		switch {
		case strings.Contains(content, "django"):
			return "Django"
		case strings.Contains(content, "fastapi"):
			return "FastAPI"
		case strings.Contains(content, "flask"):
			return "Flask"
		}
	}
	return "Python"
}

func detectRubyFramework(dir string) string {
	content := readFile(filepath.Join(dir, "Gemfile"))
	if strings.Contains(content, "rails") {
		return "Rails"
	}
	if strings.Contains(content, "sinatra") {
		return "Sinatra"
	}
	return "Ruby"
}

func detectRustFramework(dir string) string {
	content := readFile(filepath.Join(dir, "Cargo.toml"))
	if strings.Contains(content, "actix-web") {
		return "Actix"
	}
	if strings.Contains(content, "axum") {
		return "Axum"
	}
	return "Rust"
}

func detectPHPFramework(dir string) string {
	content := readFile(filepath.Join(dir, "composer.json"))
	if strings.Contains(content, "laravel") {
		return "Laravel"
	}
	if strings.Contains(content, "symfony") {
		return "Symfony"
	}
	return "PHP"
}

func detectServices(info *ProjectInfo, dir string) {
	// Check docker-compose if present
	for _, f := range []string{"docker-compose.yml", "docker-compose.yaml"} {
		content := readFile(filepath.Join(dir, f))
		if strings.Contains(content, "postgres") || strings.Contains(content, "postgresql") {
			info.HasDatabase = true
		}
		if strings.Contains(content, "redis") {
			info.HasRedis = true
		}
	}

	// Check language-specific dependency files
	for _, pkgFile := range info.PackageFiles {
		content := readFile(filepath.Join(dir, pkgFile))
		lcontent := strings.ToLower(content)
		if strings.Contains(lcontent, "postgres") || strings.Contains(lcontent, "psycopg") || strings.Contains(lcontent, "pg ") {
			info.HasDatabase = true
		}
		if strings.Contains(lcontent, "redis") || strings.Contains(lcontent, "celery") {
			info.HasRedis = true
		}
		if strings.Contains(lcontent, "celery") || strings.Contains(lcontent, "sidekiq") || strings.Contains(lcontent, "bull") {
			info.HasWorker = true
		}
	}

	// Check env file for DB hints
	if info.EnvFileExists {
		for _, envFile := range []string{".env", ".env.example", ".env.sample"} {
			if scanEnvForDB(filepath.Join(dir, envFile)) {
				info.HasDatabase = true
			}
		}
	}
}

func scanEnvForDB(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.ToUpper(scanner.Text())
		if strings.HasPrefix(line, "DATABASE_URL") || strings.HasPrefix(line, "DB_HOST") || strings.HasPrefix(line, "POSTGRES") {
			return true
		}
	}
	return false
}

func defaultPort(framework string) int {
	switch framework {
	case "Django", "FastAPI", "Flask":
		return 8000
	case "Rails":
		return 3000
	case "Express", "Fastify", "NestJS":
		return 3000
	case "Next.js", "Nuxt":
		return 3000
	case "Gin", "Echo", "Fiber", "Chi":
		return 8080
	case "Laravel":
		return 8000
	default:
		return 8080
	}
}

func isPackageFile(name string) bool {
	switch name {
	case "go.mod", "package.json", "requirements.txt", "Pipfile", "pyproject.toml",
		"Gemfile", "pom.xml", "build.gradle", "Cargo.toml", "composer.json":
		return true
	}
	return false
}

func exists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
