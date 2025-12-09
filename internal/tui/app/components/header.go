package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/andrew-sameh/brewsync/internal/tui/styles"
	"github.com/andrew-sameh/brewsync/pkg/version"
)

// HeaderModel represents the global header bar
type HeaderModel struct {
	machineName string
	width       int
	showIgnored bool
}

// NewHeader creates a new header model
func NewHeader(machineName string, width int) HeaderModel {
	return HeaderModel{
		machineName: machineName,
		width:       width,
		showIgnored: false,
	}
}

// SetMachine updates the machine name
func (m *HeaderModel) SetMachine(name string) {
	m.machineName = name
}

// SetWidth updates the header width
func (m *HeaderModel) SetWidth(width int) {
	m.width = width
}

// SetShowIgnored updates the showIgnored state
func (m *HeaderModel) SetShowIgnored(show bool) {
	m.showIgnored = show
}

// View renders the header
func (m HeaderModel) View() string {
	// Left side: app name and version
	appName := styles.HeaderStyle.Render("📦 BrewSync")
	ver := styles.SidebarDimmedStyle.Render("v" + version.Version)
	leftSide := appName + " " + ver

	// Right side: machine name
	rightSide := styles.SidebarDimmedStyle.Render("Machine: ") +
		styles.HeaderStyle.Render("💻 "+m.machineName)

	// Calculate spacing
	leftWidth := lipgloss.Width(leftSide)
	rightWidth := lipgloss.Width(rightSide)
	separator := styles.BorderStyle.Render(" │ ")
	sepWidth := lipgloss.Width(separator)

	// Build the header line
	totalContent := leftWidth + sepWidth + rightWidth
	padding := m.width - totalContent
	if padding < 0 {
		padding = 0
	}

	var sb strings.Builder
	sb.WriteString(leftSide)
	sb.WriteString(separator)
	sb.WriteString(strings.Repeat(" ", padding))
	sb.WriteString(rightSide)

	return sb.String()
}

// RenderFullHeader renders the complete header with borders
func (m HeaderModel) RenderFullHeader() string {
	const (
		topLeft     = "┌"
		topRight    = "┐"
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

	// Top border
	sb.WriteString(styles.BorderStyle.Render(topLeft + strings.Repeat(horizontal, innerWidth) + topRight))
	sb.WriteString("\n")

	// Header content
	content := m.View()
	contentWidth := lipgloss.Width(content)
	if contentWidth > innerWidth {
		content = truncateString(content, innerWidth)
	} else if contentWidth < innerWidth {
		content = content + strings.Repeat(" ", innerWidth-contentWidth)
	}

	sb.WriteString(styles.BorderStyle.Render(vertical))
	sb.WriteString("  ")
	sb.WriteString(content)
	// Adjust for the 2-space padding we added
	sb.WriteString(strings.Repeat(" ", innerWidth-lipgloss.Width(content)-2))
	sb.WriteString(styles.BorderStyle.Render(vertical))
	sb.WriteString("\n")

	// Bottom border (which connects to sidebar separator)
	sb.WriteString(styles.BorderStyle.Render(midLeft + strings.Repeat(horizontal, innerWidth) + midRight))

	return sb.String()
}

// SimpleHeader returns a simple one-line header
func (m HeaderModel) SimpleHeader() string {
	ignoredStatus := "👁 Hidden"
	if m.showIgnored {
		ignoredStatus = "👁 Shown"
	}
	return fmt.Sprintf("📦 BrewSync v%s   │   💻 %s   │   Ignored: %s", version.Version, m.machineName, ignoredStatus)
}
