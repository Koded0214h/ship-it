package cmd

import (
	"os"
	"testing"

	"github.com/Koded0214h/ship/internal/config"
)

func TestApplyAINeverReusesAKeyAcrossProviders(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: "openai", APIKey: "openai-key", Model: "gpt-4o"}}
	if err := applyAI(cfg, "anthropic", "", "gpt-4o"); err == nil {
		t.Fatal("expected a new key to be required")
	}
	if err := applyAI(cfg, "openai", "", "gpt-4o"); err != nil || cfg.AI.APIKey != "openai-key" {
		t.Fatalf("same provider should keep its key: %v %+v", err, cfg.AI)
	}
	if err := applyAI(cfg, "gemini", "g-key", "gpt-4o"); err != nil || cfg.AI.Model != "" {
		t.Fatalf("stale model kept: %v %+v", err, cfg.AI)
	}
}

func TestSanitize(t *testing.T) {
	if got := sanitize("a\x1b[2Kb\rc\n\td"); got != "a[2Kbc\n\td" {
		t.Fatalf("got %q", got)
	}
}

func TestLogsCommand(t *testing.T) {
	got, err := logsCommand(50, "app; rm -rf /", false, false)
	want := `docker compose -f docker-compose.yml logs --tail=50 -- 'app; rm -rf /'`
	if err != nil || got != want {
		t.Fatalf("got %q, %v", got, err)
	}
	if got, _ := logsCommand(10, "", true, true); got != "tail -n 10 deploy.log" {
		t.Fatalf("build log command: %q", got)
	}
	if _, err := logsCommand(-1, "", false, false); err == nil {
		t.Fatal("accepted a negative tail")
	}
}

func TestValidKeyPath(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"key", "key.pub"} {
		if err := os.WriteFile(dir+"/"+name, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := validKeyPath(dir + "/key"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{dir, dir + "/missing", dir + "/key.pub"} {
		if validKeyPath(bad) == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}
