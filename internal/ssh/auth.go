package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// PassphraseFunc asks for the passphrase of the key at path.
type PassphraseFunc func(path string) (string, error)

// authMethod loads the key at path. A passphrase-protected key is taken from
// ssh-agent when the agent holds it; otherwise the passphrase is asked for.
// The returned function releases the agent connection.
func authMethod(path string, passphrase PassphraseFunc) (ssh.AuthMethod, func(), error) {
	done := func() {}
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, done, fmt.Errorf("reading SSH key %s: %w", path, err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err == nil {
		return ssh.PublicKeys(signer), done, nil
	}
	var missing *ssh.PassphraseMissingError
	if !errors.As(err, &missing) {
		return nil, done, fmt.Errorf("parsing SSH key: %w", err)
	}

	public := missing.PublicKey
	if public == nil {
		if data, readErr := os.ReadFile(path + ".pub"); readErr == nil {
			public, _, _, _, _ = ssh.ParseAuthorizedKey(data)
		}
	}
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" && public != nil {
		if conn, dialErr := net.Dial("unix", sock); dialErr == nil {
			signers, _ := agent.NewClient(conn).Signers()
			for _, s := range signers {
				if bytes.Equal(s.PublicKey().Marshal(), public.Marshal()) {
					return ssh.PublicKeys(s), func() { conn.Close() }, nil
				}
			}
			conn.Close()
		}
	}

	if passphrase == nil {
		return nil, done, fmt.Errorf("SSH key %s needs a passphrase: add it to ssh-agent with 'ssh-add %s'", path, path)
	}
	secret, err := passphrase(path)
	if err != nil {
		return nil, done, err
	}
	signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(secret))
	if err != nil {
		return nil, done, fmt.Errorf("unlocking SSH key %s: %w", path, err)
	}
	return ssh.PublicKeys(signer), done, nil
}
