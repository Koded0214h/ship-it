package cmd

import (
	"fmt"

	"github.com/kodedlabs/ship/internal/config"
	sshclient "github.com/kodedlabs/ship/internal/ssh"
	"github.com/kodedlabs/ship/internal/ui"
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Test and verify your server SSH connection",
	RunE:  runConnect,
}

func runConnect(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Printf("Connecting to %s@%s...\n", cfg.Server.User, cfg.Server.Host)

	client, err := sshclient.Connect(sshclient.Config{
		Host:    cfg.Server.Host,
		User:    cfg.Server.User,
		KeyPath: cfg.Server.KeyPath,
		Port:    cfg.Server.Port,
	})
	if err != nil {
		fmt.Println(ui.ErrorStyle.Render("✗ Connection failed: " + err.Error()))
		return nil
	}
	defer client.Close()

	fmt.Println(ui.SuccessStyle.Render("✓ Connected successfully!"))
	fmt.Println()

	info := client.GetServerInfo()
	fmt.Println(ui.BoldStyle.Render("Server information:"))
	fmt.Printf("  OS:      %s\n", info.OS)
	fmt.Printf("  Arch:    %s\n", info.Arch)
	fmt.Printf("  Disk:    %s\n", info.Disk)
	fmt.Printf("  Memory:  %s\n", info.Memory)
	fmt.Printf("  Docker:  %s\n", ui.Check(info.Docker))

	if !info.Docker {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("⚠ Docker is not installed on your server."))
		fmt.Println(ui.MutedStyle.Render("  Ship will install Docker automatically when you deploy."))
	}

	return nil
}
