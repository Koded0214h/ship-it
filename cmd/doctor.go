package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Koded0214h/ship/internal/ai"
	"github.com/Koded0214h/ship/internal/config"
	sshclient "github.com/Koded0214h/ship/internal/ssh"
	"github.com/Koded0214h/ship/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostics on your Ship setup",
	RunE:  runDoctor,
}

// minFreeDiskKB is what a build comfortably needs.
const minFreeDiskKB = 2 * 1024 * 1024

func runDoctor(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	fmt.Println(ui.BrandStyle.Render("Running Ship diagnostics..."))
	fmt.Println()

	allOk := true
	check := func(name string, fn func() (string, error)) bool {
		detail, err := fn()
		if err != nil {
			fmt.Printf("  %s %s\n", ui.CrossMark, name)
			fmt.Printf("     %s\n", ui.ErrorStyle.Render(sanitize(err.Error())))
			allOk = false
			return false
		}
		line := fmt.Sprintf("  %s %s", ui.CheckMark, name)
		if detail != "" {
			line += ui.MutedStyle.Render("  " + sanitize(detail))
		}
		fmt.Println(line)
		return true
	}

	var cfg *config.Config
	if !check("Config file", func() (string, error) {
		loaded, _, err := loadValidConfig()
		cfg = loaded
		return ".ship/config.yaml is valid", err
	}) {
		return fmt.Errorf("fix the config before running other checks")
	}

	var client *sshclient.Client
	if check("SSH connection", func() (string, error) {
		var err error
		client, err = connect(cfg)
		return fmt.Sprintf("%s@%s", cfg.Server.User, cfg.Server.Host), err
	}) {
		defer client.Close()
		serverChecks(ctx, client, cfg, check)
	}

	check("AI provider API key", func() (string, error) {
		if cfg.AI.APIKey == "" {
			return "skipped — no AI configured", nil
		}
		provider, err := ai.New(cfg.AI.Provider, cfg.AI.APIKey, cfg.AI.Model)
		if err != nil {
			return "", err
		}
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		if err := provider.Ping(ctx); err != nil {
			return "", fmt.Errorf("%s key invalid or unreachable: %v", provider.Name(), err)
		}
		return provider.Name() + " key valid", nil
	})

	if cfg.App.Domain != "" {
		check("Domain points to the server", func() (string, error) {
			return domainPointsTo(ctx, net.DefaultResolver.LookupHost, cfg.App.Domain, cfg.Server.Host)
		})
	}

	check("App answers from here", func() (string, error) {
		host := cfg.App.Domain
		if host == "" {
			host = cfg.Server.Host
		}
		url := "http://" + net.JoinHostPort(host, strconv.Itoa(cfg.App.ListenPort()))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}}
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("cannot reach %s (not deployed yet, or the firewall blocks the port): %v", url, err)
		}
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			return "", fmt.Errorf("%s answered HTTP %d", url, resp.StatusCode)
		}
		return fmt.Sprintf("%s — HTTP %d", url, resp.StatusCode), nil
	})

	fmt.Println()
	if !allOk {
		return fmt.Errorf("some checks failed — review the errors above")
	}
	fmt.Println(ui.SuccessStyle.Render("✓ All checks passed."))
	return nil
}

// serverChecks only reads from the server.
func serverChecks(ctx context.Context, client *sshclient.Client, cfg *config.Config, check func(string, func() (string, error)) bool) {
	check("Docker on server", func() (string, error) {
		info := client.GetServerInfo(ctx)
		if !info.Docker {
			return "", fmt.Errorf("Docker is missing, not running, or not usable by %s — https://docs.docker.com/engine/install/", cfg.Server.User)
		}
		if !info.Compose {
			return "", fmt.Errorf("Docker Compose v2 is not installed")
		}
		return "Docker and Compose available", nil
	})

	check("Disk space", func() (string, error) {
		out, err := client.Output(ctx, "df -Pk / | awk 'NR==2 {print $4}'")
		if err != nil {
			return "", err
		}
		free, err := strconv.Atoi(strings.TrimSpace(out))
		if err != nil {
			return "", fmt.Errorf("unexpected df output %q", out)
		}
		detail := fmt.Sprintf("%.1f GB free", float64(free)/(1024*1024))
		if free < minFreeDiskKB {
			return "", fmt.Errorf("only %s; builds may fail", detail)
		}
		return detail, nil
	})

	check("App containers", func() (string, error) {
		out, err := client.Output(ctx, inAppDir(cfg.App.Name, "docker compose -f docker-compose.yml ps -a --format '{{.Service}} {{.State}}'"))
		if err != nil {
			return "not deployed yet", nil
		}
		return containerSummary(out)
	})
}

// containerSummary fails when any service of the app is not running.
func containerSummary(out string) (string, error) {
	var all, bad []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		service, state, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}
		all = append(all, service)
		if state != "running" {
			bad = append(bad, service+" is "+state)
		}
	}
	if len(bad) > 0 {
		return "", fmt.Errorf("%s — see 'ship logs'", strings.Join(bad, ", "))
	}
	if len(all) == 0 {
		return "not deployed yet", nil
	}
	return strings.Join(all, ", ") + " running", nil
}

type lookupFunc func(ctx context.Context, host string) ([]string, error)

// domainPointsTo compares DNS for the domain with the server's addresses.
func domainPointsTo(ctx context.Context, lookup lookupFunc, domain, server string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	addresses := func(host string) ([]string, error) {
		if net.ParseIP(host) != nil {
			return []string{host}, nil
		}
		return lookup(ctx, host)
	}
	want, err := addresses(server)
	if err != nil {
		return "", fmt.Errorf("cannot resolve server %s: %v", server, err)
	}
	got, err := addresses(domain)
	if err != nil {
		return "", fmt.Errorf("cannot resolve %s: %v", domain, err)
	}
	for _, address := range got {
		if slices.Contains(want, address) {
			return domain + " → " + address, nil
		}
	}
	return "", fmt.Errorf("%s resolves to %s, not to the server (%s). A proxy such as Cloudflare also causes this",
		domain, strings.Join(got, ", "), strings.Join(want, ", "))
}
