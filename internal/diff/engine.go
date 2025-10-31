package diff

import (
	"fmt"
	"strings"

	"github.com/hv11297/diffwatch/internal/state"
	"github.com/pmezard/go-difflib/difflib"
)

// LineType represents the type of change in a line
type LineType int

const (
	LineUnchanged LineType = iota
	LineAdded
	LineDeleted
	LineModified
)

// DiffLine represents a single line in the diff with metadata
type DiffLine struct {
	Type       LineType
	OldLineNum int // 0 if not applicable
	NewLineNum int // 0 if not applicable
	Content    string
	OldContent string // For modified lines, to show character-level diff
}

// Result represents the result of a diff operation
type Result struct {
	Path      string
	OldState  *state.FileState
	NewState  *state.FileState
	Unified   string
	Lines     []DiffLine // Structured diff lines for better rendering
	HasDiff   bool
	IsNew     bool // File was created
	IsDeleted bool // File was deleted
	IsBinary  bool // File is binary (don't show diff content)
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
		Lines:    make([]DiffLine, 0),
	}

	// Handle file deletion
	if !newState.Exists && oldState.Exists {
		result.HasDiff = true
		result.IsDeleted = true

		// Check if deleted file was binary
		if isBinary(oldState.Content) {
			result.IsBinary = true
			result.Unified = fmt.Sprintf("Binary file %s deleted\n", oldState.Path)
			return result, nil
		}

		result.Unified = fmt.Sprintf("--- %s\n+++ (deleted)\n", oldState.Path)

		// Add deleted lines
		oldLines := strings.Split(string(oldState.Content), "\n")
		for i, line := range oldLines {
			result.Lines = append(result.Lines, DiffLine{
				Type:       LineDeleted,
				OldLineNum: i + 1,
				Content:    line,
			})
		}
		return result, nil
	}

	// Handle file creation
	if newState.Exists && !oldState.Exists {
		result.HasDiff = true
		result.IsNew = true

		// Check if new file is binary
		if isBinary(newState.Content) {
			result.IsBinary = true
			result.Unified = fmt.Sprintf("Binary file %s created\n", newState.Path)
			return result, nil
		}

		result.Unified = fmt.Sprintf("--- (new file)\n+++ %s\n", newState.Path)

		// Add new lines
		newLines := strings.Split(string(newState.Content), "\n")
		for i, line := range newLines {
			result.Lines = append(result.Lines, DiffLine{
				Type:       LineAdded,
				NewLineNum: i + 1,
				Content:    line,
			})
		}
		return result, nil
	}

	// Both exist, compute diff
	if oldState.Exists && newState.Exists {
		// Check if either version is binary
		oldIsBinary := isBinary(oldState.Content)
		newIsBinary := isBinary(newState.Content)

		if oldIsBinary || newIsBinary {
			result.IsBinary = true
			result.HasDiff = true

			if oldIsBinary && newIsBinary {
				result.Unified = fmt.Sprintf("Binary file %s modified\n", newState.Path)
			} else if newIsBinary {
				result.Unified = fmt.Sprintf("File %s changed from text to binary\n", newState.Path)
			} else {
				result.Unified = fmt.Sprintf("File %s changed from binary to text\n", newState.Path)
			}
			return result, nil
		}

		oldLines := strings.Split(string(oldState.Content), "\n")
		newLines := strings.Split(string(newState.Content), "\n")

		// Generate unified diff for the Unified field
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

		// Generate structured diff lines
		result.Lines = e.computeStructuredDiff(oldLines, newLines)

		return result, nil
	}

	// No diff
	return result, nil
}

// computeStructuredDiff creates a structured representation of the diff
func (e *Engine) computeStructuredDiff(oldLines, newLines []string) []DiffLine {
	// Use difflib's sequence matcher to get opcodes
	matcher := difflib.NewMatcher(oldLines, newLines)
	opcodes := matcher.GetOpCodes()

	var lines []DiffLine

	for _, opcode := range opcodes {
		tag := opcode.Tag
		i1, i2, j1, j2 := opcode.I1, opcode.I2, opcode.J1, opcode.J2

		switch tag {
		case 'e': // equal
			for i := i1; i < i2; i++ {
				lines = append(lines, DiffLine{
					Type:       LineUnchanged,
					OldLineNum: i + 1,
					NewLineNum: j1 + (i - i1) + 1,
					Content:    oldLines[i],
				})
			}

		case 'd': // delete
			for i := i1; i < i2; i++ {
				lines = append(lines, DiffLine{
					Type:       LineDeleted,
					OldLineNum: i + 1,
					Content:    oldLines[i],
				})
			}

		case 'i': // insert
			for j := j1; j < j2; j++ {
				lines = append(lines, DiffLine{
					Type:       LineAdded,
					NewLineNum: j + 1,
					Content:    newLines[j],
				})
			}

		case 'r': // replace (modification)
			// For simple replacements, show as delete + add
			for i := i1; i < i2; i++ {
				lines = append(lines, DiffLine{
					Type:       LineDeleted,
					OldLineNum: i + 1,
					Content:    oldLines[i],
				})
			}
			for j := j1; j < j2; j++ {
				lines = append(lines, DiffLine{
					Type:       LineAdded,
					NewLineNum: j + 1,
					Content:    newLines[j],
				})
			}
		}
	}

	return lines
}

// isBinary checks if content appears to be binary data
// It checks for null bytes and high ratio of non-printable characters
func isBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Check first 8KB (or whole file if smaller)
	checkSize := 8192
	if len(content) < checkSize {
		checkSize = len(content)
	}

	sample := content[:checkSize]

	// If we find a null byte, it's likely binary
	for _, b := range sample {
		if b == 0 {
			return true
		}
	}

	// Count non-printable characters (excluding common whitespace)
	nonPrintable := 0
	for _, b := range sample {
		// Allow common whitespace: tab(9), newline(10), carriage return(13), space(32)
		if b < 32 && b != 9 && b != 10 && b != 13 {
			nonPrintable++
		} else if b > 126 && b < 128 {
			nonPrintable++
		}
	}

	// If more than 30% is non-printable, consider it binary
	return float64(nonPrintable)/float64(len(sample)) > 0.30
}
