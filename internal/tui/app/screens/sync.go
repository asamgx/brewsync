package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/config"
	"github.com/asamgx/brewsync/internal/installer"
	"github.com/asamgx/brewsync/internal/tui/styles"
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

// SyncColumn represents which column is focused
type SyncColumn int

const (
	SyncColumnAdditions SyncColumn = iota
	SyncColumnRemovals
)

// syncItem represents a displayable item in a sync column (header or package)
type syncItem struct {
	isHeader    bool
	headerType  brewfile.PackageType
	headerCount int
	pkg         brewfile.Package
	isIgnored   bool
}

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
	addItems     []syncItem // Flattened additions with headers
	remItems     []syncItem // Flattened removals with headers
	column       SyncColumn // Current column focus
	addCursor    int        // Cursor in additions
	remCursor    int        // Cursor in removals
	addOffset    int        // Scroll offset for additions
	remOffset    int        // Scroll offset for removals
	err          error
	installed    int
	removed      int
	failed       int
	showConfirm  bool
	showIgnored  bool

	// Execution state
	spinner       spinner.Model
	currentPkg    string
	currentAction string // "Installing" or "Removing"
	progress      int
	total         int
	results       []syncResult
}

type syncResult struct {
	pkg     brewfile.Package
	action  string // "installed" or "removed"
	success bool
	err     error
}

// NewSyncModel creates a new sync model
func NewSyncModel(cfg *config.Config) *SyncModel {
	source := ""
	if cfg != nil {
		source = cfg.DefaultSource
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.CatMauve)

	return &SyncModel{
		config:  cfg,
		width:   80,
		height:  24,
		source:  source,
		phase:   SyncPhaseLoading,
		spinner: s,
	}
}

type syncLoadedMsg struct {
	additions brewfile.Packages
	removals  brewfile.Packages
	protected brewfile.Packages
	err       error
}

type syncProgressMsg struct {
	pkg     brewfile.Package
	action  string
	current int
	total   int
	err     error
	done    bool
}

type syncDoneMsg struct {
	installed int
	removed   int
	failed    int
	results   []syncResult
}

// Init initializes the sync model
func (m *SyncModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
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
		},
	)
}

// Update handles messages
func (m *SyncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.phase == SyncPhaseLoading || m.phase == SyncPhaseExecuting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
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
			m.buildItems()
			// Start in additions if available, otherwise removals
			if len(m.addItems) == 0 && len(m.remItems) > 0 {
				m.column = SyncColumnRemovals
			}
		}
		return m, nil

	case ShowIgnoredMsg:
		m.showIgnored = msg.Show
		m.additions = m.filterPackages(m.allAdditions)
		m.removals = m.filterPackages(m.allRemovals)
		m.buildItems()
		return m, nil

	case syncProgressMsg:
		m.currentPkg = msg.pkg.Name
		m.currentAction = msg.action
		m.progress = msg.current
		m.total = msg.total

		if msg.done {
			// This package is done, add to results
			result := syncResult{
				pkg:     msg.pkg,
				action:  msg.action,
				success: msg.err == nil,
				err:     msg.err,
			}
			m.results = append(m.results, result)
			if msg.err != nil {
				m.failed++
			} else if msg.action == "Installing" {
				m.installed++
			} else {
				m.removed++
			}
		}
		return m, nil

	case syncDoneMsg:
		m.phase = SyncPhaseDone
		m.installed = msg.installed
		m.removed = msg.removed
		m.failed = msg.failed
		m.results = msg.results
		return m, nil

	case tea.KeyMsg:
		// Handle confirmation dialog
		if m.showConfirm {
			switch msg.String() {
			case "y", "Y":
				m.showConfirm = false
				m.phase = SyncPhaseExecuting
				return m, tea.Batch(m.spinner.Tick, m.executeSync())
			case "n", "N", "esc":
				m.showConfirm = false
				return m, nil
			}
			return m, nil
		}

		// Preview phase navigation
		if m.phase == SyncPhasePreview {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
				m.moveUp()
			case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
				m.moveDown()
			case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
				m.moveLeft()
			case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
				m.moveRight()
			case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
				m.jumpToTop()
			case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
				m.jumpToBottom()
			case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
				if len(m.additions) > 0 || len(m.removals) > 0 {
					m.showConfirm = true
				}
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				return m, func() tea.Msg { return Navigate("dashboard") }
			}
			return m, nil
		}

		// Done phase
		if m.phase == SyncPhaseDone {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter", "esc"))):
				return m, func() tea.Msg { return Navigate("dashboard") }
			}
		}
	}

	return m, nil
}

