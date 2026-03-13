package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/yogabagas/business-idea-mvp/internal/agentflow/agent"
	"github.com/yogabagas/business-idea-mvp/internal/agentflow/clickup"
	"github.com/yogabagas/business-idea-mvp/internal/agentflow/config"
	"github.com/yogabagas/business-idea-mvp/internal/agentflow/github"
	"github.com/yogabagas/business-idea-mvp/internal/agentflow/gitlab"
	"github.com/yogabagas/business-idea-mvp/internal/agentflow/gitops"
	"github.com/yogabagas/business-idea-mvp/internal/agentflow/repo"
	"github.com/yogabagas/business-idea-mvp/internal/agentflow/telegram"
)

func ForwardTaskToAssistant(taskID, repoTarget, taskDesc, botToken, chatID string) error {
	msg := fmt.Sprintf(
		`<b>[ClickUp]</b>\nTaskID: <code>%s</code>\nRepo: <code>%s</code>\n\nDescription:\n<pre>%s</pre>`,
		taskID, repoTarget, taskDesc)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       msg,
		"parse_mode": "HTML",
	}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// Run runs the full flow for a ClickUp task: fetch task → resolve repo (from description or config) → create branch → agent → commit → push → PR/MR → Telegram.
func Run(ctx context.Context, cfg *config.AgentFlowConfig, taskID string) (prURL string, err error) {
	// 1. Fetch task from ClickUp
	cu := clickup.NewClient(cfg.ClickUpAPIToken)
	taskName, taskDesc, err := cu.GetTaskDescription(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("get task: %w", err)
	}
	branchName := branchNameFromTask(taskID)

	// Forward context/task ke assistant (Telegram bot)
	repoTarget := cfg.TargetRepoURL
	go ForwardTaskToAssistant(taskID, repoTarget, taskDesc, cfg.TelegramAssistantBotToken, cfg.TelegramAssistantChatID)

	// 2. Resolve repo: from task description or fall back to config
	var cloneURL, gitToken, provider, githubRepo, gitlabProject, gitlabBaseURL string
	if res := repo.ParseFromDescription(taskDesc, cfg); res != nil {
		cloneURL = res.CloneURL
		gitToken = res.Token
		provider = res.Provider
		githubRepo = res.GitHubRepo
		gitlabProject = res.GitLabProject
		gitlabBaseURL = res.GitLabBaseURL
	}
	if cloneURL == "" {
		cloneURL = cfg.TargetRepoURL
		gitToken = cfg.GitToken()
		provider = strings.ToLower(strings.TrimSpace(cfg.GitProvider))
		if provider != config.GitProviderGitLab {
			provider = config.GitProviderGitHub
		}
		githubRepo = cfg.GitHubRepo
		gitlabProject = cfg.GitLabProject
		gitlabBaseURL = cfg.GitLabBaseURL
	}
	if cloneURL == "" {
		return "", fmt.Errorf("no repo configured: set TARGET_REPO_URL or specify repo in task description")
	}

	// 3. Prepare repo (clone/pull, checkout base, create branch)
	gitRepo := gitops.NewRepo(cfg.WorkDir, cloneURL, cfg.BaseBranch, gitToken)
	repoPath, err := gitRepo.Prepare(ctx, branchName)
	if err != nil {
		return "", fmt.Errorf("prepare repo: %w", err)
	}

	// 4. Optional: clone common repo for agent context
	commonPath := ""
	if cfg.CommonRepoURL != "" {
		commonRepo := gitops.NewRepo(cfg.WorkDir, cfg.CommonRepoURL, "main", gitToken)
		commonPath, _ = commonRepo.Prepare(ctx, "main")
	}

	// 5. Run agent (manual/assistant)
	llmBaseURL, llmAPIKey, llmModel := cfg.LLMBaseURLAndKey()
	runner := agent.NewRunner(llmBaseURL, llmAPIKey, llmModel, commonPath)
	result, err := runner.Run(ctx, repoPath, taskID, taskName, taskDesc)
	if err != nil {
		return "", fmt.Errorf("agent: %w", err)
	}

	// 6. Commit and push
	if err := gitRepo.CommitPush(ctx, branchName, result.CommitMessage); err != nil {
		return "", fmt.Errorf("commit push: %w", err)
	}

	// 7. Create PR (GitHub) or MR (GitLab)
	title := fmt.Sprintf("[%s] %s", taskID, taskName)
	body := fmt.Sprintf("## %s\n\n%s\n\n---\nOpened by Agent Flow (ClickUp → In Progress).", taskName, taskDesc)
	if provider == config.GitProviderGitLab {
		prURL, err = gitlab.CreateMR(ctx, gitlabBaseURL, gitToken, gitlabProject, branchName, cfg.BaseBranch, title, body)
	} else {
		prURL, err = github.CreatePR(ctx, gitToken, githubRepo, branchName, cfg.BaseBranch, title, body)
	}
	if err != nil {
		return "", fmt.Errorf("create PR/MR: %w", err)
	}

	// 8. Notify Telegram
	tg := telegram.NewNotifier(cfg.TelegramBotToken, cfg.TelegramChatID)
	msg := fmt.Sprintf("🦾 <b>Pull Request Review</b>\n<a href=\"%s\">Review PR</a>", prURL)
	if err := tg.Send(ctx, msg); err != nil {
		log.Printf("telegram send failed: %v", err)
	}
	return prURL, nil
}

func branchNameFromTask(taskID string) string {
	s := strings.ReplaceAll(taskID, " ", "-")
	if !strings.HasPrefix(s, "feature/") && !strings.HasPrefix(s, "fix/") {
		s = "feature/" + s
	}
	return s
}
