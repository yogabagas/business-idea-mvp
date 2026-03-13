package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CreatePR creates a pull request and returns the PR URL.
func CreatePR(ctx context.Context, token, ownerRepo, headBranch, baseBranch, title, body string) (prURL string, err error) {
	if token == "" || ownerRepo == "" {
		return "", fmt.Errorf("github: GITHUB_TOKEN and GITHUB_REPO required")
	}
	owner, repo := splitOwnerRepo(ownerRepo)
	if owner == "" || repo == "" {
		return "", fmt.Errorf("github: GITHUB_REPO must be owner/repo")
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, repo)
	reqBody := map[string]string{
		"title": title,
		"head":  headBranch,
		"base":  baseBranch,
		"body":  body,
	}
	if baseBranch == "" {
		reqBody["base"] = "develop"
	}
	enc, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(enc))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnprocessableEntity {
		// PR may already exist for this head→base; try to find it.
		existing, err := findExistingPR(ctx, token, owner, repo, headBranch, baseBranch)
		if err == nil && existing != "" {
			return existing, nil
		}
		return "", fmt.Errorf("github create PR: 422 Unprocessable Entity (PR may already exist or no diff between branches)")
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var e struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return "", fmt.Errorf("github create PR: %s %s", resp.Status, e.Message)
	}
	var pr struct {
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return "", err
	}
	return pr.HTMLURL, nil
}

// findExistingPR looks up an open PR for the given head→base and returns its URL.
func findExistingPR(ctx context.Context, token, owner, repo, head, base string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?state=open&head=%s:%s&base=%s", owner, repo, owner, head, base)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var prs []struct {
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return "", err
	}
	if len(prs) > 0 {
		return prs[0].HTMLURL, nil
	}
	return "", fmt.Errorf("no existing PR found")
}

func splitOwnerRepo(s string) (owner, repo string) {
	for i, c := range s {
		if c == '/' {
			return s[:i], s[i+1:]
		}
	}
	return "", ""
}
