# SADE First-Pass: Generate Architecture Documentation

You are initializing the architecture documentation for a software project using SADE (Software Architecture Documentation Engine).

## CRITICAL: Start with architecture.json FIRST

**Before creating any nodes/*.md files, you MUST create `.sade/architecture.json` first.**

The workflow must be:
1. **FIRST**: Create `.sade/architecture.json` with the complete graph structure (nodes + edges)
2. **SECOND**: Create `.sade/nodes/*.md` files for each node

This order is required because the architecture.json file is what SADE loads to determine if the first-pass completed successfully.

## Required Top-Level Nodes

Always create these top-level categories (skip any that genuinely don't apply):

| Node ID | Label | Purpose |
|---------|-------|---------|
| `root` | Project Root | Top-level overview, entry points, build config |
| `internals` | Core / Internals | Business logic, domain models, core packages |
| `ui` | UI & Frontend | Components, views, styles, client-side state |
| `infra` | Infrastructure | Build system, CI/CD, Docker, deployment, scripts |
| `config` | Configuration | Config files, environment handling, constants |

Within each top-level node, create child nodes for major subsystems. For example:
- `internals` → `internals/auth`, `internals/api`, `internals/database`
- `ui` → `ui/components`, `ui/stores`, `ui/pages`

## architecture.json Format

```json
{
  "version": "1.0",
  "nodes": [
    {
      "id": "root",
      "label": "Project Root",
      "description": "Top-level overview",
      "files": ["README.md", "package.json", "go.mod"]
    },
    {
      "id": "internals",
      "label": "Core / Internals",
      "description": "Core business logic and backend packages"
    },
    {
      "id": "internals/auth",
      "label": "Authentication",
      "description": "Login, session management, JWT",
      "parent": "internals",
      "files": ["src/auth/**"]
    }
  ],
  "edges": [
    { "source": "ui", "target": "internals", "type": "depends" },
    { "source": "internals/auth", "target": "config", "type": "depends" }
  ]
}
```

## nodes/*.md Format

For each node, create a markdown file named `<node-id>.md` (replace `/` with `-` in filenames):

```markdown
# <Node Label>

Brief description of this node's responsibilities.

## Overview

What this area of the codebase does, its purpose, and key design decisions.

## Key Files

List the most important files and what they do.

## Dependencies

What this node depends on and what depends on it.

## Files

- path/to/file1.go
- path/to/file2.svelte
- src/module/**
```

## Rules

1. Every file in the project should be mapped to exactly one node
2. Use glob patterns (`**`) for directories with many files of the same concern
3. Keep descriptions concise but informative — this is for developers being onboarded
4. Create edges for real dependencies (imports, API calls), not just proximity
5. Edge types: `depends` (uses), `contains` (parent-child), `calls` (runtime)
6. The `parent` field on a node means it's visually nested inside that parent in the graph
7. Write documentation that helps someone unfamiliar with the codebase understand the architecture quickly
