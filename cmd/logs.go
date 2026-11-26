package cmd

import (
	"fmt"
	"os"

	sshclient "github.com/Koded0214h/ship/internal/ssh"
	"github.com/Koded0214h/ship/internal/ui"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream logs from your deployed application",
	RunE:  runLogs,
}

func runLogs(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadValidConfig()
	if err != nil {
		return err
	}

	client, err := connect(cfg)
	if err != nil {
		return fmt.Errorf("SSH connect: %w", err)
	}
	defer client.Close()

	logCmd := `cd "$HOME"/ship/` + sshclient.Quote(cfg.App.Name) + " && docker compose -f docker-compose.yml logs -f --tail=100"

	fmt.Println(ui.MutedStyle.Render("Streaming logs from " + cfg.Server.Host + " (Ctrl+C to stop)"))
	fmt.Println()

	err = client.Stream(cmd.Context(), logCmd, os.Stdout)
	if cmd.Context().Err() != nil {
		return nil // stopped by the user
	}
	return err
}
