package ssh

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	conn *ssh.Client
	Host string
	User string
}

type Config struct {
	Host    string
	User    string
	KeyPath string
	Port    int

	// KnownHostsPath defaults to ~/.ssh/known_hosts.
	KnownHostsPath string
	// ConfirmHostKey is asked before trusting a server seen for the first
	// time. When nil, unknown servers are refused.
	ConfirmHostKey ConfirmHostKeyFunc
	// Passphrase is asked when the key is protected and not in ssh-agent.
	Passphrase PassphraseFunc
}

func Connect(cfg Config) (*Client, error) {
	keyPath := expandHome(cfg.KeyPath)
	auth, release, err := authMethod(keyPath, cfg.Passphrase)
	if err != nil {
		return nil, err
	}
	defer release()

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	hostKeys, algorithms, err := hostKeyVerifier(cfg.KnownHostsPath, addr, cfg.ConfirmHostKey)
	if err != nil {
		return nil, err
	}
	conn, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User:              cfg.User,
		Auth:              []ssh.AuthMethod{auth},
		HostKeyCallback:   hostKeys,
		HostKeyAlgorithms: algorithms,
		Timeout:           15 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("SSH connect to %s: %w", addr, err)
	}
	return &Client{conn: conn, Host: cfg.Host, User: cfg.User}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Quote makes s safe to use as a single POSIX shell word.
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// run executes cmd and waits for it, closing the session if ctx ends first.
func (c *Client) run(ctx context.Context, cmd string, stdin io.Reader, stdout, stderr io.Writer) error {
	sess, err := c.conn.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()
	sess.Stdin = stdin
	sess.Stdout = stdout
	sess.Stderr = stderr

	if err := sess.Start(cmd); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- sess.Wait() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		sess.Close()
		return ctx.Err()
	}
}

// Output runs cmd and returns its standard output.
func (c *Client) Output(ctx context.Context, cmd string) (string, error) {
	var out, errOut bytes.Buffer
	if err := c.run(ctx, cmd, nil, &out, &errOut); err != nil {
		if msg := strings.TrimSpace(errOut.String()); msg != "" {
			return out.String(), fmt.Errorf("%w: %s", err, msg)
		}
		return out.String(), err
	}
	return out.String(), nil
}

// Stream runs cmd, copying both standard output and standard error to out.
func (c *Client) Stream(ctx context.Context, cmd string, out io.Writer) error {
	w := &syncWriter{w: out}
	return c.run(ctx, cmd, nil, w, w)
}

// WriteFile writes content to a remote file and sets its permissions. The file
// is never readable by others while it is being written.
func (c *Client) WriteFile(ctx context.Context, remotePath, content string, mode os.FileMode) error {
	cmd := fmt.Sprintf("mkdir -p %s && (umask 077 && cat > %s) && chmod %o %s",
		Quote(path.Dir(remotePath)), Quote(remotePath), mode.Perm(), Quote(remotePath))
	var errOut bytes.Buffer
	if err := c.run(ctx, cmd, strings.NewReader(content), io.Discard, &errOut); err != nil {
		return fmt.Errorf("write file %s: %w\n%s", remotePath, err, errOut.String())
	}
	return nil
}

// Lock takes an exclusive lock on lockPath using flock on the server. The lock
// is released by calling the returned function, or automatically if the
// connection drops.
func (c *Client) Lock(ctx context.Context, lockPath string) (func(), error) {
	sess, err := c.conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	stdin, err := sess.StdinPipe()
	if err != nil {
		sess.Close()
		return nil, err
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		sess.Close()
		return nil, err
	}
	// cat holds the lock until its stdin closes.
	if err := sess.Start(fmt.Sprintf("flock -n %s sh -c 'echo locked; cat >/dev/null'", Quote(lockPath))); err != nil {
		sess.Close()
		return nil, err
	}

	got := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadString('\n')
		got <- strings.TrimSpace(line)
	}()
	select {
	case line := <-got:
		if line != "locked" {
			sess.Close()
			return nil, fmt.Errorf("another deployment of this app appears to be running (could not lock %s)", lockPath)
		}
	case <-ctx.Done():
		sess.Close()
		return nil, ctx.Err()
	}
	return func() {
		stdin.Close()
		_ = sess.Wait()
		sess.Close()
	}, nil
}

// ServerInfo returns basic server information.
type ServerInfo struct {
	OS      string
	Arch    string
	Disk    string
	Memory  string
	Docker  bool
	Compose bool
}

func (c *Client) GetServerInfo(ctx context.Context) ServerInfo {
	info := ServerInfo{}
	if out, err := c.Output(ctx, "uname -sr"); err == nil {
		info.OS = strings.TrimSpace(out)
	}
	if out, err := c.Output(ctx, "uname -m"); err == nil {
		info.Arch = strings.TrimSpace(out)
	}
	if out, err := c.Output(ctx, "df -h / | tail -1 | awk '{print $4}'"); err == nil {
		info.Disk = strings.TrimSpace(out) + " available"
	}
	if out, err := c.Output(ctx, "free -h | grep Mem | awk '{print $7}'"); err == nil {
		info.Memory = strings.TrimSpace(out) + " available"
	}
	// "docker info" also fails when the daemon is down or the user may not use it.
	if _, err := c.Output(ctx, "docker info >/dev/null 2>&1"); err == nil {
		info.Docker = true
	}
	if _, err := c.Output(ctx, "docker compose version >/dev/null 2>&1"); err == nil {
		info.Compose = true
	}
	return info
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
