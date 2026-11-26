package generator

import (
	"fmt"

	"github.com/Koded0214h/ship/internal/ai"
	"github.com/Koded0214h/ship/internal/detector"
)

func BuildServices(info *detector.ProjectInfo) []string {
	services := []string{"app"}
	if info.HasDatabase {
		services = append(services, "postgres")
	}
	if info.HasRedis {
		services = append(services, "redis")
	}
	if info.HasWorker && workerCommand(info) != "" {
		services = append(services, "worker")
	}
	return services
}

func BuildEnvVars(info *detector.ProjectInfo) map[string]string {
	vars := map[string]string{}
	if info.HasDatabase {
		vars["POSTGRES_DB"] = "app"
		vars["POSTGRES_USER"] = "app"
		vars["POSTGRES_PASSWORD"] = "changeme"
		vars["DATABASE_URL"] = "postgres://app:changeme@postgres:5432/app"
	}
	if info.HasRedis {
		vars["REDIS_URL"] = "redis://redis:6379/0"
	}
	return vars
}

func BuildSummary(info *detector.ProjectInfo, domain string, ssl bool) string {
	extras := ""
	switch {
	case info.HasDatabase && info.HasRedis:
		extras = " with PostgreSQL and Redis"
	case info.HasDatabase:
		extras = " with PostgreSQL"
	case info.HasRedis:
		extras = " with Redis"
	}
	target := "your server"
	if domain != "" && ssl {
		target = domain + " over HTTPS"
	} else if domain != "" {
		target = domain
	}
	techLabel := info.Language
	if info.Framework != "" && info.Framework != info.Language {
		techLabel = info.Language + " " + info.Framework
	}
	return fmt.Sprintf("Deploying a %s app%s to %s.", techLabel, extras, target)
}

func BuildPlan(info *detector.ProjectInfo, appName, domain string) *ai.DeploymentPlan {
	return &ai.DeploymentPlan{
		Dockerfile:    Dockerfile(info),
		DockerCompose: DockerCompose(info, appName, info.Port, false),
		Services:      BuildServices(info),
		Domain:        domain,
		EnvVars:       BuildEnvVars(info),
		Summary:       BuildSummary(info, domain, false),
	}
}
