package cmd

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Koded0214h/ship/internal/ai"
	"github.com/Koded0214h/ship/internal/config"
	"github.com/Koded0214h/ship/internal/detector"
)

func TestDomainPointsTo(t *testing.T) {
	lookup := func(_ context.Context, host string) ([]string, error) {
		return map[string][]string{"good.test": {"203.0.113.7"}, "other.test": {"198.51.100.1"}}[host], nil
	}
	if _, err := domainPointsTo(context.Background(), lookup, "good.test", "203.0.113.7"); err != nil {
		t.Fatal(err)
	}
	if _, err := domainPointsTo(context.Background(), lookup, "other.test", "203.0.113.7"); err == nil {
		t.Fatal("a domain pointing elsewhere must fail")
	}
}

func TestContainerSummary(t *testing.T) {
	if got, err := containerSummary("app running\nnginx running\n"); err != nil || got != "app, nginx running" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := containerSummary("app restarting\nnginx running\n"); err == nil {
		t.Fatal("a restarting service must fail")
	}
	if got, err := containerSummary(""); err != nil || got != "not deployed yet" {
		t.Fatalf("%q %v", got, err)
	}
}

// With --yes and --dry-run nothing may prompt: there is no terminal here, so a
// prompt would fail the test.
func TestNonInteractiveChoices(t *testing.T) {
	t.Cleanup(func() { assumeYes, dryRun = false, false })
	assumeYes = true

	cfg := &config.Config{App: config.AppConfig{Name: "demo", Port: 9000, Domain: "a.test"}}
	info := &detector.ProjectInfo{Language: "Go", Framework: "Go", Port: 8080, RootDir: t.TempDir()}
	if err := resolvePort(cfg, info); err != nil || info.Port != 9000 {
		t.Fatalf("app.port must win: %d %v", info.Port, err)
	}
	plan, err := deterministicPlan(info, cfg)
	if err != nil || plan.Domain != "a.test" {
		t.Fatalf("%+v %v", plan, err)
	}

	root := t.TempDir()
	if err := os.WriteFile(root+"/Dockerfile", []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}
	own := &ai.DeploymentPlan{Dockerfile: "generated"}
	if err := chooseDockerfile(own, root); err != nil || own.Dockerfile != "" {
		t.Fatalf("the user's Dockerfile must be kept: %q %v", own.Dockerfile, err)
	}
}

func TestDryRunChangesNothing(t *testing.T) {
	t.Cleanup(func() { assumeYes, dryRun = false, false })
	dryRun = true
	dir := t.TempDir()
	t.Chdir(dir)
	// An unreachable server proves no connection is attempted.
	saved := &config.Config{
		App:    config.AppConfig{Name: "demo", Port: 9000},
		Server: config.ServerConfig{Host: "192.0.2.1", User: "u", Port: 22},
	}
	if err := config.SaveNew(saved); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("go.mod", []byte("module demo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(".ship/config.yaml")

	deployCmd.SetContext(context.Background())
	if err := runDeploy(deployCmd, nil); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(".ship/config.yaml")
	if !strings.EqualFold(string(before), string(after)) {
		t.Errorf("config changed:\n%s", after)
	}
	entries, _ := os.ReadDir(".")
	if len(entries) != 2 {
		t.Errorf("files were created: %v", entries)
	}
}
