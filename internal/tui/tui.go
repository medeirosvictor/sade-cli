package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/medeirosvictor/sade-cli/pkg/arch"
	"github.com/medeirosvictor/sade-cli/pkg/config"
	"github.com/medeirosvictor/sade-cli/pkg/watcher"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12"))

	nodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("12")).
			Bold(true)

	lockedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	eventStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
)

// FileEventMsg wraps a file event for the TUI
type FileEventMsg struct {
	Event watcher.Event
}

// TickMsg is sent periodically to refresh the view
type TickMsg time.Time

// Model is the bubbletea model for the TUI
type Model struct {
	projectRoot string
	cfg         *config.Config
	nodes       []*arch.NodeDoc
	selected    int
	events      []string
	agentStatus string
	width       int
	height      int
	quitting    bool
}

// NewModel creates a new TUI model
func NewModel(projectRoot string, cfg *config.Config) Model {
	nodes, _ := arch.LoadAllNodes(projectRoot)

	return Model{
		projectRoot: projectRoot,
		cfg:         cfg,
		nodes:       nodes,
		selected:    0,
		events:      []string{},
		agentStatus: "idle",
		width:       80,
		height:      24,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.nodes)-1 {
				m.selected++
			}
		case "r":
			nodes, _ := arch.LoadAllNodes(m.projectRoot)
			m.nodes = nodes
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case FileEventMsg:
		relPath := strings.TrimPrefix(msg.Event.Path, m.projectRoot+"/")
		eventStr := fmt.Sprintf("%s %s", eventIcon(msg.Event.Type), relPath)
		m.events = append([]string{eventStr}, m.events...)
		if len(m.events) > 10 {
			m.events = m.events[:10]
		}

	case TickMsg:
		nodes, _ := arch.LoadAllNodes(m.projectRoot)
		m.nodes = nodes
		return m, tickCmd()
	}

	return m, nil
}

// View implements tea.Model
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder

	// Header
	sb.WriteString(titleStyle.Render("SADE Watch"))
	sb.WriteString(dimStyle.Render(fmt.Sprintf("  %s", m.projectRoot)))
	sb.WriteString("\n\n")

	// Status line
	sb.WriteString(statusStyle.Render(fmt.Sprintf("Agent: %s  |  Status: %s  |  Nodes: %d",
		m.cfg.Agent, m.agentStatus, len(m.nodes))))
	sb.WriteString("\n\n")

	// Nodes list
	sb.WriteString(titleStyle.Render("Nodes"))
	sb.WriteString("\n")

	if len(m.nodes) == 0 {
		sb.WriteString(dimStyle.Render("  (no nodes yet)"))
		sb.WriteString("\n")
	} else {
		for i, node := range m.nodes {
			prefix := "  "
			style := nodeStyle

			if i == m.selected {
				prefix = "▸ "
				style = selectedStyle
			}

			line := fmt.Sprintf("%s%s", prefix, node.ID)
			if node.Locked {
				line += " 🔒"
			}
			if len(node.Files) > 0 {
				line += dimStyle.Render(fmt.Sprintf(" (%d files)", len(node.Files)))
			}

			if node.Locked && i != m.selected {
				sb.WriteString(lockedStyle.Render(line))
			} else {
				sb.WriteString(style.Render(line))
			}
			sb.WriteString("\n")
		}
	}

	// Recent events
	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("Recent Changes"))
	sb.WriteString("\n")

	if len(m.events) == 0 {
		sb.WriteString(dimStyle.Render("  (watching for changes...)"))
		sb.WriteString("\n")
	} else {
		for _, event := range m.events {
			sb.WriteString(eventStyle.Render("  " + event))
			sb.WriteString("\n")
		}
	}

	// Footer
	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("j/k: navigate  r: refresh  q: quit"))
	sb.WriteString("\n")

	return sb.String()
}

// SetAgentStatus updates the agent status display
func (m *Model) SetAgentStatus(status string) {
	m.agentStatus = status
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
