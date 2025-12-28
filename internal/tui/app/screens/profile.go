package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/asamgx/brewsync/internal/config"
	"github.com/asamgx/brewsync/internal/profile"
	"github.com/asamgx/brewsync/internal/tui/styles"
)

// ProfileModel is the model for the profile screen
type ProfileModel struct {
	config   *config.Config
	width    int
	height   int
	profiles []*profile.Profile
	cursor   int
	loading  bool
	err      error
}

// NewProfileModel creates a new profile model
func NewProfileModel(cfg *config.Config) *ProfileModel {
	return &ProfileModel{
		config:  cfg,
		width:   80,
		height:  24,
		loading: true,
	}
}

type profilesLoadedMsg struct {
	profiles []*profile.Profile
	err      error
}

// Init initializes the profile model
func (m *ProfileModel) Init() tea.Cmd {
	return func() tea.Msg {
		names, err := profile.List()
		if err != nil {
			return profilesLoadedMsg{err: err}
		}

		var profiles []*profile.Profile
		for _, name := range names {
			p, err := profile.Load(name)
			if err != nil {
				continue // Skip profiles that fail to load
			}
			profiles = append(profiles, p)
		}

		return profilesLoadedMsg{profiles: profiles}
	}
}

// Update handles messages
func (m *ProfileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case profilesLoadedMsg:
		m.loading = false
		m.profiles = msg.profiles
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
			if m.cursor < len(m.profiles)-1 {
				m.cursor++
			}
		}
	}

	return m, nil
}

// SetSize updates the profile dimensions
func (m *ProfileModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the profile screen (legacy)
func (m *ProfileModel) View() string {
	return m.ViewContent(m.width, m.height)
}

// ViewContent renders just the content area (for use in layout)
func (m *ProfileModel) ViewContent(width, height int) string {
	var b strings.Builder

	if m.loading {
		b.WriteString(styles.DimmedStyle.Render("Loading profiles..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
		return b.String()
	}

	if len(m.profiles) == 0 {
		b.WriteString(styles.DimmedStyle.Render("No profiles found."))
		b.WriteString("\n\n")
		b.WriteString("Create a profile with: brewsync profile create <name>")
		return b.String()
	}

	// Profile list
	for i, p := range m.profiles {
		prefix := "  "
		if i == m.cursor {
			prefix = styles.CursorStyle.Render("> ")
		}

		// Count packages
		total := len(p.Packages.Tap) + len(p.Packages.Brew) + len(p.Packages.Cask) +
			len(p.Packages.VSCode) + len(p.Packages.Cursor) + len(p.Packages.Go) + len(p.Packages.Mas)

		line := fmt.Sprintf("%s▸ %s (%d pkgs)", prefix, p.Name, total)
		b.WriteString(line)
		b.WriteString("\n")

		if p.Description != "" {
			b.WriteString(styles.DimmedStyle.Render(fmt.Sprintf("      %s", p.Description)))
			b.WriteString("\n")
		}
	}

	return b.String()
}
