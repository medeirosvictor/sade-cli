# SADE First-Pass: Generate Node Documentation

You are initializing architecture documentation for this project using SADE.

## Your Task

Generate `.sade/nodes/*.md` files that document the project's architecture. **Do NOT create architecture.json** — that comes in a separate step.

## Steps

1. Read the project's README, file tree, and source files to understand the structure
2. Identify the major responsibility areas (nodes)
3. Create a `.sade/nodes/` directory if it doesn't exist
4. Write one markdown file per node

## Required Top-Level Nodes

Always create these (skip any that genuinely don't apply):

| Node ID | Label | Purpose |
|---------|-------|---------|
| `root` | Project Root | Top-level overview, entry points, build config |
| `internals` | Core / Internals | Business logic, domain models, core packages |
| `ui` | UI & Frontend | Components, views, styles, client-side state |
| `infra` | Infrastructure | Build system, CI/CD, Docker, deployment, scripts |
| `config` | Configuration | Config files, environment handling, constants |

Within each top-level node, create child nodes for major subsystems. For example:
- `internals` → `internals-auth`, `internals-api`, `internals-database`
- `ui` → `ui-components`, `ui-stores`, `ui-pages`

## Node File Format

Each file is `.sade/nodes/<node-id>.md`:

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
4. The node ID in the filename uses `-` instead of `/` (e.g., `internals-auth.md`)
5. Write documentation that helps someone unfamiliar with the codebase understand the architecture quickly
6. Do NOT create architecture.json — only create nodes/*.md files
