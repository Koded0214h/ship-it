package config

import (
	"fmt"
	"os"
	"path/filepath"

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
	dir := filepath.Join(configDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	path := filepath.Join(dir, configFile)
	return os.WriteFile(path, data, 0600)
}

func Validate(cfg *Config) []string {
	var errs []string
	if cfg.App.Name == "" {
		errs = append(errs, "app.name is required")
	}
	if cfg.Server.Host == "" {
		errs = append(errs, "server.host is required")
	}
	if cfg.Server.User == "" {
		errs = append(errs, "server.user is required")
	}
	if cfg.AI.Provider == "" {
		errs = append(errs, "ai.provider is required")
	}
	if cfg.AI.APIKey == "" {
		errs = append(errs, "ai.api_key is required")
	}
	return errs
}

func Exists() bool {
	_, err := findConfigFile()
	return err == nil
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
