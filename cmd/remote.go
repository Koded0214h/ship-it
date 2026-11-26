package cmd

import (
	"fmt"
	"strings"

	"github.com/Koded0214h/ship/internal/config"
	sshclient "github.com/Koded0214h/ship/internal/ssh"
)

// loadValidConfig loads and validates the config, and returns the project root.
// Validation matters: these values are used in commands run on the server.
func loadValidConfig() (*config.Config, string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, "", err
	}
	if errs := config.Validate(cfg); len(errs) > 0 {
		return nil, "", fmt.Errorf("invalid config (fix with 'ship config' or edit .ship/config.yaml):\n  • %s", strings.Join(errs, "\n  • "))
	}
	root, err := config.Root()
	if err != nil {
		return nil, "", err
	}
	return cfg, root, nil
}

// connect opens an SSH connection, asking before trusting a new server.
func connect(cfg *config.Config) (*sshclient.Client, error) {
	return sshclient.Connect(sshclient.Config{
		Host:    cfg.Server.Host,
		User:    cfg.Server.User,
		KeyPath: cfg.Server.KeyPath,
		Port:    cfg.Server.Port,
		ConfirmHostKey: func(host, keyType, fingerprint string) (bool, error) {
			return askYesNo("Trust this server?",
				fmt.Sprintf("First connection to %s.\n%s fingerprint: %s\nCompare it with your VPS provider's console, or run on the server:\n  ssh-keygen -lf /etc/ssh/ssh_host_%s_key.pub",
					host, keyType, fingerprint, hostKeyFile(keyType)))
		},
	})
}

func hostKeyFile(keyType string) string {
	switch {
	case strings.Contains(keyType, "ed25519"):
		return "ed25519"
	case strings.Contains(keyType, "ecdsa"):
		return "ecdsa"
	default:
		return "rsa"
	}
}
