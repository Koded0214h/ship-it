package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kodedlabs/ship/internal/detector"
	openai "github.com/sashabaranov/go-openai"
)

type openaiProvider struct {
	client *openai.Client
	model  string
}

func newOpenAIProvider(apiKey, model string) *openaiProvider {
	if model == "" {
		model = openai.GPT4o
	}
	return &openaiProvider{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

func (p *openaiProvider) Name() string { return "OpenAI" }

func (p *openaiProvider) Ping(ctx context.Context) error {
	_, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     p.model,
		MaxTokens: 10,
		Messages:  []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "hi"}},
	})
	return err
}

func (p *openaiProvider) GenerateDeploymentPlan(ctx context.Context, info *detector.ProjectInfo, userPrompt string) (*DeploymentPlan, error) {
	projectJSON, _ := json.MarshalIndent(info, "", "  ")

	userMsg := fmt.Sprintf(`Project Info:
%s

User request:
%s

Generate the complete deployment plan as a single JSON object.`, string(projectJSON), userPrompt)

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: p.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt()},
			{Role: openai.ChatMessageRoleUser, Content: userMsg},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		MaxTokens: 8192,
	})
	if err != nil {
		return nil, fmt.Errorf("openai API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from OpenAI")
	}

	raw := resp.Choices[0].Message.Content
	var plan DeploymentPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return nil, fmt.Errorf("parsing deployment plan: %w", err)
	}
	return &plan, nil
}
