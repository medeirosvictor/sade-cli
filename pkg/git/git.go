// Package git provides git operations for change detection.
package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileStatus represents a file's git status
type FileStatus struct {
	Path    string `json:"path"`
	Status  string `json:"status"`
	OldPath string `json:"oldPath,omitempty"`
	Staged  bool   `json:"staged"`
}

// IsRepo checks whether the given directory is inside a git repository
func IsRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

// Root returns the root directory of the git repository
func Root(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Status returns the list of changed files
func Status(dir string) ([]FileStatus, error) {
	cmd := exec.Command("git", "status", "--porcelain", "-u")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []FileStatus

	for _, line := range lines {
		if len(line) < 4 {
			continue
		}

		indexStatus := line[0]
		workTreeStatus := line[1]
		path := strings.TrimSpace(line[3:])

		var status string
		var staged bool
		var oldPath string

		if strings.Contains(path, " -> ") {
			parts := strings.SplitN(path, " -> ", 2)
			oldPath = parts[0]
			path = parts[1]
		}

		switch {
		case indexStatus == '?' && workTreeStatus == '?':
			status = "??"
		case indexStatus == 'A':
			status, staged = "A", true
		case indexStatus == 'D':
			status, staged = "D", true
		case indexStatus == 'R':
			status, staged = "R", true
		case indexStatus == 'M':
			status, staged = "M", true
		case workTreeStatus == 'M':
			status = "M"
		case workTreeStatus == 'D':
			status = "D"
		case workTreeStatus == 'A':
			status = "A"
		default:
			status = strings.TrimSpace(string([]byte{indexStatus, workTreeStatus}))
		}

		files = append(files, FileStatus{
			Path:    filepath.ToSlash(path),
			Status:  status,
			OldPath: filepath.ToSlash(oldPath),
			Staged:  staged,
		})
	}

	return files, nil
}

// HasChanges returns true if there are any uncommitted changes
func HasChanges(dir string) (bool, error) {
	files, err := Status(dir)
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}

// DiffSummary returns a brief summary of changes for prompts
func DiffSummary(dir string) (string, error) {
	files, err := Status(dir)
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "No changes detected.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d file(s) changed:\n", len(files)))

	for _, f := range files {
		sb.WriteString(fmt.Sprintf("  %s %s\n", f.Status, f.Path))
	}

	return sb.String(), nil
}