// executeSync runs the actual sync operation
func (m *SyncModel) executeSync() tea.Cmd {
	return func() tea.Msg {
		mgr := installer.NewManager()
		var results []syncResult
		var installed, removed, failed int

		// Install additions
		for _, pkg := range m.additions {
			// We can't send messages from here directly, so we'll just execute
			err := mgr.Install(pkg)
			result := syncResult{
				pkg:     pkg,
				action:  "installed",
				success: err == nil,
				err:     err,
			}
			results = append(results, result)
			if err != nil {
				failed++
			} else {
				installed++
			}
		}

		// Remove removals
		for _, pkg := range m.removals {
			err := mgr.Uninstall(pkg)
			result := syncResult{
				pkg:     pkg,
				action:  "removed",
				success: err == nil,
				err:     err,
			}
			results = append(results, result)
			if err != nil {
				failed++
			} else {
				removed++
			}
		}

		return syncDoneMsg{
			installed: installed,
			removed:   removed,
			failed:    failed,
			results:   results,
		}
	}
}

// buildItems creates flattened lists with category headers
func (m *SyncModel) buildItems() {
	m.addItems = m.buildItemsForPackages(m.additions, true)
	m.remItems = m.buildItemsForPackages(m.removals, false)
}

// buildItemsForPackages creates a flattened list with headers for a package list
func (m *SyncModel) buildItemsForPackages(pkgs brewfile.Packages, isAdditions bool) []syncItem {
	var items []syncItem
	byType := pkgs.ByType()
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

	// Get all packages for ignore checking
	var allPkgs brewfile.Packages
	if isAdditions {
		allPkgs = m.allAdditions
	} else {
		allPkgs = m.allRemovals
	}

	for _, t := range types {
		typePkgs := byType[t]
		if len(typePkgs) == 0 {
			continue
		}

		// Count visible and ignored packages
		var visiblePkgs []brewfile.Package
		var ignoredPkgs []brewfile.Package

		// Check all packages of this type from the full list
		allByType := allPkgs.ByType()
		for _, pkg := range allByType[t] {
			isIgnored := m.config != nil && (m.config.IsCategoryIgnored(m.config.CurrentMachine, string(pkg.Type)) ||
				m.config.IsPackageIgnored(m.config.CurrentMachine, pkg.ID()))
			if isIgnored {
				ignoredPkgs = append(ignoredPkgs, pkg)
			} else {
				visiblePkgs = append(visiblePkgs, pkg)
			}
		}

		// Skip this category if no visible items and not showing ignored
		if !m.showIgnored && len(visiblePkgs) == 0 {
			continue
		}

		// Calculate header count based on what's shown
		headerCount := len(visiblePkgs)
		if m.showIgnored {
			headerCount = len(visiblePkgs) + len(ignoredPkgs)
		}

		// Add header
		items = append(items, syncItem{
			isHeader:    true,
			headerType:  t,
			headerCount: headerCount,
		})

		// Add visible packages first
		for _, pkg := range visiblePkgs {
			items = append(items, syncItem{pkg: pkg, isIgnored: false})
		}

		// Add ignored packages if showing ignored
		if m.showIgnored {
			for _, pkg := range ignoredPkgs {
				items = append(items, syncItem{pkg: pkg, isIgnored: true})
			}
		}
	}
	return items
}

// Navigation methods
func (m *SyncModel) moveUp() {
	if m.column == SyncColumnAdditions {
		if m.addCursor > 0 {
			m.addCursor--
			m.adjustAddOffset()
		}
	} else {
		if m.remCursor > 0 {
			m.remCursor--
			m.adjustRemOffset()
		}
	}
}

