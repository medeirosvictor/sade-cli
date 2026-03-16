// Package agent handles coding agent detection and invocation.
package agent

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type Task struct {
  Name     string
  Prompt   string
  Validate func() error
}

// Provider defines a coding agent CLI
type Provider struct {
	ID    string
	Name  string
	Cmd   string
	Check string   // Arg to check version (e.g., "--version")
	Args  []string // Base args for non-interactive mode
}

type Agent struct {
  Provider Provider
  Root     string
}



var providers = []Provider{
	{
		ID:    "pi",
		Name:  "pi",
		Cmd:   "pi",
		Check: "--version",
		Args:  []string{"-p", "--append-system-prompt"},
	},
	{
		ID:    "claude",
		Name:  "Claude Code",
		Cmd:   "claude",
		Check: "--version",
		Args:  []string{"--print", "--append-system-prompt"},
	},
	{
		ID:    "codex",
		Name:  "Codex CLI",
		Cmd:   "codex",
		Check: "--version",
		Args:  []string{},
	},
	{
		ID:    "opencode",
		Name:  "OpenCode",
		Cmd:   "opencode",
		Check: "--version",
		Args:  []string{},
	},
	{
		ID:    "gemini",
		Name:  "Gemini CLI",
		Cmd:   "gemini",
		Check: "--version",
		Args:  []string{},
	},
	{
		ID:    "aider",
		Name:  "Aider",
		Cmd:   "aider",
		Check: "--version",
		Args:  []string{"--yes", "--no-auto-commits"},
	},
}

// Detected holds info about an available agent
type Detected struct {
	ID      string
	Name    string
	Version string
}

// Detect scans for available coding agents
func Detect() []Detected {
	var detected []Detected

	for _, p := range providers {
		path, err := exec.LookPath(p.Cmd)
		if err != nil {
			continue
		}

		cmd := exec.Command(path, p.Check)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		version := strings.TrimSpace(string(output))
		if version == "" {
			version = "installed"
		}
		if len(version) > 50 {
			version = version[:47] + "..."
		}

		detected = append(detected, Detected{
			ID:      p.ID,
			Name:    p.Name,
			Version: version,
		})
	}

	return detected
}

// GetProvider returns a provider by ID
func GetProvider(id string) *Provider {
	for _, p := range providers {
		if p.ID == id {
			return &p
		}
	}
	return nil
}

// IsAvailable checks if an agent is installed
func IsAvailable(id string) bool {
	p := GetProvider(id)
	if p == nil {
		return false
	}
	_, err := exec.LookPath(p.Cmd)
	return err == nil
}

// Result holds the result of an agent invocation
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

// Runner manages agent invocations
type Runner struct {
	mu        sync.Mutex
	running   bool
	currentID string
}

// NewRunner creates a new agent runner
func NewRunner() *Runner {
	return &Runner{}
}

// IsRunning returns true if an agent is currently running
func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Invoke runs the agent with the given prompt file
func (r *Runner) Invoke(agentID, projectRoot, promptFile string) (*Result, error) {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return nil, fmt.Errorf("agent already running: %s", r.currentID)
	}
	r.running = true
	r.currentID = agentID
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.running = false
		r.currentID = ""
		r.mu.Unlock()
	}()

	provider := GetProvider(agentID)
	if provider == nil {
		return nil, fmt.Errorf("unknown agent: %s", agentID)
	}

	args := append([]string{}, provider.Args...)

	switch agentID {
	case "pi", "claude":
		args = append(args, "@"+promptFile)
	case "aider":
		args = append(args, "--message-file", promptFile)
	default:
		args = append(args, promptFile)
	}

	cmd := exec.Command(provider.Cmd, args...)
	cmd.Dir = projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Err = err
		}
	}

	return result, nil
}
