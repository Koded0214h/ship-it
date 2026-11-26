package deploy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Koded0214h/ship/internal/ai"
	"github.com/Koded0214h/ship/internal/config"
	"github.com/Koded0214h/ship/internal/generator"
	sshclient "github.com/Koded0214h/ship/internal/ssh"
)

// Remote is what the executor needs from a server. *ssh.Client implements it.
type Remote interface {
	Output(ctx context.Context, cmd string) (string, error)
	Stream(ctx context.Context, cmd string, out io.Writer) error
	WriteFile(ctx context.Context, remotePath, content string, mode os.FileMode) error
	UploadDir(ctx context.Context, localDir, remoteDir string) ([]string, error)
	Lock(ctx context.Context, lockPath string) (func(), error)
}

const composeFile = "docker-compose.yml"

// compose is the only way Ship invokes Compose, so a compose.yaml uploaded with
// the user's source can never take precedence over the reviewed file.
const compose = "docker compose -f " + composeFile

type Executor struct {
	client   Remote
	cfg      *config.Config
	plan     *ai.DeploymentPlan
	appDir   string
	localDir string
	appPort  int

	// Out receives build output and notices.
	Out io.Writer

	BuildTimeout time.Duration
	PollInterval time.Duration

	unlock func()
}

type StepResult struct {
	Name    string
	Success bool
	Err     error
}

func New(client Remote, cfg *config.Config, plan *ai.DeploymentPlan, localDir string, appPort int) *Executor {
	return &Executor{
		client: client, cfg: cfg, plan: plan, localDir: localDir, appPort: appPort,
		Out: io.Discard, BuildTimeout: 30 * time.Minute, PollInterval: 5 * time.Second,
	}
}

type step struct {
	name string
	fn   func(context.Context) error
}

// steps lists what a deploy does, in order. Nothing running on the server is
// touched until "Start containers", which only happens after a successful build.
func (e *Executor) steps() []step {
	return []step{
		{"Check server requirements", e.preflight},
		{"Lock app directory", e.lockAppDir},
		{"Check port 80 is available", e.checkPort},
		{"Upload source code", e.uploadSourceCode},
		{"Write deployment files", e.writeDeploymentFiles},
		{"Create .env if missing", e.writeEnvTemplate},
		{"Build images", e.build},
		{"Start containers", e.start},
		{"Verify app container", e.verifyApp},
		{"Health check", e.healthCheck},
	}
}

// StepNames returns the steps a deploy will perform, for showing to the user.
func (e *Executor) StepNames() []string {
	var names []string
	for _, s := range e.steps() {
		names = append(names, s.name)
	}
	return names
}

// Run executes the steps and stops at the first failure.
func (e *Executor) Run(ctx context.Context, stepFn func(name string)) []StepResult {
	defer func() {
		if e.unlock != nil {
			e.unlock()
			e.unlock = nil
		}
	}()

	var results []StepResult
	for _, s := range e.steps() {
		if stepFn != nil {
			stepFn(s.name)
		}
		err := ctx.Err()
		if err == nil {
			err = s.fn(ctx)
		}
		results = append(results, StepResult{Name: s.name, Success: err == nil, Err: err})
		if err != nil {
			break
		}
	}
	return results
}

func (e *Executor) preflight(ctx context.Context) error {
	checks := []struct{ cmd, problem string }{
		{"docker info >/dev/null 2>&1", "Docker is not installed, not running, or this user may not use it (Ship does not install Docker)"},
		{"docker compose version >/dev/null 2>&1", "Docker Compose v2 is not installed"},
		{"command -v flock >/dev/null && command -v bash >/dev/null", "flock and bash are required on the server"},
	}
	for _, c := range checks {
		if _, err := e.client.Output(ctx, c.cmd); err != nil {
			return errors.New(c.problem)
		}
	}
	return nil
}

