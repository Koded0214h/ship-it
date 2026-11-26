package cmd

import (
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
