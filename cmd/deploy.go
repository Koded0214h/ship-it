package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/kodedlabs/ship/internal/ai"
	"github.com/kodedlabs/ship/internal/config"
	"github.com/kodedlabs/ship/internal/deploy"
	"github.com/kodedlabs/ship/internal/detector"
	"github.com/kodedlabs/ship/internal/generator"
	sshclient "github.com/kodedlabs/ship/internal/ssh"
	"github.com/kodedlabs/ship/internal/ui"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy your application to your VPS",
	RunE:  runDeploy,
}

func runDeploy(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if errs := config.Validate(cfg); len(errs) > 0 {
		fmt.Println(ui.ErrorStyle.Render("Config errors:"))
		for _, e := range errs {
			fmt.Println("  " + ui.ErrorStyle.Render("• "+e))
		}
		return fmt.Errorf("invalid config — run 'ship init' to reconfigure")
	}

	// Step 1: Detect project
	fmt.Println(ui.BrandStyle.Render("Analyzing project..."))
	cwd, _ := os.Getwd()
	info, err := detector.Detect(cwd)
	if err != nil {
		return fmt.Errorf("project detection: %w", err)
	}

	printProjectSummary(info)

	// Step 2 + 3: Build the deployment plan — deterministically when no key is set,
	// otherwise via the configured AI provider.
	fmt.Println()
	var plan *ai.DeploymentPlan
	if cfg.AI.APIKey == "" {
		plan, err = deterministicPlan(info, cfg)
		if err != nil {
			return err
		}
	} else {
		var userPrompt string
		promptForm := huh.NewForm(
			huh.NewGroup(
				huh.NewText().
					Title("Describe your deployment").
					Description("Tell Ship what you want. E.g: 'Deploy with PostgreSQL and Redis. Enable HTTPS. Deploy on push to main.'").
					Placeholder("Deploy my app with PostgreSQL, enable HTTPS, and set up CI/CD with GitHub Actions.").
					Value(&userPrompt).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("please describe your deployment")
						}
						return nil
					}),
			),
		)
		if err := promptForm.Run(); err != nil {
			return err
		}

		fmt.Println()
		plan, err = runWithSpinner("Generating deployment plan", func() (*ai.DeploymentPlan, error) {
			provider, err := ai.New(cfg.AI.Provider, cfg.AI.APIKey, cfg.AI.Model)
			if err != nil {
				return nil, err
			}
			return provider.GenerateDeploymentPlan(context.Background(), info, userPrompt)
		})
		if err != nil {
			return fmt.Errorf("generating plan: %w", err)
		}
	}

	// Step 4: Show plan
	fmt.Println()
	if err := renderPlan(plan, cfg); err != nil {
		fmt.Println(ui.ErrorStyle.Render("Warning: could not render plan markdown: " + err.Error()))
		printPlanFallback(plan)
	}

	// Step 5: Confirm
	fmt.Println()
	var confirmed bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Deploy this plan?").
				Description("Ship will execute these steps on your server.").
				Value(&confirmed),
		),
	)
	if err := confirmForm.Run(); err != nil {
		return err
	}
	if !confirmed {
		fmt.Println(ui.MutedStyle.Render("Deploy cancelled."))
		return nil
	}

	if cfg.AI.APIKey == "" && plan.Domain != cfg.App.Domain {
		cfg.App.Domain = plan.Domain
		if err := config.Save(cfg); err != nil {
			fmt.Println(ui.WarningStyle.Render("Warning: could not save domain to config: " + err.Error()))
		}
	}

	// Step 6: Save GitHub Actions workflow locally
	if plan.GitHubActions != "" {
		localPath := ".github/workflows/deploy.yml"
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err == nil {
			if err := os.WriteFile(localPath, []byte(plan.GitHubActions), 0644); err == nil {
				fmt.Println(ui.SuccessStyle.Render("✓ GitHub Actions workflow saved to " + localPath))
			}
		}
	}

	// Step 7: Connect and deploy
	fmt.Println()
	fmt.Println(ui.BrandStyle.Render("Deploying to " + cfg.Server.Host + "..."))

	client, err := sshclient.Connect(sshclient.Config{
		Host:    cfg.Server.Host,
		User:    cfg.Server.User,
		KeyPath: cfg.Server.KeyPath,
		Port:    cfg.Server.Port,
	})
	if err != nil {
		return fmt.Errorf("SSH connect: %w", err)
	}
	defer client.Close()

	executor := deploy.New(client, cfg, plan, cwd, info.Port)
	results := executor.Run(func(name string) {
		fmt.Printf("  %s %s\n", ui.Arrow, name)
	})

	// Step 8: Results
	fmt.Println()
	allOk := true
	for _, r := range results {
		if r.Success {
			fmt.Printf("  %s %s\n", ui.CheckMark, r.Name)
		} else {
			fmt.Printf("  %s %s: %v\n", ui.CrossMark, r.Name, r.Err)
			allOk = false
		}
	}

	fmt.Println()
	if allOk {
		host := cfg.Server.Host
		if plan.Domain != "" {
			host = plan.Domain
		}
		scheme := "http"
		if plan.SSLEnabled {
			scheme = "https"
		}
		fmt.Println(ui.SuccessStyle.Render("✓ Deployed successfully!"))
		fmt.Println(ui.BrandStyle.Render(fmt.Sprintf("  %s://%s", scheme, host)))
	} else {
		fmt.Println(ui.ErrorStyle.Render("✗ Deployment completed with errors."))
		fmt.Println(ui.MutedStyle.Render("  Run 'ship doctor' to diagnose issues."))
	}

	return nil
}

