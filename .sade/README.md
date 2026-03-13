# .sade/

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
