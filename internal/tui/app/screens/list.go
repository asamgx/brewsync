package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/config"
	"github.com/asamgx/brewsync/internal/tui/styles"
)

// listItem represents a displayable item in the list (header or package)
type listItem struct {
	isHeader    bool
	headerType  brewfile.PackageType
	headerCount int
	pkg         brewfile.Package
}

// ListModel is the model for the package list screen
type ListModel struct {
	config   *config.Config
	width    int
	height   int
	packages brewfile.Packages
	items    []listItem // Flattened list for navigation
	cursor   int
	offset   int // For scrolling
	loading  bool
	err      error

	// Confirmation dialog
	showConfirm   bool
	confirmAction string // "uninstall"
	confirmPkg    brewfile.Package

	// Task state
	taskRunning bool
}

// NewListModel creates a new list model
func NewListModel(cfg *config.Config) *ListModel {
	return &ListModel{
		config:  cfg,
		width:   80,
		height:  24,
		loading: true,
	}
}

type listLoadedMsg struct {
	packages brewfile.Packages
	err      error
}

// Init initializes the list model
func (m *ListModel) Init() tea.Cmd {
	return func() tea.Msg {
		if m.config == nil {
			return listLoadedMsg{err: fmt.Errorf("no config loaded")}
		}

		machine, ok := m.config.GetCurrentMachine()
		if !ok {
			return listLoadedMsg{err: fmt.Errorf("current machine not found")}
		}

		packages, err := brewfile.Parse(machine.Brewfile)
		return listLoadedMsg{packages: packages, err: err}
	}
}

// Update handles messages
func (m *ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case listLoadedMsg:
		m.loading = false
		m.packages = msg.packages
		m.err = msg.err
		m.buildItems()
		return m, nil

	case PackageActionStartMsg:
		m.taskRunning = true
		return m, nil

	case PackageActionDoneMsg:
		m.taskRunning = false
		// Reload list after action
		return m, m.Init()

	case tea.KeyMsg:
		// Handle confirmation dialog
		if m.showConfirm {
			return m.handleConfirmInput(msg)
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			m.moveUp()
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			m.moveDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
			// Jump to top
			m.cursor = 0
			m.offset = 0
		case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
			// Jump to bottom
			if len(m.items) > 0 {
				m.cursor = len(m.items) - 1
				m.adjustOffset()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("X"))):
			// Uninstall current package
			if !m.taskRunning {
				pkg := m.getCurrentPackage()
				if pkg != nil {
					m.confirmPkg = *pkg
					m.confirmAction = "uninstall"
					m.showConfirm = true
				}
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			return m, func() tea.Msg { return Navigate("dashboard") }
		}
	}

	return m, nil
}

// handleConfirmInput handles input in the confirmation dialog
func (m *ListModel) handleConfirmInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.showConfirm = false
		// Send package action message
		return m, func() tea.Msg {
			return PackageActionMsg{
				PkgType: string(m.confirmPkg.Type),
				PkgName: m.confirmPkg.Name,
				Action:  m.confirmAction,
			}
		}
	case "n", "N", "esc":
		m.showConfirm = false
	}
	return m, nil
}

// getCurrentPackage returns the package at the current cursor position
func (m *ListModel) getCurrentPackage() *brewfile.Package {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	item := m.items[m.cursor]
	if item.isHeader {
		return nil
	}
	return &item.pkg
}

// buildItems creates a flattened list of items for navigation
func (m *ListModel) buildItems() {
	m.items = nil
	byType := m.packages.ByType()
	types := []brewfile.PackageType{
		brewfile.TypeTap,
		brewfile.TypeBrew,
		brewfile.TypeCask,
		brewfile.TypeVSCode,
		brewfile.TypeCursor,
		brewfile.TypeAntigravity,
		brewfile.TypeGo,
		brewfile.TypeMas,
	}

	for _, t := range types {
		pkgs := byType[t]
		if len(pkgs) == 0 {
			continue
		}
		// Add header
		m.items = append(m.items, listItem{
			isHeader:    true,
			headerType:  t,
			headerCount: len(pkgs),
		})
		// Add packages
		for _, pkg := range pkgs {
			m.items = append(m.items, listItem{pkg: pkg})
		}
	}
}

// moveUp moves cursor up
func (m *ListModel) moveUp() {
	if m.cursor > 0 {
		m.cursor--
		m.adjustOffset()
	}
}

// moveDown moves cursor down
func (m *ListModel) moveDown() {
	if m.cursor < len(m.items)-1 {
		m.cursor++
		m.adjustOffset()
	}
}

