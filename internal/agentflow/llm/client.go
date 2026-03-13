package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	OpenAIBaseURL = "https://api.openai.com/v1"
	GroqBaseURL   = "https://api.groq.com/openai/v1"
)

// --- Message types (support both plain and tool-calling) ---

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// --- Tool definition types (OpenAI function-calling format) ---

type Tool struct {
	Type     string       `json:"type"`
	Function FunctionDef  `json:"function"`
}

type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// --- Request / response ---

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Tools       []Tool        `json:"tools,omitempty"`
	Temperature float32       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type chatCompletionResponse struct {
	Choices []choiceItem `json:"choices"`
	Usage   *UsageInfo   `json:"usage,omitempty"`
}

type choiceItem struct {
	Message ChatMessage `json:"message"`
}

type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatOpts are optional parameters for Chat / ChatWithTools.
type ChatOpts struct {
	MaxTokens   int
	Temperature float32
}

// ChatResult is the full result from ChatWithTools, including usage info.
type ChatResult struct {
	Message ChatMessage
	Usage   *UsageInfo
}

// Chat sends a simple chat completion (no tools). Returns content string.
func Chat(ctx context.Context, baseURL, apiKey, model string, messages []ChatMessage, opts ...ChatOpts) (string, error) {
	result, err := ChatWithTools(ctx, baseURL, apiKey, model, messages, nil, resolveOpts(opts))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Message.Content), nil
}

// ChatWithTools sends a chat completion with optional tool definitions.
// Returns the full ChatResult including any tool_calls the model wants to invoke.
func ChatWithTools(ctx context.Context, baseURL, apiKey, model string, messages []ChatMessage, tools []Tool, opts ChatOpts) (*ChatResult, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("llm: api key required")
	}
	if model == "" {
		model = "gpt-4o-mini"
		if baseURL == GroqBaseURL {
			model = "llama-3.3-70b-versatile"
		}
	}
	maxTokens := 2048
	temperature := float32(0.3)
	if opts.MaxTokens > 0 {
		maxTokens = opts.MaxTokens
	}
	if opts.Temperature > 0 {
		temperature = opts.Temperature
	}

	reqBody := chatCompletionRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
	if len(tools) > 0 {
		reqBody.Tools = tools
	}

	enc, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	url := baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(enc))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		_, _ = errBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("llm %s: %s %s", url, resp.Status, errBody.String())
	}
	var out chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("llm: no choices in response")
	}
	return &ChatResult{
		Message: out.Choices[0].Message,
		Usage:   out.Usage,
	}, nil
}

func resolveOpts(opts []ChatOpts) ChatOpts {
	if len(opts) > 0 {
		return opts[0]
	}
	return ChatOpts{}
}