func (m *SyncModel) moveDown() {
	if m.column == SyncColumnAdditions {
		if m.addCursor < len(m.addItems)-1 {
			m.addCursor++
			m.adjustAddOffset()
		}
	} else {
		if m.remCursor < len(m.remItems)-1 {
			m.remCursor++
			m.adjustRemOffset()
		}
	}
}

func (m *SyncModel) moveLeft() {
	if len(m.addItems) > 0 {
		m.column = SyncColumnAdditions
	}
}

func (m *SyncModel) moveRight() {
	if len(m.remItems) > 0 {
		m.column = SyncColumnRemovals
	}
}

func (m *SyncModel) jumpToTop() {
	if m.column == SyncColumnAdditions {
		m.addCursor = 0
		m.addOffset = 0
	} else {
		m.remCursor = 0
		m.remOffset = 0
	}
}

func (m *SyncModel) jumpToBottom() {
	if m.column == SyncColumnAdditions {
		if len(m.addItems) > 0 {
			m.addCursor = len(m.addItems) - 1
			m.adjustAddOffset()
		}
	} else {
		if len(m.remItems) > 0 {
			m.remCursor = len(m.remItems) - 1
			m.adjustRemOffset()
		}
	}
}

func (m *SyncModel) adjustAddOffset() {
	visibleHeight := m.getColumnHeight()
	if m.addCursor < m.addOffset {
		m.addOffset = m.addCursor
	}
	if m.addCursor >= m.addOffset+visibleHeight {
		m.addOffset = m.addCursor - visibleHeight + 1
	}
}

func (m *SyncModel) adjustRemOffset() {
	visibleHeight := m.getColumnHeight()
	if m.remCursor < m.remOffset {
		m.remOffset = m.remCursor
	}
	if m.remCursor >= m.remOffset+visibleHeight {
		m.remOffset = m.remCursor - visibleHeight + 1
	}
}

func (m *SyncModel) getColumnHeight() int {
	h := m.height - 6 // Title, action bar, and padding
	if h < 1 {
		h = 1
	}
	return h
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

	switch m.phase {
	case SyncPhaseLoading:
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(styles.DimmedStyle.Render("Loading..."))
		return b.String()

	case SyncPhasePreview:
		return m.renderPreview(width, height)

	case SyncPhaseExecuting:
		return m.renderExecuting()

	case SyncPhaseDone:
		return m.renderDone()
	}

	if m.err != nil {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
	}

	return b.String()
}

