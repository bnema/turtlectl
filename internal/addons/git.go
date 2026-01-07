package addons

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var (
	ErrNotGitRepo      = errors.New("not a git repository")
	ErrFFNotPossible   = errors.New("fast-forward not possible, local changes exist")
	ErrNoRemote        = errors.New("no remote configured")
	ErrAlreadyUpToDate = errors.New("already up to date")
)

// CloneRepo clones a git repository to the specified path
// progressWriter can be nil to disable progress output
func CloneRepo(url, destPath string, progressWriter io.Writer) error {
	_, err := git.PlainClone(destPath, false, &git.CloneOptions{
		URL:      url,
		Progress: progressWriter,
		Depth:    0, // Full clone for updates to work
	})

	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// UpdateRepo performs a fast-forward update on a git repository
// progressWriter can be nil to disable progress output
func UpdateRepo(repoPath string, progressWriter io.Writer) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNotGitRepo, err)
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Check for local modifications
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if !status.IsClean() {
		return ErrFFNotPossible
	}

	// Fetch from origin
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Progress:   progressWriter,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// Get current branch reference
	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get the remote tracking branch
	branchName := head.Name().Short()
	remoteRef := plumbing.NewRemoteReferenceName("origin", branchName)

	remoteRefObj, err := repo.Reference(remoteRef, true)
	if err != nil {
		// Try common default branches
		for _, defaultBranch := range []string{"main", "master"} {
			remoteRef = plumbing.NewRemoteReferenceName("origin", defaultBranch)
			remoteRefObj, err = repo.Reference(remoteRef, true)
			if err == nil {
				break
			}
		}
		if err != nil {
			return fmt.Errorf("failed to find remote branch: %w", err)
		}
	}

	// Check if we're already up to date
	if head.Hash() == remoteRefObj.Hash() {
		return ErrAlreadyUpToDate
	}

	// Perform fast-forward by resetting to remote
	err = worktree.Reset(&git.ResetOptions{
		Commit: remoteRefObj.Hash(),
		Mode:   git.HardReset,
	})
	if err != nil {
		return fmt.Errorf("failed to fast-forward: %w", err)
	}

	return nil
}

// IsGitRepo checks if a directory is a git repository
func IsGitRepo(path string) bool {
	_, err := git.PlainOpen(path)
	return err == nil
}

// GetRepoRemoteURL gets the origin remote URL of a git repository
func GetRepoRemoteURL(repoPath string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", ErrNotGitRepo
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		return "", ErrNoRemote
	}

	urls := remote.Config().URLs
	if len(urls) == 0 {
		return "", ErrNoRemote
	}

	return urls[0], nil
}

// GetCurrentCommit returns the current HEAD commit hash
func GetCurrentCommit(repoPath string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", ErrNotGitRepo
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	return head.Hash().String()[:8], nil
}

// ExtractRepoName extracts the repository name from a git URL
func ExtractRepoName(gitURL string) string {
	// Remove .git suffix
	name := strings.TrimSuffix(gitURL, ".git")

	// Get the last path component
	parts := strings.Split(name, "/")
	if len(parts) > 0 {
		name = parts[len(parts)-1]
	}

	// Remove common suffixes like -master, -main
	for _, suffix := range []string{"-master", "-main", "-trunk"} {
		name = strings.TrimSuffix(name, suffix)
	}

	return name
}

// ValidateGitURL checks if a string looks like a valid git URL
func ValidateGitURL(url string) error {
	url = strings.ToLower(url)

	// Check for common git URL patterns
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "git://") {
		return nil
	}

	return fmt.Errorf("invalid git URL: must start with https://, git@, or git://")
}

// NormalizeGitURL ensures the URL ends with .git
func NormalizeGitURL(url string) string {
	if !strings.HasSuffix(url, ".git") {
		return url + ".git"
	}
	return url
}

// CleanupFailedClone removes a partially cloned directory
func CleanupFailedClone(path string) error {
	// Check if directory exists and is likely a failed clone
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	// Check if it's a valid git repo
	if IsGitRepo(path) {
		return nil // Don't remove valid repos
	}

	return os.RemoveAll(path)
}

// CheckForUpdates checks if a repository has updates available without applying them
// Returns true if updates are available, false if up to date
func CheckForUpdates(repoPath string) (bool, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrNotGitRepo, err)
	}

	// Fetch from origin (updates remote refs without changing local)
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return false, fmt.Errorf("failed to fetch: %w", err)
	}

	// Get current HEAD
	head, err := repo.Head()
	if err != nil {
		return false, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get the remote tracking branch
	branchName := head.Name().Short()
	remoteRef := plumbing.NewRemoteReferenceName("origin", branchName)

	remoteRefObj, err := repo.Reference(remoteRef, true)
	if err != nil {
		// Try common default branches
		for _, defaultBranch := range []string{"main", "master"} {
			remoteRef = plumbing.NewRemoteReferenceName("origin", defaultBranch)
			remoteRefObj, err = repo.Reference(remoteRef, true)
			if err == nil {
				break
			}
		}
		if err != nil {
			return false, fmt.Errorf("failed to find remote branch: %w", err)
		}
	}

	// Compare hashes
	return head.Hash() != remoteRefObj.Hash(), nil
}

// VerifyRepoIntegrity checks if a git repository is valid and not corrupted
func VerifyRepoIntegrity(repoPath string) error {
	// Check .git directory exists
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("missing .git directory")
	}

	// Try to open the repo
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("corrupted repository: %w", err)
	}

	// Try to get HEAD
	_, err = repo.Head()
	if err != nil {
		return fmt.Errorf("corrupted HEAD: %w", err)
	}

	return nil
}
