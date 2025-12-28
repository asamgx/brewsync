package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/asamgx/brewsync/internal/tui/styles"
)

const (
	// SidebarWidth is the fixed width of the sidebar
	SidebarWidth = 20

	// MinContentWidth is the minimum width for the content area
	MinContentWidth = 40
)

// Layout handles the main TUI layout rendering
type Layout struct {
	width  int
	height int
}

// NewLayout creates a new layout
func NewLayout(width, height int) Layout {
	return Layout{
		width:  width,
		height: height,
	}
}

// SetSize updates the layout dimensions
func (l *Layout) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// ContentWidth returns the width available for content
func (l Layout) ContentWidth() int {
	// Total width - sidebar width - 3 borders (left, middle, right)
	return l.width - SidebarWidth - 3
}

// ContentHeight returns the height available for content
func (l Layout) ContentHeight() int {
	// Total height - top border (1) - header (1) - header separator (1)
	//              - footer separator (1) - footer (1) - bottom border (1) = 6 lines overhead
	return l.height - 6
}

// SidebarHeight returns the height available for sidebar
func (l Layout) SidebarHeight() int {
	return l.ContentHeight()
}

// Render renders the full layout with header, sidebar, content, and footer
func (l Layout) Render(header, sidebar, content, footer string) string {
	const (
		topLeft     = "┌"
		topRight    = "┐"
		bottomLeft  = "└"
		bottomRight = "┘"
		horizontal  = "─"
		vertical    = "│"
		midLeft     = "├"
		midRight    = "┤"
		teeDown     = "┬"
		teeUp       = "┴"
		cross       = "┼"
	)

	innerWidth := l.width - 2
	contentWidth := l.ContentWidth()
	contentHeight := l.ContentHeight()

	var sb strings.Builder

	// === TOP BORDER ===
	sb.WriteString(styles.BorderStyle.Render(topLeft))
	sb.WriteString(styles.BorderStyle.Render(strings.Repeat(horizontal, innerWidth)))
	sb.WriteString(styles.BorderStyle.Render(topRight))
	sb.WriteString("\n")

	// === HEADER ROW ===
	headerContent := header
	headerWidth := lipgloss.Width(headerContent)
	if headerWidth < innerWidth {
		headerContent = headerContent + strings.Repeat(" ", innerWidth-headerWidth)
	}
	sb.WriteString(styles.BorderStyle.Render(vertical))
	sb.WriteString(headerContent)
	sb.WriteString(styles.BorderStyle.Render(vertical))
	sb.WriteString("\n")

	// === HEADER-CONTENT SEPARATOR ===
	sb.WriteString(styles.BorderStyle.Render(midLeft))
	sb.WriteString(styles.BorderStyle.Render(strings.Repeat(horizontal, SidebarWidth)))
	sb.WriteString(styles.BorderStyle.Render(teeDown))
	sb.WriteString(styles.BorderStyle.Render(strings.Repeat(horizontal, contentWidth)))
	sb.WriteString(styles.BorderStyle.Render(midRight))
	sb.WriteString("\n")

	// === SIDEBAR + CONTENT ROWS ===
	sidebarLines := strings.Split(sidebar, "\n")
	contentLines := strings.Split(content, "\n")

	for i := 0; i < contentHeight; i++ {
		// Get sidebar line
		var sidebarLine string
		if i < len(sidebarLines) {
			sidebarLine = sidebarLines[i]
		}
		sidebarLineWidth := lipgloss.Width(sidebarLine)
		if sidebarLineWidth < SidebarWidth {
			sidebarLine = sidebarLine + strings.Repeat(" ", SidebarWidth-sidebarLineWidth)
		} else if sidebarLineWidth > SidebarWidth {
			sidebarLine = truncateString(sidebarLine, SidebarWidth)
		}

		// Get content line
		var contentLine string
		if i < len(contentLines) {
			contentLine = contentLines[i]
		}
		contentLineWidth := lipgloss.Width(contentLine)
		if contentLineWidth < contentWidth {
			contentLine = contentLine + strings.Repeat(" ", contentWidth-contentLineWidth)
		} else if contentLineWidth > contentWidth {
			contentLine = truncateString(contentLine, contentWidth)
		}

		// Build the row
		sb.WriteString(styles.BorderStyle.Render(vertical))
		sb.WriteString(sidebarLine)
		sb.WriteString(styles.BorderStyle.Render(vertical))
		sb.WriteString(contentLine)
		sb.WriteString(styles.BorderStyle.Render(vertical))
		sb.WriteString("\n")
	}

	// === CONTENT-FOOTER SEPARATOR ===
	sb.WriteString(styles.BorderStyle.Render(midLeft))
	sb.WriteString(styles.BorderStyle.Render(strings.Repeat(horizontal, SidebarWidth)))
	sb.WriteString(styles.BorderStyle.Render(teeUp))
	sb.WriteString(styles.BorderStyle.Render(strings.Repeat(horizontal, contentWidth)))
	sb.WriteString(styles.BorderStyle.Render(midRight))
	sb.WriteString("\n")

	// === FOOTER ROW ===
	footerContent := footer
	footerWidth := lipgloss.Width(footerContent)
	if footerWidth < innerWidth {
		footerContent = footerContent + strings.Repeat(" ", innerWidth-footerWidth)
	}
	sb.WriteString(styles.BorderStyle.Render(vertical))
	sb.WriteString(footerContent)
	sb.WriteString(styles.BorderStyle.Render(vertical))
	sb.WriteString("\n")

	// === BOTTOM BORDER ===
	sb.WriteString(styles.BorderStyle.Render(bottomLeft))
	sb.WriteString(styles.BorderStyle.Render(strings.Repeat(horizontal, innerWidth)))
	sb.WriteString(styles.BorderStyle.Render(bottomRight))

	return sb.String()
}

