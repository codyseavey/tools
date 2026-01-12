package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// Default Azure OpenAI resource
const (
	DefaultOpenAIResourceID = "/subscriptions/dc216e0e-5d8f-470b-8f7d-fddec411fc68/resourceGroups/evue2-mgmtopenai-rg/providers/Microsoft.CognitiveServices/accounts/evue2-mgmtopenai"
	DefaultOpenAIEndpoint   = "https://evue2-mgmtopenai.openai.azure.com"
	DefaultDeploymentName   = "gpt-4o-mini" // Common deployment name, adjust as needed
	OpenAIAPIVersion        = "2024-02-15-preview"
)

// OpenAIClient handles Azure OpenAI API calls
type OpenAIClient struct {
	endpoint       string
	deploymentName string
	credential     azcore.TokenCredential
	httpClient     *http.Client
}

// ChatMessage represents a message in a chat completion
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents a request to the chat completions API
type ChatCompletionRequest struct {
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
}

// ChatCompletionResponse represents the response from chat completions API
type ChatCompletionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// NewOpenAIClient creates a new Azure OpenAI client
func NewOpenAIClient(credential azcore.TokenCredential, endpoint, deploymentName string) *OpenAIClient {
	if endpoint == "" {
		endpoint = DefaultOpenAIEndpoint
	}
	if deploymentName == "" {
		deploymentName = DefaultDeploymentName
	}

	return &OpenAIClient{
		endpoint:       strings.TrimSuffix(endpoint, "/"),
		deploymentName: deploymentName,
		credential:     credential,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewOpenAIClientWithDefaults creates a client with default Azure OpenAI settings
func NewOpenAIClientWithDefaults(credential azcore.TokenCredential) *OpenAIClient {
	return NewOpenAIClient(credential, DefaultOpenAIEndpoint, DefaultDeploymentName)
}

// getToken retrieves an access token for Azure OpenAI
func (c *OpenAIClient) getToken(ctx context.Context) (string, error) {
	token, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://cognitiveservices.azure.com/.default"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}
	return token.Token, nil
}

// Complete sends a chat completion request
func (c *OpenAIClient) Complete(ctx context.Context, messages []ChatMessage, maxTokens int) (string, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return "", err
	}

	reqBody := ChatCompletionRequest{
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: 0.3, // Lower temperature for more deterministic completions
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		c.endpoint, c.deploymentName, OpenAIAPIVersion)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var completionResp ChatCompletionResponse
	if err := json.Unmarshal(body, &completionResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if completionResp.Error != nil {
		return "", fmt.Errorf("API error: %s", completionResp.Error.Message)
	}

	if len(completionResp.Choices) == 0 {
		return "", fmt.Errorf("no completion returned")
	}

	return completionResp.Choices[0].Message.Content, nil
}

// SuggestKQLQuery suggests a KQL query completion based on the current input
func (c *OpenAIClient) SuggestKQLQuery(ctx context.Context, partialQuery string, availableTables []string) (string, error) {
	systemPrompt := `You are a KQL (Kusto Query Language) expert assistant for Azure Log Analytics.
Your task is to complete or suggest KQL queries based on partial input.

Guidelines:
- Complete the query in a syntactically correct way
- Keep suggestions concise and relevant
- If the query looks complete, suggest improvements or variations
- Use common Log Analytics tables when appropriate
- Focus on practical, commonly-used query patterns
- Only output the query suggestion, no explanations`

	if len(availableTables) > 0 {
		tableList := strings.Join(availableTables, ", ")
		systemPrompt += fmt.Sprintf("\n\nAvailable tables in this workspace: %s", tableList)
	}

	userPrompt := fmt.Sprintf("Complete or suggest a KQL query based on this input:\n%s", partialQuery)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	return c.Complete(ctx, messages, 500)
}

// ExplainKQLQuery explains what a KQL query does
func (c *OpenAIClient) ExplainKQLQuery(ctx context.Context, query string) (string, error) {
	systemPrompt := `You are a KQL (Kusto Query Language) expert.
Explain what the given query does in simple terms.
Be concise but thorough. Format your response clearly.`

	userPrompt := fmt.Sprintf("Explain this KQL query:\n%s", query)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	return c.Complete(ctx, messages, 500)
}

// FixKQLQuery suggests fixes for a KQL query with errors
func (c *OpenAIClient) FixKQLQuery(ctx context.Context, query, errorMsg string) (string, error) {
	systemPrompt := `You are a KQL (Kusto Query Language) expert.
Given a query with an error, provide a corrected version.
Only output the corrected query, no explanations.`

	userPrompt := fmt.Sprintf("Fix this KQL query:\n%s\n\nError: %s", query, errorMsg)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	return c.Complete(ctx, messages, 500)
}
