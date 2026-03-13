package config

import (
	"os"
	"strings"
)

// GitProvider is the Git host for creating PR/MR (github or gitlab).
const (
	GitProviderGitHub = "github"
	GitProviderGitLab = "gitlab"
)

// LLM API base URLs (OpenAI-compatible).
const (
	OpenAIBaseURL = "https://api.openai.com/v1"
	GroqBaseURL   = "https://api.groq.com/openai/v1"
)

// AgentFlowConfig holds all settings for the ClickUp → Agent → PR → Telegram pipeline.
// Provide tokens and secrets via env when ready to run.
type AgentFlowConfig struct {
	// Server
	AgentFlowPort string

	// ClickUp — set CLICKUP_API_TOKEN when ready
	ClickUpAPIToken  string
	InProgressStatus string

	// Telegram — set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID when ready
	TelegramBotToken string
	TelegramChatID   string

	// Telegram Assistant Bot (for implementation, set TELEGRAM_ASSISTANT_BOT_TOKEN)
	TelegramAssistantBotToken string
	TelegramAssistantChatID   string

	// Git provider: "github" or "gitlab" (default github)
	GitProvider string
	// TARGET_REPO_URL: clone URL (GitHub or GitLab); token is injected for auth
	TargetRepoURL string
	BaseBranch    string
	CommonRepoURL string
	WorkDir       string

	// GitHub (when GitProvider == github)
	GitHubToken string
	GitHubRepo  string // "owner/repo"

	// GitLab (when GitProvider == gitlab)
	GitLabToken   string
	GitLabProject string // "group/repo" or "owner/repo"
	GitLabBaseURL string // optional, e.g. https://gitlab.company.com (default https://gitlab.com)

	// LLM: use Groq and/or OpenAI. If AGENT_LLM_PROVIDER is set, use that ("groq" or "openai"); else use groq if GROQ_API_KEY set, else openai if OPENAI_API_KEY set.
	AgentLLMProvider string // "groq" | "openai" | "" (auto)
	GroqAPIKey       string
	GroqModel        string // default llama-3.3-70b-versatile
	OpenAIAPIKey     string
	OpenAIModel      string // default gpt-4o-mini
}

// GitToken returns the token to use for git clone/push (GitHub or GitLab depending on provider).
func (c *AgentFlowConfig) GitToken() string {
	if strings.ToLower(strings.TrimSpace(c.GitProvider)) == GitProviderGitLab {
		return c.GitLabToken
	}
	return c.GitHubToken
}

func LoadAgentFlow() *AgentFlowConfig {
	baseBranch := os.Getenv("AGENTFLOW_BASE_BRANCH")
	if baseBranch == "" {
		baseBranch = "develop"
	}
	workDir := os.Getenv("AGENTFLOW_WORK_DIR")
	if workDir == "" {
		workDir = "./agentflow-workspace"
	}
	inProgress := os.Getenv("AGENTFLOW_IN_PROGRESS_STATUS")
	if inProgress == "" {
		inProgress = "in progress"
	}
	port := os.Getenv("AGENTFLOW_PORT")
	if port == "" {
		port = "8081"
	}
	gitProvider := strings.ToLower(strings.TrimSpace(os.Getenv("GIT_PROVIDER")))
	if gitProvider == "" {
		gitProvider = GitProviderGitHub
	}
	if gitProvider != GitProviderGitHub && gitProvider != GitProviderGitLab {
		gitProvider = GitProviderGitHub
	}
	return &AgentFlowConfig{
		AgentFlowPort:             port,
		ClickUpAPIToken:           os.Getenv("CLICKUP_API_TOKEN"),
		InProgressStatus:          strings.ToLower(strings.TrimSpace(inProgress)),
		TelegramBotToken:          os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:            os.Getenv("TELEGRAM_CHAT_ID"),
		TelegramAssistantBotToken: os.Getenv("TELEGRAM_ASSISTANT_BOT_TOKEN"),
		TelegramAssistantChatID:   os.Getenv("TELEGRAM_ASSISTANT_CHAT_ID"),
		GitProvider:               gitProvider,
		TargetRepoURL:             os.Getenv("TARGET_REPO_URL"),
		BaseBranch:                baseBranch,
		CommonRepoURL:             os.Getenv("COMMON_REPO_URL"),
		WorkDir:                   workDir,
		GitHubToken:               os.Getenv("GITHUB_TOKEN"),
		GitHubRepo:                os.Getenv("GITHUB_REPO"),
		GitLabToken:               os.Getenv("GITLAB_TOKEN"),
		GitLabProject:             os.Getenv("GITLAB_PROJECT"),
		GitLabBaseURL:             strings.TrimSuffix(os.Getenv("GITLAB_BASE_URL"), "/"),
		AgentLLMProvider:          strings.ToLower(strings.TrimSpace(os.Getenv("AGENT_LLM_PROVIDER"))),
		GroqAPIKey:                os.Getenv("GROQ_API_KEY"),
		GroqModel:                 os.Getenv("GROQ_MODEL"),
		OpenAIAPIKey:              os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:               os.Getenv("OPENAI_MODEL"),
	}
}

// LLMBaseURLAndKey returns base URL, API key, and model for the chosen LLM provider. If none configured, returns "", "", "".
func (c *AgentFlowConfig) LLMBaseURLAndKey() (baseURL, apiKey, model string) {
	provider := strings.ToLower(strings.TrimSpace(c.AgentLLMProvider))
	useGroq := provider == "groq" || (provider != "openai" && c.GroqAPIKey != "")
	if useGroq && c.GroqAPIKey != "" {
		model = c.GroqModel
		if model == "" {
			model = "llama-3.3-70b-versatile"
		}
		return GroqBaseURL, c.GroqAPIKey, model
	}
	if (provider == "openai" || !useGroq) && c.OpenAIAPIKey != "" {
		model = c.OpenAIModel
		if model == "" {
			model = "gpt-4o-mini"
		}
		return OpenAIBaseURL, c.OpenAIAPIKey, model
	}
	return "", "", ""
}
