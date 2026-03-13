package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// CreateMR creates a merge request and returns the MR web URL.
// projectPath is the project path (e.g. "group/subgroup/repo" or "owner/repo"); it will be URL-encoded.
func CreateMR(ctx context.Context, baseURL, token, projectPath, sourceBranch, targetBranch, title, body string) (mrURL string, err error) {
	if token == "" || projectPath == "" {
		return "", fmt.Errorf("gitlab: GITLAB_TOKEN and GITLAB_PROJECT required")
	}
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	encodedID := url.PathEscape(projectPath)
	apiURL := baseURL + "/api/v4/projects/" + encodedID + "/merge_requests"
	reqBody := map[string]string{
		"source_branch": sourceBranch,
		"target_branch": targetBranch,
		"title":         title,
		"description":   body,
	}
	if targetBranch == "" {
		reqBody["target_branch"] = "develop"
	}
	enc, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(enc))
	if err != nil {
		return "", err
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var e struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return "", fmt.Errorf("gitlab create MR: %s %s", resp.Status, e.Message)
	}
	var mr struct {
		WebURL string `json:"web_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return "", err
	}
	return mr.WebURL, nil
}
