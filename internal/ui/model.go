package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

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

	events         []string     // Recent events log
	currentDiff    *diff.Result // Current diff to display
	width          int
	height         int
	err            error
	quitting       bool
	lastRenderTime time.Time              // Track last render for throttling
	pendingEvents  map[string]eventUpdate // Coalesce rapid events for same file
}

// eventUpdate tracks the most recent event for a file
type eventUpdate struct {
	event     watcher.Event
	timestamp time.Time
}

// fileEventMsg wraps a file event for the tea runtime
type fileEventMsg watcher.Event

// processCoalescedMsg triggers processing of coalesced events
type processCoalescedMsg struct{}

// errMsg wraps an error for the tea runtime
type errMsg error

// New creates a new UI model
func New(fw *watcher.FileWatcher) *Model {
	return &Model{
		watcher:       fw,
		stateManager:  state.New(),
		diffEngine:    diff.New(),
		events:        make([]string, 0),
		pendingEvents: make(map[string]eventUpdate),
		width:         80,
		height:        24,
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
	// Start a ticker to process coalesced events periodically
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return processCoalescedMsg{}
	})
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
		// Coalesce events - store only the latest event for each file
		event := watcher.Event(msg)
		m.pendingEvents[event.Path] = eventUpdate{
			event:     event,
			timestamp: time.Now(),
		}
		// Don't process immediately - wait for coalescing ticker
		return m, nil

	case processCoalescedMsg:
		// Process all pending events that haven't been updated in a while
		now := time.Now()
		processThreshold := 200 * time.Millisecond

		for path, update := range m.pendingEvents {
			if now.Sub(update.timestamp) >= processThreshold {
				m.handleFileEvent(update.event)
				delete(m.pendingEvents, path)
			}
		}

		// Schedule next coalescing tick
		return m, tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
			return processCoalescedMsg{}
		})

	case errMsg:
		m.err = msg
	}

	return m, nil
}

// handleFileEvent processes a file event and updates the diff
func (m *Model) handleFileEvent(event watcher.Event) {
	// Throttle event log updates - don't add same file multiple times in quick succession
	shouldAddToLog := true
	if len(m.events) > 0 {
		// Check if last event was for the same file within last second
		lastEvent := m.events[len(m.events)-1]
		if strings.Contains(lastEvent, event.Path) {
			timeSinceLastRender := time.Since(m.lastRenderTime)
			if timeSinceLastRender < 500*time.Millisecond {
				shouldAddToLog = false
			}
		}
	}

	if shouldAddToLog {
		// Add to event log
		eventStr := fmt.Sprintf("[%s] %s: %s",
			event.Timestamp.Format("15:04:05"),
			event.Op,
			event.Path)

		m.events = append(m.events, eventStr)
		if len(m.events) > 10 {
			m.events = m.events[1:]
		}
		m.lastRenderTime = time.Now()
	}

	// For remove events, we can't stat the file (it's gone)
	// but we can still process it to show deletion diff
	if event.Op == "remove" {
		// Update state and compute deletion diff
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
		return
	}

	// For non-remove events, stat the file to get info
	info, err := os.Stat(event.Path)
	if err != nil {
		// File might have been deleted or is inaccessible
		if os.IsNotExist(err) {
			m.err = fmt.Errorf("file not found: %s", event.Path)
		}
		return
	}

	if info.IsDir() {
		return
	}

	// Skip files larger than 1MB for diff computation
	const maxDiffSize = 1 * 1024 * 1024 // 1MB
	if info.Size() > maxDiffSize {
		m.currentDiff = &diff.Result{
			Path:     event.Path,
			HasDiff:  true,
			IsBinary: false,
			Lines:    []diff.DiffLine{},
		}
		m.err = fmt.Errorf("file too large for diff (%d bytes, max %d bytes)",
			info.Size(), maxDiffSize)
		return
	}

	// Clear any previous "file too large" errors
	if m.err != nil && strings.Contains(m.err.Error(), "file too large") {
		m.err = nil
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

	watchPathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true)

	recursiveMode := "non-recursively"
	if m.watcher.IsRecursive() {
		recursiveMode = "recursively"
	}

	headerText := "DiffWatch - Real-time File Diff Viewer\n" +
		watchPathStyle.Render(fmt.Sprintf("Watching: %s (%s)", m.watcher.WatchPath(), recursiveMode))

	b.WriteString(headerStyle.Render(headerText))
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

	// Error display (but don't show "file too large" as error - it's already shown in diff)
	if m.err != nil {
		errMsg := m.err.Error()
		if !strings.Contains(errMsg, "file too large") {
			b.WriteString("\n")
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)
			b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		}
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

	// Handle files with no lines (e.g., too large files)
	if len(result.Lines) == 0 && result.HasDiff {
		largeFileStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Yellow
			Bold(true)

		statusStyle = statusStyle.Foreground(lipgloss.Color("11")) // Yellow
		b.WriteString(headerStyle.Render("📄 ") + statusStyle.Render("[FILE TOO LARGE] ") + result.Path + "\n\n")

		if m.err != nil && strings.Contains(m.err.Error(), "file too large") {
			b.WriteString(largeFileStyle.Render(m.err.Error()))
		} else {
			b.WriteString(largeFileStyle.Render("File is too large to display diff (max 1MB)"))
		}
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
