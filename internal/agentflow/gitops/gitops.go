package gitops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repo prepares a branch in a work dir: clone (or pull), checkout base, create branch, return repo path.
// Caller can then modify files and call CommitPush.
type Repo struct {
	workDir    string
	repoPath   string
	baseBranch string
	cloneURL   string
	token      string // optional, for private repos
}

func NewRepo(workDir, cloneURL, baseBranch, token string) *Repo {
	return &Repo{
		workDir:    workDir,
		baseBranch: baseBranch,
		cloneURL:   cloneURL,
		token:      token,
	}
}

// Prepare clones (or pulls) the repo, checks out baseBranch, creates and checks out branchName.
// Returns the absolute path to the repo and any error.
func (r *Repo) Prepare(ctx context.Context, branchName string) (repoPath string, err error) {
	if r.cloneURL == "" {
		return "", fmt.Errorf("gitops: TARGET_REPO_URL not set")
	}
	// Use task branch name safe for git
	branchName = strings.ReplaceAll(branchName, " ", "-")
	dirName := filepath.Base(strings.TrimSuffix(strings.TrimSuffix(r.cloneURL, "/"), ".git"))
	r.repoPath = filepath.Join(r.workDir, dirName)
	_ = os.MkdirAll(r.workDir, 0755)
	cloneURL := r.cloneURL
	if r.token != "" {
		// Insert token for HTTPS: https://TOKEN@github.com/owner/repo.git
		cloneURL = injectTokenIntoURL(cloneURL, r.token)
	}
	if _, err := os.Stat(filepath.Join(r.repoPath, ".git")); err != nil {
		if os.IsNotExist(err) {
			if err := runGit(ctx, r.workDir, "clone", "--", cloneURL, dirName); err != nil {
				return "", fmt.Errorf("git clone: %w", err)
			}
		} else {
			return "", err
		}
	} else {
		if err := runGit(ctx, r.repoPath, "fetch", "origin", r.baseBranch); err != nil {
			return "", fmt.Errorf("git fetch: %w", err)
		}
	}
	if err := runGit(ctx, r.repoPath, "checkout", "-B", branchName, "origin/"+r.baseBranch); err != nil {
		return "", fmt.Errorf("git checkout: %w", err)
	}
	return r.repoPath, nil
}

// CommitPush commits all changes with message and pushes to origin branchName.
func (r *Repo) CommitPush(ctx context.Context, branchName, commitMessage string) error {
	if r.repoPath == "" {
		return fmt.Errorf("gitops: Prepare not called")
	}
	branchName = strings.ReplaceAll(branchName, " ", "-")
	if err := runGit(ctx, r.repoPath, "add", "-A"); err != nil {
		return err
	}
	if err := runGit(ctx, r.repoPath, "commit", "-m", commitMessage); err != nil {
		// nothing to commit is possible
		if strings.Contains(err.Error(), "nothing to commit") {
			return nil
		}
		return err
	}
	if err := runGit(ctx, r.repoPath, "push", "--force-with-lease", "-u", "origin", branchName); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	return nil
}

func runGit(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}

func injectTokenIntoURL(rawURL, token string) string {
	if strings.HasPrefix(rawURL, "https://") {
		return "https://" + token + "@" + rawURL[8:]
	}
	if strings.HasPrefix(rawURL, "http://") {
		return "http://" + token + "@" + rawURL[7:]
	}
	return rawURL
}
