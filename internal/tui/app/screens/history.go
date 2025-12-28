package screens

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/asamgx/brewsync/internal/config"
	"github.com/asamgx/brewsync/internal/history"
	"github.com/asamgx/brewsync/internal/tui/styles"
)

// HistoryModel is the model for the history screen
type HistoryModel struct {
	config  *config.Config
	width   int
	height  int
	entries []history.Entry
	cursor  int
	loading bool
	err     error
}

// NewHistoryModel creates a new history model
func NewHistoryModel(cfg *config.Config) *HistoryModel {
	return &HistoryModel{
		config:  cfg,
		width:   80,
		height:  24,
		loading: true,
	}
}

type historyLoadedMsg struct {
	entries []history.Entry
	err     error
}

// Init initializes the history model
func (m *HistoryModel) Init() tea.Cmd {
	return func() tea.Msg {
		entries, err := history.Read(20)
		return historyLoadedMsg{entries: entries, err: err}
	}
}

// Update handles messages
func (m *HistoryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case historyLoadedMsg:
		m.loading = false
		m.entries = msg.entries
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "b"))):
			return m, func() tea.Msg { return Navigate("dashboard") }

		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		}
	}

	return m, nil
}

// SetSize updates the history dimensions
func (m *HistoryModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the history screen (legacy)
func (m *HistoryModel) View() string {
	return m.ViewContent(m.width, m.height)
}

// ViewContent renders just the content area (for use in layout)
func (m *HistoryModel) ViewContent(width, height int) string {
	var b strings.Builder

	if m.loading {
		b.WriteString(styles.DimmedStyle.Render("Loading history..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
		return b.String()
	}

	if len(m.entries) == 0 {
		b.WriteString(styles.DimmedStyle.Render("No history entries yet."))
		return b.String()
	}

	// Entries
	for i, entry := range m.entries {
		prefix := "  "
		if i == m.cursor {
			prefix = styles.CursorStyle.Render("> ")
		}

		// Format entry
		line := prefix + entry.Format(false)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}
