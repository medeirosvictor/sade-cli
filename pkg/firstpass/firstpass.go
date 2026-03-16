// Package firstpass handles the initial architecture generation for a project.
// It builds a prompt from project context (README, file tree), invokes the
// configured coding agent, and synthesizes architecture.json from the results.
//
// Both the CLI (`sade start`) and the GUI (`sade-app`) import this package so
// the first-pass behaviour is identical regardless of entry point.
package firstpass

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/medeirosvictor/sade-cli/pkg/agent"
	"github.com/medeirosvictor/sade-cli/pkg/arch"
	"github.com/medeirosvictor/sade-cli/pkg/config"
)

// ProgressFunc is called with status messages during the first-pass run.
// Callers can wire this to a TUI, Wails event, or stdout logger.
type ProgressFunc func(kind, message string)

// Options controls first-pass behaviour.
type Options struct {
	// ProjectRoot is the absolute path of the project.
	ProjectRoot string

	// AgentID is the coding agent to use (e.g. "pi", "claude").
	AgentID string

	// Progress receives structured status updates.
	// kind is "stdout" or "stderr". May be nil.
	Progress ProgressFunc
}

func (o *Options) emit(kind, msg string) {
	if o.Progress != nil {
		o.Progress(kind, msg)
	}
}

// NeedsFirstPass returns true when the project is initialised (.sade/ exists)
// but the architecture graph is empty (no nodes in architecture.json AND no
// node markdown files in .sade/nodes/).
func NeedsFirstPass(projectRoot string) bool {
	a, err := arch.LoadArch(projectRoot)
	if err != nil {
		return true
	}
	if len(a.Nodes) > 0 {
		return false
	}

	nodes, err := arch.LoadAllNodes(projectRoot)
	if err != nil {
		return true
	}
	return len(nodes) == 0
}

// BuildPrompt assembles the full first-pass prompt (system instructions +
// project context). The returned string can be previewed in a UI before
// running, or written straight to a temp file for the agent.
func BuildPrompt(projectRoot string) (string, error) {
	// Read existing README if available for context
	readmeContent := ""
	for _, name := range []string{"README.md", "readme.md", "Readme.md"} {
		data, err := os.ReadFile(filepath.Join(projectRoot, name))
		if err == nil {
			readmeContent = string(data)
			break
		}
	}

	// Get a quick file tree for context (top-level + one level deep)
	tree := BuildTextTree(projectRoot, 2)

	var sb strings.Builder
	sb.WriteString(SystemPrompt)
	sb.WriteString("\n\n---\n\n## Project Context\n\n")
	sb.WriteString(fmt.Sprintf("**Project root:** `%s`\n\n", filepath.Base(projectRoot)))

	if readmeContent != "" {
		sb.WriteString("### README.md\n\n")
		// Truncate very long READMEs
		if len(readmeContent) > 4000 {
			readmeContent = readmeContent[:4000] + "\n\n... (truncated)"
		}
		sb.WriteString(readmeContent)
		sb.WriteString("\n\n")
	}

	sb.WriteString("### File Tree\n\n```\n")
	sb.WriteString(tree)
	sb.WriteString("```\n")

	return sb.String(), nil
}

