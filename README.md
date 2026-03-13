# Business Idea MVP — ClickUp Agent Flow

When a **ClickUp** task is moved to **In Progress**, this service runs a pipeline: create a branch from `develop`, run an AI agent (placeholder by default), commit, push, open a **PR**, and notify **Telegram**.

## Flow

1. ClickUp task status → **In Progress** → webhook triggers the app.
2. App fetches task description from ClickUp API.
3. **Repo**: resolved from the task description (see below) or from env defaults.
4. Git: clone/pull repo, create branch `feature/<task_id>` from `develop`.
5. Agent: read task description (+ optional common repo MD); placeholder writes a file (replace with LLM later).
6. Git: commit and push.
7. **GitHub** or **GitLab**: create pull request / merge request.
8. Telegram: send message with PR/MR link.

Developers review and merge the PR/MR as usual.

## Run

```bash
cp .env.example .env
# Edit .env with your tokens (see below)

go mod tidy
go run ./cmd/clickup-agent
```

Listens on **:8081** by default (`AGENTFLOW_PORT`).

## Repo from ClickUp description (dynamic)

The **target repo** is read from the task description first. Supported formats (one per task):

- `repo: https://github.com/owner/repo.git` or `repo: https://gitlab.com/group/repo.git`
- `repository: owner/repo` (uses `GIT_PROVIDER` from env)
- `github: owner/repo`
- `gitlab: group/repo`
- A standalone line that is a full git URL

**If every task includes one of these**, you do **not** need `TARGET_REPO_URL`, `GITHUB_REPO`, or `GITLAB_PROJECT` in env. You still set `GITHUB_TOKEN` and/or `GITLAB_TOKEN` so the app can clone and open PRs/MRs for the repos named in the descriptions.

**If some tasks don’t specify a repo**, the app falls back to env: `TARGET_REPO_URL` and `GIT_PROVIDER` (and `GITHUB_REPO` / `GITLAB_PROJECT`). Set those only when you want a default repo.

For a full **task description template** (repo + goals + definition of done + rules to avoid), see **[docs/CLICKUP_DESCRIPTION_TEMPLATE.md](docs/CLICKUP_DESCRIPTION_TEMPLATE.md)**.

## Environment variables

**Common**

| Variable | Required | Description |
|----------|----------|-------------|
| `CLICKUP_API_TOKEN` | Yes | ClickUp API token. |
| `TELEGRAM_BOT_TOKEN` | Yes | Telegram bot token (@BotFather). |
| `TELEGRAM_CHAT_ID` | Yes | Chat or channel ID. |
| `TARGET_REPO_URL` | No | Fallback clone URL only when a task description has no repo line. Omit if every task specifies repo in description. |
| `GIT_PROVIDER` | No | Fallback: `github` or `gitlab` when repo not in description (default: `github`). |
| `COMMON_REPO_URL` | No | Clone URL for "memory" repo (MD context). |
| `AGENT_LLM_PROVIDER` | No | `groq` or `openai`; if unset, Groq is used when `GROQ_API_KEY` is set, else OpenAI. |
| `GROQ_API_KEY` | No | Groq API key (console.groq.com); agent uses it to generate commit message/plan. |
| `GROQ_MODEL` | No | Model name (default: `llama-3.3-70b-versatile`). |
| `OPENAI_API_KEY` | No | OpenAI API key; used when provider is openai or Groq not set. |
| `OPENAI_MODEL` | No | Model name (default: `gpt-4o-mini`). |
| `AGENTFLOW_*` | No | Port, base branch, work dir, etc. |

**GitHub** — set `GITHUB_TOKEN` if any task uses a GitHub repo (from description or fallback).

| Variable | Required | Description |
|----------|----------|-------------|
| `GITHUB_TOKEN` | Yes (if using GitHub) | Token with `repo` scope; needed to clone and open PRs. |
| `GITHUB_REPO` | No | Fallback `owner/repo` only when a task has no repo in description. |

**GitLab** — set `GITLAB_TOKEN` if any task uses a GitLab repo (from description or fallback).

| Variable | Required | Description |
|----------|----------|-------------|
| `GITLAB_TOKEN` | Yes (if using GitLab) | Token for clone and MRs. |
| `GITLAB_PROJECT` | No | Fallback project path when a task has no repo in description. |
| `GITLAB_BASE_URL` | No | GitLab base URL (default: `https://gitlab.com`). |

## ClickUp webhook

Create a webhook in ClickUp pointing to `https://your-server/webhook/clickup`, event **Task Status Updated**. For local dev use ngrok or similar.

## Project layout

- `cmd/clickup-agent/` — HTTP server and webhook handler.
- `internal/agentflow/` — config, ClickUp client, Telegram, git, GitHub PR / GitLab MR, agent, pipeline.
- `docs/AGENTFLOW.md` — detailed setup (webhook, Telegram, adding an LLM).
