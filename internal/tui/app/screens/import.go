package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/andrew-sameh/brewsync/internal/brewfile"
	"github.com/andrew-sameh/brewsync/internal/config"
	"github.com/andrew-sameh/brewsync/internal/tui/selection"
	"github.com/andrew-sameh/brewsync/internal/tui/styles"
)

// ImportPhase represents the current phase of the import flow
type ImportPhase int

const (
	ImportPhaseLoading ImportPhase = iota
	ImportPhaseSelect
	ImportPhaseInstalling
	ImportPhaseDone
)

// ImportModel is the model for the import screen
type ImportModel struct {
	config      *config.Config
	width       int
	height      int
	source      string
	phase       ImportPhase
	allPackages brewfile.Packages // All packages including ignored
	packages    brewfile.Packages // Filtered packages (respects showIgnored)
	selection   *selection.Model
	err         error
	installed   int
	failed      int
	showIgnored bool
}

// NewImportModel creates a new import model
func NewImportModel(cfg *config.Config) *ImportModel {
	source := ""
	if cfg != nil {
		source = cfg.DefaultSource
	}
	return &ImportModel{
		config: cfg,
		width:  80,
		height: 24,
		source: source,
		phase:  ImportPhaseLoading,
	}
}

type importLoadedMsg struct {
	packages brewfile.Packages
	err      error
}

type importDoneMsg struct {
	installed int
	failed    int
}

// Init initializes the import model
func (m *ImportModel) Init() tea.Cmd {
	return func() tea.Msg {
		if m.config == nil {
			return importLoadedMsg{err: fmt.Errorf("no config loaded")}
		}

		currentMachine, ok := m.config.GetCurrentMachine()
		if !ok {
			return importLoadedMsg{err: fmt.Errorf("current machine not found")}
		}

		sourceMachine, ok := m.config.GetMachine(m.source)
		if !ok {
			return importLoadedMsg{err: fmt.Errorf("source machine %q not found", m.source)}
		}

		// Parse both Brewfiles
		currentPkgs, err := brewfile.Parse(currentMachine.Brewfile)
		if err != nil {
			// Treat as empty if file doesn't exist
			currentPkgs = brewfile.Packages{}
		}

		sourcePkgs, err := brewfile.Parse(sourceMachine.Brewfile)
		if err != nil {
			return importLoadedMsg{err: fmt.Errorf("failed to parse source Brewfile: %w", err)}
		}

		// Get packages to import (in source but not in current)
		diff := brewfile.Diff(sourcePkgs, currentPkgs)
		return importLoadedMsg{packages: diff.Additions}
	}
}

// Update handles messages
func (m *ImportModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.selection != nil {
			newSel, cmd := m.selection.Update(msg)
			sel := newSel.(selection.Model)
			m.selection = &sel
			return m, cmd
		}
		return m, nil

	case importLoadedMsg:
		m.err = msg.err
		m.allPackages = msg.packages
		m.packages = m.filterPackages(m.allPackages)
		if m.err == nil && len(m.packages) > 0 {
			m.phase = ImportPhaseSelect
			m.rebuildSelection()
		} else if len(m.packages) == 0 && len(m.allPackages) == 0 {
			m.phase = ImportPhaseDone
		} else if len(m.packages) == 0 {
			// All packages are ignored
			m.phase = ImportPhaseDone
		}
		return m, nil

	case ShowIgnoredMsg:
		m.showIgnored = msg.Show
		m.packages = m.filterPackages(m.allPackages)
		if m.phase == ImportPhaseSelect {
			m.rebuildSelection()
		}
		return m, nil

	case importDoneMsg:
		m.phase = ImportPhaseDone
		m.installed = msg.installed
		m.failed = msg.failed
		return m, nil

	case tea.KeyMsg:
		// Handle selection mode
		if m.phase == ImportPhaseSelect && m.selection != nil {
			newSel, cmd := m.selection.Update(msg)
			sel := newSel.(selection.Model)
			m.selection = &sel

			// Check if selection is done
			if m.selection.Confirmed() {
				// Would start installation here
				m.phase = ImportPhaseDone
				return m, nil
			}
			if m.selection.Cancelled() {
				return m, func() tea.Msg { return Navigate("dashboard") }
			}
			return m, cmd
		}

		// Other phases
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "b"))):
			return m, func() tea.Msg { return Navigate("dashboard") }
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.phase == ImportPhaseDone {
				return m, func() tea.Msg { return Navigate("dashboard") }
			}
		}
	}

	return m, nil
}

// SetSize updates the import dimensions
func (m *ImportModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the import screen (legacy)
func (m *ImportModel) View() string {
	return m.ViewContent(m.width, m.height)
}

// ViewContent renders just the content area (for use in layout)
func (m *ImportModel) ViewContent(width, height int) string {
	var b strings.Builder

	switch m.phase {
	case ImportPhaseLoading:
		b.WriteString(styles.DimmedStyle.Render("Loading packages..."))

	case ImportPhaseSelect:
		if m.selection != nil {
			return m.selection.View()
		}

	case ImportPhaseInstalling:
		b.WriteString(styles.DimmedStyle.Render("Installing..."))

	case ImportPhaseDone:
		if m.err != nil {
			b.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
		} else if len(m.allPackages) == 0 {
			b.WriteString(styles.SelectedStyle.Render("✓ "))
			b.WriteString("Already in sync with " + m.source + "!")
		} else if len(m.packages) == 0 {
			b.WriteString(styles.SelectedStyle.Render("✓ "))
			b.WriteString("Already in sync with " + m.source + "!")
			b.WriteString("\n")
			b.WriteString(styles.DimmedStyle.Render(fmt.Sprintf("(%d ignored packages, press h to show)", len(m.allPackages))))
		} else {
			b.WriteString(styles.SelectedStyle.Render(fmt.Sprintf("✓ Installed %d packages", m.installed)))
			if m.failed > 0 {
				b.WriteString("\n")
				b.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("✗ Failed %d packages", m.failed)))
			}
		}
	}

	return b.String()
}

// filterPackages filters packages based on showIgnored setting
func (m *ImportModel) filterPackages(pkgs brewfile.Packages) brewfile.Packages {
	if m.showIgnored || m.config == nil {
		return pkgs
	}

	var filtered brewfile.Packages
	for _, pkg := range pkgs {
		isIgnored := m.config.IsCategoryIgnored(m.config.CurrentMachine, string(pkg.Type)) ||
			m.config.IsPackageIgnored(m.config.CurrentMachine, pkg.ID())
		if !isIgnored {
			filtered = append(filtered, pkg)
		}
	}
	return filtered
}

// rebuildSelection creates/updates the selection model with current packages
func (m *ImportModel) rebuildSelection() {
	sel := selection.New(fmt.Sprintf("Import from %s", m.source), m.packages)
	// Pre-select all packages
	selected := make(map[string]bool)
	for _, pkg := range m.packages {
		selected[pkg.ID()] = true
	}
	sel.SetSelected(selected)
	m.selection = &sel
}
