package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/asamgx/brewsync/internal/tui/styles"
)

// FooterModel represents the status bar and keybindings footer
type FooterModel struct {
	width       int
	keybindings []KeyBinding
	statusMsg   string
	statusType  string // info, success, error, warning
	// Task indicator
	taskRunning bool
	taskAction  string // "Installing", "Uninstalling"
	taskPkg     string // Package name
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

// SetTaskRunning sets the task indicator
func (m *FooterModel) SetTaskRunning(running bool, action, pkg string) {
	m.taskRunning = running
	m.taskAction = action
	m.taskPkg = pkg
}

// ClearTask clears the task indicator
func (m *FooterModel) ClearTask() {
	m.taskRunning = false
	m.taskAction = ""
	m.taskPkg = ""
}

// IsTaskRunning returns whether a task is running
func (m *FooterModel) IsTaskRunning() bool {
	return m.taskRunning
}

// View renders the footer
func (m FooterModel) View() string {
	var parts []string

	for _, kb := range m.keybindings {
		part := styles.HeaderStyle.Render(kb.Key) + " " + styles.FooterStyle.Render(kb.Desc)
		parts = append(parts, part)
	}

	content := strings.Join(parts, styles.BorderStyle.Render("  │  "))

	// Add task indicator on the right if running
	if m.taskRunning {
		taskIndicator := m.renderTaskIndicator()
		indicatorWidth := lipgloss.Width(taskIndicator)
		contentWidth := lipgloss.Width(content)
		// Calculate space: total width minus content minus indicator minus some padding
		availableSpace := m.width - contentWidth - indicatorWidth - 6

		if availableSpace > 0 {
			content = content + strings.Repeat(" ", availableSpace) + taskIndicator
		} else {
			// If not enough space, put task indicator after a separator
			content = content + styles.BorderStyle.Render("  │  ") + taskIndicator
		}
	}

	return content
}

// renderTaskIndicator renders the running task indicator
func (m FooterModel) renderTaskIndicator() string {
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	// Use a simple static spinner char for now (animation would need tick messages)
	spinner := spinnerChars[0]

	taskStyle := lipgloss.NewStyle().
		Foreground(styles.CatYellow).
		Bold(true)

	pkgStyle := lipgloss.NewStyle().
		Foreground(styles.CatMauve)

	return taskStyle.Render(spinner+" "+m.taskAction) + " " + pkgStyle.Render(m.taskPkg)
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
		{Key: "H", Desc: "Toggle Ignored"},
		{Key: "Esc", Desc: "Dashboard"},
		{Key: "q", Desc: "Quit"},
	}
}

// DashboardKeybindings returns keybindings for the dashboard
func DashboardKeybindings() []KeyBinding {
	return []KeyBinding{
		{Key: "1-0/!", Desc: "Screens"},
		{Key: "H", Desc: "Toggle Ignored"},
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
		{Key: "X", Desc: "Uninstall"},
		{Key: "g/G", Desc: "Top/Bottom"},
		{Key: "Esc", Desc: "Dashboard"},
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
		{Key: "h/l", Desc: "Columns"},
		{Key: "i", Desc: "Install"},
		{Key: "X", Desc: "Uninstall"},
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
