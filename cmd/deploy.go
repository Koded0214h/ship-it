package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/Koded0214h/ship/internal/ai"
	"github.com/Koded0214h/ship/internal/config"
	"github.com/Koded0214h/ship/internal/deploy"
	"github.com/Koded0214h/ship/internal/detector"
	"github.com/Koded0214h/ship/internal/generator"
	"github.com/Koded0214h/ship/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

const aiTimeout = 3 * time.Minute

var (
	noAI       bool
	assumeYes  bool
	dryRun     bool
	describeAs string
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy your application to your VPS",
	Long: `Deploy the project that contains .ship/config.yaml to your server.

Ship shows the files it will use and asks before doing anything. The new image
is built while the current containers keep running. HTTPS is not supported yet.

With --yes nothing is asked: your own Dockerfile is kept, app.port is used, and
a failed AI plan stops the deploy. An unknown server is still refused, so run
'ship connect' once first.`,
	RunE: runDeploy,
}

func init() {
	deployCmd.Flags().BoolVar(&noAI, "no-ai", false, "use the built-in templates even if an AI provider is configured")
	deployCmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "do not ask questions (for scripts)")
	deployCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the plan without connecting or changing anything")
	deployCmd.Flags().StringVar(&describeAs, "describe", "", "deployment description for the AI provider, instead of being asked")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cfg, root, err := loadValidConfig()
	if err != nil {
		return err
	}

	// Step 1: Detect project
	fmt.Println(ui.BrandStyle.Render("Analyzing project..."))
	info, err := detector.Detect(root)
	if err != nil {
		return fmt.Errorf("project detection: %w", err)
	}
	if err := resolvePort(cfg, info); err != nil {
		return err
	}
	printProjectSummary(info)

	// Step 2: Build the plan — from templates, or from the configured AI provider.
	fmt.Println()
	var plan *ai.DeploymentPlan
	fromTemplate := noAI || cfg.AI.APIKey == ""
	if !fromTemplate {
		plan, err = aiPlan(ctx, cfg, info, root)
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) || ctx.Err() != nil || assumeYes {
				return err
			}
			fmt.Println(ui.ErrorStyle.Render("The AI plan could not be used: " + sanitize(err.Error())))
			ok, askErr := askYesNo("Continue with Ship's built-in template instead?",
				"The template does not use your written description.")
			if askErr != nil {
				return askErr
			}
			if !ok {
				return fmt.Errorf("generating plan: %w", err)
			}
			fromTemplate = true
		}
	}
	if fromTemplate {
		plan, err = deterministicPlan(info, cfg)
		if err != nil {
			return err
		}
		if err := deploy.ValidatePlan(plan, root); err != nil {
			return err
		}
	}

	if plan.SSLEnabled {
		fmt.Println(ui.WarningStyle.Render("HTTPS is not supported yet — this will deploy over HTTP."))
		plan.SSLEnabled = false
	}
	if cfg.App.Domain != "" && !fromTemplate {
		plan.Domain = cfg.App.Domain
	}

	// Step 3: Respect the user's own files.
	if err := chooseDockerfile(plan, root); err != nil {
		return err
	}
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			fmt.Println(ui.WarningStyle.Render("Note: " + name + " is not used. Ship deploys with the compose file shown below; your local copy is not changed."))
		}
	}

	if plan.DockerCompose, err = deploy.SetPublicPort(plan.DockerCompose, cfg.App.PublicPort); err != nil {
		return err
	}

	// Step 4: Show exactly what will be used, then confirm.
	preview := deploy.New(nil, cfg, plan, root, info.Port)
	printPlan(plan, preview, fromTemplate)

	if dryRun {
		fmt.Println(ui.MutedStyle.Render("Dry run: nothing was uploaded or changed."))
		return nil
	}
	if !assumeYes {
		confirmed, err := askYesNo(
			fmt.Sprintf("Deploy %s to %s@%s?", cfg.App.Name, cfg.Server.User, cfg.Server.Host),
			"Check the files above. Containers are replaced only after a successful build.")
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println(ui.MutedStyle.Render("Deploy cancelled."))
			return nil
		}
	}

	if fromTemplate && plan.Domain != cfg.App.Domain {
		cfg.App.Domain = plan.Domain
		if err := config.Save(cfg); err != nil {
			fmt.Println(ui.WarningStyle.Render("Warning: could not save domain to config: " + err.Error()))
		}
	}

	// Step 5: Connect and deploy
	fmt.Println()
	fmt.Println(ui.BrandStyle.Render("Deploying to " + cfg.Server.Host + "..."))

	client, err := connectWith(cfg, !assumeYes)
	if err != nil {
		return fmt.Errorf("SSH connect: %w", err)
	}
	defer client.Close()

	executor := deploy.New(client, cfg, plan, root, info.Port)
	executor.Out = os.Stdout
	results := executor.Run(ctx, func(name string) {
		fmt.Printf("  %s %s\n", ui.Arrow, name)
	})

	// Step 6: Results
	fmt.Println()
	var failure error
	for _, r := range results {
		if r.Success {
			fmt.Printf("  %s %s\n", ui.CheckMark, r.Name)
		} else {
			fmt.Printf("  %s %s: %v\n", ui.CrossMark, r.Name, r.Err)
			failure = fmt.Errorf("deployment failed at step %q", r.Name)
		}
	}

	fmt.Println()
	if failure != nil {
		fmt.Println(ui.MutedStyle.Render("  Run 'ship logs' for container output or 'ship doctor' to check the server."))
		return failure
	}
	host := cfg.Server.Host
	if plan.Domain != "" {
		host = plan.Domain
	}
	fmt.Println(ui.SuccessStyle.Render("✓ Deployed successfully!"))
	if port := cfg.App.ListenPort(); port != 80 {
		host = fmt.Sprintf("%s:%d", host, port)
	}
	fmt.Println(ui.BrandStyle.Render("  http://" + host))
	return nil
}

