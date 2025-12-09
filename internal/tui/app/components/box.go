package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/andrew-sameh/brewsync/internal/tui/styles"
)

// Box renders a bordered box with an optional title
type Box struct {
	Title  string
	Width  int
	Height int
}

// NewBox creates a new box component
func NewBox(title string, width, height int) Box {
	return Box{
		Title:  title,
		Width:  width,
		Height: height,
	}
}

// Render renders content inside a bordered box
func (b Box) Render(content string) string {
	// Box drawing characters
	const (
		topLeft     = "┌"
		topRight    = "┐"
		bottomLeft  = "└"
		bottomRight = "┘"
		horizontal  = "─"
		vertical    = "│"
	)

	// Calculate inner width (accounting for borders)
	innerWidth := b.Width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	var sb strings.Builder

	// Top border with title
	if b.Title != "" {
		titleStr := "─ " + b.Title + " "
		remaining := innerWidth - len(titleStr)
		if remaining < 0 {
			remaining = 0
		}
		sb.WriteString(styles.BorderStyle.Render(topLeft + titleStr + strings.Repeat(horizontal, remaining) + topRight))
	} else {
		sb.WriteString(styles.BorderStyle.Render(topLeft + strings.Repeat(horizontal, innerWidth) + topRight))
	}
	sb.WriteString("\n")

	// Content lines
	lines := strings.Split(content, "\n")
	contentHeight := b.Height - 2 // Account for top and bottom borders
	if contentHeight < 1 {
		contentHeight = len(lines)
	}

	for i := 0; i < contentHeight; i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}

		// Pad or truncate line to fit inner width
		lineLen := lipgloss.Width(line)
		if lineLen > innerWidth {
			line = truncateString(line, innerWidth)
		} else if lineLen < innerWidth {
			line = line + strings.Repeat(" ", innerWidth-lineLen)
		}

		sb.WriteString(styles.BorderStyle.Render(vertical))
		sb.WriteString(line)
		sb.WriteString(styles.BorderStyle.Render(vertical))
		sb.WriteString("\n")
	}

	// Bottom border
	sb.WriteString(styles.BorderStyle.Render(bottomLeft + strings.Repeat(horizontal, innerWidth) + bottomRight))

	return sb.String()
}

// RenderSimple renders content in a simple box without height constraints
func (b Box) RenderSimple(content string) string {
	const (
		topLeft     = "┌"
		topRight    = "┐"
		bottomLeft  = "└"
		bottomRight = "┘"
		horizontal  = "─"
		vertical    = "│"
	)

	innerWidth := b.Width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	var sb strings.Builder

	// Top border with title
	if b.Title != "" {
		titleStr := "─ " + b.Title + " "
		remaining := innerWidth - len(titleStr)
		if remaining < 0 {
			remaining = 0
		}
		sb.WriteString(styles.BorderStyle.Render(topLeft + titleStr + strings.Repeat(horizontal, remaining) + topRight))
	} else {
		sb.WriteString(styles.BorderStyle.Render(topLeft + strings.Repeat(horizontal, innerWidth) + topRight))
	}
	sb.WriteString("\n")

	// Content lines
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		lineLen := lipgloss.Width(line)
		if lineLen > innerWidth {
			line = truncateString(line, innerWidth)
		} else if lineLen < innerWidth {
			line = line + strings.Repeat(" ", innerWidth-lineLen)
		}

		sb.WriteString(styles.BorderStyle.Render(vertical))
		sb.WriteString(line)
		sb.WriteString(styles.BorderStyle.Render(vertical))
		sb.WriteString("\n")
	}

	// Bottom border
	sb.WriteString(styles.BorderStyle.Render(bottomLeft + strings.Repeat(horizontal, innerWidth) + bottomRight))

	return sb.String()
}

// truncateString truncates a string to a maximum width
func truncateString(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	// Simple truncation - could be improved for multi-byte chars
	runes := []rune(s)
	if len(runes) > maxWidth-3 && maxWidth > 3 {
		return string(runes[:maxWidth-3]) + "..."
	}
	if len(runes) > maxWidth {
		return string(runes[:maxWidth])
	}
	return s
}
