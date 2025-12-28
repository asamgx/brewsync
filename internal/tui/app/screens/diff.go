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

// DiffColumn represents which column is focused
type DiffColumn int

const (
	DiffColumnAdditions DiffColumn = iota
	DiffColumnRemovals
)

// diffItem represents a displayable item in a diff column (header or package)
type diffItem struct {
	isHeader    bool
	headerType  brewfile.PackageType
	headerCount int
	pkg         brewfile.Package
	isIgnored   bool
}

// DiffModel is the model for the diff screen
type DiffModel struct {
	config       *config.Config
	width        int
	height       int
	source       string
	additions    brewfile.Packages
	removals     brewfile.Packages
	addItems     []diffItem // Flattened additions with headers
	remItems     []diffItem // Flattened removals with headers
	column       DiffColumn // Current column focus
	addCursor    int        // Cursor in additions
	remCursor    int        // Cursor in removals
	addOffset    int        // Scroll offset for additions
	remOffset    int        // Scroll offset for removals
	loading      bool
	err          error
	showIgnored  bool

	// Confirmation dialog
	showConfirm   bool
	confirmAction string // "install" or "uninstall"
	confirmPkg    brewfile.Package

	// Task state
	taskRunning bool
}

// NewDiffModel creates a new diff model
func NewDiffModel(cfg *config.Config) *DiffModel {
	source := ""
	if cfg != nil {
		source = cfg.DefaultSource
	}
	return &DiffModel{
		config:  cfg,
		width:   80,
		height:  24,
		source:  source,
		loading: true,
	}
}

type diffLoadedMsg struct {
	additions brewfile.Packages
	removals  brewfile.Packages
	err       error
}

// Init initializes the diff model
func (m *DiffModel) Init() tea.Cmd {
	return func() tea.Msg {
		if m.config == nil {
			return diffLoadedMsg{err: fmt.Errorf("no config loaded")}
		}

		currentMachine, ok := m.config.GetCurrentMachine()
		if !ok {
			return diffLoadedMsg{err: fmt.Errorf("current machine not found")}
		}

		sourceMachine, ok := m.config.GetMachine(m.source)
		if !ok {
			return diffLoadedMsg{err: fmt.Errorf("source machine %q not found", m.source)}
		}

		// Parse both Brewfiles
		currentPkgs, err := brewfile.Parse(currentMachine.Brewfile)
		if err != nil {
			return diffLoadedMsg{err: fmt.Errorf("failed to parse current Brewfile: %w", err)}
		}

		sourcePkgs, err := brewfile.Parse(sourceMachine.Brewfile)
		if err != nil {
			return diffLoadedMsg{err: fmt.Errorf("failed to parse source Brewfile: %w", err)}
		}

		diff := brewfile.Diff(sourcePkgs, currentPkgs)
		return diffLoadedMsg{
			additions: diff.Additions,
			removals:  diff.Removals,
		}
	}
}

