package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/andrew-sameh/brewsync/internal/config"
	"github.com/andrew-sameh/brewsync/internal/tui/styles"
)

// SetupStep represents the current step in the setup wizard
type SetupStep int

const (
	SetupStepWelcome SetupStep = iota
	SetupStepMachineName
	SetupStepHostname
	SetupStepBrewfile
	SetupStepConfirm
	SetupStepCreating
	SetupStepDone
)

// SetupModel is the model for the first-time setup wizard
type SetupModel struct {
	width       int
	height      int
	step        SetupStep
	err         error

	// Form fields
	machineName textinput.Model
	hostname    textinput.Model
	brewfile    textinput.Model
	focusIndex  int

	// Detected values
	detectedHostname string
}

// NewSetupModel creates a new setup model
func NewSetupModel() *SetupModel {
	// Detect hostname
	hostname := ""
	if h, err := config.GetLocalHostname(); err == nil {
		hostname = h
	}

	// Create text inputs
	machineNameInput := textinput.New()
	machineNameInput.Placeholder = "e.g., mini, air, work"
	machineNameInput.CharLimit = 20
	machineNameInput.Width = 30

	hostnameInput := textinput.New()
	hostnameInput.Placeholder = "e.g., Andrews-Mac-mini"
	hostnameInput.CharLimit = 50
	hostnameInput.Width = 40
	hostnameInput.SetValue(hostname)

	// Suggest default brewfile path
	home, _ := os.UserHomeDir()
	defaultBrewfile := filepath.Join(home, "dotfiles", "_brew_machine", "Brewfile")

	brewfileInput := textinput.New()
	brewfileInput.Placeholder = defaultBrewfile
	brewfileInput.CharLimit = 200
	brewfileInput.Width = 50
	brewfileInput.SetValue(defaultBrewfile)

	// Suggest machine name from hostname
	suggestedName := suggestMachineName(hostname)
	machineNameInput.SetValue(suggestedName)

	return &SetupModel{
		width:            80,
		height:           24,
		step:             SetupStepWelcome,
		machineName:      machineNameInput,
		hostname:         hostnameInput,
		brewfile:         brewfileInput,
		detectedHostname: hostname,
	}
}

func suggestMachineName(hostname string) string {
	// Extract machine name from hostname like "Andrews-Mac-mini"
	hostname = strings.ToLower(hostname)
	hostname = strings.ReplaceAll(hostname, "-", " ")

	parts := strings.Fields(hostname)
	if len(parts) > 0 {
		// Get the last part (usually the machine type)
		last := parts[len(parts)-1]
		if last == "mini" || last == "air" || last == "pro" || last == "imac" || last == "studio" {
			return last
		}
	}

	return ""
}

// Init initializes the setup model
func (m *SetupModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.step {
		case SetupStepWelcome:
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				m.step = SetupStepMachineName
				m.machineName.Focus()
				return m, textinput.Blink
			}

		case SetupStepMachineName:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if m.machineName.Value() != "" {
					m.step = SetupStepHostname
					m.machineName.Blur()
					m.hostname.Focus()
					return m, textinput.Blink
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.step = SetupStepWelcome
				m.machineName.Blur()
			default:
				m.machineName, cmd = m.machineName.Update(msg)
				return m, cmd
			}

		case SetupStepHostname:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				m.step = SetupStepBrewfile
				m.hostname.Blur()
				m.brewfile.Focus()
				// Update brewfile suggestion based on machine name
				home, _ := os.UserHomeDir()
				m.brewfile.SetValue(filepath.Join(home, "dotfiles", "_brew_"+m.machineName.Value(), "Brewfile"))
				return m, textinput.Blink
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.step = SetupStepMachineName
				m.hostname.Blur()
				m.machineName.Focus()
				return m, textinput.Blink
			default:
				m.hostname, cmd = m.hostname.Update(msg)
				return m, cmd
			}

		case SetupStepBrewfile:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if m.brewfile.Value() != "" {
					m.step = SetupStepConfirm
					m.brewfile.Blur()
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.step = SetupStepHostname
				m.brewfile.Blur()
				m.hostname.Focus()
				return m, textinput.Blink
			default:
				m.brewfile, cmd = m.brewfile.Update(msg)
				return m, cmd
			}

		case SetupStepConfirm:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter", "y"))):
				m.step = SetupStepCreating
				return m, m.createConfig()
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "n"))):
				m.step = SetupStepBrewfile
				m.brewfile.Focus()
				return m, textinput.Blink
			}

		case SetupStepDone:
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				return m, func() tea.Msg { return SetupCompleteMsg{} }
			}
		}
	}

	return m, nil
}

