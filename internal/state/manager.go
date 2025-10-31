package state

import (
	"fmt"
	"os"
	"sync"
)

// FileState represents the state of a file
type FileState struct {
	Path    string
	Content []byte
	Exists  bool
}

// Manager manages file states for diffing
type Manager struct {
	states map[string]*FileState
	mu     sync.RWMutex
}

// New creates a new state manager
func New() *Manager {
	return &Manager{
		states: make(map[string]*FileState),
	}
}

// Get retrieves the current state of a file
func (m *Manager) Get(path string) (*FileState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.states[path]
	return state, ok
}

// Update reads the file and updates its state, returning the old state
func (m *Manager) Update(path string) (*FileState, *FileState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get old state
	oldState := m.states[path]
	if oldState == nil {
		oldState = &FileState{
			Path:   path,
			Exists: false,
		}
	}

	// Read new state
	newState := &FileState{
		Path:   path,
		Exists: true,
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			newState.Exists = false
		} else {
			return oldState, newState, fmt.Errorf("reading file: %w", err)
		}
	} else {
		newState.Content = content
	}

	// Update stored state
	m.states[path] = newState

	return oldState, newState, nil
}

// Remove removes a file from state tracking
func (m *Manager) Remove(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.states, path)
}

// Clear removes all tracked states
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.states = make(map[string]*FileState)
}
