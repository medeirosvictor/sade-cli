// Package agent handles coding agent detection and invocation.
package agent

import (
	"bytes"
	"fmt"
	"os"
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
  mu       sync.Mutex
  running  bool
}

// IsRunning returns true if the agent is currently executing a task.
func (a *Agent) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

func NewAgent(providerID, projectRoot string) (*Agent, error) {
	if providerID == "" {
		return nil, fmt.Errorf("providerID cannot be empty")
	}
	if projectRoot == "" {
		return nil, fmt.Errorf("projectRoot cannot be empty")
	}

	provider := GetProvider(providerID)
	if provider == nil {
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}

	return &Agent {
		Provider: *provider,
		Root: projectRoot,
	}, nil
}

func (a *Agent) Run(task Task) (*Result, error) {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return nil, fmt.Errorf("agent already running")
	}
	a.running = true
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}()

	tmpFile, err := os.CreateTemp("", "sade-task-*.md")
	if err != nil {
		return nil, fmt.Errorf("creating temp prompt: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(task.Prompt); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("writing prompt: %w", err)
	}
	tmpFile.Close()

	args := append([]string{}, a.Provider.Args...)
	switch a.Provider.ID {
	case "pi", "claude":
		args = append(args, "@"+tmpFile.Name())
	case "aider":
		args = append(args, "--message-file", tmpFile.Name())
	default:
		args = append(args, tmpFile.Name())
	}

	cmd := exec.Command(a.Provider.Cmd, args...)
	cmd.Dir = a.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	result := &Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Err = err
			return result, fmt.Errorf("running %s: %w\nstdout: %s\nstderr: %s", a.Provider.Name, err, result.Stdout, result.Stderr)
		}
	}

	if task.Validate != nil {
		if err := task.Validate(); err != nil {
			return result, fmt.Errorf("task %q validation failed: %w\nstdout: %s\nstderr: %s", task.Name, err, result.Stdout, result.Stderr)
		}
	}

	return result, nil
}

var providers = []Provider{
	{
		ID:    "pi",
		Name:  "pi",
		Cmd:   "pi",
		Check: "--version",
		Args:  []string{"-p"},
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