// adjustOffset ensures cursor is visible
func (m *ListModel) adjustOffset() {
	visibleHeight := m.height - 2 // Leave room for scroll indicator
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	// Scroll up if cursor is above visible area
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	// Scroll down if cursor is below visible area
	if m.cursor >= m.offset+visibleHeight {
		m.offset = m.cursor - visibleHeight + 1
	}
}

// SetSize updates the list dimensions
func (m *ListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the list screen (legacy)
func (m *ListModel) View() string {
	return m.ViewContent(m.width, m.height)
}

// ViewContent renders just the content area (for use in layout)
func (m *ListModel) ViewContent(width, height int) string {
	var b strings.Builder

	if m.loading {
		b.WriteString(styles.DimmedStyle.Render("Loading packages..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
		return b.String()
	}

	if len(m.items) == 0 {
		b.WriteString(styles.DimmedStyle.Render("No packages found."))
		return b.String()
	}

	visibleHeight := height - 2
	// Reserve space for confirmation dialog if showing (4 lines: 2 blank + bordered dialog ~2 lines)
	if m.showConfirm {
		visibleHeight -= 5
	}
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	// Render visible items
	endIdx := m.offset + visibleHeight
	if endIdx > len(m.items) {
		endIdx = len(m.items)
	}

	for i := m.offset; i < endIdx; i++ {
		item := m.items[i]
		isCursor := i == m.cursor

		if item.isHeader {
			// Type header with icon
			icon := getTypeIcon(item.headerType)
			headerStyle := styles.GetCategoryStyle(string(item.headerType)).Bold(true)
			prefix := "  "
			if isCursor {
				prefix = styles.CursorStyle.Render("> ")
			}
			b.WriteString(prefix)
			b.WriteString(headerStyle.Render(fmt.Sprintf("%s %s (%d)", icon, item.headerType, item.headerCount)))
		} else {
			// Package line
			prefix := "    "
			if isCursor {
				prefix = styles.CursorStyle.Render("> ") + "  "
			}

			nameStyle := lipgloss.NewStyle()
			if isCursor {
				nameStyle = nameStyle.Foreground(styles.CatMauve).Bold(true)
			}

			line := prefix + nameStyle.Render(item.pkg.Name)

			if item.pkg.Description != "" {
				descWidth := width - lipgloss.Width(line) - 6
				if descWidth > 15 {
					desc := " — " + truncate(item.pkg.Description, descWidth)
					line += lipgloss.NewStyle().Foreground(styles.MutedColor).Render(desc)
				}
			}
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(m.items) > visibleHeight {
		scrollInfo := fmt.Sprintf(" %d/%d ", m.cursor+1, len(m.items))
		b.WriteString("\n")
		b.WriteString(styles.DimmedStyle.Render(scrollInfo))
	}

	// Confirmation dialog overlay
	if m.showConfirm {
		b.WriteString("\n\n")
		b.WriteString(m.renderConfirmDialog())
	}

	return b.String()
}

// renderConfirmDialog renders the confirmation dialog
func (m *ListModel) renderConfirmDialog() string {
	actionLabel := "Uninstall"
	actionColor := styles.CatRed
	if m.confirmAction == "install" {
		actionLabel = "Install"
		actionColor = styles.CatGreen
	}

	// Build dialog content
	actionStyle := lipgloss.NewStyle().Foreground(actionColor).Bold(true)
	pkgStyle := lipgloss.NewStyle().Foreground(styles.CatMauve).Bold(true)
	promptStyle := lipgloss.NewStyle().Foreground(styles.CatYellow)

	line1 := actionStyle.Render(actionLabel) + " " + pkgStyle.Render(fmt.Sprintf("%s:%s", m.confirmPkg.Type, m.confirmPkg.Name)) + "?"
	line2 := promptStyle.Render("Press ") + lipgloss.NewStyle().Bold(true).Render("y") + promptStyle.Render(" to confirm, ") + lipgloss.NewStyle().Bold(true).Render("n") + promptStyle.Render(" to cancel")

	// Create bordered dialog box
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(actionColor).
		Padding(0, 2)

	return dialogStyle.Render(line1 + "\n" + line2)
}

func getTypeIcon(t brewfile.PackageType) string {
	switch t {
	case brewfile.TypeTap:
		return "🚰"
	case brewfile.TypeBrew:
		return "🍺"
	case brewfile.TypeCask:
		return "📦"
	case brewfile.TypeVSCode:
		return "💻"
	case brewfile.TypeCursor:
		return "✏️"
	case brewfile.TypeAntigravity:
		return "🚀"
	case brewfile.TypeGo:
		return "🔷"
	case brewfile.TypeMas:
		return "🍎"
	default:
		return "•"
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
