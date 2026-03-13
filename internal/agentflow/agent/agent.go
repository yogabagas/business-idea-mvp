package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yogabagas/business-idea-mvp/internal/agentflow/llm"
)

type RunResult struct {
	CommitMessage string
}

type Runner struct {
	llmBaseURL string
	llmAPIKey  string
	llmModel   string
	commonRepo string
}

func NewRunner(llmBaseURL, llmAPIKey, llmModel, commonRepoPath string) *Runner {
	return &Runner{
		llmBaseURL: llmBaseURL,
		llmAPIKey:  llmAPIKey,
		llmModel:   llmModel,
		commonRepo: commonRepoPath,
	}
}

// Run executes the agent: scans the repo, asks the LLM to plan changes,
// reads relevant files, asks the LLM to generate code, then applies changes.
func (r *Runner) Run(ctx context.Context, repoPath, taskID, taskName, taskDescription string) (*RunResult, error) {
	if r.llmBaseURL == "" || r.llmAPIKey == "" {
		return r.runPlaceholder(repoPath, taskID, taskName, taskDescription)
	}

	var commonContext string
	if r.commonRepo != "" {
		commonContext = readMarkdownFiles(r.commonRepo)
	}

	// Step 1: Build file tree of the target repo
	tree := buildFileTree(repoPath, 4)
	log.Printf("[agent] repo tree (%d entries):\n%s", strings.Count(tree, "\n"), tree)

	// Step 2: Ask LLM to plan: which files to read, modify, create
	plan, err := r.planChanges(ctx, taskName, taskDescription, tree, commonContext)
	if err != nil {
		return nil, fmt.Errorf("plan changes: %w", err)
	}
	log.Printf("[agent] plan:\n%s", plan)

	filesToRead := parseFileList(plan, "READ:")
	filesToModify := parseFileList(plan, "MODIFY:")
	filesToCreate := parseFileList(plan, "CREATE:")

	// Merge read + modify lists for context
	contextFiles := uniqueStrings(append(filesToRead, filesToModify...))
	fileContents := readRepoFiles(repoPath, contextFiles)

	// Step 3: Ask LLM to generate code
	codeOutput, err := r.generateCode(ctx, taskName, taskDescription, plan, fileContents, filesToModify, filesToCreate)
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}
	log.Printf("[agent] code generation done, parsing output...")

	// Step 4: Parse and apply file changes
	changes := parseFileChanges(codeOutput)
	if len(changes) == 0 {
		return nil, fmt.Errorf("LLM produced no file changes")
	}
	for path, content := range changes {
		absPath := filepath.Join(repoPath, path)
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return nil, fmt.Errorf("mkdir for %s: %w", path, err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("write %s: %w", path, err)
		}
		log.Printf("[agent] wrote %s (%d bytes)", path, len(content))
	}

	commitMsg := parseCommitMessage(codeOutput)
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("feat: %s [%s]", truncate(taskName, 50), taskID)
	}

	return &RunResult{CommitMessage: commitMsg}, nil
}

// runPlaceholder is the fallback when no LLM is configured.
func (r *Runner) runPlaceholder(repoPath, taskID, taskName, taskDescription string) (*RunResult, error) {
	content := fmt.Sprintf("Task: %s\nTask ID: %s\n\nDescription:\n%s\n", taskName, taskID, taskDescription)
	path := filepath.Join(repoPath, ".agentflow-placeholder-"+sanitize(taskID)+".txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, err
	}
	return &RunResult{CommitMessage: fmt.Sprintf("chore: agent placeholder for %s", taskID)}, nil
}

func (r *Runner) planChanges(ctx context.Context, taskName, taskDesc, fileTree, commonCtx string) (string, error) {
	system := `You are a senior software engineer. Given a task and a repository file tree, produce a plan.

Your response MUST use this exact format:

PLAN:
- bullet 1
- bullet 2

READ:
- path/to/file1.go
- path/to/file2.go

MODIFY:
- path/to/existing_file.go

CREATE:
- path/to/new_file.go

COMMIT: feat(scope): short description

Rules:
- READ: existing files you need to see to understand the codebase (max 10).
- MODIFY: existing files you will change.
- CREATE: new files that don't exist yet.
- Every path must be relative to the repo root (as shown in the tree).
- Only list files that are relevant to the task. Be precise.`

	user := fmt.Sprintf("## Task\n%s\n\n## Description\n%s\n\n## Repository File Tree\n```\n%s\n```", taskName, taskDesc, fileTree)
	if commonCtx != "" {
		user += "\n\n## Additional Context (from memory repo)\n" + truncate(commonCtx, 2000)
	}

	msgs := []llm.ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}
	return llm.Chat(ctx, r.llmBaseURL, r.llmAPIKey, r.llmModel, msgs, llm.ChatOpts{MaxTokens: 2048})
}

