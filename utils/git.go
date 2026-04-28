package utils

import (
	"bytes"
	"context"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitFileChange struct {
	Status Status
	File   string
}

type RefKind int

const (
	RefLocal  RefKind = iota
	RefRemote
	RefTag
)

type BranchInfo struct {
	Name    string  // short name (e.g. "main", "origin/main", "v1.0")
	Kind    RefKind // local branch, remote branch, or tag
	Commit  string  // short commit hash
	Author  string
	Date    string  // relative time
	Message string  // commit subject / tag message
}

func IsGitRepo(filePath string) bool {
	if _, err := exec.LookPath("git"); err != nil {
		log.Println("No git found in PATH, diff will be disabled.")
		return false
	}

	// Get the absolute path and directory
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		log.Printf("Failed to get absolute path: %v", err)
		return false
	}
	dir := filepath.Dir(absPath)

	// Run git diff
	cmd := BuildCmd(context.Background(), "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(output)) != "true" {
		if err != nil {
			log.Println("Failed to detect git repo: ", err)
		}
		return false
	}

	return true

}

func CurrentGitBranch(projectPath string) (string, error) {
	cmd := BuildCmd(context.Background(), "git", "branch", "--show-current")
	cmd.Dir = projectPath

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// CurrentCommitHash returns the short hash of HEAD (e.g. "4f3a2b1").
func CurrentCommitHash(projectPath string) (string, error) {
	cmd := BuildCmd(context.Background(), "git", "rev-parse", "--short", "HEAD")
	cmd.Dir = projectPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// HeadRefName returns a human-readable name for HEAD when in detached state.
// It tries to find a matching tag or remote branch; falls back to the short
// commit hash. Returns empty string if HEAD is on a local branch (not detached).
func HeadRefName(projectPath string) string {
	// git describe --all --exact-match returns the ref name if HEAD
	// exactly matches a tag or branch tip.
	cmd := BuildCmd(context.Background(), "git", "describe", "--all", "--exact-match", "HEAD")
	cmd.Dir = projectPath
	out, err := cmd.Output()
	if err == nil {
		name := strings.TrimSpace(string(out))
		// Strip prefixes: heads/ tags/ remotes/
		name = strings.TrimPrefix(name, "heads/")
		name = strings.TrimPrefix(name, "tags/")
		name = strings.TrimPrefix(name, "remotes/")
		return name
	}

	// Fall back to short hash.
	hash, err := CurrentCommitHash(projectPath)
	if err != nil {
		return ""
	}
	return hash
}

type Status int

const (
	StatusUnchanged Status = iota
	StatusUntracked
	StatusModified
	StatusStaged
	StatusStagedModified // Staged changes exist, AND further unstaged changes exist
	StatusAdded
	StatusDeleted
	StatusRenamed
	StatusConflict
)

// String representation for debugging or UI tooltips
func (s Status) String() string {
	return []string{
		"Unchanged", "Untracked", "Modified", "Staged",
		"Staged + Modified", "Added", "Deleted", "Renamed", "Conflict",
	}[s]
}

func mapCharsToStatus(x, y rune) Status {
	switch {
	// Untracked
	case x == '?' && y == '?':
		return StatusUntracked

	// Conflicts (U is common in merge conflicts)
	case x == 'U' || y == 'U' || (x == 'A' && y == 'A') || (x == 'D' && y == 'D'):
		return StatusConflict

	// Added to index
	case x == 'A' && y == ' ':
		return StatusAdded

	// Renamed
	case x == 'R':
		return StatusRenamed

	case x == ' ' && y == 'M':
		return StatusModified
	case x == 'M' && y == ' ':
		return StatusStaged
	case x == 'M' && y == 'M':
		return StatusStagedModified

	// Deletions
	case x == 'D' || y == 'D':
		return StatusDeleted

	default:
		return StatusUnchanged
	}
}

// GitFileStatus returns the status of the file.
func GitFileStatus(filePath string) Status {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return StatusUnchanged
	}

	dir := filepath.Dir(absPath)
	filename := filepath.Base(absPath)

	// --porcelain=v1 for stable output.
	cmd := BuildCmd(context.Background(), "git", "status", "--porcelain=v1", filename)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return StatusUnchanged
	}
	if len(out) < 2 {
		return StatusUnchanged
	}

	// X = Staged status, Y = Unstaged status
	x, y := rune(out[0]), rune(out[1])

	return mapCharsToStatus(x, y)
}

// GitRepoStatus returns the status of the repository.
func GitRepoStatus(projectDir string) []GitFileChange {
	// --porcelain=v1 for stable output.
	cmd := BuildCmd(context.Background(), "git", "status", "--porcelain=v1")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	entries := bytes.Split(out, []byte{'\n'})
	changes := make([]GitFileChange, 0)

	for _, entry := range entries {
		if len(entry) < 2 {
			continue
		}

		x := rune(entry[0])
		y := rune(entry[1])

		changes = append(changes, GitFileChange{
			Status: mapCharsToStatus(x, y),
			File:   string(entry[3:]),
		})
	}

	return changes
}

// ListGitBranches returns all local and remote branches with commit metadata,
// sorted by most recently committed.
func ListGitBranches(projectDir string) ([]BranchInfo, error) {
	cmd := BuildCmd(context.Background(), "git", "for-each-ref",
		"--sort=-committerdate",
		"--format=%(refname:short)|%(refname)|%(objectname:short)|%(authorname)|%(committerdate:relative)|%(subject)",
		"refs/heads", "refs/remotes", "refs/tags",
	)
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}

	lines := strings.Split(trimmed, "\n")
	branches := make([]BranchInfo, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 6)
		if len(parts) < 6 {
			continue
		}
		kind := RefLocal
		if strings.HasPrefix(parts[1], "refs/remotes/") {
			kind = RefRemote
		} else if strings.HasPrefix(parts[1], "refs/tags/") {
			kind = RefTag
		}
		branches = append(branches, BranchInfo{
			Name:    parts[0],
			Kind:    kind,
			Commit:  parts[2],
			Author:  parts[3],
			Date:    parts[4],
			Message: parts[5],
		})
	}

	return branches, nil
}

// SwitchGitBranch checks out the named branch.
func SwitchGitBranch(projectDir, branch string) error {
	cmd := BuildCmd(context.Background(), "git", "checkout", branch)
	cmd.Dir = projectDir
	return cmd.Run()
}
