package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/andrew-sameh/brewsync/internal/brewfile"
	"github.com/andrew-sameh/brewsync/internal/config"
	"github.com/andrew-sameh/brewsync/internal/tui/styles"
)

// SyncPhase represents the current phase of the sync flow
type SyncPhase int

const (
	SyncPhaseLoading SyncPhase = iota
	SyncPhasePreview
	SyncPhaseConfirm
	SyncPhaseExecuting
	SyncPhaseDone
)

// SyncModel is the model for the sync screen
type SyncModel struct {
	config       *config.Config
	width        int
	height       int
	source       string
	phase        SyncPhase
	allAdditions brewfile.Packages // All additions including ignored
	allRemovals  brewfile.Packages // All removals including ignored
	additions    brewfile.Packages // Filtered additions
	removals     brewfile.Packages // Filtered removals
	protected    brewfile.Packages
	err          error
	installed    int
	removed      int
	failed       int
	showConfirm  bool
	showIgnored  bool
}

// NewSyncModel creates a new sync model
func NewSyncModel(cfg *config.Config) *SyncModel {
	source := ""
	if cfg != nil {
		source = cfg.DefaultSource
	}
	return &SyncModel{
		config: cfg,
		width:  80,
		height: 24,
		source: source,
		phase:  SyncPhaseLoading,
	}
}

type syncLoadedMsg struct {
	additions brewfile.Packages
	removals  brewfile.Packages
	protected brewfile.Packages
	err       error
}

type syncDoneMsg struct {
	installed int
	removed   int
	failed    int
}

// Init initializes the sync model
func (m *SyncModel) Init() tea.Cmd {
	return func() tea.Msg {
		if m.config == nil {
			return syncLoadedMsg{err: fmt.Errorf("no config loaded")}
		}

		currentMachine, ok := m.config.GetCurrentMachine()
		if !ok {
			return syncLoadedMsg{err: fmt.Errorf("current machine not found")}
		}

		sourceMachine, ok := m.config.GetMachine(m.source)
		if !ok {
			return syncLoadedMsg{err: fmt.Errorf("source machine %q not found", m.source)}
		}

		// Parse both Brewfiles
		currentPkgs, err := brewfile.Parse(currentMachine.Brewfile)
		if err != nil {
			currentPkgs = brewfile.Packages{}
		}

		sourcePkgs, err := brewfile.Parse(sourceMachine.Brewfile)
		if err != nil {
			return syncLoadedMsg{err: fmt.Errorf("failed to parse source Brewfile: %w", err)}
		}

		diff := brewfile.Diff(sourcePkgs, currentPkgs)

		// Filter out machine-specific packages from removals
		var removals, protected brewfile.Packages
		machineSpecific := m.config.GetMachineSpecificPackages()
		currentMachineSpecific := machineSpecific[m.config.CurrentMachine]

		for _, pkg := range diff.Removals {
			isProtected := false
			for _, ms := range currentMachineSpecific {
				if pkg.ID() == ms {
					isProtected = true
					break
				}
			}
			if isProtected {
				protected = append(protected, pkg)
			} else {
				removals = append(removals, pkg)
			}
		}

		return syncLoadedMsg{
			additions: diff.Additions,
			removals:  removals,
			protected: protected,
		}
	}
}

// Update handles messages
func (m *SyncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case syncLoadedMsg:
		m.err = msg.err
		m.allAdditions = msg.additions
		m.allRemovals = msg.removals
		m.protected = msg.protected
		m.additions = m.filterPackages(m.allAdditions)
		m.removals = m.filterPackages(m.allRemovals)
		if m.err == nil {
			m.phase = SyncPhasePreview
		}
		return m, nil

	case ShowIgnoredMsg:
		m.showIgnored = msg.Show
		m.additions = m.filterPackages(m.allAdditions)
		m.removals = m.filterPackages(m.allRemovals)
		return m, nil

	case syncDoneMsg:
		m.phase = SyncPhaseDone
		m.installed = msg.installed
		m.removed = msg.removed
		m.failed = msg.failed
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "b"))):
			if m.showConfirm {
				m.showConfirm = false
				return m, nil
			}
			return m, func() tea.Msg { return Navigate("dashboard") }

		case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
			if m.phase == SyncPhasePreview && (len(m.additions) > 0 || len(m.removals) > 0) {
				m.showConfirm = true
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("y"))):
			if m.showConfirm {
				m.showConfirm = false
				m.phase = SyncPhaseExecuting
				// Would execute sync here
				return m, func() tea.Msg {
					return syncDoneMsg{
						installed: len(m.additions),
						removed:   len(m.removals),
					}
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
			if m.showConfirm {
				m.showConfirm = false
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.phase == SyncPhaseDone {
				return m, func() tea.Msg { return Navigate("dashboard") }
			}
		}
	}

	return m, nil
}

