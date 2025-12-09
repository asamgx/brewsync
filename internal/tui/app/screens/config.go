package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/andrew-sameh/brewsync/internal/config"
	"github.com/andrew-sameh/brewsync/internal/tui/styles"
)

// ConfigSection represents which section is focused
type ConfigSection int

const (
	ConfigSectionMachines ConfigSection = iota
	ConfigSectionGeneral
	ConfigSectionAutoDump
)

// ConfigModel is the model for the config screen
type ConfigModel struct {
	config   *config.Config
	width    int
	height   int
	section  ConfigSection
	cursor   int
	machines []string
}

// NewConfigModel creates a new config model
func NewConfigModel(cfg *config.Config) *ConfigModel {
	var machines []string
	if cfg != nil {
		for name := range cfg.Machines {
			machines = append(machines, name)
		}
	}

	return &ConfigModel{
		config:   cfg,
		width:    80,
		height:   24,
		machines: machines,
	}
}

// Init initializes the config model
func (m *ConfigModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "b"))):
			return m, func() tea.Msg { return Navigate("dashboard") }

		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			m.section = (m.section + 1) % 3
			m.cursor = 0

		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			maxItems := m.getMaxItems()
			if m.cursor < maxItems-1 {
				m.cursor++
			}
		}
	}

	return m, nil
}

func (m *ConfigModel) getMaxItems() int {
	switch m.section {
	case ConfigSectionMachines:
		return len(m.machines)
	case ConfigSectionGeneral:
		return 4 // current_machine, default_source, categories, output
	case ConfigSectionAutoDump:
		return 5 // enabled, after_install, commit, push, message
	}
	return 0
}

// SetSize updates the config dimensions
func (m *ConfigModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the config screen (legacy)
func (m *ConfigModel) View() string {
	return m.ViewContent(m.width, m.height)
}

// ViewContent renders just the content area (for use in layout)
func (m *ConfigModel) ViewContent(width, height int) string {
	var b strings.Builder

	if m.config == nil {
		b.WriteString(styles.ErrorStyle.Render("No config loaded"))
		return b.String()
	}

	// Machines section
	machinesHeader := "MACHINES"
	if m.section == ConfigSectionMachines {
		machinesHeader = styles.SelectedStyle.Render("► " + machinesHeader)
	} else {
		machinesHeader = styles.DimmedStyle.Render("  " + machinesHeader)
	}
	b.WriteString(machinesHeader)
	b.WriteString("\n")
	b.WriteString(styles.DimmedStyle.Render(strings.Repeat("─", width-4)))
	b.WriteString("\n")

	for i, name := range m.machines {
		machine, _ := m.config.GetMachine(name)
		prefix := "  "
		if m.section == ConfigSectionMachines && i == m.cursor {
			prefix = styles.CursorStyle.Render("> ")
		}

		current := ""
		if name == m.config.CurrentMachine {
			current = styles.SelectedStyle.Render(" (current)")
		}

		line := fmt.Sprintf("%s▸ %s%s\n", prefix, name, current)
		b.WriteString(line)
		b.WriteString(styles.DimmedStyle.Render(fmt.Sprintf("      %s\n", machine.Brewfile)))
	}
	b.WriteString("\n")

	// General section
	generalHeader := "GENERAL"
	if m.section == ConfigSectionGeneral {
		generalHeader = styles.SelectedStyle.Render("► " + generalHeader)
	} else {
		generalHeader = styles.DimmedStyle.Render("  " + generalHeader)
	}
	b.WriteString(generalHeader)
	b.WriteString("\n")
	b.WriteString(styles.DimmedStyle.Render(strings.Repeat("─", width-4)))
	b.WriteString("\n")

	generalItems := []struct {
		label string
		value string
	}{
		{"Default Source", m.config.DefaultSource},
		{"Categories", strings.Join(m.config.DefaultCategories, ", ")},
	}

	for i, item := range generalItems {
		prefix := "  "
		if m.section == ConfigSectionGeneral && i == m.cursor {
			prefix = styles.CursorStyle.Render("> ")
		}

		labelStyle := lipgloss.NewStyle().Foreground(styles.MutedColor).Width(18)
		b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, labelStyle.Render(item.label), item.value))
	}
	b.WriteString("\n")

	// Auto-dump section
	autoDumpHeader := "AUTO-DUMP"
	if m.section == ConfigSectionAutoDump {
		autoDumpHeader = styles.SelectedStyle.Render("► " + autoDumpHeader)
	} else {
		autoDumpHeader = styles.DimmedStyle.Render("  " + autoDumpHeader)
	}
	b.WriteString(autoDumpHeader)
	b.WriteString("\n")
	b.WriteString(styles.DimmedStyle.Render(strings.Repeat("─", width-4)))
	b.WriteString("\n")

	autoDumpItems := []struct {
		label string
		value string
	}{
		{"Enabled", boolToYesNo(m.config.AutoDump.Enabled)},
		{"After Install", boolToYesNo(m.config.AutoDump.AfterInstall)},
		{"Auto Commit", boolToYesNo(m.config.AutoDump.Commit)},
	}

	for i, item := range autoDumpItems {
		prefix := "  "
		if m.section == ConfigSectionAutoDump && i == m.cursor {
			prefix = styles.CursorStyle.Render("> ")
		}

		labelStyle := lipgloss.NewStyle().Foreground(styles.MutedColor).Width(18)
		b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, labelStyle.Render(item.label), item.value))
	}

	return b.String()
}

func boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
