package generator

import (
	"strings"
	"testing"
)

func contains(s, substr string) bool { return strings.Contains(s, substr) }

func TestNginxConfigNoDomain(t *testing.T) {
	cfg := NginxConfig("", "myapp", 8080, false)
	if !contains(cfg, "server_name _;") {
		t.Errorf("expected wildcard server_name for empty domain, got:\n%s", cfg)
	}
	if contains(cfg, "ssl_certificate") {
		t.Errorf("did not expect SSL directives with no domain:\n%s", cfg)
	}
	if !contains(cfg, "server app:8080;") {
		t.Errorf("expected upstream to point at app:8080, got:\n%s", cfg)
	}
}

func TestNginxConfigDomainNoSSL(t *testing.T) {
	cfg := NginxConfig("example.com", "myapp", 3000, false)
	if !contains(cfg, "server_name example.com;") {
		t.Errorf("expected server_name example.com, got:\n%s", cfg)
	}
	if contains(cfg, "ssl_certificate") {
		t.Errorf("did not expect SSL directives when ssl=false, got:\n%s", cfg)
	}
}

func TestNginxConfigDomainWithSSL(t *testing.T) {
	cfg := NginxConfig("example.com", "myapp", 3000, true)
	if !contains(cfg, "server_name example.com;") {
		t.Errorf("expected server_name example.com, got:\n%s", cfg)
	}
	if !contains(cfg, "/etc/letsencrypt/live/example.com/fullchain.pem") {
		t.Errorf("expected cert path for example.com, got:\n%s", cfg)
	}
	if !contains(cfg, "/etc/letsencrypt/live/example.com/privkey.pem") {
		t.Errorf("expected key path for example.com, got:\n%s", cfg)
	}
	if !contains(cfg, "return 301 https://$host$request_uri;") {
		t.Errorf("expected HTTP->HTTPS redirect, got:\n%s", cfg)
	}
}

func TestNginxConfigSSLIgnoredWithoutDomain(t *testing.T) {
	// ssl=true but no domain — there's nothing to issue a cert for, so this
	// must fall back to the plain HTTP config rather than emitting a broken
	// SSL block with an empty server_name.
	cfg := NginxConfig("", "myapp", 8080, true)
	if contains(cfg, "ssl_certificate") {
		t.Errorf("did not expect SSL directives when domain is empty, got:\n%s", cfg)
	}
}