func (m *SyncModel) renderPreview(width, height int) string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatMauve)
	b.WriteString(titleStyle.Render(fmt.Sprintf("Sync: %s → %s", m.source, m.config.CurrentMachine)))
	b.WriteString("\n\n")

	// No changes
	if len(m.additions) == 0 && len(m.removals) == 0 {
		b.WriteString(styles.SelectedStyle.Render("✓ "))
		b.WriteString("Already in sync!")

		// Show ignored count
		ignoredCount := len(m.allAdditions) - len(m.additions) + len(m.allRemovals) - len(m.removals)
		if !m.showIgnored && ignoredCount > 0 {
			b.WriteString("\n")
			b.WriteString(styles.DimmedStyle.Render(fmt.Sprintf("(%d ignored changes, press H to show)", ignoredCount)))
		}

		// Show protected
		if len(m.protected) > 0 {
			b.WriteString("\n\n")
			b.WriteString(styles.WarningStyle.Render(fmt.Sprintf("Protected packages (%d):", len(m.protected))))
			b.WriteString("\n")
			for _, pkg := range m.protected {
				b.WriteString(styles.DimmedStyle.Render(fmt.Sprintf("  ⚠ %s:%s (machine-specific)", pkg.Type, pkg.Name)))
				b.WriteString("\n")
			}
		}
		return b.String()
	}

	// Calculate column widths
	colWidth := (width - 4) / 2
	if colWidth < 20 {
		colWidth = 20
	}
	visibleHeight := m.getColumnHeight()

	// Build left column (additions)
	leftLines := m.renderColumn(
		m.addItems,
		fmt.Sprintf("TO INSTALL (+%d)", len(m.additions)),
		"+",
		colWidth,
		visibleHeight,
		m.addCursor,
		m.addOffset,
		m.column == SyncColumnAdditions,
		styles.AddedStyle,
	)

	// Build right column (removals)
	rightLines := m.renderColumn(
		m.remItems,
		fmt.Sprintf("TO REMOVE (-%d)", len(m.removals)),
		"−",
		colWidth,
		visibleHeight,
		m.remCursor,
		m.remOffset,
		m.column == SyncColumnRemovals,
		styles.RemovedStyle,
	)

	// Combine columns side by side
	maxLines := visibleHeight + 2
	if len(leftLines) > maxLines {
		maxLines = len(leftLines)
	}
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	for i := 0; i < maxLines; i++ {
		left := ""
		right := ""
		if i < len(leftLines) {
			left = leftLines[i]
		}
		if i < len(rightLines) {
			right = rightLines[i]
		}

		leftWidth := lipgloss.Width(left)
		if leftWidth < colWidth {
			left += strings.Repeat(" ", colWidth-leftWidth)
		}

		b.WriteString(left)
		b.WriteString("  ")
		b.WriteString(right)
		b.WriteString("\n")
	}

	// Show protected packages if any
	if len(m.protected) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.WarningStyle.Render(fmt.Sprintf("⚠ %d protected packages will not be removed (machine-specific)", len(m.protected))))
		b.WriteString("\n")
	}

	// Action bar
	b.WriteString("\n")
	if m.showConfirm {
		confirmStyle := lipgloss.NewStyle().
			Background(styles.CatSurface0).
			Foreground(styles.CatYellow).
			Padding(0, 1).
			Bold(true)
		b.WriteString(confirmStyle.Render(fmt.Sprintf("Apply %d changes? (y/n)", len(m.additions)+len(m.removals))))
	} else {
		actionStyle := lipgloss.NewStyle().Foreground(styles.CatSubtext0)
		b.WriteString(actionStyle.Render("Press "))
		b.WriteString(lipgloss.NewStyle().Foreground(styles.CatGreen).Bold(true).Render("a"))
		b.WriteString(actionStyle.Render(" to apply • "))
		b.WriteString(lipgloss.NewStyle().Foreground(styles.CatText).Render("esc"))
		b.WriteString(actionStyle.Render(" to cancel • "))
		b.WriteString(lipgloss.NewStyle().Foreground(styles.CatText).Render("h/l"))
		b.WriteString(actionStyle.Render(" switch columns • "))
		b.WriteString(lipgloss.NewStyle().Foreground(styles.CatText).Render("j/k"))
		b.WriteString(actionStyle.Render(" navigate"))
	}

	return b.String()
}

func (m *SyncModel) renderColumn(
	items []syncItem,
	title string,
	prefix string,
	width int,
	visibleHeight int,
	cursor int,
	offset int,
	focused bool,
	baseStyle lipgloss.Style,
) []string {
	var lines []string

	// Column header
	headerStyle := baseStyle.Bold(true)
	if focused {
		headerStyle = headerStyle.Underline(true)
	}
	lines = append(lines, headerStyle.Render(title))
	lines = append(lines, styles.DimmedStyle.Render(strings.Repeat("─", width-2)))

	if len(items) == 0 {
		lines = append(lines, styles.DimmedStyle.Render("(none)"))
		return lines
	}

	// Package list with scrolling
	endIdx := offset + visibleHeight
	if endIdx > len(items) {
		endIdx = len(items)
	}

	for i := offset; i < endIdx; i++ {
		item := items[i]
		isCursor := i == cursor && focused

		if item.isHeader {
			// Category header with icon
			icon := getTypeIcon(item.headerType)
			catStyle := styles.GetCategoryStyle(string(item.headerType)).Bold(true)
			linePrefix := "  "
			if isCursor {
				linePrefix = styles.CursorStyle.Render("> ")
			}
			line := linePrefix + catStyle.Render(fmt.Sprintf("%s %s (%d)", icon, item.headerType, item.headerCount))
			lines = append(lines, line)
		} else {
			// Package line
			linePrefix := "    "
			if isCursor {
				linePrefix = styles.CursorStyle.Render("> ") + "  "
			}

			// Truncate name to fit column
			maxNameLen := width - 12
			name := item.pkg.Name
			if len(name) > maxNameLen {
				name = name[:maxNameLen-3] + "..."
			}

			var line string
			if item.isIgnored {
				nameStyle := styles.DimmedStyle
				line = linePrefix + nameStyle.Render(prefix+" "+name+" (ignored)")
			} else {
				nameStyle := baseStyle
				if isCursor {
					nameStyle = lipgloss.NewStyle().Foreground(styles.CatMauve).Bold(true)
				}
				line = linePrefix + nameStyle.Render(prefix+" "+name)
			}
			lines = append(lines, line)
		}
	}

	// Scroll indicator at bottom
	if len(items) > visibleHeight {
		scrollInfo := fmt.Sprintf(" %d/%d ", cursor+1, len(items))
		lines = append(lines, styles.DimmedStyle.Render(scrollInfo))
	}

	return lines
}

