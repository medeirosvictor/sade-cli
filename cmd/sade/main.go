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
	"github.com/medeirosvictor/sade-cli/pkg/firstpass"
	"github.com/medeirosvictor/sade-cli/pkg/git"
	"github.com/medeirosvictor/sade-cli/pkg/scaffold"
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

	// ── Step 1: Scaffold ──────────────────────────────────────────────
	if projectRoot == "" {
		projectRoot = cwd

		if !git.IsRepo(projectRoot) {
			fmt.Println("⚠ Warning: Not a git repository. SADE works best with git.")
			fmt.Println()
		}

		fmt.Println("╭─────────────────────────────────────────╮")
		fmt.Println("│           SADE Initialization           │")
		fmt.Println("╰─────────────────────────────────────────╯")
		fmt.Println()

		cfg, err = scaffold.Init(projectRoot, "")
		if err != nil {
			fatal("Scaffolding failed: %v", err)
		}
		fmt.Println("✓ Created .sade/ directory")
	} else {
		cfg, err = config.Load(projectRoot)
		if err != nil {
			fatal("Could not load config: %v", err)
		}
	}

	// ── Step 2: Agent selection ───────────────────────────────────────
	if cfg.Agent == "" {
		cfg = ensureAgent(projectRoot, cfg)
	}

	// Verify agent is available
	if !agent.IsAvailable(cfg.Agent) {
		fmt.Printf("⚠ Agent '%s' is not available.\n\n", cfg.Agent)
		cfg = ensureAgent(projectRoot, cfg)
	}

	// ── Step 3: First-pass if architecture is empty ───────────────────
	if firstpass.NeedsFirstPass(projectRoot) {
		fmt.Println()
		fmt.Println("No architecture documentation found. Running first-pass…")
		fmt.Println()

		architecture, err := firstpass.Run(firstpass.Options{
			ProjectRoot: projectRoot,
			AgentID:     cfg.Agent,
			Progress: func(kind, msg string) {
				if kind == "stderr" {
					fmt.Printf("  ⚠ %s\n", msg)
				} else {
					fmt.Printf("  %s\n", msg)
				}
			},
		})
		if err != nil {
			fmt.Printf("\n⚠ First-pass failed: %v\n", err)
			fmt.Println("  You can run 'sade start' again later to retry.")
			fmt.Println("  Continuing to watch mode…")
			fmt.Println()
		} else {
			fmt.Printf("\n✓ First-pass complete: %d nodes, %d edges\n\n",
				len(architecture.Nodes), len(architecture.Edges))
		}
	}

	// ── Step 4: Watch ─────────────────────────────────────────────────
	provider := agent.GetProvider(cfg.Agent)

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

	ag, err := agent.NewAgent(cfg.Agent, projectRoot)
	if err != nil {
		fatal("Could not create agent: %v", err)
	}

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
			if !cfg.PulseEnabled || ag.IsRunning() {
				continue
			}

			if time.Since(lastEvent) < time.Duration(cfg.PulseMs)*time.Millisecond {
				continue
			}

			fmt.Println()
			fmt.Println("⚡ Pulse: triggering documentation update...")

			go runPulse(ag, projectRoot)

		case <-housekeepingTicker.C:
			if !cfg.HousekeepingEnabled || ag.IsRunning() {
				continue
			}

			if !changesSinceUpkeep {
				continue
			}

			fmt.Println()
			fmt.Println("🧹 Housekeeping: running periodic maintenance...")
			changesSinceUpkeep = false

			go runHousekeeping(ag, projectRoot)
		}
	}
}

func runEventLoop(w *watcher.Watcher, p *tea.Program, cfg *config.Config, projectRoot string) {
	ag, err := agent.NewAgent(cfg.Agent, projectRoot)
	if err != nil {
		return // can't create agent, stop event loop
	}

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
			if !cfg.PulseEnabled || ag.IsRunning() {
				continue
			}

			if time.Since(lastEvent) < time.Duration(cfg.PulseMs)*time.Millisecond {
				continue
			}

			go runPulse(ag, projectRoot)

		case <-housekeepingTicker.C:
			if !cfg.HousekeepingEnabled || ag.IsRunning() {
				continue
			}

			if !changesSinceUpkeep {
				continue
			}

			changesSinceUpkeep = false
			go runHousekeeping(ag, projectRoot)
		}
	}
}

func runPulse(ag *agent.Agent, projectRoot string) {
	paths := config.GetPaths(projectRoot)

	ctx, err := upkeep.BuildPulseContext(projectRoot)
	if err != nil {
		fmt.Printf("  ⚠ Could not build context: %v\n", err)
		return
	}

	promptBytes, err := os.ReadFile(paths.PulsePrompt)
	if err != nil {
		fmt.Printf("  ⚠ Could not read pulse prompt: %v\n", err)
		return
	}
	prompt := string(promptBytes) + "\n\n---\n\n" + ctx

	_, err = ag.Run(agent.Task{
		Name:   "pulse",
		Prompt: prompt,
	})
	if err != nil {
		fmt.Printf("  ⚠ Pulse error: %v\n", err)
		return
	}

	fmt.Println("  ✓ Pulse complete")
}

func runHousekeeping(ag *agent.Agent, projectRoot string) {
	paths := config.GetPaths(projectRoot)

	ctx, err := upkeep.BuildHousekeepContext(projectRoot)
	if err != nil {
		fmt.Printf("  ⚠ Could not build context: %v\n", err)
		return
	}

	promptBytes, err := os.ReadFile(paths.HousekeepPrompt)
	if err != nil {
		fmt.Printf("  ⚠ Could not read housekeeping prompt: %v\n", err)
		return
	}
	prompt := string(promptBytes) + "\n\n---\n\n" + ctx

	_, err = ag.Run(agent.Task{
		Name:   "housekeeping",
		Prompt: prompt,
	})
	if err != nil {
		fmt.Printf("  ⚠ Housekeeping error: %v\n", err)
		return
	}

	fmt.Println("  ✓ Housekeeping complete")
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
