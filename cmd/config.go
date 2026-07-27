package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/kodedlabs/ship/internal/config"
	"github.com/kodedlabs/ship/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and edit Ship settings",
	RunE:  runConfig,
}

func runConfig(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	printCurrentConfig(cfg)
	fmt.Println()

	var section string
	sectionForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What do you want to update?").
				Options(
					huh.NewOption("AI provider & API key", "ai"),
					huh.NewOption("App settings (name, domain, port)", "app"),
					huh.NewOption("Server settings (host, user, SSH key)", "server"),
					huh.NewOption("Nothing — just viewing", "none"),
				).
				Value(&section),
		),
	)
	if err := sectionForm.Run(); err != nil {
		return err
	}

	switch section {
	case "ai":
		if err := editAI(cfg); err != nil {
			return err
		}
	case "app":
		if err := editApp(cfg); err != nil {
			return err
		}
	case "server":
		if err := editServer(cfg); err != nil {
			return err
		}
	case "none":
		return nil
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✓ Config saved to .ship/config.yaml"))
	return nil
}

func editAI(cfg *config.Config) error {
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("AI provider").
				Options(
					huh.NewOption("Claude (Anthropic)", "anthropic"),
					huh.NewOption("OpenAI", "openai"),
					huh.NewOption("Google Gemini", "gemini"),
					huh.NewOption("None — deploy without AI", ""),
				).
				Value(&cfg.AI.Provider),
			huh.NewInput().
				Title("API key").
				Description("Leave blank if you selected 'None' above").
				EchoMode(huh.EchoModePassword).
				Value(&cfg.AI.APIKey),
			huh.NewInput().
				Title("Model").
				Description("Leave blank to use the default model for the provider").
				Placeholder("e.g. claude-sonnet-4-6 / gpt-4o / gemini-2.5-flash").
				Value(&cfg.AI.Model),
		),
	).Run()
	if err != nil {
		return err
	}
	if cfg.AI.Provider == "" {
		cfg.AI.APIKey = ""
		cfg.AI.Model = ""
	} else if cfg.AI.APIKey == "" {
		return fmt.Errorf("API key is required when a provider is selected")
	}
	return nil
}

func editApp(cfg *config.Config) error {
	portStr := strconv.Itoa(cfg.App.Port)
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("App name").
				Value(&cfg.App.Name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("app name is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Domain").
				Placeholder("api.example.com").
				Value(&cfg.App.Domain),
			huh.NewInput().
				Title("Port").
				Value(&portStr).
				Validate(func(s string) error {
					_, err := strconv.Atoi(s)
					if err != nil {
						return fmt.Errorf("must be a number")
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return err
	}
	cfg.App.Port, _ = strconv.Atoi(portStr)
	return nil
}

func editServer(cfg *config.Config) error {
	portStr := strconv.Itoa(cfg.Server.Port)
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Server host").
				Placeholder("123.456.789.0").
				Value(&cfg.Server.Host).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("server host is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("SSH user").
				Value(&cfg.Server.User),
			huh.NewInput().
				Title("SSH key path").
				Value(&cfg.Server.KeyPath),
			huh.NewInput().
				Title("SSH port").
				Value(&portStr).
				Validate(func(s string) error {
					_, err := strconv.Atoi(s)
					if err != nil {
						return fmt.Errorf("must be a number")
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return err
	}
	cfg.Server.Port, _ = strconv.Atoi(portStr)
	return nil
}

func printCurrentConfig(cfg *config.Config) {
	label := lipgloss.NewStyle().Foreground(ui.Muted).Width(14)
	value := lipgloss.NewStyle().Foreground(ui.White)
	masked := func(s string) string {
		if len(s) <= 8 {
			return "••••••••"
		}
		return s[:4] + "••••••••" + s[len(s)-4:]
	}

	fmt.Println(ui.BrandStyle.Render("Current config") + ui.MutedStyle.Render("  (.ship/config.yaml)"))
	fmt.Println()

	fmt.Println(ui.BoldStyle.Render("  App"))
	fmt.Printf("  %s %s\n", label.Render("Name"), value.Render(cfg.App.Name))
	fmt.Printf("  %s %s\n", label.Render("Domain"), value.Render(orDash(cfg.App.Domain)))
	fmt.Printf("  %s %s\n", label.Render("Port"), value.Render(strconv.Itoa(cfg.App.Port)))

	fmt.Println()
	fmt.Println(ui.BoldStyle.Render("  Server"))
	fmt.Printf("  %s %s\n", label.Render("Host"), value.Render(cfg.Server.Host))
	fmt.Printf("  %s %s\n", label.Render("User"), value.Render(cfg.Server.User))
	fmt.Printf("  %s %s\n", label.Render("SSH key"), value.Render(cfg.Server.KeyPath))
	fmt.Printf("  %s %s\n", label.Render("Port"), value.Render(strconv.Itoa(cfg.Server.Port)))

	fmt.Println()
	fmt.Println(ui.BoldStyle.Render("  AI"))
	fmt.Printf("  %s %s\n", label.Render("Provider"), value.Render(cfg.AI.Provider))
	fmt.Printf("  %s %s\n", label.Render("API key"), value.Render(masked(cfg.AI.APIKey)))
	fmt.Printf("  %s %s\n", label.Render("Model"), value.Render(orDash(cfg.AI.Model)))

	// Warn if config looks wrong
	if cfg.AI.Provider == "gemini" && len(cfg.AI.APIKey) > 0 && cfg.AI.APIKey[:2] != "AI" {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("  ⚠ Gemini API keys should start with 'AIza'. Get yours at: aistudio.google.com"))
	}

	if _, err := os.Stat(cfg.Server.KeyPath); os.IsNotExist(err) {
		expandedPath := expandKeyPath(cfg.Server.KeyPath)
		if _, err2 := os.Stat(expandedPath); os.IsNotExist(err2) {
			fmt.Println()
			fmt.Println(ui.WarningStyle.Render("  ⚠ SSH key not found at " + cfg.Server.KeyPath))
		}
	}
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func expandKeyPath(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + path[1:]
		}
	}
	return path
}