func (e *Executor) lockAppDir(ctx context.Context) error {
	if err := config.ValidateAppName(e.cfg.App.Name); err != nil {
		return err
	}
	out, err := e.client.Output(ctx, `printf %s "$HOME"`)
	if err != nil {
		return fmt.Errorf("resolving home dir: %w", err)
	}
	e.appDir = path.Join(strings.TrimSpace(out), "ship", e.cfg.App.Name)
	if _, err := e.client.Output(ctx, "mkdir -p "+sshclient.Quote(e.appDir)); err != nil {
		return fmt.Errorf("creating %s: %w", e.appDir, err)
	}
	unlock, err := e.client.Lock(ctx, e.appDir+"/.ship.lock")
	if err != nil {
		return err
	}
	e.unlock = unlock
	return nil
}

// inApp builds a command that runs in the app directory.
func (e *Executor) inApp(cmd string) string {
	return "cd " + sshclient.Quote(e.appDir) + " && " + cmd
}

// checkPort fails early if something other than this app already holds port 80.
func (e *Executor) checkPort(ctx context.Context) error {
	// On a first deploy there is no compose file yet, so errors mean "not ours".
	ours, _ := e.client.Output(ctx, e.inApp(compose+" ps -q nginx 2>/dev/null"))
	if strings.TrimSpace(ours) != "" {
		return nil
	}
	listening, _ := e.client.Output(ctx, "ss -Hltn 'sport = :80' 2>/dev/null")
	if strings.TrimSpace(listening) != "" {
		return errors.New("port 80 on the server is already in use by another process or app. " +
			"Ship supports one app per server for now and will not stop anything it did not start")
	}
	return nil
}

func (e *Executor) uploadSourceCode(ctx context.Context) error {
	skipped, err := e.client.UploadDir(ctx, e.localDir, e.appDir)
	for _, s := range skipped {
		fmt.Fprintf(e.Out, "     not uploaded: %s\n", s)
	}
	return err
}

// dockerignoreRules keep secrets out of images, including a server-side .env.
const dockerignoreRules = `
# Added by Ship: keep secrets out of the image
.git
**/.env
**/.env.*
**/*.env
**/*.pem
**/*.key
!**/.env.example
!**/.env.sample
!**/.env.template
`

func (e *Executor) writeDeploymentFiles(ctx context.Context) error {
	existing, err := os.ReadFile(filepath.Join(e.localDir, ".dockerignore"))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading .dockerignore: %w", err)
	}

	files := []struct{ name, content string }{
		{".dockerignore", string(existing) + dockerignoreRules},
		{"Dockerfile", e.plan.Dockerfile}, // empty: keep the one from the source upload
		{composeFile, e.plan.DockerCompose},
		{"nginx.conf", e.NginxConfig()},
	}
	for _, f := range files {
		if f.content == "" {
			continue
		}
		if err := e.client.WriteFile(ctx, e.appDir+"/"+f.name, f.content, 0644); err != nil {
			return err
		}
	}
	return nil
}

// NginxConfig is the proxy config a deploy writes. HTTPS is not supported yet,
// so it is always Ship's own HTTP config.
func (e *Executor) NginxConfig() string {
	return generator.NginxConfig(e.plan.Domain, e.cfg.App.Name, e.appPort, false)
}

var envKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// envFile renders the initial .env, replacing the placeholder database password
// with a random one everywhere it appears.
func envFile(vars map[string]string) (string, error) {
	oldPassword := vars["POSTGRES_PASSWORD"]
	newPassword := ""
	if _, ok := vars["POSTGRES_PASSWORD"]; ok {
		raw := make([]byte, 24)
		if _, err := rand.Read(raw); err != nil {
			return "", fmt.Errorf("generating database password: %w", err)
		}
		newPassword = hex.EncodeToString(raw)
	}

	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("# Generated by Ship. Review the values below.\n")
	for _, k := range keys {
		v := vars[k]
		if !envKey.MatchString(k) || strings.ContainsAny(v, "\r\n") {
			return "", fmt.Errorf("invalid environment variable %q in the plan", k)
		}
		switch {
		case k == "POSTGRES_PASSWORD":
			v = newPassword
		case newPassword != "" && oldPassword != "":
			v = strings.ReplaceAll(v, ":"+oldPassword+"@", ":"+newPassword+"@")
		}
		fmt.Fprintf(&sb, "%s=%s\n", k, v)
	}
	return sb.String(), nil
}

