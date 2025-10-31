package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hv11297/diffwatch/internal/diff"
	"github.com/hv11297/diffwatch/internal/state"
	"github.com/hv11297/diffwatch/internal/watcher"
)

// Model represents the UI state
type Model struct {
	watcher      *watcher.FileWatcher
	stateManager *state.Manager
	diffEngine   *diff.Engine

	events      []string     // Recent events log
	currentDiff *diff.Result // Current diff to display
	width       int
	height      int
	err         error
	quitting    bool
}

// fileEventMsg wraps a file event for the tea runtime
type fileEventMsg watcher.Event

// errMsg wraps an error for the tea runtime
type errMsg error

// New creates a new UI model
func New(fw *watcher.FileWatcher) *Model {
	return &Model{
		watcher:      fw,
		stateManager: state.New(),
		diffEngine:   diff.New(),
		events:       make([]string, 0),
		width:        80,
		height:       24,
	}
}

// Start starts the bubbletea program
func (m *Model) Start() error {
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Start listening for file events in background
	go m.listenForEvents(p)

	_, err := p.Run()
	return err
}

// Quit signals the program to quit
func (m *Model) Quit() {
	m.quitting = true
}

// listenForEvents listens for file system events and sends them to the tea program
func (m *Model) listenForEvents(p *tea.Program) {
	for {
		select {
		case event, ok := <-m.watcher.Events():
			if !ok {
				return
			}
			p.Send(fileEventMsg(event))

		case err, ok := <-m.watcher.Errors():
			if !ok {
				return
			}
			p.Send(errMsg(err))
		}
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case fileEventMsg:
		m.handleFileEvent(watcher.Event(msg))

	case errMsg:
		m.err = msg
	}

	return m, nil
}

// handleFileEvent processes a file event and updates the diff
func (m *Model) handleFileEvent(event watcher.Event) {
	// Add to event log
	eventStr := fmt.Sprintf("[%s] %s: %s",
		event.Timestamp.Format("15:04:05"),
		event.Op,
		event.Path)

	m.events = append(m.events, eventStr)
	if len(m.events) > 10 {
		m.events = m.events[1:]
	}

	// Skip directories - we only track file changes for diffs
	if info, err := os.Stat(event.Path); err == nil && info.IsDir() {
		return
	}

	// Update state and compute diff
	oldState, newState, err := m.stateManager.Update(event.Path)
	if err != nil {
		m.err = err
		return
	}

	result, err := m.diffEngine.Compute(oldState, newState)
	if err != nil {
		m.err = err
		return
	}

	if result.HasDiff {
		m.currentDiff = result
	}
}