func (m *SetupModel) createConfig() tea.Cmd {
	return func() tea.Msg {
		// Ensure config directory exists
		if err := config.EnsureDir(); err != nil {
			m.err = err
			m.step = SetupStepDone
			return nil
		}

		// Create config
		configPath, _ := config.ConfigPath()
		configContent := fmt.Sprintf(`machines:
  %s:
    hostname: "%s"
    brewfile: "%s"

current_machine: auto
default_source: %s
default_categories: [tap, brew, cask, vscode, cursor, antigravity, go, mas]

auto_dump:
  enabled: false
  after_install: false
  commit: false
  push: false
  commit_message: "brewsync: update {machine} Brewfile"

dump:
  use_brew_bundle: true

conflict_resolution: ask

output:
  color: true
  verbose: false
  show_descriptions: true
`, m.machineName.Value(), m.hostname.Value(), m.brewfile.Value(), m.machineName.Value())

		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			m.err = err
			m.step = SetupStepDone
			return nil
		}

		// Create default ignore file
		if err := config.CreateDefaultIgnoreFile(); err != nil {
			// Non-fatal, continue
		}

		// Ensure Brewfile directory exists
		brewfileDir := filepath.Dir(m.brewfile.Value())
		if err := os.MkdirAll(brewfileDir, 0755); err != nil {
			m.err = err
			m.step = SetupStepDone
			return nil
		}

		m.step = SetupStepDone
		return nil
	}
}

// View renders the setup wizard
func (m *SetupModel) View() string {
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.PrimaryColor).
		MarginBottom(1)

	switch m.step {
	case SetupStepWelcome:
		b.WriteString(headerStyle.Render("Welcome to BrewSync!"))
		b.WriteString("\n\n")
		b.WriteString("This wizard will help you set up BrewSync for the first time.\n")
		b.WriteString("You'll need to provide:\n\n")
		b.WriteString("  • A name for this machine (e.g., 'mini', 'air')\n")
		b.WriteString("  • The hostname for auto-detection\n")
		b.WriteString("  • The path to your Brewfile\n")
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Press Enter to continue..."))

	case SetupStepMachineName:
		b.WriteString(headerStyle.Render("Step 1: Machine Name"))
		b.WriteString("\n\n")
		b.WriteString("Enter a short name to identify this machine:\n\n")
		b.WriteString(m.machineName.View())
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Enter to continue • Esc to go back"))

	case SetupStepHostname:
		b.WriteString(headerStyle.Render("Step 2: Hostname"))
		b.WriteString("\n\n")
		b.WriteString("Enter the hostname for auto-detection:\n")
		b.WriteString(styles.DimmedStyle.Render(fmt.Sprintf("(Detected: %s)\n\n", m.detectedHostname)))
		b.WriteString(m.hostname.View())
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Enter to continue • Esc to go back"))

	case SetupStepBrewfile:
		b.WriteString(headerStyle.Render("Step 3: Brewfile Path"))
		b.WriteString("\n\n")
		b.WriteString("Enter the path to your Brewfile:\n\n")
		b.WriteString(m.brewfile.View())
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Enter to continue • Esc to go back"))

	case SetupStepConfirm:
		b.WriteString(headerStyle.Render("Confirm Configuration"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("Machine Name: %s\n", styles.SelectedStyle.Render(m.machineName.Value())))
		b.WriteString(fmt.Sprintf("Hostname:     %s\n", styles.SelectedStyle.Render(m.hostname.Value())))
		b.WriteString(fmt.Sprintf("Brewfile:     %s\n", styles.SelectedStyle.Render(m.brewfile.Value())))
		b.WriteString("\n")
		b.WriteString("Create configuration? (y/n)")

	case SetupStepCreating:
		b.WriteString(headerStyle.Render("Creating Configuration..."))

	case SetupStepDone:
		if m.err != nil {
			b.WriteString(styles.ErrorStyle.Render("Setup Failed"))
			b.WriteString("\n\n")
			b.WriteString(styles.ErrorStyle.Render(m.err.Error()))
		} else {
			b.WriteString(headerStyle.Render("Setup Complete!"))
			b.WriteString("\n\n")
			b.WriteString(styles.SelectedStyle.Render("✓"))
			b.WriteString(" Configuration created successfully!\n\n")
			b.WriteString("Next steps:\n")
			b.WriteString("  • Run 'dump' to capture your installed packages\n")
			b.WriteString("  • Or 'import' to sync from another machine\n")
		}
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Press Enter to continue..."))
	}

	return b.String()
}
