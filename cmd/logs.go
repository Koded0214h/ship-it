package cmd

import (
	"fmt"
	"os"

	"github.com/kodedlabs/ship/internal/config"
	sshclient "github.com/kodedlabs/ship/internal/ssh"
	"github.com/kodedlabs/ship/internal/ui"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream logs from your deployed application",
	RunE:  runLogs,
}

func runLogs(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

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

	appDir := fmt.Sprintf("/opt/ship/%s", cfg.App.Name)
	logCmd := fmt.Sprintf("cd %s && docker compose logs -f --tail=100", appDir)

	fmt.Println(ui.MutedStyle.Render("Streaming logs from " + cfg.Server.Host + " (Ctrl+C to stop)"))
	fmt.Println()

	return client.ExecStream(logCmd, os.Stdout)
}
