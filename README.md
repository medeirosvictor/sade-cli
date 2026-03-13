# sade-cli

Watches your codebase and maintains architectural documentation using coding agents.

## Install

```bash
go install github.com/medeirosvictor/sade-cli/cmd/sade@latest
```

Or build from source:

```bash
git clone https://github.com/medeirosvictor/sade-cli
cd sade-cli
go build -o sade ./cmd/sade
```

## Usage

```bash
cd your-project
sade start
```

That's it. On first run, SADE will:
1. Create `.sade/` directory
2. Detect available coding agents (pi, claude, codex, aider)
3. Start watching for changes

For interactive TUI:

```bash
sade start -i
```

## How It Works

**Pulse**: After file changes settle (default 15s), asks the agent to update documentation.

**Housekeeping**: Every 30 minutes, runs maintenance to check consistency.

## Configuration

`.sade/config.json`:

```json
{
  "agent": "pi",
  "pulse_ms": 15000,
  "housekeeping_ms": 1800000,
  "pulse_enabled": true,
  "housekeeping_enabled": true
}
```

## Prompts

Customize agent behavior:

- `.sade/pulse-prompt.md` — reactive updates
- `.sade/housekeeping-prompt.md` — periodic maintenance

## Locks

Prevent modification:

- Add `{locked}` at the top of any `.sade/nodes/*.md`
- Set `"locked": true` on nodes/edges in `architecture.json`

## Commands

```
sade start       Start (scaffolds if needed)
sade start -i    Start with TUI
sade status      Show config
sade version     Show version
```

## Using as a Library

```go
import (
    "github.com/medeirosvictor/sade-cli/pkg/watcher"
    "github.com/medeirosvictor/sade-cli/pkg/arch"
    "github.com/medeirosvictor/sade-cli/pkg/config"
)
```

## Development

For development, add this wrapper to `~/.zshrc` to auto-rebuild on every invocation:

```bash
# SADE CLI (auto-rebuild wrapper)
sade() {
  (cd /path/to/sade-cli && go build -o sade ./cmd/sade 2>&1 && ./sade "$@")
}
```

Then `source ~/.zshrc` and run `sade start` anywhere.

Once stable, switch to `go install ./cmd/sade` for a static binary.
