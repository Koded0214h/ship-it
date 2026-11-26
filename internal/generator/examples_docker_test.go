//go:build docker

package generator

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Koded0214h/ship/internal/detector"
)

// Builds each example with the generated Dockerfile, runs it and requests "/".
func TestExamplesBuildAndServe(t *testing.T) {
	entries, err := os.ReadDir("../../examples")
	if err != nil {
		t.Fatal(err)
	}
	docker := func(args ...string) (string, error) {
		out, err := exec.Command("docker", args...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			// Work on a copy so nothing is written into the example.
			dir := t.TempDir()
			if err := os.CopyFS(dir, os.DirFS(filepath.Join("../../examples", name))); err != nil {
				t.Fatal(err)
			}
			info, err := detector.Detect(dir)
			if err != nil {
				t.Fatal(err)
			}
			dockerfile := Dockerfile(info)
			if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
				t.Fatal(err)
			}
			tag := "ship-example-" + name
			if out, err := docker("build", "-t", tag, dir); err != nil {
				t.Fatalf("build failed:\n%s\n--- output ---\n%s", dockerfile, tail(out, 30))
			}
			t.Cleanup(func() { docker("rmi", "-f", tag) })

			port := strconv.Itoa(info.Port)
			id, err := docker("run", "-d", "-p", "127.0.0.1::"+port, "-e", "PORT="+port, tag)
			if err != nil {
				t.Fatalf("run: %s", id)
			}
			t.Cleanup(func() { docker("rm", "-f", id) })
			addr, err := docker("port", id, port)
			if err != nil {
				logs, _ := docker("logs", id)
				t.Fatalf("container is not running:\n%s\n%s", dockerfile, tail(logs, 20))
			}

			client := &http.Client{Timeout: 3 * time.Second}
			var last string
			for i := 0; i < 20; i++ {
				resp, err := client.Get("http://" + strings.Split(addr, "\n")[0] + "/")
				if err == nil {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					if resp.StatusCode == 200 && strings.TrimSpace(string(body)) == "ok" {
						return
					}
					last = resp.Status + " " + string(body)
				} else {
					last = err.Error()
				}
				time.Sleep(time.Second)
			}
			logs, _ := docker("logs", id)
			t.Fatalf("no 'ok' response (%s)\n%s\n%s", last, dockerfile, tail(logs, 20))
		})
	}
}

func tail(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
