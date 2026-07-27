package generator

import (
	"testing"

	"github.com/kodedlabs/ship/internal/detector"
)

func TestBuildServices(t *testing.T) {
	cases := []struct {
		name string
		info *detector.ProjectInfo
		want []string
	}{
		{"app only", &detector.ProjectInfo{}, []string{"app"}},
		{"database", &detector.ProjectInfo{HasDatabase: true}, []string{"app", "postgres"}},
		{"redis", &detector.ProjectInfo{HasRedis: true}, []string{"app", "redis"}},
		{"all", &detector.ProjectInfo{HasDatabase: true, HasRedis: true, HasWorker: true}, []string{"app", "postgres", "redis", "worker"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := BuildServices(c.info)
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("got %v, want %v", got, c.want)
				}
			}
		})
	}
}

func TestBuildEnvVars(t *testing.T) {
	if got := BuildEnvVars(&detector.ProjectInfo{}); len(got) != 0 {
		t.Fatalf("expected no env vars, got %v", got)
	}

	db := BuildEnvVars(&detector.ProjectInfo{HasDatabase: true})
	for _, k := range []string{"POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD", "DATABASE_URL"} {
		if db[k] == "" {
			t.Fatalf("expected %s to be set, got %v", k, db)
		}
	}

	redis := BuildEnvVars(&detector.ProjectInfo{HasRedis: true})
	if redis["REDIS_URL"] == "" {
		t.Fatalf("expected REDIS_URL to be set, got %v", redis)
	}
}

func TestBuildSummary(t *testing.T) {
	cases := []struct {
		name   string
		info   *detector.ProjectInfo
		domain string
		ssl    bool
		want   string
	}{
		{
			"no extras no domain",
			&detector.ProjectInfo{Language: "Python", Framework: "FastAPI"},
			"", false,
			"Deploying a Python FastAPI app to your server.",
		},
		{
			"db and redis with domain and ssl",
			&detector.ProjectInfo{Language: "Go", Framework: "Gin", HasDatabase: true, HasRedis: true},
			"api.example.com", true,
			"Deploying a Go Gin app with PostgreSQL and Redis to api.example.com over HTTPS.",
		},
		{
			"db with domain no ssl",
			&detector.ProjectInfo{Language: "Node.js", Framework: "Express", HasDatabase: true},
			"example.com", false,
			"Deploying a Node.js Express app with PostgreSQL to example.com.",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := BuildSummary(c.info, c.domain, c.ssl); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}
