package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/config"
	"github.com/asamgx/brewsync/internal/installer"
	"github.com/asamgx/brewsync/internal/tui/styles"
)

// SetupStep represents the current step in the setup wizard
type SetupStep int

const (
	SetupStepWelcome SetupStep = iota
	SetupStepMachineName
	SetupStepHostname
	SetupStepBrewfileChoice    // Choose to create or enter existing
	SetupStepBrewfilePath      // Only shown if user chooses to enter existing path
	SetupStepAddSourceMachine  // Ask if user wants to add a source machine
	SetupStepSourceName        // Source machine name
	SetupStepSourceHostname    // Source machine hostname
	SetupStepSourceBrewfile    // Source machine brewfile path
	SetupStepSetDefaultSource  // Set source as default?
	SetupStepConfirm
	SetupStepCreating
	SetupStepDumping // Run dump to create Brewfile
	SetupStepDone
)

// BrewfileChoice represents the user's choice for Brewfile handling
type BrewfileChoice int

const (
	BrewfileChoiceCreate BrewfileChoice = iota
	BrewfileChoiceExisting
)

// setupConfigResultMsg is sent when config creation completes
type setupConfigResultMsg struct {
	err error
}

// setupDumpResultMsg is sent when dump completes
type setupDumpResultMsg struct {
	err    error
	counts map[string]int
	total  int
}

// SetupModel is the model for the first-time setup wizard
type SetupModel struct {
	width  int
	height int
	step   SetupStep
	err    error

	// Current machine fields
	machineName textinput.Model
	hostname    textinput.Model
	brewfile    textinput.Model

	// Brewfile choice
	brewfileChoice    BrewfileChoice
	createBrewfile    bool // true = create via dump, false = use existing path
	runDumpAfterSetup bool // flag to trigger dump after setup

	// Source machine fields
	addSourceMachine   bool
	sourceName         textinput.Model
	sourceHostname     textinput.Model
	sourceBrewfile     textinput.Model
	setSourceAsDefault bool

	// Detected values
	detectedHostname string

	// Progress tracking
	spinner         spinner.Model
	progressMessage string

	// Dump results (when createBrewfile is true)
	dumpCounts map[string]int
	dumpTotal  int
}