func deterministicPlan(info *detector.ProjectInfo, cfg *config.Config) (*ai.DeploymentPlan, error) {
	domain := cfg.App.Domain
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Domain (optional)").
				Description("Leave blank to deploy on your server's IP address. Deterministic mode deploys over HTTP for now.").
				Placeholder("api.example.com").
				Value(&domain),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}
	return generator.BuildPlan(info, cfg.App.Name, cfg.Server.User, domain, false), nil
}

func printProjectSummary(info *detector.ProjectInfo) {
	fmt.Println()
	fmt.Println(ui.BoxStyle.Render(fmt.Sprintf(
		"%s\n\nLanguage:   %s\nFramework:  %s\nPort:       %d\nDatabase:   %s\nRedis:      %s",
		ui.BoldStyle.Render("Project detected"),
		info.Language,
		info.Framework,
		info.Port,
		boolYN(info.HasDatabase),
		boolYN(info.HasRedis),
	)))
}

func boolYN(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func renderPlan(plan *ai.DeploymentPlan, cfg *config.Config) error {
	md := buildPlanMarkdown(plan, cfg)
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return err
	}
	out, err := renderer.Render(md)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

func buildPlanMarkdown(plan *ai.DeploymentPlan, cfg *config.Config) string {
	var sb strings.Builder
	sb.WriteString("## Deployment Plan\n\n")
	if plan.Summary != "" {
		sb.WriteString(plan.Summary + "\n\n")
	}

	sb.WriteString("### Services\n\n")
	for _, s := range plan.Services {
		sb.WriteString(fmt.Sprintf("- %s\n", s))
	}

	sb.WriteString("\n### Steps\n\n")
	for i, s := range plan.Steps {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, s))
	}

	if plan.SSLEnabled {
		sb.WriteString("\n### HTTPS\n\n✓ SSL/TLS enabled via Let's Encrypt\n")
	}

	if len(plan.EnvVars) > 0 {
		sb.WriteString("\n### Required Environment Variables\n\n")
		for k, v := range plan.EnvVars {
			sb.WriteString(fmt.Sprintf("- `%s` = `%s`\n", k, v))
		}
	}

	if plan.GitHubActions != "" {
		sb.WriteString("\n### CI/CD\n\n✓ GitHub Actions workflow will be created at `.github/workflows/deploy.yml`\n")
	}

	return sb.String()
}

func printPlanFallback(plan *ai.DeploymentPlan) {
	style := lipgloss.NewStyle().Foreground(ui.Brand)
	fmt.Println(style.Render("Deployment Plan:"))
	if plan.Summary != "" {
		fmt.Println(plan.Summary)
	}
	for i, step := range plan.Steps {
		fmt.Printf("  %d. %s\n", i+1, step)
	}
}

// runWithSpinner runs fn while showing an animated spinner line.
func runWithSpinner[T any](label string, fn func() (T, error)) (T, error) {
	type result struct {
		val T
		err error
	}
	ch := make(chan result, 1)
	go func() {
		v, err := fn()
		ch <- result{v, err}
	}()

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	tick := time.NewTicker(80 * time.Millisecond)
	defer tick.Stop()

	i := 0
	for {
		select {
		case r := <-ch:
			// Clear spinner line, print final status
			fmt.Printf("\r  %s %s                    \n", ui.CheckMark, label)
			if r.err != nil {
				var zero T
				return zero, r.err
			}
			return r.val, nil
		case <-tick.C:
			frame := ui.BrandStyle.Render(frames[i%len(frames)])
			fmt.Printf("\r  %s %s...", frame, label)
			i++
		}
	}
}

