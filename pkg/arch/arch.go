// Package arch handles .sade/architecture.json and nodes/*.md files.
package arch

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Architecture represents .sade/architecture.json
type Architecture struct {
	Version      string     `json:"version"`
	LastUpdated  *time.Time `json:"last_updated,omitempty"`
	Nodes        []Node     `json:"nodes"`
	Edges        []Edge     `json:"edges"`
}

// Node represents a responsibility group in the architecture graph
type Node struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Locked      bool     `json:"locked,omitempty"`
	Parent      *string  `json:"parent,omitempty"`
	Files       []string `json:"files,omitempty"`
}

// Edge represents a relationship between nodes
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // "depends", "contains", "calls"
	Locked bool   `json:"locked,omitempty"`
}

// LoadArch loads architecture.json from .sade/
func LoadArch(projectRoot string) (*Architecture, error) {
	path := filepath.Join(projectRoot, ".sade", "architecture.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Empty(), nil
		}
		return nil, err
	}

	var arch Architecture
	if err := json.Unmarshal(data, &arch); err != nil {
		return nil, err
	}

	if arch.Nodes == nil {
		arch.Nodes = []Node{}
	}
	if arch.Edges == nil {
		arch.Edges = []Edge{}
	}

	return &arch, nil
}

// Empty returns an empty architecture
func Empty() *Architecture {
	return &Architecture{
		Version: "1.0",
		Nodes:   []Node{},
		Edges:   []Edge{},
	}
}

// SaveArch writes architecture.json to .sade/
func SaveArch(projectRoot string, arch *Architecture) error {
	sadeDir := filepath.Join(projectRoot, ".sade")
	if err := os.MkdirAll(sadeDir, 0755); err != nil {
		return err
	}

	now := time.Now()
	arch.LastUpdated = &now

	data, err := json.MarshalIndent(arch, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(sadeDir, "architecture.json"), data, 0644)
}

// NodeDoc represents a parsed .sade/nodes/*.md file
type NodeDoc struct {
	ID          string
	Description string
	Files       []string
	Locked      bool
	Raw         string // Full file content
}

// ParseNodeDoc parses a node markdown file
func ParseNodeDoc(path string) (*NodeDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	node := &NodeDoc{
		ID:  strings.TrimSuffix(filepath.Base(path), ".md"),
		Raw: content,
	}

	// Check for {locked} marker in first 5 lines
	for i := 0; i < min(5, len(lines)); i++ {
		if strings.TrimSpace(lines[i]) == "{locked}" {
			node.Locked = true
			break
		}
	}

	// Parse sections
	var section string
	var descLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## ") {
			section = strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			continue
		}

		switch section {
		case "files":
			if strings.HasPrefix(trimmed, "- ") {
				file := strings.TrimPrefix(trimmed, "- ")
				node.Files = append(node.Files, file)
			}
		case "":
			if !strings.HasPrefix(trimmed, "#") && trimmed != "" && trimmed != "{locked}" {
				descLines = append(descLines, line)
			}
		}
	}

	node.Description = strings.TrimSpace(strings.Join(descLines, "\n"))

	return node, nil
}

// LoadAllNodes loads all node docs from .sade/nodes/
func LoadAllNodes(projectRoot string) ([]*NodeDoc, error) {
	nodesDir := filepath.Join(projectRoot, ".sade", "nodes")

	entries, err := os.ReadDir(nodesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var nodes []*NodeDoc
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		node, err := ParseNodeDoc(filepath.Join(nodesDir, entry.Name()))
		if err != nil {
			continue
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// LockedNodeIDs returns IDs of all locked nodes
func LockedNodeIDs(projectRoot string) ([]string, error) {
	nodes, err := LoadAllNodes(projectRoot)
	if err != nil {
		return nil, err
	}

	var locked []string
	for _, n := range nodes {
		if n.Locked {
			locked = append(locked, n.ID)
		}
	}

	return locked, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