// writeEnvTemplate never overwrites an existing .env on the server.
func (e *Executor) writeEnvTemplate(ctx context.Context) error {
	remotePath := e.appDir + "/.env"
	q := sshclient.Quote(remotePath)
	out, err := e.client.Output(ctx, fmt.Sprintf("if [ -e %s ]; then chmod 600 %s && echo exists; else echo missing; fi", q, q))
	if err != nil {
		return fmt.Errorf("checking %s: %w", remotePath, err)
	}
	switch strings.TrimSpace(out) {
	case "exists":
		fmt.Fprintf(e.Out, "     keeping the existing %s\n", remotePath)
		return nil
	case "missing":
		content, err := envFile(e.plan.EnvVars)
		if err != nil {
			return err
		}
		if err := e.client.WriteFile(ctx, remotePath, content, 0600); err != nil {
			return err
		}
		fmt.Fprintf(e.Out, "     created %s — your local .env is never uploaded, so add your values there\n", remotePath)
		return nil
	default:
		return fmt.Errorf("unexpected reply while checking %s: %q", remotePath, out)
	}
}

func (e *Executor) build(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, e.BuildTimeout)
	defer cancel()
	// pipefail keeps the build's exit status while also saving the log.
	script := "set -o pipefail; " + compose + " build 2>&1 | tee deploy.log"
	if err := e.client.Stream(ctx, e.inApp("bash -c "+sshclient.Quote(script)), e.Out); err != nil {
		return fmt.Errorf("image build failed; running containers were not changed (log: %s/deploy.log): %w", e.appDir, err)
	}
	return nil
}

func (e *Executor) start(ctx context.Context) error {
	if err := e.client.Stream(ctx, e.inApp(compose+" up -d"), e.Out); err != nil {
		return fmt.Errorf("starting containers: %w", err)
	}
	// nginx resolves the app's address only at startup, and reads its config then.
	if err := e.client.Stream(ctx, e.inApp(compose+" up -d --force-recreate --no-deps nginx"), e.Out); err != nil {
		return fmt.Errorf("restarting nginx: %w", err)
	}
	return nil
}

// verifyApp requires the app container itself to stay running without
// restarting for several checks in a row.
func (e *Executor) verifyApp(ctx context.Context) error {
	const attempts, wantStable = 18, 3
	cmd := e.inApp(`id=$(` + compose + ` ps -a -q app) && [ -n "$id" ] && docker inspect -f '{{.State.Status}} {{.RestartCount}}' $id`)

	stable, last, state := 0, "", "no app container"
	for i := 0; i < attempts && stable < wantStable; i++ {
		if err := sleep(ctx, e.PollInterval); err != nil {
			return err
		}
		out, err := e.client.Output(ctx, cmd)
		current := strings.TrimSpace(out)
		if err != nil || !strings.HasPrefix(current, "running ") {
			stable, last = 0, ""
			if current != "" {
				state = current
			}
			continue
		}
		if current == last {
			stable++
		} else {
			stable = 1
		}
		last, state = current, current
	}
	if stable >= wantStable {
		return nil
	}
	logs, _ := e.client.Output(ctx, e.inApp(compose+" logs --no-color --tail=40 app 2>&1"))
	return fmt.Errorf("the app container is not staying up (last state: %s)\n\n%s", state, logs)
}

// healthCheckURL probes the server itself, so a domain that points somewhere
// else cannot make a broken deploy look healthy.
func healthCheckURL(serverHost string) string {
	if strings.Count(serverHost, ":") > 1 { // bare IPv6 address
		serverHost = "[" + serverHost + "]"
	}
	return "http://" + serverHost
}

func (e *Executor) healthCheck(ctx context.Context) error {
	url := healthCheckURL(e.cfg.Server.Host)

	// Don't follow redirects — a 3xx from the app counts as "app is up"
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	var lastErr error
	for i := 0; i < 6; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		if e.plan.Domain != "" {
			req.Host = e.plan.Domain
		}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		if err := sleep(ctx, e.PollInterval); err != nil {
			return err
		}
	}
	return fmt.Errorf("health check failed for %s: %w", url, lastErr)
}

func sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
