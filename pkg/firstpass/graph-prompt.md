# SADE First-Pass: Generate Architecture Graph

You are completing the architecture documentation for this project using SADE.

## Your Task

Read the existing `.sade/nodes/*.md` files and the project's file tree, then generate `.sade/architecture.json`.

## Steps

1. Read all files in `.sade/nodes/` to understand the documented architecture
2. Read the project's file tree to verify file mappings
3. Create `.sade/architecture.json` with the complete graph structure

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
      "id": "internals-auth",
      "label": "Authentication",
      "description": "Login, session management, JWT",
      "parent": "internals",
      "files": ["src/auth/**"]
    }
  ],
  "edges": [
    { "source": "ui", "target": "internals", "type": "depends" },
    { "source": "internals-auth", "target": "config", "type": "depends" }
  ]
}
```

## Rules

1. Every node in the nodes/*.md files must appear in architecture.json
2. The `id` field must match the node filename (without `.md`)
3. The `parent` field should reflect nesting (e.g., `internals-auth` has parent `internals`)
4. Create edges for real dependencies (imports, API calls), not just proximity
5. Edge types: `depends` (uses), `contains` (parent-child), `calls` (runtime)
6. Every file in the project should be mapped to exactly one node via the `files` array
7. Use glob patterns (`**`) for directories with many files of the same concern
8. Do NOT modify the nodes/*.md files — only create architecture.json
