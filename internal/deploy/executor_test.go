package deploy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Koded0214h/ship/internal/ai"
	"github.com/Koded0214h/ship/internal/config"
	"github.com/Koded0214h/ship/internal/detector"
	"github.com/Koded0214h/ship/internal/generator"
)

// fakeRemote records commands and fails those containing failOn.
type fakeRemote struct {
	commands []string
	files    map[string]string
	failOn   string
	state    func() string // reply to "docker inspect"
	envFile  bool
	port80   string
	released bool
}

func (f *fakeRemote) record(cmd string) error {
	f.commands = append(f.commands, cmd)
	if f.failOn != "" && strings.Contains(cmd, f.failOn) {
		return errors.New("exit status 1")
	}
	return nil
}

func (f *fakeRemote) Output(_ context.Context, cmd string) (string, error) {
	if err := f.record(cmd); err != nil {
		return "", err
	}
	switch {
	case strings.HasPrefix(cmd, "printf"):
		return "/home/u", nil
	case strings.Contains(cmd, "docker inspect"):
		if f.state != nil {
			return f.state(), nil
		}
		return "running 0\n", nil
	case strings.HasPrefix(cmd, "if [ -e"):
		if f.envFile {
			return "exists\n", nil
		}
		return "missing\n", nil
	case strings.HasPrefix(cmd, "ss "):
		return f.port80, nil
	}
	return "", nil
}

func (f *fakeRemote) Stream(_ context.Context, cmd string, _ io.Writer) error { return f.record(cmd) }

func (f *fakeRemote) WriteFile(_ context.Context, p, content string, _ os.FileMode) error {
	if f.files == nil {
		f.files = map[string]string{}
	}
	f.files[p] = content
	return f.record("write " + p)
}

func (f *fakeRemote) UploadDir(context.Context, string, string) ([]string, error) {
	return nil, f.record("upload")
}

func (f *fakeRemote) Lock(context.Context, string) (func(), error) {
	return func() { f.released = true }, f.record("lock")
}

func newExecutor(t *testing.T, remote Remote, status int) *Executor {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
	t.Cleanup(server.Close)
	cfg := &config.Config{
		App:    config.AppConfig{Name: "demo"},
		Server: config.ServerConfig{Host: strings.TrimPrefix(server.URL, "http://")},
	}
	plan := generator.BuildPlan(&detector.ProjectInfo{Language: "Go", Port: 8080, HasDatabase: true, RootDir: t.TempDir()}, "demo", "")
	e := New(remote, cfg, plan, t.TempDir(), 8080)
	e.PollInterval = 0
	return e
}

func last(results []StepResult) StepResult { return results[len(results)-1] }

func assertNeverDestructive(t *testing.T, commands []string) {
	t.Helper()
	for _, c := range commands {
		for _, bad := range []string{"docker stop", "docker rm", "xargs", "prune", "compose down", "system prune"} {
			if strings.Contains(c, bad) {
				t.Errorf("destructive command: %s", c)
			}
		}
	}
}

func started(commands []string) bool {
	return slices.ContainsFunc(commands, func(c string) bool { return strings.Contains(c, " up -d") })
}

func TestRunSucceeds(t *testing.T) {
	remote := &fakeRemote{}
	results := newExecutor(t, remote, 200).Run(context.Background(), nil)
	if r := last(results); !r.Success || r.Name != "Health check" {
		t.Fatalf("last step %q: %v", r.Name, r.Err)
	}
	if !remote.released {
		t.Error("lock was not released")
	}
	assertNeverDestructive(t, remote.commands)
}

func TestFailuresBeforeStartLeaveContainersAlone(t *testing.T) {
	for name, failOn := range map[string]string{
		"no docker": "docker info", "locked": "lock", "upload": "upload", "build": " build",
	} {
		t.Run(name, func(t *testing.T) {
			remote := &fakeRemote{failOn: failOn}
			results := newExecutor(t, remote, 200).Run(context.Background(), nil)
			if last(results).Success {
				t.Fatal("expected a failure")
			}
			if started(remote.commands) {
				t.Errorf("containers were changed after a failure: %v", remote.commands)
			}
			assertNeverDestructive(t, remote.commands)
		})
	}
}

