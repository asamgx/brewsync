package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/andrew-sameh/brewsync/internal/tui/styles"
)

// MenuItem represents a single item in the sidebar menu
type MenuItem struct {
	Label     string
	Screen    int  // Screen enum value
	Separator bool // If true, renders as a separator line
}

// SidebarModel represents the sidebar display component (display-only, no navigation)
type SidebarModel struct {
	items  []MenuItem
	active int // Currently active screen
	width  int
	height int
}

// NewSidebar creates a new sidebar model
func NewSidebar(items []MenuItem, width int) SidebarModel {
	return SidebarModel{
		items:  items,
		active: 0,
		width:  width,
		height: 24,
	}
}

// SetActive sets the currently active screen
func (m *SidebarModel) SetActive(screen int) {
	m.active = screen
}

// SetSize updates the sidebar dimensions
func (m *SidebarModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update is a no-op since sidebar is display-only
func (m SidebarModel) Update(msg tea.Msg) (SidebarModel, tea.Cmd) {
	return m, nil
}

// View renders the sidebar
func (m SidebarModel) View() string {
	var sb strings.Builder

	for _, item := range m.items {
		if item.Separator {
			// Render separator line
			sep := strings.Repeat("─", m.width-2)
			sb.WriteString(styles.SeparatorStyle.Render("  " + sep))
			sb.WriteString("\n")
			continue
		}

		// Determine styling based on state
		var line string
		isActive := item.Screen == m.active

		// Active indicator
		if isActive {
			line = styles.ActiveIndicator.String() + " "
		} else {
			line = "  "
		}

		// Label styling
		label := item.Label
		if isActive {
			label = styles.SidebarActiveStyle.Render(label)
		} else {
			label = styles.SidebarStyle.Render(label)
		}

		line += label

		// Pad to width
		lineWidth := lipgloss.Width(line)
		if lineWidth < m.width {
			line += strings.Repeat(" ", m.width-lineWidth)
		}

		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

// DefaultMenuItems returns the default menu items for BrewSync
func DefaultMenuItems() []MenuItem {
	return []MenuItem{
		{Label: "1 Dashboard", Screen: 0},
		{Label: "2 Import", Screen: 1},
		{Label: "3 Sync", Screen: 2},
		{Label: "4 Diff", Screen: 3},
		{Label: "5 Dump", Screen: 4},
		{Separator: true},
		{Label: "6 List", Screen: 5},
		{Label: "7 Ignore", Screen: 6},
		{Separator: true},
		{Label: "8 Config", Screen: 7},
		{Label: "9 History", Screen: 8},
		{Label: "0 Profiles", Screen: 9},
		{Separator: true},
		{Label: "! Doctor", Screen: 10},
	}
}
