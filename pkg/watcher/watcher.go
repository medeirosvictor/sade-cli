// Package watcher provides file system monitoring with debouncing.
package watcher

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// EventType represents the type of file system event
type EventType string

const (
	Created  EventType = "CREATED"
	Modified EventType = "MODIFIED"
	Deleted  EventType = "DELETED"
)

// Event represents a debounced file system event
type Event struct {
	Path      string    `json:"path"`
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

// Watcher monitors a directory tree for file changes with debouncing
type Watcher struct {
	fsWatcher    *fsnotify.Watcher
	rootDir      string
	eventCh      chan Event
	pending      map[string]Event
	pendingMu    sync.Mutex
	debounceTime time.Duration
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

// New creates a new file system watcher with the given debounce time in milliseconds
func New(debounceMs int) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if debounceMs <= 0 {
		debounceMs = 100
	}

	return &Watcher{
		fsWatcher:    fsw,
		eventCh:      make(chan Event, 100),
		pending:      make(map[string]Event),
		debounceTime: time.Duration(debounceMs) * time.Millisecond,
		stopCh:       make(chan struct{}),
	}, nil
}

// Watch starts watching a directory recursively
func (w *Watcher) Watch(root string) error {
	w.rootDir = root

	if err := w.fsWatcher.Add(root); err != nil {
		return err
	}

	return w.walkDir(root)
}

func (w *Watcher) walkDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(dir, name)

		if name == "" || name[0] == '.' {
			continue
		}
		if isIgnoredDir(name) {
			continue
		}

		if entry.IsDir() {
			if err := w.fsWatcher.Add(path); err != nil {
				continue
			}
			w.walkDir(path)
		}
	}
	return nil
}

// Events returns a channel of file events
func (w *Watcher) Events() <-chan Event {
	return w.eventCh
}

// RootDir returns the watched root directory
func (w *Watcher) RootDir() string {
	return w.rootDir
}

// Run starts the event processing loop
func (w *Watcher) Run() {
	w.wg.Add(2)
	go w.processEvents()
	go w.flushPending()
}

func (w *Watcher) processEvents() {
	defer w.wg.Done()

	for {
		select {
		case <-w.stopCh:
			return
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			name := filepath.Base(event.Name)
			if len(name) > 0 && name[0] == '.' {
				continue
			}

			var eventType EventType
			if event.Has(fsnotify.Create) {
				eventType = Created
			} else if event.Has(fsnotify.Write) {
				eventType = Modified
			} else if event.Has(fsnotify.Remove) {
				eventType = Deleted
			} else {
				continue
			}

			w.pendingMu.Lock()
			if _, ok := w.pending[event.Name]; !ok || eventType == Created || eventType == Deleted {
				w.pending[event.Name] = Event{
					Path:      event.Name,
					Type:      eventType,
					Timestamp: time.Now(),
				}
			}
			w.pendingMu.Unlock()
		}
	}
}

func (w *Watcher) flushPending() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.debounceTime)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			w.pendingMu.Lock()
			for _, event := range w.pending {
				select {
				case w.eventCh <- event:
				default:
				}
			}
			w.pending = make(map[string]Event)
			w.pendingMu.Unlock()
			return
		case <-ticker.C:
			w.pendingMu.Lock()
			for _, event := range w.pending {
				select {
				case w.eventCh <- event:
				default:
				}
			}
			w.pending = make(map[string]Event)
			w.pendingMu.Unlock()
		}
	}
}

// Close stops the watcher and cleans up resources
func (w *Watcher) Close() {
	close(w.stopCh)
	w.wg.Wait()

	if w.fsWatcher != nil {
		w.fsWatcher.Close()
	}
	close(w.eventCh)
}

func isIgnoredDir(name string) bool {
	switch name {
	case "node_modules", "vendor", "__pycache__", "target", "dist", "build", ".git", ".sade":
		return true
	}
	return false
}