// NewSetupModel creates a new setup model
func NewSetupModel() *SetupModel {
	// Detect hostname
	hostname := ""
	if h, err := config.GetLocalHostname(); err == nil {
		hostname = h
	}

	// Create text inputs for current machine
	machineNameInput := textinput.New()
	machineNameInput.Placeholder = "e.g., mini, air, work"
	machineNameInput.CharLimit = 20

	hostnameInput := textinput.New()
	hostnameInput.Placeholder = "e.g., Andrews-Mac-mini"
	hostnameInput.CharLimit = 50
	hostnameInput.SetValue(hostname)

	// Default brewfile path: ~/Brewfile
	home, _ := os.UserHomeDir()
	defaultBrewfile := filepath.Join(home, "Brewfile")

	brewfileInput := textinput.New()
	brewfileInput.Placeholder = defaultBrewfile
	brewfileInput.CharLimit = 200
	brewfileInput.SetValue(defaultBrewfile)

	// Create text inputs for source machine
	sourceNameInput := textinput.New()
	sourceNameInput.Placeholder = "e.g., work, server"
	sourceNameInput.CharLimit = 20

	sourceHostnameInput := textinput.New()
	sourceHostnameInput.Placeholder = "e.g., Work-MacBook-Pro"
	sourceHostnameInput.CharLimit = 50

	sourceBrewfileInput := textinput.New()
	sourceBrewfileInput.Placeholder = "e.g., ~/dotfiles/work/Brewfile"
	sourceBrewfileInput.CharLimit = 200

	// Suggest machine name from hostname
	suggestedName := suggestMachineName(hostname)
	machineNameInput.SetValue(suggestedName)

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.PrimaryColor)

	return &SetupModel{
		width:              80,
		height:             24,
		step:               SetupStepWelcome,
		machineName:        machineNameInput,
		hostname:           hostnameInput,
		brewfile:           brewfileInput,
		brewfileChoice:     BrewfileChoiceCreate,
		createBrewfile:     true,
		sourceName:         sourceNameInput,
		sourceHostname:     sourceHostnameInput,
		sourceBrewfile:     sourceBrewfileInput,
		detectedHostname:   hostname,
		addSourceMachine:   false,
		setSourceAsDefault: false,
		spinner:            s,
		progressMessage:    "",
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

	case spinner.TickMsg:
		// Update spinner animation
		if m.step == SetupStepCreating || m.step == SetupStepDumping {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case setupConfigResultMsg:
		// Config creation completed
		if msg.err != nil {
			m.err = msg.err
			m.step = SetupStepDone
			return m, nil
		}
		// If we need to run dump, start it now
		if m.runDumpAfterSetup {
			m.step = SetupStepDumping
			m.progressMessage = "Collecting installed packages..."
			return m, tea.Batch(m.spinner.Tick, m.runDump())
		}
		// Otherwise go to done
		m.step = SetupStepDone
		return m, nil

	case setupDumpResultMsg:
		// Dump completed
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.dumpCounts = msg.counts
			m.dumpTotal = msg.total
		}
		m.step = SetupStepDone
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
				m.step = SetupStepBrewfileChoice
				m.hostname.Blur()
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.step = SetupStepMachineName
				m.hostname.Blur()
				m.machineName.Focus()
				return m, textinput.Blink
			default:
				m.hostname, cmd = m.hostname.Update(msg)
				return m, cmd
			}

		case SetupStepBrewfileChoice:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("1", "c", "C"))):
				// Create new Brewfile
				m.createBrewfile = true
				m.runDumpAfterSetup = true
				home, _ := os.UserHomeDir()
				m.brewfile.SetValue(filepath.Join(home, "Brewfile"))
				m.step = SetupStepAddSourceMachine
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("2", "e", "E"))):
				// Enter existing path
				m.createBrewfile = false
				m.runDumpAfterSetup = false
				m.step = SetupStepBrewfilePath
				m.brewfile.Focus()
				return m, textinput.Blink
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.step = SetupStepHostname
				m.hostname.Focus()
				return m, textinput.Blink
			}

		case SetupStepBrewfilePath:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if m.brewfile.Value() != "" {
					m.step = SetupStepAddSourceMachine
					m.brewfile.Blur()
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.step = SetupStepBrewfileChoice
				m.brewfile.Blur()
				return m, nil
			default:
				m.brewfile, cmd = m.brewfile.Update(msg)
				return m, cmd
			}

		case SetupStepAddSourceMachine:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("y", "Y"))):
				m.addSourceMachine = true
				m.step = SetupStepSourceName
				m.sourceName.Focus()
				return m, textinput.Blink
			case key.Matches(msg, key.NewBinding(key.WithKeys("n", "N", "enter"))):
				m.addSourceMachine = false
				m.step = SetupStepConfirm
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				if m.createBrewfile {
					m.step = SetupStepBrewfileChoice
				} else {
					m.step = SetupStepBrewfilePath
					m.brewfile.Focus()
					return m, textinput.Blink
				}
				return m, nil
			}

		case SetupStepSourceName:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if m.sourceName.Value() != "" {
					m.step = SetupStepSourceHostname
					m.sourceName.Blur()
					m.sourceHostname.Focus()
					return m, textinput.Blink
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.step = SetupStepAddSourceMachine
				m.sourceName.Blur()
				return m, nil
			default:
				m.sourceName, cmd = m.sourceName.Update(msg)
				return m, cmd
			}

		case SetupStepSourceHostname:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				m.step = SetupStepSourceBrewfile
				m.sourceHostname.Blur()
				m.sourceBrewfile.Focus()
				return m, textinput.Blink
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.step = SetupStepSourceName
				m.sourceHostname.Blur()
				m.sourceName.Focus()
				return m, textinput.Blink
			default:
				m.sourceHostname, cmd = m.sourceHostname.Update(msg)
				return m, cmd
			}

		case SetupStepSourceBrewfile:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if m.sourceBrewfile.Value() != "" {
					m.step = SetupStepSetDefaultSource
					m.sourceBrewfile.Blur()
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.step = SetupStepSourceHostname
				m.sourceBrewfile.Blur()
				m.sourceHostname.Focus()
				return m, textinput.Blink
			default:
				m.sourceBrewfile, cmd = m.sourceBrewfile.Update(msg)
				return m, cmd
			}

		case SetupStepSetDefaultSource:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("y", "Y"))):
				m.setSourceAsDefault = true
				m.step = SetupStepConfirm
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("n", "N", "enter"))):
				m.setSourceAsDefault = false
				m.step = SetupStepConfirm
				return m, nil
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.step = SetupStepSourceBrewfile
				m.sourceBrewfile.Focus()
				return m, textinput.Blink
			}

		case SetupStepConfirm:
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter", "y"))):
				m.step = SetupStepCreating
				m.progressMessage = "Initializing..."
				// Start spinner and config creation
				return m, tea.Batch(m.spinner.Tick, m.createConfig())
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "n"))):
				// Go back to appropriate step
				if m.addSourceMachine {
					m.step = SetupStepSetDefaultSource
				} else {
					m.step = SetupStepAddSourceMachine
				}
				return m, nil
			}

		case SetupStepDone:
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				return m, func() tea.Msg {
					return SetupCompleteMsg{}
				}
			}
		}
	}

	return m, nil
}