// Update handles messages
func (m *DiffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case diffLoadedMsg:
		m.loading = false
		m.additions = msg.additions
		m.removals = msg.removals
		m.err = msg.err
		m.buildItems()
		// Start in additions if available, otherwise removals
		if len(m.addItems) == 0 && len(m.remItems) > 0 {
			m.column = DiffColumnRemovals
		}
		return m, nil

	case ShowIgnoredMsg:
		m.showIgnored = msg.Show
		m.buildItems() // Rebuild to update visibility
		return m, nil

	case PackageActionStartMsg:
		m.taskRunning = true
		return m, nil

	case PackageActionDoneMsg:
		m.taskRunning = false
		// Reload diff after action
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
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			m.moveLeft()
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			m.moveRight()
		case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
			m.jumpToTop()
		case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
			m.jumpToBottom()
		case key.Matches(msg, key.NewBinding(key.WithKeys("i"))):
			// Install package (only from additions column)
			if !m.taskRunning && m.column == DiffColumnAdditions {
				pkg := m.getCurrentPackage()
				if pkg != nil {
					m.confirmPkg = *pkg
					m.confirmAction = "install"
					m.showConfirm = true
				}
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("X"))):
			// Uninstall package (only from removals column - packages that exist locally)
			if !m.taskRunning && m.column == DiffColumnRemovals {
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
func (m *DiffModel) handleConfirmInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.showConfirm = false
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

// getCurrentPackage returns the package at the current cursor position in the active column
func (m *DiffModel) getCurrentPackage() *brewfile.Package {
	var items []diffItem
	var cursor int

	if m.column == DiffColumnAdditions {
		items = m.addItems
		cursor = m.addCursor
	} else {
		items = m.remItems
		cursor = m.remCursor
	}

	if cursor < 0 || cursor >= len(items) {
		return nil
	}
	item := items[cursor]
	if item.isHeader || item.isIgnored {
		return nil
	}
	return &item.pkg
}

// buildItems creates flattened lists with category headers
func (m *DiffModel) buildItems() {
	m.addItems = m.buildItemsForPackages(m.additions)
	m.remItems = m.buildItemsForPackages(m.removals)
}

// buildItemsForPackages creates a flattened list with headers for a package list
func (m *DiffModel) buildItemsForPackages(pkgs brewfile.Packages) []diffItem {
	var items []diffItem
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

	for _, t := range types {
		typePkgs := byType[t]
		if len(typePkgs) == 0 {
			continue
		}

		// Count visible and ignored packages
		var visiblePkgs []brewfile.Package
		var ignoredPkgs []brewfile.Package
		for _, pkg := range typePkgs {
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
			headerCount = len(typePkgs)
		}

		// Add header
		items = append(items, diffItem{
			isHeader:    true,
			headerType:  t,
			headerCount: headerCount,
		})

		// Add visible packages first
		for _, pkg := range visiblePkgs {
			items = append(items, diffItem{pkg: pkg, isIgnored: false})
		}

		// Add ignored packages if showing ignored
		if m.showIgnored {
			for _, pkg := range ignoredPkgs {
				items = append(items, diffItem{pkg: pkg, isIgnored: true})
			}
		}
	}
	return items
}

// moveUp moves cursor up in current column
func (m *DiffModel) moveUp() {
	if m.column == DiffColumnAdditions {
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

// moveDown moves cursor down in current column
func (m *DiffModel) moveDown() {
	if m.column == DiffColumnAdditions {
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

// moveLeft switches to additions column
func (m *DiffModel) moveLeft() {
	if len(m.addItems) > 0 {
		m.column = DiffColumnAdditions
	}
}

// moveRight switches to removals column
func (m *DiffModel) moveRight() {
	if len(m.remItems) > 0 {
		m.column = DiffColumnRemovals
	}
}

// jumpToTop jumps to top of current column
func (m *DiffModel) jumpToTop() {
	if m.column == DiffColumnAdditions {
		m.addCursor = 0
		m.addOffset = 0
	} else {
		m.remCursor = 0
		m.remOffset = 0
	}
}

// jumpToBottom jumps to bottom of current column
func (m *DiffModel) jumpToBottom() {
	if m.column == DiffColumnAdditions {
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

// adjustAddOffset ensures additions cursor is visible
func (m *DiffModel) adjustAddOffset() {
	visibleHeight := m.getColumnHeight()
	if m.addCursor < m.addOffset {
		m.addOffset = m.addCursor
	}
	if m.addCursor >= m.addOffset+visibleHeight {
		m.addOffset = m.addCursor - visibleHeight + 1
	}
}

// adjustRemOffset ensures removals cursor is visible
func (m *DiffModel) adjustRemOffset() {
	visibleHeight := m.getColumnHeight()
	if m.remCursor < m.remOffset {
		m.remOffset = m.remCursor
	}
	if m.remCursor >= m.remOffset+visibleHeight {
		m.remOffset = m.remCursor - visibleHeight + 1
	}
}

// getColumnHeight returns the visible height for each column
func (m *DiffModel) getColumnHeight() int {
	h := m.height - 4 // Just title and scroll indicator
	// Reserve space for confirmation dialog if showing
	if m.showConfirm {
		h -= 5
	}
	if h < 1 {
		h = 1
	}
	return h
}

// SetSize updates the diff dimensions
func (m *DiffModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the diff screen (legacy)
func (m *DiffModel) View() string {
	return m.ViewContent(m.width, m.height)
}

// ViewContent renders just the content area (for use in layout)
func (m *DiffModel) ViewContent(width, height int) string {
	var b strings.Builder

	// Title showing source -> target
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatMauve)
	b.WriteString(titleStyle.Render(fmt.Sprintf("Diff: %s → %s", m.source, m.config.CurrentMachine)))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(styles.DimmedStyle.Render("Computing diff..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
		return b.String()
	}

	// No changes
	if len(m.additions) == 0 && len(m.removals) == 0 {
		b.WriteString(styles.SelectedStyle.Render("✓ "))
		b.WriteString("Machines are in sync!")
		b.WriteString("\n")
		return b.String()
	}

	// Calculate column widths
	colWidth := (width - 4) / 2 // Split width with gap
	if colWidth < 20 {
		colWidth = 20
	}
	visibleHeight := m.getColumnHeight()

	// Build left column (additions)
	leftLines := m.renderColumn(
		m.addItems,
		fmt.Sprintf("TO IMPORT (+%d)", len(m.additions)),
		"+",
		colWidth,
		visibleHeight,
		m.addCursor,
		m.addOffset,
		m.column == DiffColumnAdditions,
		styles.AddedStyle,
	)

	// Build right column (removals)
	rightLines := m.renderColumn(
		m.remItems,
		fmt.Sprintf("NOT IN SOURCE (-%d)", len(m.removals)),
		"−",
		colWidth,
		visibleHeight,
		m.remCursor,
		m.remOffset,
		m.column == DiffColumnRemovals,
		styles.RemovedStyle,
	)

	// Combine columns side by side - use full height
	maxLines := visibleHeight + 2 // Header + separator + content
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

		// Pad left column to width
		leftWidth := lipgloss.Width(left)
		if leftWidth < colWidth {
			left += strings.Repeat(" ", colWidth-leftWidth)
		}

		b.WriteString(left)
		b.WriteString("  ") // Gap between columns
		b.WriteString(right)
		b.WriteString("\n")
	}

	// Confirmation dialog overlay
	if m.showConfirm {
		b.WriteString("\n")
		b.WriteString(m.renderConfirmDialog())
	}

	return b.String()
}

// renderConfirmDialog renders the confirmation dialog
func (m *DiffModel) renderConfirmDialog() string {
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

// renderColumn renders a single column with categorized packages
func (m *DiffModel) renderColumn(
	items []diffItem,
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
			maxNameLen := width - 12 // Extra space for ignored marker
			name := item.pkg.Name
			if len(name) > maxNameLen {
				name = name[:maxNameLen-3] + "..."
			}

			var line string
			if item.isIgnored {
				// Dimmed style for ignored packages
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
