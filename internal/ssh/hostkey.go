package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ConfirmHostKeyFunc decides whether to trust a server seen for the first time.
type ConfirmHostKeyFunc func(host, keyType, fingerprint string) (bool, error)

// hostKeyVerifier checks servers against known_hosts. An unknown server must be
// confirmed; a changed key is always refused. It also returns the key
// algorithms to request, so a server is asked for the key type already on file.
func hostKeyVerifier(file, addr string, confirm ConfirmHostKeyFunc) (ssh.HostKeyCallback, []string, error) {
	if file == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, nil, err
		}
		file = filepath.Join(home, ".ssh", "known_hosts")
	}
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		return nil, nil, fmt.Errorf("creating %s: %w", filepath.Dir(file), err)
	}
	f, err := os.OpenFile(file, os.O_CREATE|os.O_RDONLY, 0600)
	if err != nil {
		return nil, nil, fmt.Errorf("opening %s: %w", file, err)
	}
	f.Close()

	known, err := knownhosts.New(file)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", file, err)
	}

	callback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := known(hostname, remote, key)
		if err == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if !errors.As(err, &keyErr) {
			return err
		}
		fingerprint := ssh.FingerprintSHA256(key)
		if len(keyErr.Want) > 0 {
			return fmt.Errorf("HOST KEY CHANGED for %s (got %s %s). This can mean someone is intercepting the connection. "+
				"If you rebuilt the server, remove the old entry with: ssh-keygen -R '%s'",
				hostname, key.Type(), fingerprint, knownhosts.Normalize(hostname))
		}
		if confirm == nil {
			return fmt.Errorf("unknown server %s (%s %s): run 'ship connect' to review and trust it", hostname, key.Type(), fingerprint)
		}
		ok, err := confirm(hostname, key.Type(), fingerprint)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("server %s was not trusted", hostname)
		}
		out, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("saving host key: %w", err)
		}
		defer out.Close()
		line := knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key)
		// The leading newline guards against a file without a trailing one.
		if _, err := out.WriteString("\n" + line + "\n"); err != nil {
			return fmt.Errorf("saving host key: %w", err)
		}
		return nil
	}

	return callback, knownAlgorithms(known, addr), nil
}

// knownAlgorithms returns the algorithms matching the keys on file for addr, or
// nil (library defaults) when the server is unknown.
func knownAlgorithms(known ssh.HostKeyCallback, addr string) []string {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil
	}
	probe, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil
	}
	var keyErr *knownhosts.KeyError
	if !errors.As(known(addr, &net.TCPAddr{IP: net.IPv4zero}, probe), &keyErr) {
		return nil
	}
	var algorithms []string
	for _, k := range keyErr.Want {
		if t := k.Key.Type(); t == ssh.KeyAlgoRSA {
			algorithms = append(algorithms, ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA)
		} else {
			algorithms = append(algorithms, t)
		}
	}
	return algorithms
}

// syncWriter serialises the concurrent stdout and stderr copies of a session.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}