func (m *SetupModel) createConfig() tea.Cmd {
	// Capture values needed for the closure
	machineName := m.machineName.Value()
	hostnameVal := m.hostname.Value()
	brewfilePath := m.brewfile.Value()
	addSourceMachine := m.addSourceMachine
	sourceName := m.sourceName.Value()
	sourceHostname := m.sourceHostname.Value()
	sourceBrewfile := m.sourceBrewfile.Value()
	setSourceAsDefault := m.setSourceAsDefault

	return func() tea.Msg {
		// Expand ~ in brewfile path
		home, _ := os.UserHomeDir()
		if strings.HasPrefix(brewfilePath, "~/") {
			brewfilePath = filepath.Join(home, brewfilePath[2:])
		}

		// Ensure config directory exists
		if err := config.EnsureDir(); err != nil {
			return setupConfigResultMsg{err: fmt.Errorf("failed to create config directory: %w", err)}
		}

		// Get config path
		configPath, err := config.ConfigPath()
		if err != nil {
			return setupConfigResultMsg{err: fmt.Errorf("failed to get config path: %w", err)}
		}

		// Build machines map
		machines := map[string]interface{}{
			machineName: map[string]interface{}{
				"hostname":    hostnameVal,
				"brewfile":    brewfilePath,
				"description": fmt.Sprintf("Machine %s", machineName),
			},
		}

		// Add source machine if configured
		if addSourceMachine && sourceName != "" {
			srcBrewfile := sourceBrewfile
			if strings.HasPrefix(srcBrewfile, "~/") {
				srcBrewfile = filepath.Join(home, srcBrewfile[2:])
			}
			machines[sourceName] = map[string]interface{}{
				"hostname":    sourceHostname,
				"brewfile":    srcBrewfile,
				"description": fmt.Sprintf("Machine %s", sourceName),
			}
		}

		// Determine default source
		defaultSource := machineName
		if addSourceMachine && setSourceAsDefault {
			defaultSource = sourceName
		}

		// Build initial config with all default settings
		initialConfig := map[string]interface{}{
			"machines":            machines,
			"current_machine":     "auto",
			"default_source":      defaultSource,
			"default_categories":  config.DefaultCategories,
			"conflict_resolution": string(config.ConflictAsk),
			"auto_dump": map[string]interface{}{
				"enabled":        false,
				"after_install":  false,
				"commit":         false,
				"push":           false,
				"commit_message": config.DefaultCommitMessage,
			},
			"dump": map[string]interface{}{
				"use_brew_bundle": true,
			},
			"machine_specific": map[string]interface{}{},
			"output": map[string]interface{}{
				"color":             true,
				"verbose":           false,
				"show_descriptions": true,
			},
			"hooks": map[string]interface{}{
				"pre_install":  "",
				"post_install": "",
				"pre_dump":     "",
				"post_dump":    "",
			},
		}

		// Marshal to YAML
		data, err := yaml.Marshal(initialConfig)
		if err != nil {
			return setupConfigResultMsg{err: fmt.Errorf("failed to marshal config: %w", err)}
		}

		// Write config file
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return setupConfigResultMsg{err: fmt.Errorf("failed to write config: %w", err)}
		}

		// Create default ignore file (non-fatal if it fails)
		_ = config.CreateDefaultIgnoreFile()

		// Ensure Brewfile directory exists
		brewfileDir := filepath.Dir(brewfilePath)
		if err := os.MkdirAll(brewfileDir, 0755); err != nil {
			return setupConfigResultMsg{err: fmt.Errorf("failed to create Brewfile directory: %w", err)}
		}

		// Success!
		return setupConfigResultMsg{err: nil}
	}
}

