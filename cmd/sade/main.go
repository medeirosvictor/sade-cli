package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/medeirosvictor/sade-cli/internal/tui"
	"github.com/medeirosvictor/sade-cli/pkg/agent"
	"github.com/medeirosvictor/sade-cli/pkg/config"
	"github.com/medeirosvictor/sade-cli/pkg/git"
	"github.com/medeirosvictor/sade-cli/pkg/upkeep"
	"github.com/medeirosvictor/sade-cli/pkg/watcher"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	interactive := false
	for _, arg := range os.Args[2:] {
		if arg == "-i" || arg == "--interactive" {
			interactive = true
		}
	}

	switch cmd {
	case "start":
		cmdStart(interactive)
	case "status":
		cmdStatus()
	case "version", "--version", "-v":
		fmt.Printf("sade %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`sade - Software Architecture Documentation Engine

Usage:
  sade start         Start watching (scaffolds .sade/ if needed)
  sade start -i      Start with interactive TUI
  sade status        Show current configuration
  sade version       Show version

Examples:
  cd your-project && sade start
  sade start -i`)
}

func cmdStart(interactive bool) {
	cwd, err := os.Getwd()
	if err != nil {
		fatal("Could not get current directory: %v", err)
	}

	projectRoot := config.FindRoot(cwd)
	var cfg *config.Config

	// Scaffold if needed
	if projectRoot == "" {
		projectRoot = cwd
		cfg = ensureScaffolded(projectRoot)
	} else {
		cfg, err = config.Load(projectRoot)
		if err != nil {
			fatal("Could not load config: %v", err)
		}
	}

	// Ensure agent is selected
	if cfg.Agent == "" {
		cfg = ensureAgent(projectRoot, cfg)
	}

	// Verify agent is available
	if !agent.IsAvailable(cfg.Agent) {
		fmt.Printf("⚠ Agent '%s' is not available.\n\n", cfg.Agent)
		cfg = ensureAgent(projectRoot, cfg)
	}

	provider := agent.GetProvider(cfg.Agent)

	fmt.Println()
	fmt.Println("╭─────────────────────────────────────────╮")
	fmt.Println("│              SADE Watcher               │")
	fmt.Println("╰─────────────────────────────────────────╯")
	fmt.Println()
	fmt.Printf("  Project: %s\n", projectRoot)
	fmt.Printf("  Agent:   %s\n", provider.Name)
	fmt.Printf("  Pulse:   %dms\n", cfg.PulseMs)
	fmt.Printf("  Housekeeping: %dms\n", cfg.HousekeepingMs)
	fmt.Println()

	if interactive {
		fmt.Println("Starting interactive mode...")
		fmt.Println()
		runInteractive(projectRoot, cfg)
	} else {
		fmt.Println("Watching for changes... (Ctrl+C to stop)")
		fmt.Println()
		runWatchLoop(projectRoot, cfg)
	}
}

func ensureScaffolded(projectRoot string) *config.Config {
	fmt.Println("╭─────────────────────────────────────────╮")
	fmt.Println("│           SADE Initialization           │")
	fmt.Println("╰─────────────────────────────────────────╯")
	fmt.Println()

	if !git.IsRepo(projectRoot) {
		fmt.Println("⚠ Warning: Not a git repository. SADE works best with git.")
		fmt.Println()
	}

	cfg := config.Default()

	if err := config.EnsureDirs(projectRoot); err != nil {
		fatal("Could not create directories: %v", err)
	}

	if err := upkeep.EnsurePromptFiles(projectRoot); err != nil {
		fatal("Could not create prompt files: %v", err)
	}

	// Create README
	paths := config.GetPaths(projectRoot)
	readmePath := paths.SadeDir + "/README.md"
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
		os.WriteFile(readmePath, []byte(readme), 0644)
	}

	fmt.Println("✓ Created .sade/ directory")

	return cfg
}

func ensureAgent(projectRoot string, cfg *config.Config) *config.Config {
	fmt.Println("Scanning for coding agents...")
	agents := agent.Detect()

	if len(agents) == 0 {
		fmt.Println("\n⚠ No coding agents detected.")
		fmt.Println("SADE requires a coding agent CLI. Supported agents:")
		fmt.Println("  - pi (github.com/mariozechner/pi-coding-agent)")
		fmt.Println("  - claude (Anthropic Claude Code)")
		fmt.Println("  - codex (OpenAI Codex CLI)")
		fmt.Println("  - aider (github.com/paul-gauthier/aider)")
		fmt.Println("\nInstall one and run 'sade start' again.")
		os.Exit(1)
	}

	fmt.Printf("\nFound %d agent(s):\n\n", len(agents))
	for i, a := range agents {
		fmt.Printf("  [%d] %s (%s)\n", i+1, a.Name, a.Version)
	}

	var selected *agent.Detected
	if len(agents) == 1 {
		selected = &agents[0]
		fmt.Printf("\nUsing %s (only agent available)\n", selected.Name)
	} else {
		fmt.Print("\nSelect agent [1]: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		idx := 0
		if input != "" {
			var err error
			idx, err = strconv.Atoi(input)
			if err != nil || idx < 1 || idx > len(agents) {
				fatal("Invalid selection")
			}
			idx--
		}
		selected = &agents[idx]
	}

	cfg.Agent = selected.ID

	if err := config.Save(projectRoot, cfg); err != nil {
		fatal("Could not save config: %v", err)
	}

	fmt.Printf("✓ Configured agent: %s\n", selected.Name)

	return cfg
}

func runInteractive(projectRoot string, cfg *config.Config) {
	w, err := watcher.New(100)
	if err != nil {
		fatal("Could not create watcher: %v", err)
	}

	if err := w.Watch(projectRoot); err != nil {
		fatal("Could not watch directory: %v", err)
	}

	w.Run()

	model := tui.NewModel(projectRoot, cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	go runEventLoop(w, p, cfg, projectRoot)

	if _, err := p.Run(); err != nil {
		fatal("TUI error: %v", err)
	}

	w.Close()
}

func runWatchLoop(projectRoot string, cfg *config.Config) {
	w, err := watcher.New(100)
	if err != nil {
		fatal("Could not create watcher: %v", err)
	}
	defer w.Close()

	if err := w.Watch(projectRoot); err != nil {
		fatal("Could not watch directory: %v", err)
	}

	w.Run()

	runner := agent.NewRunner()

	var lastEvent time.Time
	pulseTimer := time.NewTimer(time.Hour)
	pulseTimer.Stop()

	housekeepingTicker := time.NewTicker(time.Duration(cfg.HousekeepingMs) * time.Millisecond)
	defer housekeepingTicker.Stop()

	changesSinceUpkeep := false

	for {
		select {
		case event, ok := <-w.Events():
			if !ok {
				return
			}

			if strings.Contains(event.Path, "/.sade/") {
				continue
			}

			lastEvent = time.Now()
			changesSinceUpkeep = true

			fmt.Printf("  %s %s %s\n", eventIcon(event.Type), event.Type, relativePath(projectRoot, event.Path))

			if cfg.PulseEnabled {
				pulseTimer.Reset(time.Duration(cfg.PulseMs) * time.Millisecond)
			}

		case <-pulseTimer.C:
			if !cfg.PulseEnabled || runner.IsRunning() {
				continue
			}

			if time.Since(lastEvent) < time.Duration(cfg.PulseMs)*time.Millisecond {
				continue
			}

			fmt.Println()
			fmt.Println("⚡ Pulse: triggering documentation update...")

			go runPulse(runner, cfg.Agent, projectRoot)

		case <-housekeepingTicker.C:
			if !cfg.HousekeepingEnabled || runner.IsRunning() {
				continue
			}

			if !changesSinceUpkeep {
				continue
			}

			fmt.Println()
			fmt.Println("🧹 Housekeeping: running periodic maintenance...")
			changesSinceUpkeep = false

			go runHousekeeping(runner, cfg.Agent, projectRoot)
		}
	}
}

func runEventLoop(w *watcher.Watcher, p *tea.Program, cfg *config.Config, projectRoot string) {
	runner := agent.NewRunner()
	var lastEvent time.Time
	pulseTimer := time.NewTimer(time.Hour)
	pulseTimer.Stop()

	housekeepingTicker := time.NewTicker(time.Duration(cfg.HousekeepingMs) * time.Millisecond)
	defer housekeepingTicker.Stop()

	changesSinceUpkeep := false

	for {
		select {
		case event, ok := <-w.Events():
			if !ok {
				return
			}

			if strings.Contains(event.Path, "/.sade/") {
				continue
			}

			lastEvent = time.Now()
			changesSinceUpkeep = true

			p.Send(tui.FileEventMsg{Event: event})

			if cfg.PulseEnabled {
				pulseTimer.Reset(time.Duration(cfg.PulseMs) * time.Millisecond)
			}

		case <-pulseTimer.C:
			if !cfg.PulseEnabled || runner.IsRunning() {
				continue
			}

			if time.Since(lastEvent) < time.Duration(cfg.PulseMs)*time.Millisecond {
				continue
			}

			go runPulse(runner, cfg.Agent, projectRoot)

		case <-housekeepingTicker.C:
			if !cfg.HousekeepingEnabled || runner.IsRunning() {
				continue
			}

			if !changesSinceUpkeep {
				continue
			}

			changesSinceUpkeep = false
			go runHousekeeping(runner, cfg.Agent, projectRoot)
		}
	}
}

func runPulse(runner *agent.Runner, agentID, projectRoot string) {
	paths := config.GetPaths(projectRoot)

	ctx, err := upkeep.BuildPulseContext(projectRoot)
	if err != nil {
		fmt.Printf("  ⚠ Could not build context: %v\n", err)
		return
	}

	promptFile, err := upkeep.WriteTempPrompt(projectRoot, paths.PulsePrompt, ctx)
	if err != nil {
		fmt.Printf("  ⚠ Could not write prompt: %v\n", err)
		return
	}

	result, err := runner.Invoke(agentID, projectRoot, promptFile)
	if err != nil {
		fmt.Printf("  ⚠ Agent error: %v\n", err)
		return
	}

	if result.ExitCode == 0 {
		fmt.Println("  ✓ Pulse complete")
	} else {
		fmt.Printf("  ⚠ Agent exited with code %d\n", result.ExitCode)
	}

	upkeep.CleanupTempPrompts(projectRoot)
}

func runHousekeeping(runner *agent.Runner, agentID, projectRoot string) {
	paths := config.GetPaths(projectRoot)

	ctx, err := upkeep.BuildHousekeepContext(projectRoot)
	if err != nil {
		fmt.Printf("  ⚠ Could not build context: %v\n", err)
		return
	}

	promptFile, err := upkeep.WriteTempPrompt(projectRoot, paths.HousekeepPrompt, ctx)
	if err != nil {
		fmt.Printf("  ⚠ Could not write prompt: %v\n", err)
		return
	}

	result, err := runner.Invoke(agentID, projectRoot, promptFile)
	if err != nil {
		fmt.Printf("  ⚠ Agent error: %v\n", err)
		return
	}

	if result.ExitCode == 0 {
		fmt.Println("  ✓ Housekeeping complete")
	} else {
		fmt.Printf("  ⚠ Agent exited with code %d\n", result.ExitCode)
	}

	upkeep.CleanupTempPrompts(projectRoot)
}

func cmdStatus() {
	cwd, err := os.Getwd()
	if err != nil {
		fatal("Could not get current directory: %v", err)
	}

	projectRoot := config.FindRoot(cwd)
	if projectRoot == "" {
		fmt.Println("No .sade/ directory found. Run 'sade start' to initialize.")
		return
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		fatal("Could not load config: %v", err)
	}

	fmt.Println("╭─────────────────────────────────────────╮")
	fmt.Println("│              SADE Status                │")
	fmt.Println("╰─────────────────────────────────────────╯")
	fmt.Println()
	fmt.Printf("  Project:      %s\n", projectRoot)
	fmt.Printf("  Agent:        %s\n", cfg.Agent)
	fmt.Printf("  Pulse:        %dms (%s)\n", cfg.PulseMs, enabledStr(cfg.PulseEnabled))
	fmt.Printf("  Housekeeping: %dms (%s)\n", cfg.HousekeepingMs, enabledStr(cfg.HousekeepingEnabled))
	fmt.Println()

	if cfg.Agent != "" {
		if agent.IsAvailable(cfg.Agent) {
			fmt.Printf("  ✓ Agent '%s' is available\n", cfg.Agent)
		} else {
			fmt.Printf("  ⚠ Agent '%s' is NOT available\n", cfg.Agent)
		}
	}

	paths := config.GetPaths(projectRoot)
	nodes, _ := os.ReadDir(paths.Nodes)
	nodeCount := 0
	for _, n := range nodes {
		if !n.IsDir() && strings.HasSuffix(n.Name(), ".md") {
			nodeCount++
		}
	}
	fmt.Printf("  Nodes:        %d\n", nodeCount)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

func eventIcon(t watcher.EventType) string {
	switch t {
	case watcher.Created:
		return "+"
	case watcher.Modified:
		return "~"
	case watcher.Deleted:
		return "-"
	default:
		return "?"
	}
}

func relativePath(base, path string) string {
	if strings.HasPrefix(path, base) {
		rel := strings.TrimPrefix(path, base)
		return strings.TrimPrefix(rel, "/")
	}
	return path
}

func enabledStr(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
