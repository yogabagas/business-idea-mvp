# ClickUp task description template (AI-friendly)

Use this structure in your ClickUp task description so the Agent Flow knows **which repo** to use and the AI agent knows **what to do** and **what to avoid**.

Copy the template below into the task description and fill the sections. The pipeline parses **Repository** to choose the repo; the rest is for the AI (and humans) to understand scope and constraints.

---

## Template (copy below)

```markdown
## Repository
repo: https://github.com/owner/repo.git
<!-- Or: github: owner/repo | gitlab: group/repo | repository: owner/repo -->

## Goals (Definition of done)
- [ ] Goal 1: clear, measurable outcome.
- [ ] Goal 2: another outcome.
- [ ] Goal 3: e.g. "All new code has unit tests."

## What to do
- Step or requirement 1.
- Step or requirement 2.
- Reference existing patterns (e.g. "Follow the same structure as X.").

## Rules / What to avoid
- Do not change Y; only touch Z.
- Do not add new dependencies without reason.
- Follow project style: lint/format X, naming Y.
- Other hard constraints or conventions.

## Context (optional)
- Link to doc, ticket, or "See common-repo/ARCHITECTURE.md."
- Relevant background for the AI.
```

---

## Section guide

### Repository (required for dynamic repo)

**Purpose:** Tells the pipeline which Git repo to clone and where to open the PR/MR.

**Use exactly one of these** (on its own line, with the key as shown):

| Format | Example | When to use |
|--------|---------|-------------|
| Full URL | `repo: https://github.com/owner/repo.git` | Any GitHub repo |
| Full URL | `repo: https://gitlab.com/group/repo.git` | Any GitLab repo |
| Short GitHub | `github: owner/repo` | GitHub; pipeline uses your `GITHUB_TOKEN` |
| Short GitLab | `gitlab: group/repo` | GitLab; pipeline uses your `GITLAB_TOKEN` |
| Generic | `repository: owner/repo` | Uses `GIT_PROVIDER` from env (github or gitlab) |

If you omit this block, the app uses the default repo from env (`TARGET_REPO_URL` / `GITHUB_REPO` / `GITLAB_PROJECT`).

---

### Goals (Definition of done)

**Purpose:** Gives the AI a clear checklist of outcomes. Each item should be verifiable.

**Good examples:**
- "Add endpoint `GET /api/users/:id` returning user JSON."
- "New components use the design tokens from `theme.ts`."
- "All new functions have a short doc comment and a unit test."

**Avoid:** Vague goals like "Improve the code" or "Make it better."

---

### What to do

**Purpose:** Concrete steps, requirements, or references so the AI knows how to implement.

**Good examples:**
- "Add validation: email format, password min 8 chars."
- "Reuse the pattern from `services/auth.go` for the new service."
- "Update the README section 'Running locally' with the new env var."

---

### Rules / What to avoid

**Purpose:** Constraints so the AI stays within project rules and doesn’t do unwanted things.

**Good examples:**
- "Do not modify `internal/legacy/`; only add under `internal/v2/`."
- "Do not add new npm packages; use existing utils only."
- "Use `go fmt` and existing linter config; no new lint rules."
- "Do not change API request/response shapes; only add optional fields."
- "Keep all new code in English (comments, logs, errors)."

---

### Context (optional)

**Purpose:** Links or short notes (docs, ADRs, other tickets) so the AI has background.

**Examples:**
- "See docs/API-CONVENTIONS.md for error format."
- "Related: ClickUp task ABC-123 (already merged)."
- "The common-repo is cloned for context; prefer patterns from there."

---

## Full example

```markdown
## Repository
github: myorg/backend-api

## Goals (Definition of done)
- [ ] New endpoint GET /api/v2/products/:id returns product JSON or 404.
- [ ] Response shape matches docs/API.md#product.
- [ ] New code has unit tests; coverage does not drop.

## What to do
- Add handler in `internal/handlers/products.go` following the pattern of `GetUser`.
- Use `svc.ProductByID(ctx, id)` from existing service; do not add DB calls in the handler.
- Return 404 when product not found; use helper `respondNotFound(w)`.

## Rules / What to avoid
- Do not change existing endpoints or request/response fields.
- Do not add new dependencies.
- Use existing logger; no new log format.
- Keep handler under 30 lines; put logic in service layer.

## Context
- API conventions: docs/API-CONVENTIONS.md.
- Product model: internal/models/product.go.
```

---

## Tips for AI readability

1. **One repo per task** — Use a single Repository line so the pipeline and AI are aligned.
2. **Order matters for the parser** — Put the Repository block near the top so the repo line is found quickly.
3. **Be specific** — File paths, function names, and "same as X" references help the AI.
4. **Explicit don’ts** — "Do not X" is easier for the AI than inferring from "only do Y."
5. **Checklist for done** — Goals as `- [ ]` make it clear when the task is complete.
