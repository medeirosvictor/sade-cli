// Package scaffold handles the creation of the .sade/ directory structure.
// Both the CLI (`sade start`) and the GUI (`sade-app`) call Init to ensure
// a project has the required scaffolding before watching or first-pass.
package scaffold

import (
	"os"
	"path/filepath"

	"github.com/medeirosvictor/sade-cli/pkg/config"
	"github.com/medeirosvictor/sade-cli/pkg/upkeep"
)

// Init ensures the full .sade/ directory structure exists. It is idempotent —
// calling it on an already-scaffolded project is a no-op for each item.
//
// NOTE: We intentionally do NOT create architecture.json here.
// The first-pass step will create it (or the agent will).
// This allows us to detect whether first-pass has run.
//
// Steps:
//  1. Create .sade/ and .sade/nodes/ directories
//  2. Write default config.json (with agentID if non-empty)
//  3. Create pulse-prompt.md and housekeeping-prompt.md
//  4. Create README.md
//
// Returns the loaded (or newly created) Config.
func Init(projectRoot, agentID string) (*config.Config, error) {
	// 1. Create directory structure
	if err := config.EnsureDirs(projectRoot); err != nil {
		return nil, err
	}

	// 2. Write / update config.json
	cfg, err := config.Load(projectRoot)
	if err != nil {
		cfg = config.Default()
	}
	if agentID != "" {
		cfg.Agent = agentID
	}
	if err := config.Save(projectRoot, cfg); err != nil {
		return nil, err
	}

	// 3. Create default prompt files
	if err := upkeep.EnsurePromptFiles(projectRoot); err != nil {
		return nil, err
	}

	// 4. Create README.md (if missing)
	// Note: We intentionally do NOT create architecture.json here.
	// The first-pass agent will create it.
	paths := config.GetPaths(projectRoot)
	readmePath := filepath.Join(paths.SadeDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		readme := `# .sade/

SADE architectural documentation.

## Files

- config.json — agent + timing settings
- architecture.json — graph structure (nodes + edges)
- nodes/*.md — documentation per responsibility
- pulse-prompt.md — instructions for reactive updates
- housekeeping-prompt.md — instructions for periodic maintenance

## Locks

- Add {locked} at the top of any nodes/*.md to protect it
- Set "locked": true on nodes/edges in architecture.json

## Commands

    sade start      # background mode
    sade start -i   # interactive TUI
    sade status     # show config
`
		if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
