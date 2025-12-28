package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/config"
	"github.com/asamgx/brewsync/internal/installer"
	"github.com/asamgx/brewsync/internal/tui/styles"
)

// DumpModel is the model for the dump screen
type DumpModel struct {
	config   *config.Config
	width    int
	height   int
	spinner  spinner.Model
	step     string
	steps    []string
	started  bool // Whether the dump has been started
	done     bool
	err      error
	counts   map[string]int
	total    int
}

// NewDumpModel creates a new dump model
func NewDumpModel(cfg *config.Config) *DumpModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	return &DumpModel{
		config:  cfg,
		width:   80,
		height:  24,
		spinner: s,
		step:    "Initializing...",
		steps:   []string{},
		counts:  make(map[string]int),
	}
}

type dumpStepMsg struct {
	step string
}

type dumpCompleteMsg struct {
	counts map[string]int
	total  int
	err    error
}

// Init initializes the dump model
func (m *DumpModel) Init() tea.Cmd {
	// Don't auto-run dump - wait for user to press Enter or 'd'
	return nil
}

func (m *DumpModel) runDump() tea.Cmd {
	return func() tea.Msg {
		if m.config == nil {
			return dumpCompleteMsg{err: fmt.Errorf("no config loaded")}
		}

		machine, ok := m.config.GetCurrentMachine()
		if !ok {
			return dumpCompleteMsg{err: fmt.Errorf("current machine not configured")}
		}

		brewfilePath := machine.Brewfile
		if brewfilePath == "" {
			return dumpCompleteMsg{err: fmt.Errorf("no Brewfile path configured")}
		}

		// Ensure directory exists
		dir := filepath.Dir(brewfilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return dumpCompleteMsg{err: fmt.Errorf("failed to create directory: %w", err)}
		}

		// Collect all packages
		allPackages, err := collectAllPackages(m.config, brewfilePath)
		if err != nil {
			return dumpCompleteMsg{err: err}
		}

		// Write Brewfile
		writer := brewfile.NewWriter(allPackages)
		if err := writer.Write(brewfilePath); err != nil {
			return dumpCompleteMsg{err: fmt.Errorf("failed to write Brewfile: %w", err)}
		}

		// Count by type
		counts := make(map[string]int)
		for _, pkg := range allPackages {
			counts[string(pkg.Type)]++
		}

		return dumpCompleteMsg{
			counts: counts,
			total:  len(allPackages),
		}
	}
}

// collectAllPackages collects all installed packages
func collectAllPackages(cfg *config.Config, brewfilePath string) (brewfile.Packages, error) {
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

// Update handles messages
func (m *DumpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if !m.done {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case dumpStepMsg:
		m.steps = append(m.steps, m.step)
		m.step = msg.step
		return m, nil

	case dumpCompleteMsg:
		m.done = true
		m.counts = msg.counts
		m.total = msg.total
		m.err = msg.err
		if m.err == nil {
			m.steps = append(m.steps, m.step)
			m.step = "Complete!"
		}
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", "d"))):
			// Start dump if not started, or go back if done
			if !m.started {
				m.started = true
				m.step = "Starting dump..."
				return m, tea.Batch(m.spinner.Tick, m.runDump())
			}
			if m.done {
				return m, func() tea.Msg { return Navigate("dashboard") }
			}
		}
	}

	return m, nil
}

// SetSize updates the dump dimensions
func (m *DumpModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the dump screen (legacy)
func (m *DumpModel) View() string {
	return m.ViewContent(m.width, m.height)
}

// ViewContent renders just the content area (for use in layout)
func (m *DumpModel) ViewContent(width, height int) string {
	var b strings.Builder

	// Show prompt if not started
	if !m.started {
		b.WriteString(styles.TitleStyle.Render("Dump Packages"))
		b.WriteString("\n\n")
		b.WriteString("This will update your Brewfile with all currently installed packages.\n\n")
		b.WriteString("The following will be collected:\n")
		b.WriteString("  • Homebrew taps, formulae, and casks\n")
		b.WriteString("  • VSCode and Cursor extensions\n")
		b.WriteString("  • Go tools\n")
		b.WriteString("  • Mac App Store apps\n\n")
		b.WriteString(styles.SelectedStyle.Render("Press Enter or 'd' to start dump"))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
		return b.String()
	}

	// Progress
	for _, step := range m.steps {
		b.WriteString(styles.SelectedStyle.Render("✓ " + step))
		b.WriteString("\n")
	}

	if !m.done {
		b.WriteString(m.spinner.View() + " " + m.step)
		b.WriteString("\n")
	}

	// Results
	if m.done && m.err == nil {
		b.WriteString("\n")
		b.WriteString(m.renderCounts())
		b.WriteString("\n\n")
		b.WriteString(styles.SelectedStyle.Render(fmt.Sprintf("✓ Dumped %d packages to Brewfile", m.total)))
		b.WriteString("\n\n")
		b.WriteString("Press Enter to return to dashboard")
	}

	return b.String()
}

func (m *DumpModel) renderCounts() string {
	var parts []string
	types := []struct {
		name string
		icon string
	}{
		{"tap", "🚰"},
		{"brew", "🍺"},
		{"cask", "📦"},
		{"vscode", "💻"},
		{"cursor", "✏️"},
		{"antigravity", "🚀"},
		{"go", "🔷"},
		{"mas", "🍎"},
	}

	for _, t := range types {
		if count, ok := m.counts[t.name]; ok && count > 0 {
			parts = append(parts, fmt.Sprintf("%s %s: %d", t.icon, t.name, count))
		}
	}

	return strings.Join(parts, "   ")
}
