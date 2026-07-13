package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kodedlabs/ship/internal/ai"
	"github.com/kodedlabs/ship/internal/config"
	sshclient "github.com/kodedlabs/ship/internal/ssh"
	"github.com/kodedlabs/ship/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostics on your Ship setup",
	RunE:  runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println(ui.BrandStyle.Render("Running Ship diagnostics..."))
	fmt.Println()

	allOk := true
	check := func(name string, fn func() (string, error)) {
		detail, err := fn()
		if err != nil {
			fmt.Printf("  %s %s\n", ui.CrossMark, name)
			fmt.Printf("     %s\n", ui.ErrorStyle.Render(err.Error()))
			allOk = false
		} else {
			line := fmt.Sprintf("  %s %s", ui.CheckMark, name)
			if detail != "" {
				line += ui.MutedStyle.Render("  "+detail)
			}
			fmt.Println(line)
		}
	}

	check("Config file", func() (string, error) {
		cfg, err := config.Load()
		if err != nil {
			return "", err
		}
		if errs := config.Validate(cfg); len(errs) > 0 {
			return "", fmt.Errorf("%v", errs)
		}
		return ".ship/config.yaml found", nil
	})

	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		fmt.Println()
		fmt.Println(ui.ErrorStyle.Render("Cannot continue — config not loaded."))
		return nil
	}

	check("SSH connection", func() (string, error) {
		client, err := sshclient.Connect(sshclient.Config{
			Host:    cfg.Server.Host,
			User:    cfg.Server.User,
			KeyPath: cfg.Server.KeyPath,
			Port:    cfg.Server.Port,
		})
		if err != nil {
			return "", err
		}
		defer client.Close()
		return fmt.Sprintf("%s@%s", cfg.Server.User, cfg.Server.Host), nil
	})

	check("Docker on server", func() (string, error) {
		client, err := sshclient.Connect(sshclient.Config{
			Host:    cfg.Server.Host,
			User:    cfg.Server.User,
			KeyPath: cfg.Server.KeyPath,
			Port:    cfg.Server.Port,
		})
		if err != nil {
			return "", fmt.Errorf("SSH not available")
		}
		defer client.Close()
		info := client.GetServerInfo()
		if !info.Docker {
			return "", fmt.Errorf("Docker not found — run: apt-get install -y docker.io docker-compose-plugin")
		}
		return "Docker installed", nil
	})

	check("AI provider API key", func() (string, error) {
		if cfg.AI.APIKey == "" {
			return "skipped — deterministic mode", nil
		}
		provider, err := ai.New(cfg.AI.Provider, cfg.AI.APIKey, cfg.AI.Model)
		if err != nil {
			return "", err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := provider.Ping(ctx); err != nil {
			return "", fmt.Errorf("%s key invalid or unreachable: %v", provider.Name(), err)
		}
		return provider.Name() + " key valid", nil
	})

	if cfg.App.Domain != "" {
		check("Domain DNS", func() (string, error) {
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get("http://" + cfg.App.Domain)
			if err != nil {
				return "", fmt.Errorf("cannot reach %s: %v", cfg.App.Domain, err)
			}
			resp.Body.Close()
			return cfg.App.Domain + " is reachable", nil
		})
	}

	fmt.Println()
	if allOk {
		fmt.Println(ui.SuccessStyle.Render("✓ All checks passed! You're ready to deploy."))
	} else {
		fmt.Println(ui.WarningStyle.Render("⚠ Some checks failed. Review the errors above."))
	}

	return nil
}
