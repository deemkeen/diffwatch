package ui

import (
	"fmt"
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

	events      []string // Recent events log
	currentDiff string   // Current diff to display
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
		m.currentDiff = result.Unified
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

	if m.currentDiff != "" {
		// Colorize diff
		coloredDiff := colorizeDiff(m.currentDiff)
		b.WriteString(diffStyle.Render(coloredDiff))
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

// colorizeDiff adds color to diff output
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