func (r *Runner) generateCode(ctx context.Context, taskName, taskDesc, plan, fileContents string, toModify, toCreate []string) (string, error) {
	system := `You are a senior software engineer. Generate the actual code changes for the given task.

CRITICAL RULES:
1. Output EVERY file that needs to be written (modified or created) using this exact format:

>>>>>> FILE: path/to/file.go
package main

// full file content here...
<<<<<< END FILE

2. After all files, output the commit message:

COMMIT: feat(scope): short description

3. Output the COMPLETE file content for each file — not diffs, not partial snippets.
4. Maintain the existing code style, imports, and conventions.
5. Only output files that actually change or are new. Do not output unchanged files.
6. Make sure the code compiles and is correct.`

	user := fmt.Sprintf("## Task\n%s\n\n## Description\n%s\n\n## Plan\n%s\n", taskName, taskDesc, plan)

	if fileContents != "" {
		user += "\n## Existing Files\n" + fileContents
	}

	if len(toModify) > 0 {
		user += "\n## Files to Modify\n- " + strings.Join(toModify, "\n- ")
	}
	if len(toCreate) > 0 {
		user += "\n## Files to Create\n- " + strings.Join(toCreate, "\n- ")
	}

	msgs := []llm.ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}
	return llm.Chat(ctx, r.llmBaseURL, r.llmAPIKey, r.llmModel, msgs, llm.ChatOpts{MaxTokens: 16384})
}

// --- Parsing helpers ---

func parseFileList(plan, header string) []string {
	var files []string
	inSection := false
	for _, line := range strings.Split(plan, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmed), header) {
			inSection = true
			after := strings.TrimSpace(trimmed[len(header):])
			if after != "" {
				files = append(files, cleanPath(after))
			}
			continue
		}
		if inSection {
			if trimmed == "" || (!strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "*")) {
				inSection = false
				continue
			}
			p := strings.TrimLeft(trimmed, "-* ")
			p = strings.TrimSpace(p)
			// Strip backticks that LLMs sometimes add
			p = strings.Trim(p, "`")
			if p != "" {
				files = append(files, cleanPath(p))
			}
		}
	}
	return files
}

func parseFileChanges(output string) map[string]string {
	changes := make(map[string]string)
	lines := strings.Split(output, "\n")
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, ">>>>>> FILE:") {
			filePath := cleanPath(strings.TrimSpace(trimmed[len(">>>>>> FILE:"):]))
			if filePath == "" {
				continue
			}
			var contentLines []string
			i++
			for i < len(lines) {
				if strings.TrimSpace(lines[i]) == "<<<<<< END FILE" {
					break
				}
				contentLines = append(contentLines, lines[i])
			}
			content := strings.Join(contentLines, "\n")
			// Trim a leading/trailing code fence if the LLM wrapped it
			content = trimCodeFence(content)
			changes[filePath] = content
		}
	}
	return changes
}

func parseCommitMessage(output string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmed), "COMMIT:") {
			return strings.TrimSpace(trimmed[7:])
		}
	}
	return ""
}

func trimCodeFence(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) >= 2 {
		first := strings.TrimSpace(lines[0])
		if strings.HasPrefix(first, "```") {
			lines = lines[1:]
		}
		last := strings.TrimSpace(lines[len(lines)-1])
		if last == "```" {
			lines = lines[:len(lines)-1]
		}
	}
	return strings.Join(lines, "\n")
}

// --- Repo scanning helpers ---

func buildFileTree(root string, maxDepth int) string {
	var lines []string
	_ = walkDir(root, root, 0, maxDepth, &lines)
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "__pycache__": true,
	".idea": true, ".vscode": true, "dist": true, "build": true,
	".agentflow-workspace": true, "agentflow-workspace": true,
}

func walkDir(root, dir string, depth, maxDepth int, lines *[]string) error {
	if depth > maxDepth {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".agentflow-placeholder") {
			continue
		}
		rel, _ := filepath.Rel(root, filepath.Join(dir, name))
		if e.IsDir() {
			if skipDirs[name] {
				continue
			}
			*lines = append(*lines, rel+"/")
			_ = walkDir(root, filepath.Join(dir, name), depth+1, maxDepth, lines)
		} else {
			*lines = append(*lines, rel)
		}
	}
	return nil
}

func readRepoFiles(root string, paths []string) string {
	var parts []string
	for _, p := range paths {
		absPath := filepath.Join(root, p)
		data, err := os.ReadFile(absPath)
		if err != nil {
			log.Printf("[agent] skip unreadable file: %s (%v)", p, err)
			continue
		}
		content := string(data)
		if len(content) > 15000 {
			content = content[:15000] + "\n... (truncated)"
		}
		parts = append(parts, fmt.Sprintf("### %s\n```\n%s\n```\n", p, content))
	}
	return strings.Join(parts, "\n")
}

func readMarkdownFiles(dir string) string {
	var out []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			b, _ := os.ReadFile(path)
			out = append(out, string(b))
		}
		return nil
	})
	return strings.Join(out, "\n\n---\n\n")
}

// --- Utilities ---

func uniqueStrings(ss []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func cleanPath(p string) string {
	p = strings.Trim(p, "`\"' ")
	p = filepath.Clean(p)
	return p
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, " ", "-")
	if len(s) > 32 {
		s = s[:32]
	}
	return s
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
