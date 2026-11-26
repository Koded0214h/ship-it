package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Koded0214h/ship/internal/config"
	"github.com/Koded0214h/ship/internal/detector"
	"github.com/Koded0214h/ship/internal/sshconfig"
	"github.com/Koded0214h/ship/internal/sshkey"
	"github.com/Koded0214h/ship/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Ship in your project",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println(ui.BrandStyle.Render("Initializing Ship..."))
	fmt.Println()

	if _, err := os.Stat(".ship/config.yaml"); err == nil {
		ok, err := askYesNo("Replace the existing config?", ".ship/config.yaml already exists here.")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println(ui.MutedStyle.Render("Left unchanged. Use 'ship config' to edit it."))
			return nil
		}
	}

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
				Validate(config.ValidateAppName),
			huh.NewInput().
				Title("Domain").
				Description("Your app's domain (e.g. api.example.com). Leave blank to use server IP.").
				Placeholder("api.example.com").
				Value(&cfg.App.Domain).
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					return config.ValidateDomain(s)
				}),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Server host").
				Description("IP address, hostname, or an alias from your ~/.ssh/config").
				Placeholder("203.0.113.10").
				Value(&cfg.Server.Host).
				Validate(config.ValidateHost),
		),
	)
	if err := step1.Run(); err != nil {
		return err
	}

	suggestedKey, err := useSSHAlias(cmd.Context(), cfg)
	if err != nil {
		return err
	}

	userForm := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("SSH user").
			Placeholder("ubuntu").
			Value(&cfg.Server.User).
			Validate(config.ValidateUser),
	))
	if err := userForm.Run(); err != nil {
		return err
	}

	// Step 2: SSH key — detect, pick, or generate
	keyPath, err := resolveSSHKey(suggestedKey)
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
					huh.NewOption("Skip — deploy without AI", ""),
				).
				Value(&aiProvider),
		),
	)
	if err := step3.Run(); err != nil {
		return err
	}

	if aiProvider != "" {
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

	// 0 means "detect at deploy time".
	if info, err := detector.Detect("."); err == nil {
		cfg.App.Port = info.Port
	}

	if err := config.SaveNew(cfg); err != nil {
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

// useSSHAlias offers to fill in the connection from ~/.ssh/config and returns
// the key configured there, if any.
func useSSHAlias(ctx context.Context, cfg *config.Config) (string, error) {
	host, ok := sshconfig.Lookup(ctx, cfg.Server.Host)
	if !ok || config.ValidateHost(host.HostName) != nil {
		return "", nil
	}
	if host.Proxied {
		fmt.Println(ui.WarningStyle.Render("⚠ This alias uses a jump host or proxy command, which Ship does not support."))
	}
	use, err := askYesNo(fmt.Sprintf("Use the SSH config entry %q?", cfg.Server.Host),
		fmt.Sprintf("host %s, user %s, port %d. Ship saves these values, not the alias.", host.HostName, host.User, host.Port))
	if err != nil || !use {
		return "", err
	}
	cfg.Server.Host = host.HostName
	cfg.Server.User = host.User
	if host.Port > 0 {
		cfg.Server.Port = host.Port
	}
	return host.IdentityFile, nil
}

const (
	generateOption = "__generate__"
	typeOption     = "__type__"
)

// resolveSSHKey lets the user choose a key, listing the suggested one first.
func resolveSSHKey(suggested string) (string, error) {
	existing := sshkey.FindExisting()
	if suggested != "" && !slices.Contains(existing, suggested) {
		existing = append([]string{suggested}, existing...)
	}

	opts := make([]huh.Option[string], 0, len(existing)+2)
	for _, p := range existing {
		opts = append(opts, huh.NewOption(p, p))
	}
	// Offer generation only when it would not collide with an existing key.
	if !slices.ContainsFunc(existing, func(p string) bool { return filepath.Base(p) == "ship_ed25519" }) {
		opts = append(opts, huh.NewOption("Generate a new key for me", generateOption))
	}
	opts = append(opts, huh.NewOption("Enter a path to another key", typeOption))

	choice := suggested
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

	switch choice {
	case generateOption:
		return generateKey()
	case typeOption:
		return askKeyPath()
	}
	return choice, nil
}

func askKeyPath() (string, error) {
	var path string
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Path to your private SSH key").
			Value(&path).
			Validate(validKeyPath),
	)).Run()
	return path, err
}

func validKeyPath(path string) error {
	info, err := os.Stat(expandKeyPath(path))
	if err != nil || info.IsDir() {
		return fmt.Errorf("no key file at that path")
	}
	if strings.HasSuffix(path, ".pub") {
		return fmt.Errorf("that is the public key; give the private key (without .pub)")
	}
	return nil
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
	if err := confirm.Run(); err != nil {
		return "", err
	}

	return info.PrivatePath, nil
}
