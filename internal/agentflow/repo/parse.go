package repo

import (
	"regexp"
	"strings"

	"github.com/yogabagas/business-idea-mvp/internal/agentflow/config"
)

const (
	hostGitHub   = "github.com"
	hostGitLab   = "gitlab.com"
	gitLabComURL = "https://gitlab.com"
)

// ResolvedRepo is the repo to use for a run (from description or config defaults).
type ResolvedRepo struct {
	CloneURL       string // URL to clone (with .git)
	Provider       string // config.GitProviderGitHub or config.GitProviderGitLab
	GitHubRepo     string // "owner/repo" for GitHub API
	GitLabProject  string // "group/repo" for GitLab API
	GitLabBaseURL  string // for GitLab API (empty = use config default)
	Token          string // token to use for clone and PR/MR
}

// ParseFromDescription extracts repo from the task description (markdown).
// Supported formats (case-insensitive keys):
//
//   - repo: https://github.com/owner/repo.git
//   - repo: https://gitlab.com/group/repo.git
//   - repository: owner/repo
//   - github: owner/repo
//   - gitlab: group/repo
//   - A standalone line that is a full git URL (github.com or gitlab.com)
//
// Returns nil if nothing found; then use config defaults.
func ParseFromDescription(description string, cfg *config.AgentFlowConfig) *ResolvedRepo {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return nil
	}
	lines := strings.Split(desc, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Strip markdown list or code
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// repo: URL or owner/repo
		if v, ok := parseKey(line, []string{"repo:", "repository:", "repo_url:"}); ok {
			v = strings.TrimSpace(v)
			if isGitURL(v) {
				r := repoFromURL(v, cfg)
				if r != nil {
					r.Token = tokenForProvider(r.Provider, cfg)
					return r
				}
			}
			// "owner/repo" form — use default provider from config
			if idx := strings.Index(v, "/"); idx > 0 && idx < len(v)-1 {
				provider := strings.ToLower(strings.TrimSpace(cfg.GitProvider))
				if provider != config.GitProviderGitLab {
					provider = config.GitProviderGitHub
				}
				return &ResolvedRepo{
					CloneURL:      buildCloneURL(provider, v, cfg.GitLabBaseURL),
					Provider:      provider,
					GitHubRepo:    pick(provider == config.GitProviderGitHub, v, ""),
					GitLabProject: pick(provider == config.GitProviderGitLab, v, ""),
					GitLabBaseURL: pick(provider == config.GitProviderGitLab, cfg.GitLabBaseURL, ""),
					Token:         cfg.GitToken(),
				}
			}
		}
		// github: owner/repo
		if v, ok := parseKey(line, []string{"github:"}); ok {
			v = strings.TrimSpace(v)
			if idx := strings.Index(v, "/"); idx > 0 && idx < len(v)-1 {
				return &ResolvedRepo{
					CloneURL:   "https://github.com/" + v + ".git",
					Provider:   config.GitProviderGitHub,
					GitHubRepo: v,
					Token:      cfg.GitHubToken,
				}
			}
		}
		// gitlab: group/repo
		if v, ok := parseKey(line, []string{"gitlab:"}); ok {
			v = strings.TrimSpace(v)
			if idx := strings.Index(v, "/"); idx > 0 && idx < len(v)-1 {
				base := cfg.GitLabBaseURL
				if base == "" {
					base = gitLabComURL
				}
				return &ResolvedRepo{
					CloneURL:      base + "/" + v + ".git",
					Provider:      config.GitProviderGitLab,
					GitLabProject: v,
					GitLabBaseURL: base,
					Token:         cfg.GitLabToken,
				}
			}
		}
		// Standalone URL line
		if isGitURL(line) {
			r := repoFromURL(line, cfg)
			if r != nil {
				r.Token = tokenForProvider(r.Provider, cfg)
				return r
			}
		}
	}
	return nil
}

func parseKey(line string, keys []string) (string, bool) {
	lower := strings.ToLower(line)
	for _, k := range keys {
		if strings.HasPrefix(lower, strings.ToLower(k)) {
			return strings.TrimSpace(line[len(k):]), true
		}
	}
	return "", false
}

var gitURLRe = regexp.MustCompile(`(?i)https?://[^\s]+\.git`)

func isGitURL(s string) bool {
	return strings.Contains(s, hostGitHub) || strings.Contains(s, hostGitLab) || gitURLRe.MatchString(s)
}

func repoFromURL(rawURL string, cfg *config.AgentFlowConfig) *ResolvedRepo {
	rawURL = strings.TrimSpace(rawURL)
	rawURL = strings.TrimSuffix(rawURL, ".git")
	// Normalize: https://github.com/owner/repo or https://gitlab.com/group/repo
	if strings.Contains(rawURL, hostGitHub) {
		path := extractPath(rawURL, hostGitHub)
		if path == "" {
			return nil
		}
		return &ResolvedRepo{
			CloneURL:   rawURL + ".git",
			Provider:   config.GitProviderGitHub,
			GitHubRepo: path,
		}
	}
	if strings.Contains(rawURL, hostGitLab) {
		path := extractPath(rawURL, hostGitLab)
		if path == "" {
			return nil
		}
		base := gitLabComURL
		return &ResolvedRepo{
			CloneURL:      rawURL + ".git",
			Provider:      config.GitProviderGitLab,
			GitLabProject: path,
			GitLabBaseURL: base,
		}
	}
	// Self-hosted GitLab: https://gitlab.company.com/group/repo (not gitlab.com)
	if strings.Contains(rawURL, "gitlab") && !strings.Contains(rawURL, hostGitLab) {
		after := rawURL
		if i := strings.Index(after, "://"); i >= 0 {
			after = after[i+3:]
		}
		j := strings.Index(after, "/")
		if j > 0 && j < len(after)-1 {
			path := after[j+1:]
			base := rawURL[:len(rawURL)-len(path)-1]
			return &ResolvedRepo{
				CloneURL:      rawURL + ".git",
				Provider:      config.GitProviderGitLab,
				GitLabProject: path,
				GitLabBaseURL: strings.TrimSuffix(base, "/"),
			}
		}
	}
	return nil
}

func extractPath(url, host string) string {
	i := strings.Index(url, host)
	if i < 0 {
		return ""
	}
	path := url[i+len(host):]
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	return path
}

func buildCloneURL(provider, projectPath, gitlabBaseURL string) string {
	if provider == config.GitProviderGitLab {
		base := gitlabBaseURL
		if base == "" {
			base = gitLabComURL
		}
		return strings.TrimSuffix(base, "/") + "/" + projectPath + ".git"
	}
	return "https://" + hostGitHub + "/" + projectPath + ".git"
}

func tokenForProvider(provider string, cfg *config.AgentFlowConfig) string {
	if provider == config.GitProviderGitLab {
		return cfg.GitLabToken
	}
	return cfg.GitHubToken
}

func pick(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
