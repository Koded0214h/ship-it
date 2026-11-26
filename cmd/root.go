package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/Koded0214h/ship/internal/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "ship",
	Short:   "Deploy your apps to your own VPS using natural language",
	Version: version,
	Long:    banner(),
	// Errors are printed once by Execute; usage text would bury them.
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	// "go install" builds have no ldflags; take the version from the module.
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			rootCmd.Version = info.Main.Version
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		stop() // a second Ctrl+C exits immediately
	}()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, ui.ErrorStyle.Render(err.Error()))
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
}

func banner() string {
	title := lipgloss.NewStyle().
		Foreground(ui.Brand).
		Bold(true).
		Render("Ship 🚀")
	sub := ui.MutedStyle.Render("Deploy to your VPS using natural language.")
	return fmt.Sprintf("%s\n%s", title, sub)
}