// resolvePort makes app.port authoritative, asking when it disagrees with what
// was detected.
func resolvePort(cfg *config.Config, info *detector.ProjectInfo) error {
	if cfg.App.Port == 0 || cfg.App.Port == info.Port {
		return nil
	}
	if assumeYes || dryRun {
		info.Port = cfg.App.Port
		return nil
	}
	port := cfg.App.Port
	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[int]().
			Title("Which port does your app listen on?").
			Description("Your config and the detected framework default disagree. Change app.port with 'ship config'.").
			Options(
				huh.NewOption(fmt.Sprintf("%d — from .ship/config.yaml", cfg.App.Port), cfg.App.Port),
				huh.NewOption(fmt.Sprintf("%d — usual port for %s", info.Port, info.Framework), info.Port),
			).
			Value(&port),
	)).Run()
	if err != nil {
		return err
	}
	if port != cfg.App.Port {
		cfg.App.Port = port
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
	}
	info.Port = port
	return nil
}

func aiPlan(ctx context.Context, cfg *config.Config, info *detector.ProjectInfo, root string) (*ai.DeploymentPlan, error) {
	provider, err := ai.New(cfg.AI.Provider, cfg.AI.APIKey, cfg.AI.Model)
	if err != nil {
		return nil, err
	}

	userPrompt := strings.TrimSpace(describeAs)
	if userPrompt == "" && assumeYes {
		return nil, errors.New("--yes with an AI provider needs --describe (or use --no-ai)")
	}
	promptForm := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Describe your deployment").
				Description("This text and your project's language, framework, port, services and manifest file names are sent to " + provider.Name() + ". Don't include secrets.").
				Placeholder("Deploy my app with PostgreSQL and Redis.").
				Value(&userPrompt).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("please describe your deployment")
					}
					return nil
				}),
		),
	)
	if userPrompt == "" {
		if err := promptForm.Run(); err != nil {
			return nil, err
		}
	}

	fmt.Println()
	ctx, cancel := context.WithTimeout(ctx, aiTimeout)
	defer cancel()
	plan, err := runWithSpinner("Generating deployment plan", func() (*ai.DeploymentPlan, error) {
		return provider.GenerateDeploymentPlan(ctx, info, userPrompt)
	})
	if err != nil {
		return nil, err
	}
	if err := deploy.ValidatePlan(plan, root); err != nil {
		return nil, fmt.Errorf("the generated plan was rejected: %w", err)
	}
	return plan, nil
}

