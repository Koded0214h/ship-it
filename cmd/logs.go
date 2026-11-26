package cmd

import (
	"fmt"
	"os"

	sshclient "github.com/Koded0214h/ship/internal/ssh"
	"github.com/spf13/cobra"
)

var logOptions struct {
	tail     int
	service  string
	noFollow bool
	build    bool
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream logs from your deployed application",
	Example: `  ship logs
  ship logs --service app --tail 500 --no-follow
  ship logs --build      # output of the last image build`,
	RunE: runLogs,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the containers of your deployed application",
	RunE:  runStatus,
}

func init() {
	logsCmd.Flags().IntVar(&logOptions.tail, "tail", 100, "number of lines to show from the end")
	logsCmd.Flags().StringVar(&logOptions.service, "service", "", "only this service, for example app or nginx")
	logsCmd.Flags().BoolVar(&logOptions.noFollow, "no-follow", false, "print and exit")
	logsCmd.Flags().BoolVar(&logOptions.build, "build", false, "show the output of the last image build")
}

// inAppDir builds a command that runs in the app's directory on the server. The
// app name has been validated and is quoted as well.
func inAppDir(app, command string) string {
	return `cd "$HOME"/ship/` + sshclient.Quote(app) + " && " + command
}

func logsCommand(tail int, service string, follow, build bool) (string, error) {
	if tail < 0 {
		return "", fmt.Errorf("--tail must not be negative")
	}
	if build {
		return fmt.Sprintf("tail -n %d deploy.log", tail), nil
	}
	cmd := fmt.Sprintf("docker compose -f docker-compose.yml logs --tail=%d", tail)
	if follow {
		cmd += " -f"
	}
	if service != "" {
		cmd += " -- " + sshclient.Quote(service)
	}
	return cmd, nil
}

func runLogs(cmd *cobra.Command, args []string) error {
	command, err := logsCommand(logOptions.tail, logOptions.service, !logOptions.noFollow, logOptions.build)
	if err != nil {
		return err
	}
	return streamFromApp(cmd, command)
}

func runStatus(cmd *cobra.Command, args []string) error {
	return streamFromApp(cmd, "docker compose -f docker-compose.yml ps -a")
}

func streamFromApp(cmd *cobra.Command, command string) error {
	cfg, _, err := loadValidConfig()
	if err != nil {
		return err
	}
	client, err := connect(cfg)
	if err != nil {
		return fmt.Errorf("SSH connect: %w", err)
	}
	defer client.Close()

	err = client.Stream(cmd.Context(), inAppDir(cfg.App.Name, command), os.Stdout)
	if cmd.Context().Err() != nil {
		return nil // stopped by the user
	}
	return err
}
