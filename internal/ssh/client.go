package ssh

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
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
}

func Connect(cfg Config) (*Client, error) {
	keyPath := expandHome(cfg.KeyPath)
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading SSH key %s: %w", keyPath, err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parsing SSH key: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	conn, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("SSH connect to %s: %w", addr, err)
	}
	return &Client{conn: conn, Host: cfg.Host, User: cfg.User}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Exec(cmd string) (stdout, stderr string, err error) {
	sess, err := c.conn.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	var outBuf, errBuf bytes.Buffer
	sess.Stdout = &outBuf
	sess.Stderr = &errBuf
	err = sess.Run(cmd)
	return outBuf.String(), errBuf.String(), err
}

func (c *Client) ExecStream(cmd string, out io.Writer) error {
	sess, err := c.conn.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()
	sess.Stdout = out
	sess.Stderr = out
	return sess.Run(cmd)
}

// Upload writes content to remotePath on the server using SCP.
func (c *Client) Upload(content, remotePath string) error {
	sess, err := c.conn.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	dir := filepath.Dir(remotePath)
	filename := filepath.Base(remotePath)
	size := len(content)

	go func() {
		w, _ := sess.StdinPipe()
		defer w.Close()
		fmt.Fprintf(w, "C0644 %d %s\n", size, filename)
		io.WriteString(w, content)
		fmt.Fprint(w, "\x00")
	}()

	return sess.Run(fmt.Sprintf("mkdir -p %s && scp -qt %s", dir, dir))
}

// UploadFile uploads a local file to the remote server.
func (c *Client) UploadFile(localPath, remotePath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	return c.Upload(string(data), remotePath)
}

// WriteFileSudo writes content to a remote path that requires root, using sudo tee.
func (c *Client) WriteFileSudo(remotePath, content string) error {
	dir := filepath.Dir(remotePath)
	if _, _, err := c.Exec(fmt.Sprintf("sudo mkdir -p %s", dir)); err != nil {
		return fmt.Errorf("sudo mkdir %s: %w", dir, err)
	}

	sess, err := c.conn.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	var errBuf bytes.Buffer
	sess.Stderr = &errBuf
	w, err := sess.StdinPipe()
	if err != nil {
		return err
	}

	if err := sess.Start(fmt.Sprintf("sudo tee %s > /dev/null", remotePath)); err != nil {
		return err
	}
	if _, err := io.WriteString(w, content); err != nil {
		return err
	}
	w.Close()
	if err := sess.Wait(); err != nil {
		return fmt.Errorf("sudo tee %s: %w\n%s", remotePath, err, errBuf.String())
	}
	return nil
}

// MkdirAll creates directories on the remote server.
func (c *Client) MkdirAll(path string) error {
	_, _, err := c.Exec(fmt.Sprintf("mkdir -p %s", path))
	return err
}

// WriteFile writes content to a remote file.
func (c *Client) WriteFile(remotePath, content string) error {
	dir := filepath.Dir(remotePath)
	if _, _, err := c.Exec(fmt.Sprintf("mkdir -p %s", dir)); err != nil {
		return err
	}
	// Use heredoc-style approach via stdin
	sess, err := c.conn.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	var errBuf bytes.Buffer
	sess.Stderr = &errBuf
	w, err := sess.StdinPipe()
	if err != nil {
		return err
	}

	if err := sess.Start(fmt.Sprintf("cat > %s", remotePath)); err != nil {
		return err
	}
	if _, err := io.WriteString(w, content); err != nil {
		return err
	}
	w.Close()
	if err := sess.Wait(); err != nil {
		return fmt.Errorf("write file %s: %w\n%s", remotePath, err, errBuf.String())
	}
	return nil
}

// UploadDir tars localDir and extracts it into remoteDir on the server.
// It skips common build artifacts and large files.
func (c *Client) UploadDir(localDir, remoteDir string) error {
	skipDirs := map[string]bool{
		".git": true, ".ship": true, "node_modules": true, "vendor": true,
		"__pycache__": true, ".venv": true, "venv": true, "env": true,
		"dist": true, "build": true, ".next": true, "target": true, ".idea": true,
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	err := filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		// Skip based on top-level component
		parts := strings.SplitN(rel, string(filepath.Separator), 2)
		if skipDirs[parts[0]] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories themselves (tar recreates them via file paths)
		if info.IsDir() {
			return nil
		}

		// Skip files over 30 MB
		if info.Size() > 30*1024*1024 {
			return nil
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return fmt.Errorf("building source archive: %w", err)
	}
	tw.Close()
	gw.Close()

	sess, err := c.conn.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	var errBuf bytes.Buffer
	sess.Stderr = &errBuf
	w, err := sess.StdinPipe()
	if err != nil {
		return err
	}

	if err := sess.Start(fmt.Sprintf("mkdir -p %s && tar xzf - -C %s", remoteDir, remoteDir)); err != nil {
		return err
	}
	if _, err := io.Copy(w, &buf); err != nil {
		return err
	}
	w.Close()
	if err := sess.Wait(); err != nil {
		return fmt.Errorf("extract source archive: %w\n%s", err, errBuf.String())
	}
	return nil
}

// ServerInfo returns basic server information.
type ServerInfo struct {
	OS      string
	Arch    string
	Disk    string
	Memory  string
	Docker  bool
}

func (c *Client) GetServerInfo() ServerInfo {
	info := ServerInfo{}
	if out, _, err := c.Exec("uname -sr"); err == nil {
		info.OS = strings.TrimSpace(out)
	}
	if out, _, err := c.Exec("uname -m"); err == nil {
		info.Arch = strings.TrimSpace(out)
	}
	if out, _, err := c.Exec("df -h / | tail -1 | awk '{print $4}' "); err == nil {
		info.Disk = strings.TrimSpace(out) + " available"
	}
	if out, _, err := c.Exec("free -h | grep Mem | awk '{print $7}'"); err == nil {
		info.Memory = strings.TrimSpace(out) + " available"
	}
	if _, _, err := c.Exec("docker --version"); err == nil {
		info.Docker = true
	}
	return info
}

// IsPortOpen checks if a TCP port is accepting connections.
func IsPortOpen(host string, port int) bool {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