// View renders the UI
func (m *Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Width(m.width)

	b.WriteString(headerStyle.Render("DiffWatch - Real-time File Diff Viewer"))
	b.WriteString("\n\n")

	// Event log
	eventStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	b.WriteString(eventStyle.Render("Recent Events:"))
	b.WriteString("\n")

	if len(m.events) == 0 {
		b.WriteString(eventStyle.Render("  Waiting for file changes..."))
		b.WriteString("\n")
	} else {
		for _, event := range m.events {
			b.WriteString(eventStyle.Render("  " + event))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// Diff view
	diffStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1).
		Width(m.width - 4)

	if m.currentDiff != nil {
		// Render modern diff view
		renderedDiff := m.renderModernDiff(m.currentDiff)
		b.WriteString(diffStyle.Render(renderedDiff))
	} else {
		b.WriteString(diffStyle.Render("No changes yet"))
	}

	// Error display
	if m.err != nil {
		b.WriteString("\n")
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	// Footer
	b.WriteString("\n")
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)
	b.WriteString(footerStyle.Render("Press 'q' to quit"))

	return b.String()
}

// renderModernDiff renders a diff in a modern, vimdiff-like style
func (m *Model) renderModernDiff(result *diff.Result) string {
	var b strings.Builder

	// File header with status
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14"))

	statusStyle := lipgloss.NewStyle().
		Bold(true)

	// Handle binary files specially
	if result.IsBinary {
		binaryStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Yellow
			Bold(true)

		if result.IsNew {
			statusStyle = statusStyle.Foreground(lipgloss.Color("10")) // Green
			b.WriteString(headerStyle.Render("📦 ") + statusStyle.Render("[NEW BINARY FILE] ") + result.Path + "\n\n")
		} else if result.IsDeleted {
			statusStyle = statusStyle.Foreground(lipgloss.Color("9")) // Red
			b.WriteString(headerStyle.Render("📦 ") + statusStyle.Render("[DELETED BINARY FILE] ") + result.Path + "\n\n")
		} else {
			statusStyle = statusStyle.Foreground(lipgloss.Color("11")) // Yellow
			b.WriteString(headerStyle.Render("📦 ") + statusStyle.Render("[MODIFIED BINARY FILE] ") + result.Path + "\n\n")
		}

		b.WriteString(binaryStyle.Render("Binary file detected - diff content not shown"))
		return b.String()
	}

	if result.IsNew {
		statusStyle = statusStyle.Foreground(lipgloss.Color("10")) // Green
		b.WriteString(headerStyle.Render("📄 ") + statusStyle.Render("[NEW FILE] ") + result.Path + "\n\n")
	} else if result.IsDeleted {
		statusStyle = statusStyle.Foreground(lipgloss.Color("9")) // Red
		b.WriteString(headerStyle.Render("📄 ") + statusStyle.Render("[DELETED] ") + result.Path + "\n\n")
	} else {
		statusStyle = statusStyle.Foreground(lipgloss.Color("11")) // Yellow
		b.WriteString(headerStyle.Render("📄 ") + statusStyle.Render("[MODIFIED] ") + result.Path + "\n\n")
	}

	// Styles for different line types
	addedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")). // Bright green
		Background(lipgloss.Color("22"))  // Dark green background

	deletedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")). // Bright red
		Background(lipgloss.Color("52")) // Dark red background

	unchangedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")) // Light gray

	lineNumStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(5).
		Align(lipgloss.Right)

	// Render lines
	maxLines := 50 // Limit lines to prevent overflow
	displayLines := result.Lines
	if len(displayLines) > maxLines {
		displayLines = displayLines[:maxLines]
	}

	for _, line := range displayLines {
		var lineNumStr, iconStr, content string

		switch line.Type {
		case diff.LineAdded:
			iconStr = "✓ "
			lineNumStr = lineNumStyle.Render(fmt.Sprintf("%4d ", line.NewLineNum))
			content = addedStyle.Render(iconStr + line.Content)

		case diff.LineDeleted:
			iconStr = "✗ "
			lineNumStr = lineNumStyle.Render(fmt.Sprintf("%4d ", line.OldLineNum))
			content = deletedStyle.Render(iconStr + line.Content)

		case diff.LineUnchanged:
			iconStr = "  "
			lineNumStr = lineNumStyle.Render(fmt.Sprintf("%4d ", line.NewLineNum))
			content = unchangedStyle.Render(iconStr + line.Content)

		default:
			continue
		}

		b.WriteString(lineNumStr + content + "\n")
	}

	if len(result.Lines) > maxLines {
		moreStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		b.WriteString("\n" + moreStyle.Render(fmt.Sprintf("... %d more lines (truncated for display)", len(result.Lines)-maxLines)))
	}

	return b.String()
}

// colorizeDiff adds color to diff output (legacy, keeping for backward compatibility)
func colorizeDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var colored []string

	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))    // Green
	delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))     // Red
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14")) // Cyan

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+"):
			colored = append(colored, addStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			colored = append(colored, delStyle.Render(line))
		case strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++"):
			colored = append(colored, headerStyle.Render(line))
		default:
			colored = append(colored, line)
		}
	}

	return strings.Join(colored, "\n")
}
