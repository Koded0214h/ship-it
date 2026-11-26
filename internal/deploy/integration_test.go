//go:build !windows

package deploy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Koded0214h/ship/internal/ai"
	"github.com/Koded0214h/ship/internal/config"
	sshclient "github.com/Koded0214h/ship/internal/ssh"
	"github.com/Koded0214h/ship/internal/ssh/sshtest"
)

// Runs the real commands through a shell over SSH, with a fake docker whose
// build fails while an old release is still serving.
func TestFailedBuildOverSSH(t *testing.T) {
	s := sshtest.Start(t)
	log := s.Home + "/docker.log"
	tool := func(name, body string) {
		if err := os.WriteFile(s.Bin+"/"+name, []byte("#!/bin/sh\n"+body), 0755); err != nil {
			t.Fatal(err)
		}
	}
	tool("docker", `echo "$*" >> "`+log+`"
case "$*" in *" build") echo "ERROR: failed to build"; exit 1 ;; esac
`)
	tool("flock", "shift 2; exec \"$@\"\n")
	tool("ss", "exit 0\n")

	local := t.TempDir()
	os.WriteFile(local+"/.env", []byte("DEV_SECRET=laptop\n"), 0600)
	os.WriteFile(local+"/main.go", []byte("package main"), 0644)
	appDir := s.Home + "/ship/demo"
	os.MkdirAll(appDir, 0755)
	os.WriteFile(appDir+"/.env", []byte("PROD=1\n"), 0644)

	old := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer old.Close()

	client, err := sshclient.Connect(sshclient.Config{Host: "127.0.0.1", Port: s.Port, User: "u", KeyPath: s.KeyPath,
		KnownHostsPath: s.KnownHosts, ConfirmHostKey: func(string, string, string) (bool, error) { return true, nil }})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	cfg := &config.Config{App: config.AppConfig{Name: "demo"}, Server: config.ServerConfig{Host: strings.TrimPrefix(old.URL, "http://")}}
	e := New(client, cfg, &ai.DeploymentPlan{Dockerfile: "FROM x", DockerCompose: "services: {}"}, local, 8080)
	e.PollInterval = 0
	var out strings.Builder
	e.Out = &out

	r := last(e.Run(context.Background(), nil))
	if r.Success || r.Name != "Build images" {
		t.Fatalf("got step %q success=%v err=%v; want a build failure", r.Name, r.Success, r.Err)
	}
	if !strings.Contains(out.String(), "failed to build") {
		t.Errorf("build output was not streamed:\n%s", out.String())
	}
	calls, _ := os.ReadFile(log)
	for _, bad := range []string{"stop", "rm ", "prune", " up "} {
		if strings.Contains(string(calls), bad) {
			t.Errorf("containers touched after a failed build:\n%s", calls)
		}
	}
	if env, _ := os.ReadFile(appDir + "/.env"); string(env) != "PROD=1\n" {
		t.Errorf("server .env changed: %q", env)
	}
	if info, _ := os.Stat(appDir + "/.env"); info.Mode().Perm() != 0600 {
		t.Errorf("server .env mode = %v", info.Mode())
	}
}
