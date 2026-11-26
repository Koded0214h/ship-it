package deploy

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// SetPublicPort makes the nginx service listen on port on the server instead of
// 80, so Ship can run beside something that already owns 80.
func SetPublicPort(compose string, port int) (string, error) {
	if port == 0 || port == 80 {
		return compose, nil
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(compose), &doc); err != nil {
		return "", fmt.Errorf("docker-compose.yml is not valid YAML: %w", err)
	}
	services, _ := doc["services"].(map[string]any)
	nginx, ok := services["nginx"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("docker-compose.yml needs a service named \"nginx\"")
	}
	// HTTPS is not supported, so port 80 inside the container is the only one.
	nginx["ports"] = []string{fmt.Sprintf("%d:80", port)}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
