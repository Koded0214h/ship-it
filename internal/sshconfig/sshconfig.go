// Package sshconfig resolves host aliases from the user's SSH configuration by
// asking the system's ssh, so Include and Match behave as they do there.
package sshconfig

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Host struct {
	HostName string
	User     string
	Port     int
	// IdentityFile is the first configured key that exists, or "".
	IdentityFile string
	// Proxied means ssh reaches the host through a jump or proxy command,
	// which Ship does not support.
	Proxied bool
}

// Lookup returns the settings for alias, and false when alias is not an alias:
// ssh is unavailable, or it would connect to that very name. The caller must
// have validated alias as a hostname.
func Lookup(ctx context.Context, alias string) (Host, bool) {
	if alias == "" || strings.HasPrefix(alias, "-") {
		return Host{}, false
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ssh", "-G", "--", alias).Output()
	if err != nil {
		return Host{}, false
	}
	home, _ := os.UserHomeDir()
	h := parse(string(out), home, func(p string) bool {
		info, err := os.Stat(p)
		return err == nil && !info.IsDir()
	})
	if h.HostName == "" || strings.EqualFold(h.HostName, alias) {
		return Host{}, false
	}
	return h, true
}

// parse reads the output of "ssh -G".
func parse(output, home string, exists func(string) bool) Host {
	var h Host
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}
		switch strings.ToLower(key) {
		case "hostname":
			h.HostName = value
		case "user":
			h.User = value
		case "port":
			h.Port, _ = strconv.Atoi(value)
		case "identityfile":
			if strings.HasPrefix(value, "~/") {
				value = filepath.Join(home, value[2:])
			}
			if h.IdentityFile == "" && exists(value) {
				h.IdentityFile = value
			}
		case "proxyjump", "proxycommand":
			if value != "none" {
				h.Proxied = true
			}
		}
	}
	return h
}
