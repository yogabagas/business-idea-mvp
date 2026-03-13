package clickup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const apiBase = "https://api.clickup.com/api/v2"

type Client struct {
	apiToken string
	client   *http.Client
}

func NewClient(apiToken string) *Client {
	return &Client{
		apiToken: apiToken,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

// Task is a subset of ClickUp task response (Get Task).
type Task struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"` // may be empty; rich text in API uses different structure
	Status      *Status  `json:"status"`
	CustomID    string   `json:"custom_id"`
	URL         string   `json:"url"`
}

type Status struct {
	Status string `json:"status"`
	Type   string `json:"type"`
}

// GetTask fetches a task by ID. Description might be in a different field in real API (e.g. content blocks).
func (c *Client) GetTask(ctx context.Context, taskID string) (*Task, error) {
	if c.apiToken == "" {
		return nil, fmt.Errorf("clickup: CLICKUP_API_TOKEN not set")
	}
	url := apiBase + "/task/" + taskID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("clickup get task: %s %s", resp.Status, string(body))
	}
	var out struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Status      *Status  `json:"status"`
		CustomID    string   `json:"custom_id"`
		URL         string   `json:"url"`
		// ClickUp may return description in "markdown_description" or nested content
		MarkdownDescription string `json:"markdown_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	desc := out.Description
	if desc == "" && out.MarkdownDescription != "" {
		desc = out.MarkdownDescription
	}
	return &Task{
		ID:          out.ID,
		Name:        out.Name,
		Description: desc,
		Status:      out.Status,
		CustomID:    out.CustomID,
		URL:         out.URL,
	}, nil
}

// GetTaskDescription returns task name and description for the agent (markdown-friendly).
func (c *Client) GetTaskDescription(ctx context.Context, taskID string) (name, description string, err error) {
	task, err := c.GetTask(ctx, taskID)
	if err != nil {
		return "", "", err
	}
	return task.Name, task.Description, nil
}
