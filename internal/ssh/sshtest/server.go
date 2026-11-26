// Package sshtest runs an in-process SSH server for tests. Commands execute
// locally with an isolated HOME and PATH.
package sshtest

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/pem"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

type Server struct {
	Port       int
	Home       string
	Bin        string // first on PATH: put fake tools here
	KeyPath    string // client private key
	KnownHosts string
	HostKey    ssh.PublicKey
}

func newKey(t *testing.T) (ssh.Signer, []byte) {
	t.Helper()
	_, k, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(k)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(k, "")
	if err != nil {
		t.Fatal(err)
	}
	return signer, pem.EncodeToMemory(block)
}

// Start launches a server that lasts for the test.
func Start(t *testing.T) *Server {
	t.Helper()
	root := t.TempDir()
	s := &Server{
		Home:       filepath.Join(root, "home"),
		Bin:        filepath.Join(root, "bin"),
		KeyPath:    filepath.Join(root, "id"),
		KnownHosts: filepath.Join(root, "known_hosts"),
	}
	for _, d := range []string{s.Home, s.Bin} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	_, clientPEM := newKey(t)
	if err := os.WriteFile(s.KeyPath, clientPEM, 0600); err != nil {
		t.Fatal(err)
	}
	host, _ := newKey(t)
	s.HostKey = host.PublicKey()
	cfg := &ssh.ServerConfig{PublicKeyCallback: func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) { return nil, nil }}
	cfg.AddHostKey(host)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	s.Port = l.Addr().(*net.TCPAddr).Port
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go s.serve(conn, cfg)
		}
	}()
	return s
}

func (s *Server) serve(conn net.Conn, cfg *ssh.ServerConfig) {
	_, channels, requests, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		return
	}
	go ssh.DiscardRequests(requests)
	for newChannel := range channels {
		ch, reqs, err := newChannel.Accept()
		if err != nil {
			return
		}
		go func() {
			for req := range reqs {
				if req.Type != "exec" {
					req.Reply(false, nil)
					continue
				}
				req.Reply(true, nil)
				n := binary.BigEndian.Uint32(req.Payload[:4])
				s.run(ch, string(req.Payload[4:4+n]))
			}
		}()
	}
}

func (s *Server) run(ch ssh.Channel, command string) {
	defer ch.Close()
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Env = []string{"HOME=" + s.Home, "PATH=" + s.Bin + ":/usr/bin:/bin"}
	cmd.Dir = s.Home
	cmd.Stdout, cmd.Stderr = ch, ch.Stderr()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return
	}
	status := make([]byte, 4)
	if err := cmd.Start(); err != nil {
		binary.BigEndian.PutUint32(status, 127)
	} else {
		go func() {
			io.Copy(stdin, ch)
			stdin.Close()
		}()
		if cmd.Wait() != nil {
			binary.BigEndian.PutUint32(status, 1)
		}
	}
	ch.SendRequest("exit-status", false, status)
}
