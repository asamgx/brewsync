package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/andrew-sameh/brewsync/internal/config"
	"github.com/andrew-sameh/brewsync/internal/tui/styles"
)

// IgnoreSection represents which section is focused
type IgnoreSection int

const (
	IgnoreSectionCategories IgnoreSection = iota
	IgnoreSectionPackages
)

// IgnoreModel is the model for the ignore management screen
type IgnoreModel struct {
	config           *config.Config
	width            int
	height           int
	section          IgnoreSection
	cursor           int
	globalCategories []string
	machineCategories []string
	globalPackages   []string
	machinePackages  []string
	loading          bool
}

// NewIgnoreModel creates a new ignore model
func NewIgnoreModel(cfg *config.Config) *IgnoreModel {
	return &IgnoreModel{
		config:  cfg,
		width:   80,
		height:  24,
		loading: true,
	}
}

type ignoreLoadedMsg struct {
	globalCategories  []string
	machineCategories []string
	globalPackages    []string
	machinePackages   []string
}

// Init initializes the ignore model
func (m *IgnoreModel) Init() tea.Cmd {
	return func() tea.Msg {
		result := ignoreLoadedMsg{}

		if m.config != nil {
			// Get ignored categories
			allCats := m.config.GetIgnoredCategories(m.config.CurrentMachine)
			// This is simplified - in real implementation we'd separate global vs machine
			result.globalCategories = allCats

			// Get ignored packages
			allPkgs := m.config.GetIgnoredPackages(m.config.CurrentMachine)
			result.globalPackages = allPkgs
		}

		return result
	}
}

// Update handles messages
func (m *IgnoreModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case ignoreLoadedMsg:
		m.loading = false
		m.globalCategories = msg.globalCategories
		m.machineCategories = msg.machineCategories
		m.globalPackages = msg.globalPackages
		m.machinePackages = msg.machinePackages
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "b"))):
			return m, func() tea.Msg { return Navigate("dashboard") }

		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			if m.section == IgnoreSectionCategories {
				m.section = IgnoreSectionPackages
			} else {
				m.section = IgnoreSectionCategories
			}
			m.cursor = 0

		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			maxItems := len(m.globalCategories)
			if m.section == IgnoreSectionPackages {
				maxItems = len(m.globalPackages)
			}
			if m.cursor < maxItems-1 {
				m.cursor++
			}
		}
	}

	return m, nil
}

// SetSize updates the ignore dimensions
func (m *IgnoreModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the ignore screen (legacy)
func (m *IgnoreModel) View() string {
	return m.ViewContent(m.width, m.height)
}

// ViewContent renders just the content area (for use in layout)
func (m *IgnoreModel) ViewContent(width, height int) string {
	var b strings.Builder

	if m.loading {
		b.WriteString(styles.DimmedStyle.Render("Loading..."))
		return b.String()
	}

	// Categories section
	catHeader := "IGNORED CATEGORIES"
	if m.section == IgnoreSectionCategories {
		catHeader = styles.SelectedStyle.Render("► " + catHeader)
	} else {
		catHeader = styles.DimmedStyle.Render("  " + catHeader)
	}
	b.WriteString(catHeader)
	b.WriteString("\n")
	b.WriteString(styles.DimmedStyle.Render(strings.Repeat("─", width-4)))
	b.WriteString("\n")

	if len(m.globalCategories) == 0 {
		b.WriteString(styles.DimmedStyle.Render("  No categories ignored"))
		b.WriteString("\n")
	} else {
		for i, cat := range m.globalCategories {
			prefix := "  "
			if m.section == IgnoreSectionCategories && i == m.cursor {
				prefix = styles.CursorStyle.Render("> ")
			}
			b.WriteString(fmt.Sprintf("%s▸ %s\n", prefix, cat))
		}
	}
	b.WriteString("\n")

	// Packages section
	pkgHeader := "IGNORED PACKAGES"
	if m.section == IgnoreSectionPackages {
		pkgHeader = styles.SelectedStyle.Render("► " + pkgHeader)
	} else {
		pkgHeader = styles.DimmedStyle.Render("  " + pkgHeader)
	}
	b.WriteString(pkgHeader)
	b.WriteString("\n")
	b.WriteString(styles.DimmedStyle.Render(strings.Repeat("─", width-4)))
	b.WriteString("\n")

	if len(m.globalPackages) == 0 {
		b.WriteString(styles.DimmedStyle.Render("  No packages ignored"))
		b.WriteString("\n")
	} else {
		for i, pkg := range m.globalPackages {
			prefix := "  "
			if m.section == IgnoreSectionPackages && i == m.cursor {
				prefix = styles.CursorStyle.Render("> ")
			}
			b.WriteString(fmt.Sprintf("%s▸ %s\n", prefix, pkg))
		}
	}

	return b.String()
}
