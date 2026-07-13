package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/kodedlabs/ship/internal/config"
	"github.com/kodedlabs/ship/internal/sshkey"
	"github.com/kodedlabs/ship/internal/ui"
	"github.com/spf13/cobra"
)

const skipAI = "skip"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Ship in your project",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println(ui.BrandStyle.Render("Initializing Ship..."))
	fmt.Println()

	cfg := &config.Config{}
	cfg.Server.Port = 22

	var aiProvider string

	// Step 1: App + server details
	step1 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("App name").
				Description("A short identifier for your application (e.g. my-api)").
				Placeholder("my-app").
				Value(&cfg.App.Name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("app name is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Domain").
				Description("Your app's domain (e.g. api.example.com). Leave blank to use server IP.").
				Placeholder("api.example.com").
				Value(&cfg.App.Domain),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Server host").
				Description("IP address or hostname of your VPS").
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
				Placeholder("ubuntu").
				Value(&cfg.Server.User).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("SSH user is required")
					}
					return nil
				}),
		),
	)
	if err := step1.Run(); err != nil {
		return err
	}

	// Step 2: SSH key — detect, pick, or generate
	keyPath, err := resolveSSHKey()
	if err != nil {
		return err
	}
	cfg.Server.KeyPath = keyPath

	// Step 3: AI provider + key
	step3 := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("AI provider").
				Description("Which AI provider should generate your deployment plans? Skip if you don't have a key — Ship falls back to built-in templates.").
				Options(
					huh.NewOption("Claude (Anthropic)", "anthropic"),
					huh.NewOption("OpenAI", "openai"),
					huh.NewOption("Google Gemini", "gemini"),
					huh.NewOption("Skip — deploy without AI", skipAI),
				).
				Value(&aiProvider),
		),
	)
	if err := step3.Run(); err != nil {
		return err
	}

	if aiProvider != skipAI {
		keyForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("AI API key").
					Description("Your API key for the selected provider").
					EchoMode(huh.EchoModePassword).
					Value(&cfg.AI.APIKey).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("API key is required")
						}
						return nil
					}),
			),
		)
		if err := keyForm.Run(); err != nil {
			return err
		}
		cfg.AI.Provider = aiProvider
	}

	cfg.App.Port = 8080

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	if err := config.EnsureGitignore(); err != nil {
		fmt.Fprintln(os.Stderr, ui.WarningStyle.Render("Warning: could not update .gitignore: "+err.Error()))
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✓ Ship initialized!"))
	fmt.Println(ui.MutedStyle.Render("  Config saved to .ship/config.yaml"))
	fmt.Println(ui.MutedStyle.Render("  Run 'ship connect' to test your server connection."))
	return nil
}

const generateOption = "__generate__"

func resolveSSHKey() (string, error) {
	existing := sshkey.FindExisting()

	if len(existing) == 0 {
		// No keys found — generate automatically
		return generateKey()
	}

	// Build options: existing keys + generate new
	opts := make([]huh.Option[string], 0, len(existing)+1)
	for _, p := range existing {
		opts = append(opts, huh.NewOption(p, p))
	}
	opts = append(opts, huh.NewOption("Generate a new key for me", generateOption))

	var choice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("SSH key").
				Description("Which key should Ship use to connect to your server?").
				Options(opts...).
				Value(&choice),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}

	if choice == generateOption {
		return generateKey()
	}
	return choice, nil
}

func generateKey() (string, error) {
	fmt.Println()
	fmt.Println(ui.BrandStyle.Render("Generating SSH key pair..."))

	info, err := sshkey.Generate()
	if err != nil {
		return "", fmt.Errorf("generating SSH key: %w", err)
	}

	fmt.Println()
	fmt.Printf("  %s Key saved to %s\n", ui.CheckMark, ui.BoldStyle.Render(info.PrivatePath))
	fmt.Println()

	addCmd := fmt.Sprintf(`echo "%s" >> ~/.ssh/authorized_keys`, info.PublicKey)

	cmdBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.Brand).
		Padding(0, 1).
		Render(addCmd)

	fmt.Println(ui.BoldStyle.Render("Run this command on your server to authorize Ship:"))
	fmt.Println(cmdBox)
	fmt.Println()

	// Pause so user can read the instructions
	var ready bool
	confirm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Have you added the key to your server?").
				Description("Ship needs this key to connect. You can always re-run 'ship connect' to verify.").
				Value(&ready),
		),
	)
	_ = confirm.Run()

	return info.PrivatePath, nil
}
