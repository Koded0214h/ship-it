//go:build !windows

package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Validates generated configs with nginx itself when it is installed.
func TestNginxAcceptsGeneratedConfigs(t *testing.T) {
	nginx, err := exec.LookPath("nginx")
	if err != nil {
		t.Skip("nginx not installed")
	}
	openssl, err := exec.LookPath("openssl")
	if err != nil {
		t.Skip("openssl not installed")
	}
	dir := t.TempDir()
	crt, key := filepath.Join(dir, "c.pem"), filepath.Join(dir, "k.pem")
	if out, err := exec.Command(openssl, "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "1",
		"-subj", "/CN=example.com", "-keyout", key, "-out", crt).CombinedOutput(); err != nil {
		t.Fatalf("openssl: %v\n%s", err, out)
	}

	for name, ssl := range map[string]bool{"http": false, "https": true} {
		t.Run(name, func(t *testing.T) {
			// Substitute what only exists inside the containers.
			cfg := strings.NewReplacer(
				"server app:", "server 127.0.0.1:",
				"/etc/letsencrypt/live/example.com/fullchain.pem", crt,
				"/etc/letsencrypt/live/example.com/privkey.pem", key,
				"listen 80;", "listen 18080;",
				"listen 443 ssl;", "listen 18443 ssl;",
				"http {", "http {\n access_log off; client_body_temp_path b; proxy_temp_path p; fastcgi_temp_path f; uwsgi_temp_path u; scgi_temp_path s;",
			).Replace(NginxConfig("example.com", "myapp", 3000, ssl))
			path := filepath.Join(dir, name+".conf")
			cfg = "pid " + filepath.Join(dir, "n.pid") + ";\n" + cfg
			if err := os.WriteFile(path, []byte(cfg), 0600); err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(nginx, "-t", "-p", dir, "-c", path, "-e", filepath.Join(dir, "err.log")).CombinedOutput()
			if err != nil {
				t.Fatalf("nginx rejected the config: %v\n%s", err, out)
			}
		})
	}
}