// runDump collects all installed packages and writes them to the Brewfile
func (m *SetupModel) runDump() tea.Cmd {
	brewfilePath := m.brewfile.Value()

	return func() tea.Msg {
		// Expand ~ in brewfile path
		home, _ := os.UserHomeDir()
		if strings.HasPrefix(brewfilePath, "~/") {
			brewfilePath = filepath.Join(home, brewfilePath[2:])
		}

		// Load the config we just created
		cfg, err := config.Load()
		if err != nil {
			return setupDumpResultMsg{err: fmt.Errorf("failed to load config: %w", err)}
		}

		// Collect all packages
		allPackages, err := collectPackagesForSetup(cfg, brewfilePath)
		if err != nil {
			return setupDumpResultMsg{err: err}
		}

		// Write Brewfile
		writer := brewfile.NewWriter(allPackages)
		if err := writer.Write(brewfilePath); err != nil {
			return setupDumpResultMsg{err: fmt.Errorf("failed to write Brewfile: %w", err)}
		}

		// Count by type
		counts := make(map[string]int)
		for _, pkg := range allPackages {
			counts[string(pkg.Type)]++
		}

		return setupDumpResultMsg{
			counts: counts,
			total:  len(allPackages),
		}
	}
}

// collectPackagesForSetup collects all installed packages (similar to dump screen)
func collectPackagesForSetup(cfg *config.Config, brewfilePath string) (brewfile.Packages, error) {
	var allPackages brewfile.Packages
	brewInst := installer.NewBrewInstaller()

	// Use brew bundle dump if configured (default), otherwise collect manually
	if cfg.Dump.UseBrewBundle && brewInst.IsAvailable() {
		// Create temp file for brew bundle dump
		tmpFile := brewfilePath + ".brewbundle.tmp"
		if err := brewInst.DumpToFile(tmpFile); err == nil {
			// Parse the brew bundle output (includes taps, formulae, casks with descriptions)
			if brewPkgs, err := brewfile.Parse(tmpFile); err == nil {
				allPackages = append(allPackages, brewPkgs...)
			}
			os.Remove(tmpFile)
		}
	} else if brewInst.IsAvailable() {
		// Manual collection
		if taps, err := brewInst.ListTaps(); err == nil {
			allPackages = append(allPackages, taps...)
		}
		if formulae, err := brewInst.ListFormulae(); err == nil {
			allPackages = append(allPackages, formulae...)
		}
		if casks, err := brewInst.ListCasks(); err == nil {
			allPackages = append(allPackages, casks...)
		}
	}

	// Collect extensions
	if vscodeInst := installer.NewVSCodeInstaller(); vscodeInst.IsAvailable() {
		if extensions, err := vscodeInst.List(); err == nil {
			allPackages = allPackages.AddUnique(extensions...)
		}
	}

	if cursorInst := installer.NewCursorInstaller(); cursorInst.IsAvailable() {
		if extensions, err := cursorInst.List(); err == nil {
			allPackages = allPackages.AddUnique(extensions...)
		}
	}

	if antigravityInst := installer.NewAntigravityInstaller(); antigravityInst.IsAvailable() {
		if extensions, err := antigravityInst.List(); err == nil {
			allPackages = allPackages.AddUnique(extensions...)
		}
	}

	if goInst := installer.NewGoToolsInstaller(); goInst.IsAvailable() {
		if tools, err := goInst.List(); err == nil {
			allPackages = allPackages.AddUnique(tools...)
		}
	}

	if masInst := installer.NewMasInstaller(); masInst.IsAvailable() {
		if apps, err := masInst.List(); err == nil {
			allPackages = allPackages.AddUnique(apps...)
		}
	}

	return allPackages, nil
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
		b.WriteString("You'll configure:\n\n")
		b.WriteString("  • A name for this machine (e.g., 'mini', 'air')\n")
		b.WriteString("  • The hostname for auto-detection\n")
		b.WriteString("  • Your Brewfile location\n")
		b.WriteString("  • Optionally, a source machine to sync from\n")
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Enter to continue • Ctrl+C to exit"))

	case SetupStepMachineName:
		b.WriteString(headerStyle.Render("Step 1: Machine Name"))
		b.WriteString("\n\n")
		b.WriteString("Enter a short name to identify this machine:\n\n")
		b.WriteString(m.machineName.View())
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Enter to continue • Esc to go back • Ctrl+C to exit"))

	case SetupStepHostname:
		b.WriteString(headerStyle.Render("Step 2: Hostname"))
		b.WriteString("\n\n")
		b.WriteString("Enter the hostname for auto-detection:\n")
		b.WriteString(styles.DimmedStyle.Render(fmt.Sprintf("(Detected: %s)", m.detectedHostname)) + "\n\n")
		b.WriteString(m.hostname.View())
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Enter to continue • Esc to go back • Ctrl+C to exit"))

	case SetupStepBrewfileChoice:
		b.WriteString(headerStyle.Render("Step 3: Brewfile"))
		b.WriteString("\n\n")
		b.WriteString("How would you like to set up your Brewfile?\n\n")
		b.WriteString("  " + styles.SelectedStyle.Render("[1]") + " Create a new Brewfile (captures currently installed packages)\n")
		b.WriteString("      " + styles.DimmedStyle.Render("Will save to ~/Brewfile and run dump after setup") + "\n\n")
		b.WriteString("  " + styles.SelectedStyle.Render("[2]") + " Enter path to existing Brewfile\n")
		b.WriteString("      " + styles.DimmedStyle.Render("Use an existing Brewfile from your dotfiles") + "\n")
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Press 1 or 2 to choose • Esc to go back • Ctrl+C to exit"))

	case SetupStepBrewfilePath:
		b.WriteString(headerStyle.Render("Step 3: Brewfile Path"))
		b.WriteString("\n\n")
		b.WriteString("Enter the path to your existing Brewfile:\n\n")
		b.WriteString(m.brewfile.View())
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Enter to continue • Esc to go back • Ctrl+C to exit"))

	case SetupStepAddSourceMachine:
		b.WriteString(headerStyle.Render("Step 4: Source Machine"))
		b.WriteString("\n\n")
		b.WriteString("Would you like to add a source machine to sync from?\n\n")
		b.WriteString(styles.DimmedStyle.Render("A source machine is another Mac whose packages you want to import.") + "\n")
		b.WriteString(styles.DimmedStyle.Render("You can add more machines later with 'brewsync config add-machine'.") + "\n")
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("y = Yes • n = No (skip) • Esc to go back • Ctrl+C to exit"))

	case SetupStepSourceName:
		b.WriteString(headerStyle.Render("Source Machine: Name"))
		b.WriteString("\n\n")
		b.WriteString("Enter a short name for the source machine:\n\n")
		b.WriteString(m.sourceName.View())
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Enter to continue • Esc to go back • Ctrl+C to exit"))

	case SetupStepSourceHostname:
		b.WriteString(headerStyle.Render("Source Machine: Hostname"))
		b.WriteString("\n\n")
		b.WriteString("Enter the hostname of the source machine:\n")
		b.WriteString(styles.DimmedStyle.Render("(Used for auto-detection when on that machine)") + "\n\n")
		b.WriteString(m.sourceHostname.View())
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Enter to continue • Esc to go back • Ctrl+C to exit"))

	case SetupStepSourceBrewfile:
		b.WriteString(headerStyle.Render("Source Machine: Brewfile"))
		b.WriteString("\n\n")
		b.WriteString("Enter the path to the source machine's Brewfile:\n\n")
		b.WriteString(m.sourceBrewfile.View())
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Enter to continue • Esc to go back • Ctrl+C to exit"))

	case SetupStepSetDefaultSource:
		b.WriteString(headerStyle.Render("Default Source"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("Set '%s' as the default source for imports?\n\n", m.sourceName.Value()))
		b.WriteString(styles.DimmedStyle.Render("The default source is used when running 'import' without specifying --from.") + "\n")
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("y = Yes • n = No • Esc to go back • Ctrl+C to exit"))

	case SetupStepConfirm:
		b.WriteString(headerStyle.Render("Confirm Configuration"))
		b.WriteString("\n\n")
		b.WriteString(styles.SelectedStyle.Render("Current Machine:") + "\n")
		b.WriteString(fmt.Sprintf("  Name:     %s\n", m.machineName.Value()))
		b.WriteString(fmt.Sprintf("  Hostname: %s\n", m.hostname.Value()))
		b.WriteString(fmt.Sprintf("  Brewfile: %s\n", m.brewfile.Value()))
		if m.createBrewfile {
			b.WriteString(fmt.Sprintf("  Action:   %s\n", styles.DimmedStyle.Render("Create via dump")))
		}

		if m.addSourceMachine {
			b.WriteString("\n")
			b.WriteString(styles.SelectedStyle.Render("Source Machine:") + "\n")
			b.WriteString(fmt.Sprintf("  Name:     %s\n", m.sourceName.Value()))
			b.WriteString(fmt.Sprintf("  Hostname: %s\n", m.sourceHostname.Value()))
			b.WriteString(fmt.Sprintf("  Brewfile: %s\n", m.sourceBrewfile.Value()))
			if m.setSourceAsDefault {
				b.WriteString(fmt.Sprintf("  Default:  %s\n", styles.SelectedStyle.Render("Yes")))
			}
		}

		b.WriteString("\n")
		b.WriteString("Create configuration?\n\n")
		b.WriteString(styles.HelpStyle.Render("y/Enter = Yes • n/Esc = No • Ctrl+C to exit"))

	case SetupStepCreating:
		b.WriteString(headerStyle.Render("Creating Configuration"))
		b.WriteString("\n\n")
		b.WriteString(m.spinner.View() + " ")
		b.WriteString("Setting up BrewSync...\n\n")
		b.WriteString(styles.DimmedStyle.Render("  • Creating config directory") + "\n")
		b.WriteString(styles.DimmedStyle.Render("  • Writing config.yaml") + "\n")
		b.WriteString(styles.DimmedStyle.Render("  • Creating ignore.yaml") + "\n")
		b.WriteString(styles.DimmedStyle.Render("  • Setting up Brewfile directory") + "\n")

	case SetupStepDumping:
		b.WriteString(headerStyle.Render("Creating Brewfile"))
		b.WriteString("\n\n")
		b.WriteString(m.spinner.View() + " ")
		b.WriteString("Collecting installed packages...\n\n")
		b.WriteString(styles.DimmedStyle.Render("This may take a moment. Scanning:") + "\n")
		b.WriteString(styles.DimmedStyle.Render("  • Homebrew taps, formulae, and casks") + "\n")
		b.WriteString(styles.DimmedStyle.Render("  • VSCode and Cursor extensions") + "\n")
		b.WriteString(styles.DimmedStyle.Render("  • Go tools") + "\n")
		b.WriteString(styles.DimmedStyle.Render("  • Mac App Store apps") + "\n")

	case SetupStepDone:
		if m.err != nil {
			b.WriteString(styles.ErrorStyle.Render("Setup Failed"))
			b.WriteString("\n\n")
			b.WriteString(styles.ErrorStyle.Render(m.err.Error()))
		} else {
			b.WriteString(headerStyle.Render("Setup Complete!"))
			b.WriteString("\n\n")
			b.WriteString(styles.SelectedStyle.Render("✓"))
			b.WriteString(" Configuration created successfully!\n")

			// Show dump results if we ran dump
			if m.runDumpAfterSetup && m.dumpTotal > 0 {
				b.WriteString(styles.SelectedStyle.Render("✓"))
				b.WriteString(fmt.Sprintf(" Brewfile created with %d packages\n\n", m.dumpTotal))

				// Show counts by type
				b.WriteString(styles.DimmedStyle.Render("Packages by type:") + "\n")
				typeOrder := []string{"tap", "brew", "cask", "vscode", "cursor", "antigravity", "go", "mas"}
				for _, t := range typeOrder {
					if count, ok := m.dumpCounts[t]; ok && count > 0 {
						b.WriteString(fmt.Sprintf("  %s: %d\n", t, count))
					}
				}
				b.WriteString("\n")
				b.WriteString(fmt.Sprintf("Brewfile saved to: %s\n", m.brewfile.Value()))
			} else if !m.runDumpAfterSetup {
				b.WriteString("\nNext steps:\n")
				if m.addSourceMachine {
					b.WriteString("  • Run 'import' to sync packages from " + m.sourceName.Value() + "\n")
				}
				b.WriteString("  • Run 'dump' to capture your installed packages\n")
			}
		}
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Press Enter to continue..."))
	}

	return b.String()
}