func TestCrashLoopAndServerErrorAreFailures(t *testing.T) {
	restarts := 0
	cases := map[string]struct {
		remote *fakeRemote
		status int
		step   string
	}{
		"exited":     {&fakeRemote{state: func() string { return "exited 1" }}, 200, "Verify app container"},
		"restarting": {&fakeRemote{state: func() string { restarts++; return "running " + strings.Repeat("1", restarts) }}, 200, "Verify app container"},
		"502":        {&fakeRemote{}, 502, "Health check"},
		"port taken": {&fakeRemote{port80: "LISTEN 0 511 *:80"}, 200, "Check port 80 is available"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r := last(newExecutor(t, c.remote, c.status).Run(context.Background(), nil))
			if r.Success || r.Name != c.step {
				t.Fatalf("got step %q success=%v, want failure at %q", r.Name, r.Success, c.step)
			}
			if strings.Contains(r.Err.Error(), "%!") {
				t.Errorf("badly formatted error: %v", r.Err)
			}
		})
	}
}

func TestEnvFile(t *testing.T) {
	remote := &fakeRemote{envFile: true}
	newExecutor(t, remote, 200).Run(context.Background(), nil)
	if _, wrote := remote.files["/home/u/ship/demo/.env"]; wrote {
		t.Error("overwrote an existing .env")
	}

	remote = &fakeRemote{}
	newExecutor(t, remote, 200).Run(context.Background(), nil)
	env := remote.files["/home/u/ship/demo/.env"]
	if env == "" || strings.Contains(env, "changeme") {
		t.Fatalf("expected a generated password:\n%s", env)
	}
	password := strings.Split(strings.Split(env, "POSTGRES_PASSWORD=")[1], "\n")[0]
	if !strings.Contains(env, ":"+password+"@postgres") {
		t.Errorf("DATABASE_URL does not use the generated password:\n%s", env)
	}
	if _, err := envFile(map[string]string{"A": "x\nINJECTED=1"}); err == nil {
		t.Error("accepted a multi-line value")
	}
}

func TestValidatePlan(t *testing.T) {
	for _, info := range []*detector.ProjectInfo{
		{Language: "Go", Port: 8080},
		{Language: "Python", Framework: "Django", Port: 8000, HasDatabase: true, HasRedis: true, HasWorker: true},
	} {
		info.RootDir = t.TempDir()
		if err := ValidatePlan(generator.BuildPlan(info, "demo", "example.com"), info.RootDir); err != nil {
			t.Fatalf("template plan rejected: %v", err)
		}
	}

	base := "services:\n  nginx:\n    image: nginx:alpine\n  app:\n    build: .\n"
	for name, extra := range map[string]string{
		"privileged":    "    privileged: true\n",
		"host network":  "    network_mode: host\n",
		"docker socket": "    volumes: [\"/var/run/docker.sock:/var/run/docker.sock\"]\n",
		"parent dir":    "    volumes: [\"../:/data\"]\n",
		"interpolated":  "    volumes: [\"${HOME}:/data\"]\n",
		"long bind":     "    volumes: [{type: bind, source: /etc, target: /x}]\n",
		"public port":   "    ports: [\"5432:5432\"]\n",
		"capabilities":  "    cap_add: [SYS_ADMIN]\n",
		"anchored":      "    <<: {privileged: true}\n",
	} {
		plan := &ai.DeploymentPlan{Dockerfile: "FROM x", DockerCompose: base + extra}
		if err := ValidatePlan(plan, t.TempDir()); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	bad := &ai.DeploymentPlan{Dockerfile: "FROM x", DockerCompose: base, Domain: "a.com; rm -rf /"}
	if ValidatePlan(bad, t.TempDir()) == nil {
		t.Error("accepted a malicious domain")
	}
}
