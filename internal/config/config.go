package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

const configDir = ".ship"
const configFile = "config.yaml"

type Config struct {
	App    AppConfig    `yaml:"app"`
	Server ServerConfig `yaml:"server"`
	AI     AIConfig     `yaml:"ai"`
}

type AppConfig struct {
	Name   string `yaml:"name"`
	Domain string `yaml:"domain"`
	Port   int    `yaml:"port"`
}

type ServerConfig struct {
	Host    string `yaml:"host"`
	User    string `yaml:"user"`
	KeyPath string `yaml:"key_path"`
	Port    int    `yaml:"port"`
}

type AIConfig struct {
	Provider string `yaml:"provider"`
	APIKey   string `yaml:"api_key"`
	Model    string `yaml:"model"`
}

func Load() (*Config, error) {
	path, err := findConfigFile()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 22
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := findConfigFile()
	if err != nil {
		path = filepath.Join(configDir, configFile)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	return writePrivate(path, data)
}

// writePrivate also tightens the permissions of a file that already exists.
func writePrivate(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	return os.Chmod(path, 0600)
}

func SaveNew(cfg *Config) error {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	return writePrivate(filepath.Join(configDir, configFile), data)
}

// These values end up in remote shell commands and config files, so they are
// restricted to safe characters.
var (
	appNameRe  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,62}$`)
	hostnameRe = regexp.MustCompile(`^[A-Za-z0-9_]([A-Za-z0-9_-]{0,61}[A-Za-z0-9_])?(\.[A-Za-z0-9_]([A-Za-z0-9_-]{0,61}[A-Za-z0-9_])?)*\.?$`)
	userRe     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*\$?$`)
)

func ValidateAppName(s string) error {
	if s == "" {
		return fmt.Errorf("app.name is required")
	}
	if !appNameRe.MatchString(s) {
		return fmt.Errorf("app.name %q may only use letters, digits, '.', '_' and '-', and must start with a letter or digit", s)
	}
	return nil
}

func ValidateDomain(s string) error {
	if len(s) > 253 || !hostnameRe.MatchString(s) {
		return fmt.Errorf("domain %q is not a valid hostname (no scheme, path or port)", s)
	}
	return nil
}

func ValidateHost(s string) error {
	if s == "" {
		return fmt.Errorf("server.host is required")
	}
	if net.ParseIP(s) == nil && !hostnameRe.MatchString(s) {
		return fmt.Errorf("server.host %q is not a valid hostname or IP address", s)
	}
	return nil
}

func ValidateUser(s string) error {
	if s == "" {
		return fmt.Errorf("server.user is required")
	}
	if !userRe.MatchString(s) {
		return fmt.Errorf("server.user %q is not a valid user name", s)
	}
	return nil
}

func Validate(cfg *Config) []string {
	var errs []string
	add := func(err error) {
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	add(ValidateAppName(cfg.App.Name))
	add(ValidateHost(cfg.Server.Host))
	add(ValidateUser(cfg.Server.User))
	if cfg.App.Domain != "" {
		add(ValidateDomain(cfg.App.Domain))
	}
	if cfg.App.Port < 0 || cfg.App.Port > 65535 {
		errs = append(errs, "app.port must be between 0 (auto-detect) and 65535")
	}
	if cfg.Server.Port < 0 || cfg.Server.Port > 65535 {
		errs = append(errs, "server.port must be between 1 and 65535")
	}
	return errs
}

func Exists() bool {
	_, err := findConfigFile()
	return err == nil
}

// Root returns the project directory: the one containing .ship/.
func Root() (string, error) {
	path, err := findConfigFile()
	if err != nil {
		return "", err
	}
	return filepath.Dir(filepath.Dir(path)), nil
}

func findConfigFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		path := filepath.Join(dir, configDir, configFile)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no .ship/config.yaml found (run 'ship init' first)")
}

func EnsureGitignore() error {
	data, err := os.ReadFile(".gitignore")
	content := string(data)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	entry := "\n# Ship deployment config (contains secrets)\n.ship/\n"
	if len(content) > 0 && content[len(content)-1] != '\n' {
		entry = "\n" + entry
	}
	// Don't add if already present
	for _, line := range []string{".ship/", ".ship"} {
		for _, l := range splitLines(content) {
			if l == line {
				return nil
			}
		}
	}
	f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(entry)
	return err
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
