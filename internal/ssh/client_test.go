//go:build !windows

package ssh_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	sshclient "github.com/Koded0214h/ship/internal/ssh"
	"github.com/Koded0214h/ship/internal/ssh/sshtest"
)

func config(s *sshtest.Server, confirm sshclient.ConfirmHostKeyFunc) sshclient.Config {
	return sshclient.Config{Host: "127.0.0.1", Port: s.Port, User: "u", KeyPath: s.KeyPath,
		KnownHostsPath: s.KnownHosts, ConfirmHostKey: confirm}
}

func trust(string, string, string) (bool, error)  { return true, nil }
func refuse(string, string, string) (bool, error) { return false, nil }

func TestHostKeyVerification(t *testing.T) {
	s := sshtest.Start(t)

	if _, err := sshclient.Connect(config(s, nil)); err == nil {
		t.Fatal("connected to an unknown server without confirmation")
	}
	if _, err := sshclient.Connect(config(s, refuse)); err == nil {
		t.Fatal("connected to a server the user refused")
	}
	c, err := sshclient.Connect(config(s, trust))
	if err != nil {
		t.Fatalf("trusted connect: %v", err)
	}
	c.Close()
	// Now on file: no question is asked.
	c, err = sshclient.Connect(config(s, nil))
	if err != nil {
		t.Fatalf("known server: %v", err)
	}
	c.Close()

	// A different server presenting itself at a known address is refused.
	other := sshtest.Start(t)
	data, err := os.ReadFile(s.KnownHosts)
	if err != nil {
		t.Fatal(err)
	}
	moved := strings.ReplaceAll(string(data), ":"+itoa(s.Port), ":"+itoa(other.Port))
	if err := os.WriteFile(other.KnownHosts, []byte(moved), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = sshclient.Connect(config(other, trust))
	if err == nil || !strings.Contains(err.Error(), "HOST KEY CHANGED") {
		t.Fatalf("expected a changed-key refusal, got %v", err)
	}
}

func TestUploadDirExcludesSecretsAndKeepsSource(t *testing.T) {
	s := sshtest.Start(t)
	c, err := sshclient.Connect(config(s, trust))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	local := t.TempDir()
	write := func(name string) {
		p := filepath.Join(local, name)
		os.MkdirAll(filepath.Dir(p), 0755)
		if err := os.WriteFile(p, []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"main.py", ".env", ".env.production", "api/.env", "prod.env", "certs/tls.pem",
		"id_ed25519", ".npmrc", ".env.example", "web/node_modules/x.js", ".git/config", "env/settings.py",
		"myenv/pyvenv.cfg", "dist/out.js"} {
		write(f)
	}
	os.Symlink("main.py", filepath.Join(local, "link.py"))
	os.Symlink("/etc/hosts", filepath.Join(local, "escape"))

	// A quote and spaces in the path must not break the remote command.
	remote := s.Home + "/it's here"
	skipped, err := c.UploadDir(context.Background(), local, remote)
	if err != nil {
		t.Fatalf("UploadDir: %v", err)
	}
	var got []string
	filepath.Walk(remote, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(remote, p)
			got = append(got, filepath.ToSlash(rel))
		}
		return nil
	})
	slices.Sort(got)
	want := []string{".env.example", "env/settings.py", "link.py", "main.py"}
	if !slices.Equal(got, want) {
		t.Fatalf("uploaded %v, want %v", got, want)
	}
	joined := strings.Join(skipped, "\n")
	for _, expected := range []string{".env ", "api/.env", "certs/tls.pem", "escape", "dist/"} {
		if !strings.Contains(joined, expected) {
			t.Errorf("skip report is missing %q:\n%s", expected, joined)
		}
	}
}

func TestWriteFilePermissionsAndLock(t *testing.T) {
	s := sshtest.Start(t)
	os.WriteFile(s.Bin+"/flock", []byte("#!/bin/sh\n[ \"$1\" = -n ] && shift\nf=$1; shift\n[ -e \"$f.busy\" ] && exit 1\nexec \"$@\"\n"), 0755)
	c, err := sshclient.Connect(config(s, trust))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx := context.Background()

	p := s.Home + "/a b/.env"
	if err := c.WriteFile(ctx, p, "K=V\n", 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(p)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %v, err = %v; want 0600", info.Mode(), err)
	}

	release, err := c.Lock(ctx, s.Home+"/lock")
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	release()
	os.WriteFile(s.Home+"/lock.busy", nil, 0600)
	if _, err := c.Lock(ctx, s.Home+"/lock"); err == nil {
		t.Fatal("expected a held lock to be refused")
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
