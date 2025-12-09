package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/andrew-sameh/brewsync/internal/tui/styles"
)

// FooterModel represents the status bar and keybindings footer
type FooterModel struct {
	width       int
	keybindings []KeyBinding
	statusMsg   string
	statusType  string // info, success, error, warning
}

// KeyBinding represents a single keybinding hint
type KeyBinding struct {
	Key  string
	Desc string
}

// NewFooter creates a new footer model
func NewFooter(width int) FooterModel {
	return FooterModel{
		width:       width,
		keybindings: DefaultKeybindings(),
	}
}

// SetWidth updates the footer width
func (m *FooterModel) SetWidth(width int) {
	m.width = width
}

// SetKeybindings updates the keybindings to display
func (m *FooterModel) SetKeybindings(bindings []KeyBinding) {
	m.keybindings = bindings
}

// SetStatus sets a status message
func (m *FooterModel) SetStatus(msg, msgType string) {
	m.statusMsg = msg
	m.statusType = msgType
}

// ClearStatus clears the status message
func (m *FooterModel) ClearStatus() {
	m.statusMsg = ""
	m.statusType = ""
}

// View renders the footer
func (m FooterModel) View() string {
	var parts []string

	for _, kb := range m.keybindings {
		part := styles.HeaderStyle.Render(kb.Key) + " " + styles.FooterStyle.Render(kb.Desc)
		parts = append(parts, part)
	}

	content := strings.Join(parts, styles.BorderStyle.Render("  │  "))

	// Pad to width if needed
	contentWidth := lipgloss.Width(content)
	if contentWidth < m.width {
		content = content + strings.Repeat(" ", m.width-contentWidth)
	}

	return content
}

// RenderFullFooter renders the complete footer with borders
func (m FooterModel) RenderFullFooter() string {
	const (
		bottomLeft  = "└"
		bottomRight = "┘"
		horizontal  = "─"
		vertical    = "│"
		midLeft     = "├"
		midRight    = "┤"
	)

	innerWidth := m.width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	var sb strings.Builder

	// Top border of footer (connects to main content)
	sb.WriteString(styles.BorderStyle.Render(midLeft + strings.Repeat(horizontal, innerWidth) + midRight))
	sb.WriteString("\n")

	// Footer content
	content := m.View()
	contentWidth := lipgloss.Width(content)
	if contentWidth > innerWidth-2 {
		// Truncate content
		content = truncateString(content, innerWidth-2)
		contentWidth = lipgloss.Width(content)
	}

	sb.WriteString(styles.BorderStyle.Render(vertical))
	sb.WriteString("  ")
	sb.WriteString(content)
	remaining := innerWidth - contentWidth - 2
	if remaining > 0 {
		sb.WriteString(strings.Repeat(" ", remaining))
	}
	sb.WriteString(styles.BorderStyle.Render(vertical))
	sb.WriteString("\n")

	// Bottom border
	sb.WriteString(styles.BorderStyle.Render(bottomLeft + strings.Repeat(horizontal, innerWidth) + bottomRight))

	return sb.String()
}

// DefaultKeybindings returns default global keybindings
func DefaultKeybindings() []KeyBinding {
	return []KeyBinding{
		{Key: "1-0/!", Desc: "Screens"},
		{Key: "h", Desc: "Toggle Ignored"},
		{Key: "Esc", Desc: "Dashboard"},
		{Key: "q", Desc: "Quit"},
	}
}

// DashboardKeybindings returns keybindings for the dashboard
func DashboardKeybindings() []KeyBinding {
	return []KeyBinding{
		{Key: "1-0/!", Desc: "Screens"},
		{Key: "h", Desc: "Toggle Ignored"},
		{Key: "q", Desc: "Quit"},
	}
}

// ContentKeybindings returns generic keybindings for content screens
func ContentKeybindings() []KeyBinding {
	return []KeyBinding{
		{Key: "j/k", Desc: "Navigate"},
		{Key: "1-0/!", Desc: "Screens"},
		{Key: "Esc", Desc: "Dashboard"},
		{Key: "q", Desc: "Quit"},
	}
}

// ListKeybindings returns keybindings for the list screen
func ListKeybindings() []KeyBinding {
	return []KeyBinding{
		{Key: "j/k", Desc: "Navigate"},
		{Key: "g/G", Desc: "Top/Bottom"},
		{Key: "Esc", Desc: "Dashboard"},
		{Key: "q", Desc: "Quit"},
	}
}

// ImportKeybindings returns keybindings for the import screen
func ImportKeybindings() []KeyBinding {
	return []KeyBinding{
		{Key: "j/k", Desc: "Navigate"},
		{Key: "Space", Desc: "Toggle"},
		{Key: "a/n", Desc: "All/None"},
		{Key: "Enter", Desc: "Install"},
		{Key: "Esc", Desc: "Dashboard"},
	}
}

// SyncKeybindings returns keybindings for the sync screen
func SyncKeybindings() []KeyBinding {
	return []KeyBinding{
		{Key: "a", Desc: "Apply"},
		{Key: "Esc", Desc: "Dashboard"},
		{Key: "q", Desc: "Quit"},
	}
}

// DiffKeybindings returns keybindings for the diff screen
func DiffKeybindings() []KeyBinding {
	return []KeyBinding{
		{Key: "j/k", Desc: "Navigate"},
		{Key: "h/l", Desc: "Switch Column"},
		{Key: "g/G", Desc: "Top/Bottom"},
		{Key: "Esc", Desc: "Dashboard"},
	}
}

// DumpKeybindings returns keybindings for the dump screen
func DumpKeybindings() []KeyBinding {
	return []KeyBinding{
		{Key: "d/Enter", Desc: "Run Dump"},
		{Key: "Esc", Desc: "Dashboard"},
		{Key: "q", Desc: "Quit"},
	}
}

// IgnoreKeybindings returns keybindings for the ignore screen
func IgnoreKeybindings() []KeyBinding {
	return []KeyBinding{
		{Key: "j/k", Desc: "Navigate"},
		{Key: "a", Desc: "Add"},
		{Key: "d", Desc: "Delete"},
		{Key: "Esc", Desc: "Dashboard"},
	}
}
