//go:build !windows

package ssh_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sshclient "github.com/Koded0214h/ship/internal/ssh"
	"github.com/Koded0214h/ship/internal/ssh/sshtest"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func lockedKey(t *testing.T) (ed25519.PrivateKey, string) {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKeyWithPassphrase(key, "", []byte("hunter2"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
		t.Fatal(err)
	}
	return key, path
}

func TestPassphraseAndAgent(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	s := sshtest.Start(t)
	key, path := lockedKey(t)
	cfg := config(s, trust)
	cfg.KeyPath = path

	if _, err := sshclient.Connect(cfg); err == nil || !strings.Contains(err.Error(), "ssh-add") {
		t.Fatalf("expected advice to use ssh-add, got %v", err)
	}
	cfg.Passphrase = func(string) (string, error) { return "wrong", nil }
	if _, err := sshclient.Connect(cfg); err == nil {
		t.Fatal("accepted a wrong passphrase")
	}
	cfg.Passphrase = func(string) (string, error) { return "hunter2", nil }
	c, err := sshclient.Connect(cfg)
	if err != nil {
		t.Fatalf("with passphrase: %v", err)
	}
	c.Close()

	// With the key in an agent, nothing is asked.
	keyring := agent.NewKeyring()
	if err := keyring.Add(agent.AddedKey{PrivateKey: key}); err != nil {
		t.Fatal(err)
	}
	// Unix socket paths are length limited, so avoid t.TempDir.
	dir, err := os.MkdirTemp("/tmp", "ag")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	listener, err := net.Listen("unix", dir+"/s")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go agent.ServeAgent(keyring, conn)
		}
	}()
	t.Setenv("SSH_AUTH_SOCK", dir+"/s")
	cfg.Passphrase = func(string) (string, error) {
		t.Error("asked for a passphrase although the agent holds the key")
		return "", nil
	}
	c, err = sshclient.Connect(cfg)
	if err != nil {
		t.Fatalf("with agent: %v", err)
	}
	c.Close()
}

func TestShipIgnore(t *testing.T) {
	s := sshtest.Start(t)
	c, err := sshclient.Connect(config(s, trust))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	local := t.TempDir()
	for _, f := range []string{"main.go", "notes.log", "deep/x.log", "docs/a.md", "certs/ca.pem", "certs/other.pem", "build/gen.go"} {
		p := filepath.Join(local, f)
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(f), 0644)
	}
	os.WriteFile(filepath.Join(local, ".shipignore"), []byte("# comment\n*.log\ndocs/\n!certs/ca.pem\n!build\n"), 0644)
	remote := s.Home + "/app"
	if _, err := c.UploadDir(context.Background(), local, remote); err != nil {
		t.Fatal(err)
	}
	for file, want := range map[string]bool{"main.go": true, "certs/ca.pem": true, "build/gen.go": true,
		"notes.log": false, "deep/x.log": false, "docs/a.md": false, "certs/other.pem": false} {
		if _, err := os.Stat(filepath.Join(remote, file)); (err == nil) != want {
			t.Errorf("%s uploaded=%v, want %v", file, err == nil, want)
		}
	}

	os.WriteFile(filepath.Join(local, ".shipignore"), []byte("**/*.log\n"), 0644)
	if _, err := c.UploadDir(context.Background(), local, remote); err == nil {
		t.Error("an unsupported pattern must be an error, not silently ignored")
	}
}
