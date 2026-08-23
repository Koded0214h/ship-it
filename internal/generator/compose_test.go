package generator

import (
	"testing"

	"github.com/Koded0214h/ship/internal/detector"
)

func TestDockerComposeAppOnly(t *testing.T) {
	got := DockerCompose(&detector.ProjectInfo{}, "myapp", 8080, false)
	if !contains(got, "container_name: myapp") {
		t.Errorf("expected app container, got:\n%s", got)
	}
	if contains(got, "postgres:") {
		t.Errorf("did not expect postgres service:\n%s", got)
	}
	if contains(got, "redis:") {
		t.Errorf("did not expect redis service:\n%s", got)
	}
	if contains(got, "worker:") {
		t.Errorf("did not expect worker service:\n%s", got)
	}
	if !contains(got, `- "80:80"`) || contains(got, `- "443:443"`) {
		t.Errorf("expected only port 80 exposed without SSL, got:\n%s", got)
	}
}

func TestDockerComposeWithSSLExposes443(t *testing.T) {
	got := DockerCompose(&detector.ProjectInfo{}, "myapp", 8080, true)
	if !contains(got, `- "443:443"`) {
		t.Errorf("expected port 443 exposed with SSL enabled, got:\n%s", got)
	}
}

func TestDockerComposeWithDatabaseAndRedis(t *testing.T) {
	info := &detector.ProjectInfo{HasDatabase: true, HasRedis: true}
	got := DockerCompose(info, "myapp", 8080, false)

	for _, want := range []string{"postgres:", "redis:", "postgres_data:", "redis_data:", "depends_on:"} {
		if !contains(got, want) {
			t.Errorf("expected %q in compose output:\n%s", want, got)
		}
	}
}

func TestDockerComposeWithWorker(t *testing.T) {
	info := &detector.ProjectInfo{HasWorker: true, Framework: "Django"}
	got := DockerCompose(info, "myapp", 8000, false)

	if !contains(got, "myapp-worker") {
		t.Errorf("expected worker container name, got:\n%s", got)
	}
	if !contains(got, "celery") {
		t.Errorf("expected Django worker command (celery), got:\n%s", got)
	}
}

func TestDockerComposeNoVolumesWithoutStatefulServices(t *testing.T) {
	got := DockerCompose(&detector.ProjectInfo{}, "myapp", 8080, false)
	if contains(got, "\nvolumes:\n") {
		t.Errorf("did not expect a top-level volumes block with no database/redis:\n%s", got)
	}
}
