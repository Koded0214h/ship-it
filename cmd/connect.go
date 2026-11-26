package cmd

import (
	"fmt"

	"github.com/Koded0214h/ship/internal/ui"
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Test and verify your server SSH connection",
	RunE:  runConnect,
}

func runConnect(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadValidConfig()
	if err != nil {
		return err
	}

	fmt.Printf("Connecting to %s@%s...\n", cfg.Server.User, cfg.Server.Host)

	client, err := connect(cfg)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer client.Close()

	fmt.Println(ui.SuccessStyle.Render("✓ Connected successfully!"))
	fmt.Println()

	info := client.GetServerInfo(cmd.Context())
	fmt.Println(ui.BoldStyle.Render("Server information:"))
	fmt.Printf("  OS:      %s\n", info.OS)
	fmt.Printf("  Arch:    %s\n", info.Arch)
	fmt.Printf("  Disk:    %s\n", info.Disk)
	fmt.Printf("  Memory:  %s\n", info.Memory)
	fmt.Printf("  Docker:  %s\n", ui.Check(info.Docker))
	fmt.Printf("  Compose: %s\n", ui.Check(info.Compose))

	if !info.Docker || !info.Compose {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("⚠ Docker with Compose v2 must be installed and usable by this user before deploying."))
		fmt.Println(ui.MutedStyle.Render("  Ship does not install it: https://docs.docker.com/engine/install/"))
	}

	return nil
}