func (m *SyncModel) renderExecuting() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatMauve)
	b.WriteString(titleStyle.Render(fmt.Sprintf("Sync: %s → %s", m.source, m.config.CurrentMachine)))
	b.WriteString("\n\n")

	// Spinner with current action
	b.WriteString(m.spinner.View())
	b.WriteString(" ")
	b.WriteString(styles.DimmedStyle.Render("Syncing packages..."))
	b.WriteString("\n\n")

	// Progress
	total := len(m.additions) + len(m.removals)
	progress := m.installed + m.removed + m.failed

	progressStyle := lipgloss.NewStyle().Foreground(styles.CatBlue)
	b.WriteString(progressStyle.Render(fmt.Sprintf("Progress: %d/%d", progress, total)))
	b.WriteString("\n\n")

	// Recent results (last 5)
	if len(m.results) > 0 {
		start := 0
		if len(m.results) > 5 {
			start = len(m.results) - 5
		}
		for _, r := range m.results[start:] {
			if r.success {
				icon := styles.SelectedStyle.Render("✓")
				action := r.action
				b.WriteString(fmt.Sprintf("%s %s %s:%s\n", icon, action, r.pkg.Type, r.pkg.Name))
			} else {
				icon := styles.ErrorStyle.Render("✗")
				b.WriteString(fmt.Sprintf("%s failed %s:%s\n", icon, r.pkg.Type, r.pkg.Name))
			}
		}
	}

	return b.String()
}

func (m *SyncModel) renderDone() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatMauve)
	b.WriteString(titleStyle.Render(fmt.Sprintf("Sync: %s → %s", m.source, m.config.CurrentMachine)))
	b.WriteString("\n\n")

	// Summary
	if m.failed == 0 {
		b.WriteString(styles.SelectedStyle.Render("✓ Sync complete!"))
	} else {
		b.WriteString(styles.WarningStyle.Render("⚠ Sync completed with errors"))
	}
	b.WriteString("\n\n")

	// Stats
	if m.installed > 0 {
		b.WriteString(styles.AddedStyle.Render(fmt.Sprintf("  +%d installed", m.installed)))
		b.WriteString("\n")
	}
	if m.removed > 0 {
		b.WriteString(styles.RemovedStyle.Render(fmt.Sprintf("  -%d removed", m.removed)))
		b.WriteString("\n")
	}
	if m.failed > 0 {
		b.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("  ✗%d failed", m.failed)))
		b.WriteString("\n")
	}

	// Show failed packages
	if m.failed > 0 {
		b.WriteString("\n")
		b.WriteString(styles.ErrorStyle.Render("Failed packages:"))
		b.WriteString("\n")
		for _, r := range m.results {
			if !r.success {
				errMsg := ""
				if r.err != nil {
					errMsg = ": " + r.err.Error()
				}
				b.WriteString(styles.DimmedStyle.Render(fmt.Sprintf("  • %s:%s%s", r.pkg.Type, r.pkg.Name, errMsg)))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.DimmedStyle.Render("Press enter to continue"))

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
