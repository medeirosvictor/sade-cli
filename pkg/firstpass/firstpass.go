// Package firstpass handles the initial architecture generation for a project.
// It builds a prompt from project context (README, file tree), invokes the
// configured coding agent, and synthesizes architecture.json from the results.
//
// Both the CLI (`sade start`) and the GUI (`sade-app`) import this package so
// the first-pass behaviour is identical regardless of entry point.
package firstpass

import (
	"fmt"
	"github.com/medeirosvictor/sade-cli/pkg/agent"
	"github.com/medeirosvictor/sade-cli/pkg/arch"
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
	return !arch.ArchExists(projectRoot) || !arch.HasNodes(projectRoot)
}

// Run executes the first-pass architecture generation. It builds the prompt,
// invokes the coding agent, reloads the architecture, and — if needed —
// synthesizes architecture.json from node markdown files.
//
// Returns the resulting Architecture on success.
func Run(opts Options) (*arch.Architecture, error) {
	ag, err := agent.NewAgent(opts.AgentID, opts.ProjectRoot)
	if err != nil { return nil, err }

	opts.emit("stdout", "Generating node documentation...")
	_, err = ag.Run(agent.Task{
		Name: "generate-nodes",
		Prompt: NodesPrompt,
		Validate: func() error {
			if !arch.HasNodes(opts.ProjectRoot) {
				return fmt.Errorf("no nodes were generated")
			}
			return nil
		},
	})

	if err != nil { return nil, fmt.Errorf("nodes task: %w", err)}

	opts.emit("stdout", "Generating architecture graph")
	_, err = ag.Run(agent.Task{
		Name: "generate-graph",
		Prompt: GraphPrompt,
		Validate: func() error {
			if !arch.ArchExists(opts.ProjectRoot) {
				return fmt.Errorf("no architecture.json created")
			}
			return nil
		},
	})
	if err != nil { return nil, fmt.Errorf("graph task: %w", err)}

	return arch.LoadArch(opts.ProjectRoot)
}

