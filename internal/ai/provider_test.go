package ai

import "testing"

func TestNewProvider(t *testing.T) {
	cases := []struct {
		alias    string
		wantName string
	}{
		{"anthropic", "Claude (Anthropic)"},
		{"claude", "Claude (Anthropic)"},
		{"openai", "OpenAI"},
		{"gemini", "Google Gemini"},
		{"google", "Google Gemini"},
	}
	for _, c := range cases {
		t.Run(c.alias, func(t *testing.T) {
			p, err := New(c.alias, "test-key", "test-model")
			if err != nil {
				t.Fatalf("New(%q): unexpected error: %v", c.alias, err)
			}
			if got := p.Name(); got != c.wantName {
				t.Errorf("Name() = %q, want %q", got, c.wantName)
			}
		})
	}
}

func TestNewProviderUnsupported(t *testing.T) {
	if _, err := New("not-a-real-provider", "key", "model"); err == nil {
		t.Fatal("expected an error for an unsupported provider, got nil")
	}
}
