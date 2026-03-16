# AGENTS.md

**sade-cli** — Watches your codebase and maintains architectural documentation using coding agents.

## Structure

```
sade-cli/
├── cmd/sade/main.go       # CLI entrypoint
├── pkg/                   # PUBLIC (importable by sade-app)
│   ├── agent/             # Agent detection + invocation
│   ├── arch/              # architecture.json + nodes/*.md
│   ├── config/            # .sade/config.json
│   ├── firstpass/         # First-pass architecture generation (shared)
│   ├── git/               # Git status + diff
│   ├── scaffold/          # .sade/ directory scaffolding (shared)
│   ├── upkeep/            # Pulse + housekeeping prompts
│   └── watcher/           # File system watching
└── internal/              # CLI-specific (not importable)
    └── tui/               # Interactive TUI
```

## Commands

```
sade start       # Scaffold → agent selection → first-pass (if empty) → watch
sade start -i    # Same but with interactive TUI
sade status      # Show config
```

## .sade/ Structure

```
.sade/
├── config.json              # agent + timings
├── architecture.json        # nodes + edges (graph)
├── nodes/*.md               # documentation
├── pulse-prompt.md          # reactive update instructions
├── housekeeping-prompt.md   # periodic maintenance instructions
├── apis/                    # (future)
└── tests/                   # (future)
```

## Development

```bash
go build -o sade ./cmd/sade
./sade start -i
```