// Run executes the first-pass architecture generation. It builds the prompt,
// invokes the coding agent, reloads the architecture, and — if needed —
// synthesizes architecture.json from node markdown files.
//
// Returns the resulting Architecture on success.
func Run(opts Options) (*arch.Architecture, error) {
	if opts.AgentID == "" {
		return nil, fmt.Errorf("no coding agent configured")
	}
	if !agent.IsAvailable(opts.AgentID) {
		return nil, fmt.Errorf("configured agent '%s' is not available", opts.AgentID)
	}

	// Build the full prompt (system prompt + project context)
	prompt, err := BuildPrompt(opts.ProjectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Write to temp file for agent consumption
	promptFile, err := writeTempPrompt(opts.ProjectRoot, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to write prompt file: %w", err)
	}

	opts.emit("stdout", fmt.Sprintf("Running first-pass documentation with %s…", opts.AgentID))

	// Invoke the agent
	runner := agent.NewRunner()
	result, err := runner.Invoke(opts.AgentID, opts.ProjectRoot, promptFile)
	if err != nil {
		opts.emit("stderr", fmt.Sprintf("Agent error: %v", err))
		return nil, fmt.Errorf("agent invocation failed: %w", err)
	}

	if result.ExitCode != 0 {
		errMsg := result.Stderr
		if errMsg == "" {
			errMsg = fmt.Sprintf("exit code %d", result.ExitCode)
		}
		opts.emit("stderr", fmt.Sprintf("Agent finished with error: %s", errMsg))
		return nil, fmt.Errorf("agent exited with code %d: %s", result.ExitCode, errMsg)
	}

	opts.emit("stdout", "First-pass complete. Reloading architecture…")

	// Reload the architecture that the agent should have created
	architecture, err := arch.LoadArch(opts.ProjectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to reload architecture: %w", err)
	}

	// Fallback: if agent created nodes/*.md but not architecture.json,
	// synthesize architecture.json from the node files.
	if len(architecture.Nodes) == 0 {
		opts.emit("stdout", "No architecture.json found — synthesizing from node files…")
		architecture, err = SynthesizeArchitecture(opts.ProjectRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to synthesize architecture: %w", err)
		}
		opts.emit("stdout", fmt.Sprintf("Synthesized architecture.json with %d nodes.", len(architecture.Nodes)))
	}

	// Clean up temp prompt
	os.Remove(promptFile)

	return architecture, nil
}

// SynthesizeArchitecture builds architecture.json from existing .sade/nodes/*.md
// files. This is the fallback when the agent creates node docs but forgets the
// graph JSON. Also useful for manual rebuilds.
func SynthesizeArchitecture(projectRoot string) (*arch.Architecture, error) {
	nodes, err := arch.LoadAllNodes(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load nodes: %w", err)
	}
	if len(nodes) == 0 {
		return arch.Empty(), nil
	}

	architecture := &arch.Architecture{
		Version: "1.0",
		Nodes:   []arch.Node{},
		Edges:   []arch.Edge{},
	}

	// Build a set of node IDs for parent resolution
	nodeIDs := make(map[string]bool)
	for _, n := range nodes {
		nodeIDs[n.ID] = true
	}

	for _, n := range nodes {
		node := arch.Node{
			ID:          n.ID,
			Label:       n.ID, // Default label = ID
			Description: n.Description,
			Locked:      n.Locked,
			Files:       n.Files,
		}

		// Try to extract a better label from the first markdown heading
		lines := strings.Split(n.Raw, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "# ") {
				node.Label = strings.TrimPrefix(trimmed, "# ")
				break
			}
		}

		// Infer parent from ID if it contains a dash (e.g. "ui-components" → parent "ui")
		if strings.Contains(n.ID, "-") {
			parts := strings.SplitN(n.ID, "-", 2)
			parentID := parts[0]
			if nodeIDs[parentID] && parentID != n.ID {
				node.Parent = &parentID
			}
		}

		architecture.Nodes = append(architecture.Nodes, node)
	}

	// Save
	if err := arch.SaveArch(projectRoot, architecture); err != nil {
		return nil, fmt.Errorf("failed to save synthesized architecture: %w", err)
	}

	return architecture, nil
}

// --- Helpers ----------------------------------------------------------------

// writeTempPrompt writes the combined prompt to .sade/.tmp/first-pass-prompt.md.
func writeTempPrompt(projectRoot, content string) (string, error) {
	paths := config.GetPaths(projectRoot)
	tmpDir := filepath.Join(paths.SadeDir, ".tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	promptFile := filepath.Join(tmpDir, "first-pass-prompt.md")
	if err := os.WriteFile(promptFile, []byte(content), 0644); err != nil {
		return "", err
	}
	return promptFile, nil
}

// BuildTextTree returns a plain-text representation of the directory tree
// for inclusion in prompts sent to coding agents.
func BuildTextTree(root string, maxDepth int) string {
	var sb strings.Builder
	walkTextTree(&sb, root, "", maxDepth, 0)
	return sb.String()
}

func walkTextTree(sb *strings.Builder, dir, prefix string, maxDepth, depth int) {
	if depth >= maxDepth {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// Filter out hidden/ignored
	var visible []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		if name == "" || name[0] == '.' {
			continue
		}
		if isPromptTreeIgnored(name) {
			continue
		}
		visible = append(visible, entry)
	}

	for i, entry := range visible {
		isLast := i == len(visible)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		sb.WriteString(prefix + connector + name + "\n")

		if entry.IsDir() {
			childPrefix := prefix + "│   "
			if isLast {
				childPrefix = prefix + "    "
			}
			walkTextTree(sb, filepath.Join(dir, entry.Name()), childPrefix, maxDepth, depth+1)
		}
	}
}

func isPromptTreeIgnored(name string) bool {
	switch name {
	case "node_modules", "vendor", "__pycache__", "target", "dist", "build",
		".git", ".sade", ".next", ".nuxt", ".cache", "coverage":
		return true
	}
	return false
}