// chooseDockerfile asks whether to keep a Dockerfile the project already has.
func chooseDockerfile(plan *ai.DeploymentPlan, root string) error {
	if _, err := os.Stat(filepath.Join(root, "Dockerfile")); err != nil || plan.Dockerfile == "" {
		return nil
	}
	if assumeYes || dryRun {
		plan.Dockerfile = ""
		return nil
	}
	keep := true
	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("Use your existing Dockerfile?").
			Description("No replaces it on the server with a generated one. Your local file is not changed.").
			Value(&keep),
	)).Run()
	if err != nil {
		return err
	}
	if keep {
		plan.Dockerfile = ""
	}
	return nil
}

func askYesNo(title, description string) (bool, error) {
	var yes bool // defaults to No
	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(title).Description(description).Value(&yes),
	)).Run()
	return yes, err
}

func deterministicPlan(info *detector.ProjectInfo, cfg *config.Config) (*ai.DeploymentPlan, error) {
	domain := cfg.App.Domain
	if assumeYes || dryRun {
		return generator.BuildPlan(info, cfg.App.Name, domain), nil
	}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Domain (optional)").
				Description("Leave blank to deploy on your server's IP address. Ship deploys over HTTP for now.").
				Placeholder("api.example.com").
				Value(&domain).
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					return config.ValidateDomain(s)
				}),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}
	return generator.BuildPlan(info, cfg.App.Name, domain), nil
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

// sanitize removes control characters so generated or remote text cannot drive
// the terminal (for example, to hide lines of a file under review).
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) {
			return -1
		}
		return r
	}, s)
}

// printPlan prints the files verbatim and the steps the executor will take. The
// steps come from Ship's code, not from the AI.
func printPlan(plan *ai.DeploymentPlan, executor *deploy.Executor, fromTemplate bool) {
	source := "AI-generated"
	if fromTemplate {
		source = "built-in template"
	}
	fmt.Println()
	fmt.Println(ui.BrandStyle.Render("Deployment plan (" + source + ")"))
	if plan.Summary != "" {
		fmt.Println(sanitize(plan.Summary))
	}

	file := func(name, content string) {
		fmt.Println()
		fmt.Println(ui.BoldStyle.Render("── " + name + " ──"))
		fmt.Println(strings.TrimRight(sanitize(content), "\n"))
	}
	if plan.Dockerfile == "" {
		fmt.Println()
		fmt.Println(ui.BoldStyle.Render("── Dockerfile ──") + " yours, unchanged")
	} else {
		file("Dockerfile", plan.Dockerfile)
	}
	file("docker-compose.yml", plan.DockerCompose)
	file("nginx.conf", executor.NginxConfig())

	if len(plan.EnvVars) > 0 {
		names := make([]string, 0, len(plan.EnvVars))
		for name := range plan.EnvVars {
			names = append(names, sanitize(name))
		}
		slices.Sort(names)
		fmt.Println()
		fmt.Println("Server .env (written only if none exists): " + strings.Join(names, ", "))
	}

	fmt.Println()
	fmt.Println(ui.BoldStyle.Render("Steps"))
	for i, s := range executor.StepNames() {
		fmt.Printf("  %d. %s\n", i+1, s)
	}
	fmt.Println()
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
			mark := ui.CheckMark
			if r.err != nil {
				mark = ui.CrossMark
			}
			fmt.Printf("\r  %s %s                    \n", mark, label)
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
