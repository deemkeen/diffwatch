package diff

import (
	"fmt"
	"strings"

	"github.com/hv11297/diffwatch/internal/state"
	"github.com/pmezard/go-difflib/difflib"
)

// Result represents the result of a diff operation
type Result struct {
	Path     string
	OldState *state.FileState
	NewState *state.FileState
	Unified  string
	HasDiff  bool
}

// Engine computes diffs between file states
type Engine struct{}

// New creates a new diff engine
func New() *Engine {
	return &Engine{}
}

// Compute computes the diff between two file states
func (e *Engine) Compute(oldState, newState *state.FileState) (*Result, error) {
	result := &Result{
		Path:     newState.Path,
		OldState: oldState,
		NewState: newState,
	}

	// Handle file deletion
	if !newState.Exists && oldState.Exists {
		result.Unified = fmt.Sprintf("--- %s\n+++ (deleted)\n", oldState.Path)
		result.HasDiff = true
		return result, nil
	}

	// Handle file creation
	if newState.Exists && !oldState.Exists {
		result.Unified = fmt.Sprintf("--- (new file)\n+++ %s\n", newState.Path)
		result.HasDiff = true
		return result, nil
	}

	// Both exist, compute diff
	if oldState.Exists && newState.Exists {
		oldLines := strings.Split(string(oldState.Content), "\n")
		newLines := strings.Split(string(newState.Content), "\n")

		diff := difflib.UnifiedDiff{
			A:        oldLines,
			B:        newLines,
			FromFile: oldState.Path,
			ToFile:   newState.Path,
			Context:  3,
		}

		unified, err := difflib.GetUnifiedDiffString(diff)
		if err != nil {
			return nil, fmt.Errorf("computing diff: %w", err)
		}

		result.Unified = unified
		result.HasDiff = len(unified) > 0

		return result, nil
	}

	// No diff
	return result, nil
}