// SetSize updates the sync dimensions
func (m *SyncModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the sync screen (legacy)
func (m *SyncModel) View() string {
	return m.ViewContent(m.width, m.height)
}

// ViewContent renders just the content area (for use in layout)
func (m *SyncModel) ViewContent(width, height int) string {
	var b strings.Builder

	// Title showing source -> target
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatMauve)
	b.WriteString(titleStyle.Render(fmt.Sprintf("Sync: %s → %s", m.source, m.config.CurrentMachine)))
	b.WriteString("\n\n")

	if m.phase == SyncPhaseLoading {
		b.WriteString(styles.DimmedStyle.Render("Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
		return b.String()
	}

	switch m.phase {
	case SyncPhasePreview:
		if len(m.additions) == 0 && len(m.removals) == 0 {
			b.WriteString(styles.SelectedStyle.Render("✓ "))
			b.WriteString("Already in sync!")
			// Show ignored count
			ignoredCount := len(m.allAdditions) - len(m.additions) + len(m.allRemovals) - len(m.removals)
			if !m.showIgnored && ignoredCount > 0 {
				b.WriteString("\n")
				b.WriteString(styles.DimmedStyle.Render(fmt.Sprintf("(%d ignored changes, press h to show)", ignoredCount)))
			}
		} else {
			// Show additions
			if len(m.additions) > 0 {
				b.WriteString(styles.AddedStyle.Render(fmt.Sprintf("To Install (+%d)", len(m.additions))))
				b.WriteString("\n")
				for _, pkg := range m.additions {
					b.WriteString(styles.AddedStyle.Render(fmt.Sprintf("  + %s: %s", pkg.Type, pkg.Name)))
					b.WriteString("\n")
				}
				b.WriteString("\n")
			}

			// Show removals
			if len(m.removals) > 0 {
				b.WriteString(styles.RemovedStyle.Render(fmt.Sprintf("To Remove (-%d)", len(m.removals))))
				b.WriteString("\n")
				for _, pkg := range m.removals {
					b.WriteString(styles.RemovedStyle.Render(fmt.Sprintf("  - %s: %s", pkg.Type, pkg.Name)))
					b.WriteString("\n")
				}
				b.WriteString("\n")
			}

			// Show protected
			if len(m.protected) > 0 {
				b.WriteString(styles.WarningStyle.Render(fmt.Sprintf("Protected (%d)", len(m.protected))))
				b.WriteString("\n")
				for _, pkg := range m.protected {
					b.WriteString(styles.WarningStyle.Render(fmt.Sprintf("  ⚠ %s: %s (machine-specific)", pkg.Type, pkg.Name)))
					b.WriteString("\n")
				}
			}
		}

		if m.showConfirm {
			b.WriteString("\n")
			b.WriteString(styles.WarningStyle.Render("Apply changes? (y/n)"))
		}

	case SyncPhaseExecuting:
		b.WriteString(styles.DimmedStyle.Render("Syncing..."))

	case SyncPhaseDone:
		b.WriteString(styles.SelectedStyle.Render("✓ Sync complete!"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("Installed: %d  Removed: %d", m.installed, m.removed))
		if m.failed > 0 {
			b.WriteString(fmt.Sprintf("  Failed: %d", m.failed))
		}
	}

	return b.String()
}

// filterPackages filters packages based on showIgnored setting
func (m *SyncModel) filterPackages(pkgs brewfile.Packages) brewfile.Packages {
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
