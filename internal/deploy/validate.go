package deploy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Koded0214h/ship/internal/ai"
	"github.com/Koded0214h/ship/internal/config"
	"gopkg.in/yaml.v3"
)

// ValidatePlan rejects plans that cannot work or that reach into the host. It
// is a guard against mistakes, not a sandbox: users should still read the files.
func ValidatePlan(plan *ai.DeploymentPlan, localDir string) error {
	if plan.Domain != "" {
		if err := config.ValidateDomain(plan.Domain); err != nil {
			return err
		}
	}
	if strings.TrimSpace(plan.Dockerfile) == "" {
		if _, err := os.Stat(filepath.Join(localDir, "Dockerfile")); err != nil {
			return errors.New("the plan has no Dockerfile and the project has none")
		}
	}
	if _, err := envFile(plan.EnvVars); err != nil {
		return err
	}
	return validateCompose(plan.DockerCompose)
}

func validateCompose(content string) error {
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return fmt.Errorf("docker-compose.yml is not valid YAML: %w", err)
	}
	services, ok := doc["services"].(map[string]any)
	if !ok {
		return errors.New("docker-compose.yml has no services")
	}
	for _, required := range []string{"app", "nginx"} {
		if _, ok := services[required]; !ok {
			return fmt.Errorf("docker-compose.yml needs a service named %q", required)
		}
	}
	if _, ok := doc["include"]; ok {
		return errors.New("docker-compose.yml may not include other files")
	}
	for _, section := range []string{"volumes", "configs", "secrets"} {
		entries, _ := doc[section].(map[string]any)
		for name, v := range entries {
			m, _ := v.(map[string]any)
			if m["driver_opts"] != nil || m["file"] != nil {
				return fmt.Errorf("docker-compose.yml: %s %q reaches into the host filesystem", section, name)
			}
		}
	}

	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		svc, ok := services[name].(map[string]any)
		if !ok {
			return fmt.Errorf("docker-compose.yml: service %q is empty", name)
		}
		if err := validateService(name, svc); err != nil {
			return fmt.Errorf("docker-compose.yml: service %q %w", name, err)
		}
	}
	return nil
}

func validateService(name string, svc map[string]any) error {
	if v, ok := svc["privileged"]; ok && v != false {
		return errors.New("is privileged")
	}
	for _, key := range []string{"network_mode", "pid", "ipc", "userns_mode", "uts", "cgroup"} {
		if v, _ := svc[key].(string); v == "host" || strings.HasPrefix(v, "container:") {
			return fmt.Errorf("sets %s: %s", key, v)
		}
	}
	for _, key := range []string{"cap_add", "devices", "device_cgroup_rules", "security_opt", "volumes_from", "extends", "use_api_socket", "provider", "cgroup_parent"} {
		if _, ok := svc[key]; ok {
			return fmt.Errorf("uses %s", key)
		}
	}

	volumes, _ := svc["volumes"].([]any)
	for _, v := range volumes {
		source := ""
		switch vol := v.(type) {
		case string:
			if parts := strings.Split(vol, ":"); len(parts) > 1 {
				source = parts[0]
			}
			if strings.Contains(vol, "$") {
				source = vol // interpolation could expand to anything
			}
		case map[string]any:
			source, _ = vol["source"].(string)
		}
		if leavesProject(source) {
			return fmt.Errorf("mounts a path outside the app directory: %v", v)
		}
	}

	if err := validateBuild(svc["build"]); err != nil {
		return err
	}

	ports, _ := svc["ports"].([]any)
	for _, p := range ports {
		s, isString := p.(string)
		switch {
		case name == "nginx" && isString && (s == "80:80" || s == "443:443"):
		case isString && strings.HasPrefix(s, "127.0.0.1:"):
		default:
			return fmt.Errorf("publishes a port to the internet: %v (only nginx may publish 80/443)", p)
		}
	}
	return nil
}

func validateBuild(build any) error {
	var paths []string
	switch b := build.(type) {
	case string:
		paths = []string{b}
	case map[string]any:
		for _, key := range []string{"context", "dockerfile"} {
			if s, ok := b[key].(string); ok {
				paths = append(paths, s)
			}
		}
		for _, key := range []string{"additional_contexts", "privileged", "entitlements", "ssh", "network"} {
			if _, ok := b[key]; ok {
				return fmt.Errorf("uses build.%s", key)
			}
		}
	}
	for _, p := range paths {
		if leavesProject(p) || strings.Contains(p, "://") {
			return fmt.Errorf("builds from outside the app directory: %s", p)
		}
	}
	return nil
}

func leavesProject(p string) bool {
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "~") || strings.Contains(p, "$") {
		return true
	}
	for _, part := range strings.Split(p, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
