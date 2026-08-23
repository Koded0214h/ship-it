package deploy

import (
	"strings"
	"testing"

	"github.com/Koded0214h/ship/internal/ai"
)

func TestSelectNginxConfigNoDomainOverridesAIPlan(t *testing.T) {
	// No domain: even if the AI plan generated an SSL config (which would be
	// broken against a bare IP), we must fall back to the known-good
	// HTTP-only config rather than trust the AI's output.
	plan := &ai.DeploymentPlan{
		Domain:      "",
		SSLEnabled:  true,
		NginxConfig: "this looks like a broken SSL config with no domain",
	}
	got := selectNginxConfig(plan, "myapp", 8080)
	if strings.Contains(got, "broken SSL config") {
		t.Fatalf("expected the AI-provided config to be overridden when there's no domain, got:\n%s", got)
	}
	if !strings.Contains(got, "server_name _;") {
		t.Fatalf("expected the fallback HTTP-only config, got:\n%s", got)
	}
}

func TestSelectNginxConfigDomainWithoutSSLOverridesAIPlan(t *testing.T) {
	// Domain present but SSL not requested: still override, same reasoning.
	plan := &ai.DeploymentPlan{
		Domain:      "example.com",
		SSLEnabled:  false,
		NginxConfig: "ai generated config",
	}
	got := selectNginxConfig(plan, "myapp", 8080)
	if got == "ai generated config" {
		t.Fatalf("expected override to the generated HTTP config, got AI plan verbatim")
	}
	if !strings.Contains(got, "server_name example.com;") {
		t.Fatalf("expected server_name example.com in fallback config, got:\n%s", got)
	}
}

func TestSelectNginxConfigDomainWithSSLUsesAIPlan(t *testing.T) {
	// Domain + SSL both present: trust the AI-generated config as-is — this
	// is the one case selectNginxConfig doesn't override.
	plan := &ai.DeploymentPlan{
		Domain:      "example.com",
		SSLEnabled:  true,
		NginxConfig: "ai generated ssl config",
	}
	got := selectNginxConfig(plan, "myapp", 8080)
	if got != "ai generated ssl config" {
		t.Fatalf("expected the AI plan's nginx config to pass through unchanged, got:\n%s", got)
	}
}

func TestHealthCheckURL(t *testing.T) {
	cases := []struct {
		name       string
		serverHost string
		plan       *ai.DeploymentPlan
		want       string
	}{
		{"no domain uses server IP over HTTP", "1.2.3.4", &ai.DeploymentPlan{}, "http://1.2.3.4"},
		{"domain without SSL stays HTTP", "1.2.3.4", &ai.DeploymentPlan{Domain: "example.com"}, "http://example.com"},
		{"domain with SSL uses HTTPS", "1.2.3.4", &ai.DeploymentPlan{Domain: "example.com", SSLEnabled: true}, "https://example.com"},
		{"SSL without domain still uses server IP over HTTP", "1.2.3.4", &ai.DeploymentPlan{SSLEnabled: true}, "http://1.2.3.4"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := healthCheckURL(c.serverHost, c.plan); got != c.want {
				t.Errorf("healthCheckURL(%q, %+v) = %q, want %q", c.serverHost, c.plan, got, c.want)
			}
		})
	}
}