// truncateString truncates a string to a maximum width
func truncateString(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	// Simple truncation
	runes := []rune(s)
	if len(runes) > maxWidth-3 && maxWidth > 3 {
		return string(runes[:maxWidth-3]) + "..."
	}
	if len(runes) > maxWidth {
		return string(runes[:maxWidth])
	}
	return s
}

// RenderSimple renders a simpler layout without complex borders (for setup wizard)
func (l Layout) RenderSimple(content string) string {
	const (
		topLeft     = "┌"
		topRight    = "┐"
		bottomLeft  = "└"
		bottomRight = "┘"
		horizontal  = "─"
		vertical    = "│"
	)

	innerWidth := l.width - 2
	innerHeight := l.height - 2

	var sb strings.Builder

	// Top border
	sb.WriteString(styles.BorderStyle.Render(topLeft))
	sb.WriteString(styles.BorderStyle.Render(strings.Repeat(horizontal, innerWidth)))
	sb.WriteString(styles.BorderStyle.Render(topRight))
	sb.WriteString("\n")

	// Content lines
	contentLines := strings.Split(content, "\n")
	for i := 0; i < innerHeight; i++ {
		var line string
		if i < len(contentLines) {
			line = contentLines[i]
		}
		lineWidth := lipgloss.Width(line)
		if lineWidth < innerWidth {
			line = line + strings.Repeat(" ", innerWidth-lineWidth)
		} else if lineWidth > innerWidth {
			line = truncateString(line, innerWidth)
		}

		sb.WriteString(styles.BorderStyle.Render(vertical))
		sb.WriteString(line)
		sb.WriteString(styles.BorderStyle.Render(vertical))
		sb.WriteString("\n")
	}

	// Bottom border
	sb.WriteString(styles.BorderStyle.Render(bottomLeft))
	sb.WriteString(styles.BorderStyle.Render(strings.Repeat(horizontal, innerWidth)))
	sb.WriteString(styles.BorderStyle.Render(bottomRight))

	return sb.String()
}
