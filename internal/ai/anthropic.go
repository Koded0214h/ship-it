package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Koded0214h/ship/internal/detector"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type anthropicProvider struct {
	client anthropic.Client
	model  string
}

func newAnthropicProvider(apiKey, model string) *anthropicProvider {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &anthropicProvider{client: client, model: model}
}

func (p *anthropicProvider) Name() string { return "Claude (Anthropic)" }

func (p *anthropicProvider) Ping(ctx context.Context) error {
	_, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 10,
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock("hi"))},
	})
	return err
}

func (p *anthropicProvider) GenerateDeploymentPlan(ctx context.Context, info *detector.ProjectInfo, userPrompt string) (*DeploymentPlan, error) {
	projectJSON, _ := json.MarshalIndent(info, "", "  ")

	userMsg := fmt.Sprintf(`Project Info:
%s

User request:
%s

Generate the complete deployment plan as a single JSON object.`, string(projectJSON), userPrompt)

	msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 8192,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt()},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic API error: %w", err)
	}

	if msg.StopReason == anthropic.StopReasonMaxTokens {
		return nil, fmt.Errorf("the response from Claude was cut off before the plan was complete")
	}
	var text strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	if text.Len() == 0 {
		return nil, fmt.Errorf("empty response from Claude")
	}

	raw := extractJSON(text.String())

	var plan DeploymentPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return nil, fmt.Errorf("parsing deployment plan: %w\nRaw response:\n%s", err, raw)
	}
	return &plan, nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "```json"); idx >= 0 {
		s = s[idx+7:]
	} else if idx := strings.Index(s, "```"); idx >= 0 {
		s = s[idx+3:]
	}
	if idx := strings.LastIndex(s, "```"); idx >= 0 {
		s = s[:idx]
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		s = s[start : end+1]
	}
	return strings.TrimSpace(s)
}
