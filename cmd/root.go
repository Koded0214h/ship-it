package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/kodedlabs/ship/internal/ui"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "ship",
	Short:   "Deploy your apps to your own VPS using natural language",
	Version: version,
	Long:    banner(),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
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
