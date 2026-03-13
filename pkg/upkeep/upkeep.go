// Package upkeep handles pulse and housekeeping prompt generation.
package upkeep

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/medeirosvictor/sade-cli/pkg/arch"
	"github.com/medeirosvictor/sade-cli/pkg/config"
	"github.com/medeirosvictor/sade-cli/pkg/git"
)

// EnsurePromptFiles creates default prompt files if they don't exist
func EnsurePromptFiles(projectRoot string) error {
	paths := config.GetPaths(projectRoot)

	if _, err := os.Stat(paths.PulsePrompt); os.IsNotExist(err) {
		if err := os.WriteFile(paths.PulsePrompt, []byte(defaultPulsePrompt), 0644); err != nil {
			return err
		}
	}

	if _, err := os.Stat(paths.HousekeepPrompt); os.IsNotExist(err) {
		if err := os.WriteFile(paths.HousekeepPrompt, []byte(defaultHousekeepPrompt), 0644); err != nil {
			return err
		}
	}

	return nil
}

// BuildPulseContext generates context for reactive updates
func BuildPulseContext(projectRoot string) (string, error) {
	var sb strings.Builder

	// Git changes
	diffSummary, err := git.DiffSummary(projectRoot)
	if err != nil {
		diffSummary = "Could not get git status."
	}

	sb.WriteString("## Recent Changes\n\n")
	sb.WriteString(diffSummary)
	sb.WriteString("\n")

	// Locked nodes
	locked, err := arch.LockedNodeIDs(projectRoot)
	if err == nil && len(locked) > 0 {
		sb.WriteString("\n## Locked Nodes (do not modify)\n\n")
		for _, id := range locked {
			sb.WriteString(fmt.Sprintf("- %s\n", id))
		}
	}

	return sb.String(), nil
}

// BuildHousekeepContext generates context for periodic maintenance
func BuildHousekeepContext(projectRoot string) (string, error) {
	var sb strings.Builder

	// Load architecture
	a, err := arch.LoadArch(projectRoot)
	if err != nil {
		return "", err
	}

	sb.WriteString("## Current State\n\n")
	sb.WriteString(fmt.Sprintf("- Nodes: %d\n", len(a.Nodes)))
	sb.WriteString(fmt.Sprintf("- Edges: %d\n", len(a.Edges)))
	sb.WriteString(fmt.Sprintf("- Last updated: %s\n", formatTime(a.LastUpdated)))

	// Locked nodes
	locked, err := arch.LockedNodeIDs(projectRoot)
	if err == nil && len(locked) > 0 {
		sb.WriteString("\n## Locked Nodes (do not modify)\n\n")
		for _, id := range locked {
			sb.WriteString(fmt.Sprintf("- %s\n", id))
		}
	}

	// List all nodes
	nodes, err := arch.LoadAllNodes(projectRoot)
	if err == nil && len(nodes) > 0 {
		sb.WriteString("\n## Existing Nodes\n\n")
		for _, n := range nodes {
			lockMark := ""
			if n.Locked {
				lockMark = " 🔒"
			}
			sb.WriteString(fmt.Sprintf("- %s%s\n", n.ID, lockMark))
		}
	}

	return sb.String(), nil
}

// WriteTempPrompt combines prompt template + context and writes to temp file
func WriteTempPrompt(projectRoot, promptFile, context string) (string, error) {
	promptContent, err := os.ReadFile(promptFile)
	if err != nil {
		return "", fmt.Errorf("could not read prompt file %s: %w", promptFile, err)
	}

	combined := string(promptContent) + "\n\n---\n\n" + context

	tmpDir := filepath.Join(projectRoot, ".sade", ".tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("prompt-%d.md", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, []byte(combined), 0644); err != nil {
		return "", err
	}

	return tmpFile, nil
}

// CleanupTempPrompts removes old temp prompt files (older than 1 hour)
func CleanupTempPrompts(projectRoot string) {
	tmpDir := filepath.Join(projectRoot, ".sade", ".tmp")
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-1 * time.Hour)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(tmpDir, entry.Name()))
		}
	}
}

func formatTime(t *time.Time) string {
	if t == nil {
		return "never"
	}
	return t.Format("2006-01-02 15:04:05")
}

var defaultPulsePrompt = `# SADE Pulse Update

You are maintaining the architectural documentation for this project.

Recent file changes have been detected. Your task is to:

1. Review the changes listed below
2. Update the relevant .sade/nodes/*.md files to reflect any new responsibilities
3. Update .sade/architecture.json if the graph structure changed
4. Do NOT modify nodes marked as {locked} or with locked: true

Keep updates minimal and focused. Only change what's necessary to keep docs accurate.

`

var defaultHousekeepPrompt = `# SADE Housekeeping

You are performing periodic maintenance on the architectural documentation.

Your task is to:

1. Review the current state of .sade/nodes/*.md and .sade/architecture.json
2. Ensure all nodes accurately describe their responsibilities
3. Check for orphaned files not mapped to any node
4. Check for empty nodes with no files
5. Preserve and do NOT modify nodes marked as {locked} or with locked: true

Be conservative. This is maintenance, not restructuring.

`
