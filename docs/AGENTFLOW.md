# ClickUp Agent Flow

When a ClickUp task is moved to **In Progress**, this service runs a pipeline: create a branch from `develop`, run an AI agent (placeholder by default), commit, push, open a PR, and notify a Telegram channel.

## Flow

1. **ClickUp** → Task status set to "In Progress" → Webhook sends `taskStatusUpdated` to your app.
2. **App** → Fetches task description from ClickUp API.
3. **Git** → Clones target repo (or pulls), creates branch `feature/<task_id>` from `develop`.
4. **Agent** → Reads task description (+ optional common repo MD). Placeholder: writes a small file; you can replace with an LLM that edits code.
5. **Git** → Commits and pushes the branch.
6. **GitHub** or **GitLab** → Creates a pull request or merge request (repo can be from task description or env).
7. **Telegram** → Sends a message with the PR/MR link.

**Dynamic repo from description:** You can set the target repo per task in the ClickUp task description. Supported lines (case-insensitive): `repo: https://github.com/owner/repo.git`, `repo: https://gitlab.com/group/repo.git`, `repository: owner/repo`, `github: owner/repo`, `gitlab: group/repo`, or a standalone line with a full git URL. If none are found, env `TARGET_REPO_URL` and `GIT_PROVIDER` (and `GITHUB_REPO` / `GITLAB_PROJECT`) are used.

**Task description template:** For a consistent, AI-friendly structure (repo + goals / definition of done + rules to avoid), use **[CLICKUP_DESCRIPTION_TEMPLATE.md](CLICKUP_DESCRIPTION_TEMPLATE.md)** in this folder.

Developers review and merge the PR/MR as usual.

## Run the service

```bash
# From repo root; ensure .env has the required tokens (see below)
go run ./cmd/clickup-agent
```

By default it listens on **:8081** (set `AGENTFLOW_PORT` to change).

## Environment variables (provide when ready)

**Common:** `CLICKUP_API_TOKEN`, `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`, `TARGET_REPO_URL`. Set `GIT_PROVIDER` to `github` or `gitlab` (default: `github`).

**GitHub** (when `GIT_PROVIDER=github`): `GITHUB_TOKEN`, `GITHUB_REPO`. `TARGET_REPO_URL` e.g. `https://github.com/owner/repo.git`.

**GitLab** (when `GIT_PROVIDER=gitlab`): `GITLAB_TOKEN`, `GITLAB_PROJECT` (e.g. `group/repo`). Optional `GITLAB_BASE_URL` for self-hosted (default `https://gitlab.com`). `TARGET_REPO_URL` e.g. `https://gitlab.com/group/repo.git`.

**Optional:** `COMMON_REPO_URL`, `AGENTFLOW_*`. **LLM (Groq or OpenAI):** set `GROQ_API_KEY` and/or `OPENAI_API_KEY` so the agent uses an LLM to generate the commit message and a short plan; optional `AGENT_LLM_PROVIDER` (groq | openai), `GROQ_MODEL`, `OPENAI_MODEL`.

## ClickUp webhook setup

1. In ClickUp: **Settings** → **Apps** → **Webhooks** (or your workspace/settings).
2. Create a webhook:
   - **URL**: `https://your-server.com/webhook/clickup` (must be HTTPS for production).
   - **Events**: enable **Task Status Updated** (and optionally **Task Updated** if you want).
3. For local testing use a tunnel (e.g. ngrok): `ngrok http 8081`, then use the ngrok URL in the webhook.

## Telegram

1. Create a bot with [@BotFather](https://t.me/BotFather), get the token.
2. Add the bot to your channel or group.
3. Get the chat ID (e.g. use [getUpdates](https://core.telegram.org/bots/api#getupdates) or a helper bot; for channels it’s usually `-100xxxxxxxxxx`).

## Endpoints

- `GET /health` — Health check.
- `POST /webhook/clickup` — ClickUp webhook. Expects JSON body with `event`, `task_id`, `history_items`. Responds 200 quickly and runs the pipeline in the background.

## LLM (Groq / OpenAI)

If `GROQ_API_KEY` or `OPENAI_API_KEY` is set, the agent calls the LLM (Groq preferred when both are set) to generate a short plan and a conventional commit message. The response is stored in the placeholder file and the commit message is used for the git commit. Set `AGENT_LLM_PROVIDER=groq` or `=openai` to force a provider; optional `GROQ_MODEL` (default `llama-3.3-70b-versatile`), `OPENAI_MODEL` (default `gpt-4o-mini`).

To go further (e.g. have the LLM suggest file edits), edit `internal/agentflow/agent/agent.go`: use the LLM output to create or modify files under `repoPath`, then return the commit message in `RunResult`.
