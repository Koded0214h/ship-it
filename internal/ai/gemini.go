package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"github.com/kodedlabs/ship/internal/detector"
	"google.golang.org/api/option"
)

type geminiProvider struct {
	apiKey string
	model  string
}

func newGeminiProvider(apiKey, model string) *geminiProvider {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &geminiProvider{apiKey: apiKey, model: model}
}

func (p *geminiProvider) Name() string { return "Google Gemini" }

func (p *geminiProvider) Ping(ctx context.Context) error {
	client, err := genai.NewClient(ctx, option.WithAPIKey(p.apiKey))
	if err != nil {
		return err
	}
	defer client.Close()
	model := client.GenerativeModel(p.model)
	_, err = model.GenerateContent(ctx, genai.Text("hi"))
	return err
}

func (p *geminiProvider) GenerateDeploymentPlan(ctx context.Context, info *detector.ProjectInfo, userPrompt string) (*DeploymentPlan, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(p.apiKey))
	if err != nil {
		return nil, fmt.Errorf("creating gemini client: %w", err)
	}
	defer client.Close()

	projectJSON, _ := json.MarshalIndent(info, "", "  ")
	prompt := fmt.Sprintf(`%s

Project Info:
%s

User request:
%s

Generate the complete deployment plan as a single JSON object.`, systemPrompt(), string(projectJSON), userPrompt)

	model := client.GenerativeModel(p.model)
	model.ResponseMIMEType = "application/json"

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gemini API error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	raw := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	raw = extractJSON(raw)

	var plan DeploymentPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return nil, fmt.Errorf("parsing deployment plan: %w\nRaw response:\n%s", err, raw)
	}
	return &plan, nil
}
