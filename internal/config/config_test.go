package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// chdir switches the working directory for the duration of the test and
// restores it afterward. os.Chdir is process-global, so config tests must
// not run in parallel with each other or with anything else that chdirs.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir(%s): %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("restoring cwd: %v", err)
		}
	})
}

func TestSaveNewAndLoadRoundTrip(t *testing.T) {
	chdir(t, t.TempDir())

	want := &Config{
		App:    AppConfig{Name: "myapp", Domain: "example.com", Port: 8080},
		Server: ServerConfig{Host: "1.2.3.4", User: "deploy", KeyPath: "~/.ssh/id_ed25519", Port: 2222},
		AI:     AIConfig{Provider: "anthropic", APIKey: "sk-test", Model: "claude-sonnet"},
	}
	if err := SaveNew(want); err != nil {
		t.Fatalf("SaveNew: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if *got != *want {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, want)
	}
}

func TestLoadDefaultsServerPort(t *testing.T) {
	chdir(t, t.TempDir())

	cfg := &Config{App: AppConfig{Name: "app"}, Server: ServerConfig{Host: "1.2.3.4", User: "root"}}
	if err := SaveNew(cfg); err != nil {
		t.Fatalf("SaveNew: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Port != 22 {
		t.Fatalf("expected default server.port 22, got %d", got.Server.Port)
	}
}

func TestLoadPreservesExplicitPort(t *testing.T) {
	chdir(t, t.TempDir())

	cfg := &Config{App: AppConfig{Name: "app"}, Server: ServerConfig{Host: "1.2.3.4", User: "root", Port: 2222}}
	if err := SaveNew(cfg); err != nil {
		t.Fatalf("SaveNew: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Port != 2222 {
		t.Fatalf("expected explicit server.port 2222 to survive, got %d", got.Server.Port)
	}
}

func TestLoadNoConfigFound(t *testing.T) {
	chdir(t, t.TempDir())

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when no .ship/config.yaml exists, got nil")
	}
}

func TestFindConfigFileWalksUpToParent(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	chdir(t, root)
	cfg := &Config{App: AppConfig{Name: "app"}, Server: ServerConfig{Host: "1.2.3.4", User: "root"}}
	if err := SaveNew(cfg); err != nil {
		t.Fatalf("SaveNew: %v", err)
	}

	// Now descend into the nested subdirectory and confirm Load still finds
	// the config file at the root via findConfigFile's upward walk.
	chdir(t, sub)
	got, err := Load()
	if err != nil {
		t.Fatalf("Load from nested dir: %v", err)
	}
	if got.App.Name != "app" {
		t.Fatalf("got %+v, want App.Name=app", got)
	}
}

func TestSaveUpdatesExistingConfigInPlace(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "nested")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	chdir(t, root)
	cfg := &Config{App: AppConfig{Name: "app"}, Server: ServerConfig{Host: "1.2.3.4", User: "root"}}
	if err := SaveNew(cfg); err != nil {
		t.Fatalf("SaveNew: %v", err)
	}

	// Save (not SaveNew) from a nested directory should locate and overwrite
	// the existing config at the root rather than creating a new one nested
	// under the subdirectory.
	chdir(t, sub)
	cfg.App.Name = "renamed"
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(filepath.Join(sub, configDir, configFile)); err == nil {
		t.Fatal("Save created a new config file in the nested dir instead of updating the existing one")
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.App.Name != "renamed" {
		t.Fatalf("expected updated App.Name=renamed, got %q", got.App.Name)
	}
}

func TestExists(t *testing.T) {
	chdir(t, t.TempDir())

	if Exists() {
		t.Fatal("Exists() should be false before any config is saved")
	}
	if err := SaveNew(&Config{}); err != nil {
		t.Fatalf("SaveNew: %v", err)
	}
	if !Exists() {
		t.Fatal("Exists() should be true after SaveNew")
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     *Config
		wantErr []string
	}{
		{
			"valid",
			&Config{App: AppConfig{Name: "app"}, Server: ServerConfig{Host: "1.2.3.4", User: "root"}},
			nil,
		},
		{
			"missing everything",
			&Config{},
			[]string{"app.name is required", "server.host is required", "server.user is required"},
		},
		{
			"missing user only",
			&Config{App: AppConfig{Name: "app"}, Server: ServerConfig{Host: "1.2.3.4"}},
			[]string{"server.user is required"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Validate(c.cfg)
			if len(got) != len(c.wantErr) {
				t.Fatalf("got %v, want %v", got, c.wantErr)
			}
			for i := range got {
				if got[i] != c.wantErr[i] {
					t.Fatalf("got %v, want %v", got, c.wantErr)
				}
			}
		})
	}
}

func TestEnsureGitignoreAddsEntryOnce(t *testing.T) {
	chdir(t, t.TempDir())

	if err := EnsureGitignore(); err != nil {
		t.Fatalf("EnsureGitignore (no .gitignore yet): %v", err)
	}
	data, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	if !containsLine(string(data), ".ship/") {
		t.Fatalf(".gitignore does not contain .ship/ entry: %q", data)
	}

	// Calling it again should not duplicate the entry.
	if err := EnsureGitignore(); err != nil {
		t.Fatalf("EnsureGitignore (second call): %v", err)
	}
	data2, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	if got, want := countOccurrences(string(data2), ".ship/"), 1; got != want {
		t.Fatalf("expected .ship/ to appear once, appeared %d times in %q", got, data2)
	}
}

func TestEnsureGitignoreSkipsWhenAlreadyPresent(t *testing.T) {
	chdir(t, t.TempDir())

	if err := os.WriteFile(".gitignore", []byte("node_modules/\n.ship/\n"), 0644); err != nil {
		t.Fatalf("writing .gitignore: %v", err)
	}
	if err := EnsureGitignore(); err != nil {
		t.Fatalf("EnsureGitignore: %v", err)
	}
	data, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	if got, want := countOccurrences(string(data), ".ship/"), 1; got != want {
		t.Fatalf("expected .ship/ to still appear once, appeared %d times in %q", got, data)
	}
}

func containsLine(content, line string) bool {
	return slices.Contains(splitLines(content), line)
}

func countOccurrences(content, line string) int {
	n := 0
	for _, l := range splitLines(content) {
		if l == line {
			n++
		}
	}
	return n
}
