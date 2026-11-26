package cmd

import (
	"fmt"
	"strings"

	"github.com/Koded0214h/ship/internal/config"
	sshclient "github.com/Koded0214h/ship/internal/ssh"
	"github.com/charmbracelet/huh"
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

// passphrases remembers what was typed, so one command asks only once.
var passphrases = map[string]string{}

// connect opens an SSH connection, asking before trusting a new server.
func connect(cfg *config.Config) (*sshclient.Client, error) {
	return connectWith(cfg, true)
}

// connectWith can run without prompting, in which case unknown servers and
// locked keys are errors.
func connectWith(cfg *config.Config, interactive bool) (*sshclient.Client, error) {
	options := sshclient.Config{
		Host:    cfg.Server.Host,
		User:    cfg.Server.User,
		KeyPath: cfg.Server.KeyPath,
		Port:    cfg.Server.Port,
	}
	if interactive {
		options.ConfirmHostKey = func(host, keyType, fingerprint string) (bool, error) {
			return askYesNo("Trust this server?",
				fmt.Sprintf("First connection to %s.\n%s fingerprint: %s\nCompare it with your VPS provider's console, or run on the server:\n  ssh-keygen -lf /etc/ssh/ssh_host_%s_key.pub",
					host, keyType, fingerprint, hostKeyFile(keyType)))
		}
		options.Passphrase = func(path string) (string, error) {
			if secret, ok := passphrases[path]; ok {
				return secret, nil
			}
			var secret string
			err := huh.NewForm(huh.NewGroup(
				huh.NewInput().Title("Passphrase for " + path).EchoMode(huh.EchoModePassword).Value(&secret),
			)).Run()
			if err == nil {
				passphrases[path] = secret
			}
			return secret, err
		}
	}
	return sshclient.Connect(options)
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
