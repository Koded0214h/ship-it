package sshkey

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

type KeyInfo struct {
	PrivatePath string
	PublicPath  string
	IsNew       bool
	PublicKey   string
}

// FindExisting scans ~/.ssh for known private key files.
func FindExisting() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	sshDir := filepath.Join(home, ".ssh")
	candidates := []string{"id_ed25519", "id_rsa", "id_ecdsa", "ship_ed25519"}

	var found []string
	for _, name := range candidates {
		path := filepath.Join(sshDir, name)
		if _, err := os.Stat(path); err == nil {
			found = append(found, path)
		}
	}
	return found
}

// Generate creates a new ED25519 key pair at ~/.ssh/ship_ed25519.
func Generate() (*KeyInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return nil, fmt.Errorf("creating ~/.ssh: %w", err)
	}

	privPath := filepath.Join(sshDir, "ship_ed25519")
	pubPath := privPath + ".pub"

	pubRaw, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	// Marshal private key as OpenSSH PEM using x/crypto
	pemBlock, err := ssh.MarshalPrivateKey(priv, "ship-deploy")
	if err != nil {
		return nil, fmt.Errorf("encoding private key: %w", err)
	}
	privPEM := pem.EncodeToMemory(pemBlock)
	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		return nil, fmt.Errorf("writing private key: %w", err)
	}

	// Marshal public key in authorized_keys format
	sshPub, err := ssh.NewPublicKey(pubRaw)
	if err != nil {
		return nil, fmt.Errorf("encoding public key: %w", err)
	}
	pubLine := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub))) + " ship-deploy\n"
	if err := os.WriteFile(pubPath, []byte(pubLine), 0644); err != nil {
		return nil, fmt.Errorf("writing public key: %w", err)
	}

	return &KeyInfo{
		PrivatePath: privPath,
		PublicPath:  pubPath,
		IsNew:       true,
		PublicKey:   strings.TrimSpace(pubLine),
	}, nil
}
