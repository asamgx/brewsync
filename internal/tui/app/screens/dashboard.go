package screens

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/andrew-sameh/brewsync/internal/brewfile"
	"github.com/andrew-sameh/brewsync/internal/config"
	"github.com/andrew-sameh/brewsync/internal/debug"
	"github.com/andrew-sameh/brewsync/internal/tui/styles"
)

// DashboardKeyMap defines keybindings for the dashboard
type DashboardKeyMap struct {
	Import  key.Binding
	Sync    key.Binding
	Diff    key.Binding
	Dump    key.Binding
	List    key.Binding
	Ignore  key.Binding
	Config  key.Binding
	History key.Binding
	Profile key.Binding
	Doctor  key.Binding
	Help    key.Binding
	Quit    key.Binding
}

// DefaultDashboardKeyMap returns the default dashboard keybindings
func DefaultDashboardKeyMap() DashboardKeyMap {
	return DashboardKeyMap{
		Import: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "import"),
		),
		Sync: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sync"),
		),
		Diff: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "diff"),
		),
		Dump: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "dump"),
		),
		List: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "list"),
		),
		Ignore: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "ignore"),
		),
		Config: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "config"),
		),
		History: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "history"),
		),
		Profile: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "profiles"),
		),
		Doctor: key.NewBinding(
			key.WithKeys("!"),
			key.WithHelp("!", "doctor"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// DashboardModel is the model for the dashboard screen
type DashboardModel struct {
	config *config.Config
	width  int
	height int
	keys   DashboardKeyMap

	// Data
	machineName     string
	hostname        string
	brewfilePath    string
	defaultSource   string
	packageCounts   map[string]int
	totalPackages   int
	lastDump        time.Time
	ignoredCats     int
	ignoredPkgs     int

	// Pending changes - stored by type for breakdown
	pendingAddsByType    map[string]int // type -> count
	pendingRemovesByType map[string]int // type -> count
	ignoredAddsByType    map[string]int // type -> count (ignored additions)
	ignoredRemovesByType map[string]int // type -> count (ignored removals)

	// State
	loading     bool
	err         error
	showIgnored bool
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(cfg *config.Config) *DashboardModel {
	m := &DashboardModel{
		config:               cfg,
		width:                80,
		height:               24,
		keys:                 DefaultDashboardKeyMap(),
		packageCounts:        make(map[string]int),
		pendingAddsByType:    make(map[string]int),
		pendingRemovesByType: make(map[string]int),
		ignoredAddsByType:    make(map[string]int),
		ignoredRemovesByType: make(map[string]int),
		loading:              true,
	}

	if cfg != nil {
		m.machineName = cfg.CurrentMachine
		m.defaultSource = cfg.DefaultSource
		if machine, ok := cfg.GetCurrentMachine(); ok {
			m.hostname = machine.Hostname
			m.brewfilePath = machine.Brewfile
		}
	}

	return m
}

// loadDataMsg is sent when data loading completes
type loadDataMsg struct {
	packageCounts        map[string]int
	totalPackages        int
	lastDump             time.Time
	pendingAddsByType    map[string]int
	pendingRemovesByType map[string]int
	ignoredAddsByType    map[string]int
	ignoredRemovesByType map[string]int
	ignoredCats          int
	ignoredPkgs          int
	err                  error
}

// Init initializes the dashboard and loads data
func (m *DashboardModel) Init() tea.Cmd {
	return m.loadData()
}

// loadData loads dashboard data in background
func (m *DashboardModel) loadData() tea.Cmd {
	debug.Log("Dashboard.loadData: starting background data load")
	return func() tea.Msg {
		debug.Log("Dashboard.loadData: inside tea.Cmd function")
		result := loadDataMsg{
			packageCounts:        make(map[string]int),
			pendingAddsByType:    make(map[string]int),
			pendingRemovesByType: make(map[string]int),
			ignoredAddsByType:    make(map[string]int),
			ignoredRemovesByType: make(map[string]int),
		}

		if m.config == nil {
			debug.Log("Dashboard.loadData: config is nil")
			result.err = fmt.Errorf("no config loaded")
			return result
		}

		debug.Log("Dashboard.loadData: config loaded, current machine: %s", m.config.CurrentMachine)

		machine, ok := m.config.GetCurrentMachine()
		if !ok {
			debug.Log("Dashboard.loadData: current machine not found in config")
			result.err = fmt.Errorf("current machine not found in config")
			return result
		}

		debug.Log("Dashboard.loadData: got machine, brewfile path: %s", machine.Brewfile)

		// Load package counts from Brewfile
		debug.Log("Dashboard.loadData: parsing brewfile...")
		packages, err := brewfile.Parse(machine.Brewfile)
		if err != nil {
			debug.Log("Dashboard.loadData: brewfile parse error: %v", err)
		} else {
			debug.Log("Dashboard.loadData: parsed %d packages", len(packages))
			for _, pkg := range packages {
				result.packageCounts[string(pkg.Type)]++
				result.totalPackages++
			}
		}

		// Load metadata for last dump time
		metaPath := filepath.Join(filepath.Dir(machine.Brewfile), ".brewsync-meta")
		debug.Log("Dashboard.loadData: loading metadata from: %s", metaPath)
		meta, err := brewfile.LoadMetadata(metaPath)
		if err != nil {
			debug.Log("Dashboard.loadData: metadata load error (non-fatal): %v", err)
		} else if meta != nil {
			result.lastDump = meta.LastDump
			debug.Log("Dashboard.loadData: last dump: %v", meta.LastDump)
		}

		// Calculate pending changes if default source is different
		if m.config.DefaultSource != "" && m.config.DefaultSource != m.config.CurrentMachine {
			debug.Log("Dashboard.loadData: calculating pending changes from source: %s", m.config.DefaultSource)
			if sourceMachine, ok := m.config.GetMachine(m.config.DefaultSource); ok {
				sourcePackages, err := brewfile.Parse(sourceMachine.Brewfile)
				if err != nil {
					debug.Log("Dashboard.loadData: source brewfile parse error: %v", err)
				} else {
					diff := brewfile.Diff(sourcePackages, packages)

					// Categorize additions by type, separating ignored
					for _, pkg := range diff.Additions {
						pkgType := string(pkg.Type)
						isIgnored := m.config.IsCategoryIgnored(m.config.CurrentMachine, pkgType) ||
							m.config.IsPackageIgnored(m.config.CurrentMachine, pkg.ID())
						if isIgnored {
							result.ignoredAddsByType[pkgType]++
						} else {
							result.pendingAddsByType[pkgType]++
						}
					}

					// Categorize removals by type, separating ignored
					for _, pkg := range diff.Removals {
						pkgType := string(pkg.Type)
						isIgnored := m.config.IsCategoryIgnored(m.config.CurrentMachine, pkgType) ||
							m.config.IsPackageIgnored(m.config.CurrentMachine, pkg.ID())
						if isIgnored {
							result.ignoredRemovesByType[pkgType]++
						} else {
							result.pendingRemovesByType[pkgType]++
						}
					}

					debug.Log("Dashboard.loadData: pending adds=%v, removes=%v", result.pendingAddsByType, result.pendingRemovesByType)
				}
			}
		}

		// Count ignored items
		result.ignoredCats = len(m.config.GetIgnoredCategories(m.config.CurrentMachine))
		result.ignoredPkgs = len(m.config.GetIgnoredPackages(m.config.CurrentMachine))

		debug.Log("Dashboard.loadData: completed, total=%d, err=%v", result.totalPackages, result.err)
		return result
	}
}

// Update handles messages
func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	debug.Log("Dashboard.Update: received msg type %T", msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		debug.Log("Dashboard.Update: window resize %dx%d", msg.Width, msg.Height)
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case loadDataMsg:
		debug.Log("Dashboard.Update: received loadDataMsg, err=%v, total=%d", msg.err, msg.totalPackages)
		m.loading = false
		m.err = msg.err
		m.packageCounts = msg.packageCounts
		m.totalPackages = msg.totalPackages
		m.lastDump = msg.lastDump
		m.pendingAddsByType = msg.pendingAddsByType
		m.pendingRemovesByType = msg.pendingRemovesByType
		m.ignoredAddsByType = msg.ignoredAddsByType
		m.ignoredRemovesByType = msg.ignoredRemovesByType
		m.ignoredCats = msg.ignoredCats
		m.ignoredPkgs = msg.ignoredPkgs
		return m, nil

	case ShowIgnoredMsg:
		m.showIgnored = msg.Show
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Import):
			return m, func() tea.Msg { return Navigate("import") }
		case key.Matches(msg, m.keys.Sync):
			return m, func() tea.Msg { return Navigate("sync") }
		case key.Matches(msg, m.keys.Diff):
			return m, func() tea.Msg { return Navigate("diff") }
		case key.Matches(msg, m.keys.Dump):
			return m, func() tea.Msg { return Navigate("dump") }
		case key.Matches(msg, m.keys.List):
			return m, func() tea.Msg { return Navigate("list") }
		case key.Matches(msg, m.keys.Ignore):
			return m, func() tea.Msg { return Navigate("ignore") }
		case key.Matches(msg, m.keys.Config):
			return m, func() tea.Msg { return Navigate("config") }
		case key.Matches(msg, m.keys.History):
			return m, func() tea.Msg { return Navigate("history") }
		case key.Matches(msg, m.keys.Profile):
			return m, func() tea.Msg { return Navigate("profile") }
		case key.Matches(msg, m.keys.Doctor):
			return m, func() tea.Msg { return Navigate("doctor") }
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	}

	return m, nil
}

// SetSize updates the dashboard dimensions
func (m *DashboardModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the dashboard (legacy, for standalone use)
func (m *DashboardModel) View() string {
	return m.ViewContent(m.width, m.height)
}

// ViewContent renders just the content area (for use in layout)
func (m *DashboardModel) ViewContent(width, height int) string {
	var b strings.Builder

	// System Health section
	healthBox := m.renderHealthSection(width - 4)
	b.WriteString(healthBox)
	b.WriteString("\n\n")

	// Inventory section
	inventoryBox := m.renderInventorySection(width - 4)
	b.WriteString(inventoryBox)
	b.WriteString("\n\n")

	// Pending Changes section
	pendingBox := m.renderPendingSection(width - 4)
	b.WriteString(pendingBox)

	return b.String()
}

// renderHealthSection renders the System Health box
func (m *DashboardModel) renderHealthSection(width int) string {
	var content strings.Builder

	labelStyle := lipgloss.NewStyle().Foreground(styles.MutedColor).Width(14)
	valueStyle := lipgloss.NewStyle().Foreground(styles.CatText)
	okStyle := lipgloss.NewStyle().Foreground(styles.CatGreen)

	// Last Dump
	content.WriteString(labelStyle.Render("Last Dump"))
	if !m.lastDump.IsZero() {
		content.WriteString(valueStyle.Render(formatTimeAgo(m.lastDump)))
		content.WriteString("  ")
		content.WriteString(okStyle.Render("🟢 OK"))
	} else {
		content.WriteString(valueStyle.Render("Never"))
	}
	content.WriteString("\n")

	// Git Status (placeholder for now)
	content.WriteString(labelStyle.Render("Git Status"))
	content.WriteString(valueStyle.Render("Clean"))
	content.WriteString("  ")
	content.WriteString(okStyle.Render("🟢 OK"))
	content.WriteString("\n")

	// Brewfile path
	content.WriteString(labelStyle.Render("Brewfile"))
	content.WriteString(valueStyle.Render(shortenPath(m.brewfilePath)))

	return renderBox("System Health", content.String(), width)
}

// renderInventorySection renders the Inventory box
func (m *DashboardModel) renderInventorySection(width int) string {
	var content strings.Builder

	if m.loading {
		content.WriteString(styles.DimmedStyle.Render("Loading..."))
		return renderBox("Inventory", content.String(), width)
	}

	if m.err != nil {
		content.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
		return renderBox("Inventory", content.String(), width)
	}

	// Package counts in a grid layout
	counts := []struct {
		icon  string
		name  string
		count int
	}{
		{"🚰", "Taps", m.packageCounts["tap"]},
		{"📦", "Casks", m.packageCounts["cask"]},
		{"🍺", "Brews", m.packageCounts["brew"]},
		{"💻", "VSCode", m.packageCounts["vscode"]},
		{"✏️", "Cursor", m.packageCounts["cursor"]},
		{"🔷", "Go", m.packageCounts["go"]},
		{"🚀", "Antigrav", m.packageCounts["antigravity"]},
		{"🍎", "MAS", m.packageCounts["mas"]},
	}

	// Two columns
	colWidth := (width - 8) / 2
	itemStyle := lipgloss.NewStyle().Width(colWidth)

	for i := 0; i < len(counts); i += 2 {
		left := counts[i]
		var leftStr string
		if left.count > 0 {
			leftStr = fmt.Sprintf("%s %-8s %3d", left.icon, left.name, left.count)
		} else {
			leftStr = fmt.Sprintf("%s %-8s   -", left.icon, left.name)
		}
		content.WriteString(itemStyle.Render(leftStr))

		if i+1 < len(counts) {
			right := counts[i+1]
			var rightStr string
			if right.count > 0 {
				rightStr = fmt.Sprintf("%s %-8s %3d", right.icon, right.name, right.count)
			} else {
				rightStr = fmt.Sprintf("%s %-8s   -", right.icon, right.name)
			}
			content.WriteString(rightStr)
		}
		content.WriteString("\n")
	}

	// Total
	content.WriteString("\n")
	totalStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatText)
	content.WriteString(strings.Repeat(" ", colWidth))
	content.WriteString(totalStyle.Render(fmt.Sprintf("Total: %d packages", m.totalPackages)))

	return renderBox("Inventory", content.String(), width)
}

// renderPendingSection renders the Pending Changes box
func (m *DashboardModel) renderPendingSection(width int) string {
	var content strings.Builder

	// Calculate totals
	totalAdds := 0
	totalRemoves := 0
	totalIgnoredAdds := 0
	totalIgnoredRemoves := 0

	for _, count := range m.pendingAddsByType {
		totalAdds += count
	}
	for _, count := range m.pendingRemovesByType {
		totalRemoves += count
	}
	for _, count := range m.ignoredAddsByType {
		totalIgnoredAdds += count
	}
	for _, count := range m.ignoredRemovesByType {
		totalIgnoredRemoves += count
	}

	// If showIgnored, include ignored items in total
	displayAdds := totalAdds
	displayRemoves := totalRemoves
	if m.showIgnored {
		displayAdds += totalIgnoredAdds
		displayRemoves += totalIgnoredRemoves
	}

	if displayAdds == 0 && displayRemoves == 0 {
		content.WriteString(styles.DimmedStyle.Render("No pending changes"))
		if m.defaultSource != "" && m.defaultSource != m.machineName {
			content.WriteString("\n")
			content.WriteString(styles.DimmedStyle.Render(fmt.Sprintf("Source: %s", m.defaultSource)))
		}
		// Show ignored count if any
		if !m.showIgnored && (totalIgnoredAdds > 0 || totalIgnoredRemoves > 0) {
			content.WriteString("\n")
			content.WriteString(styles.DimmedStyle.Render(fmt.Sprintf("(%d ignored, press h to show)", totalIgnoredAdds+totalIgnoredRemoves)))
		}
	} else {
		labelStyle := lipgloss.NewStyle().Foreground(styles.MutedColor)
		content.WriteString(labelStyle.Render(fmt.Sprintf("Source: %s", m.defaultSource)))
		content.WriteString("\n\n")

		// Render additions with breakdown
		if displayAdds > 0 {
			content.WriteString(styles.AddedStyle.Render(fmt.Sprintf("+ %d to import", displayAdds)))
			content.WriteString("\n")
			m.renderBreakdown(&content, m.pendingAddsByType, m.ignoredAddsByType, "+", styles.AddedStyle)
		}

		// Render removals with breakdown
		if displayRemoves > 0 {
			if displayAdds > 0 {
				content.WriteString("\n")
			}
			content.WriteString(styles.RemovedStyle.Render(fmt.Sprintf("− %d not in source", displayRemoves)))
			content.WriteString("\n")
			m.renderBreakdown(&content, m.pendingRemovesByType, m.ignoredRemovesByType, "−", styles.RemovedStyle)
		}

		// Show ignored hint
		if !m.showIgnored && (totalIgnoredAdds > 0 || totalIgnoredRemoves > 0) {
			content.WriteString("\n")
			content.WriteString(styles.DimmedStyle.Render(fmt.Sprintf("(%d ignored, press h to show)", totalIgnoredAdds+totalIgnoredRemoves)))
		}
	}

	return renderBox("Pending Changes", content.String(), width)
}

// renderBreakdown renders a breakdown of counts by type
func (m *DashboardModel) renderBreakdown(content *strings.Builder, byType, ignoredByType map[string]int, prefix string, style lipgloss.Style) {
	types := []struct {
		key  string
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
		count := byType[t.key]
		ignoredCount := ignoredByType[t.key]

		if m.showIgnored {
			count += ignoredCount
		}

		if count > 0 {
			content.WriteString(fmt.Sprintf("    %s %s: %d", t.icon, t.key, count))
			if m.showIgnored && ignoredCount > 0 {
				content.WriteString(styles.DimmedStyle.Render(fmt.Sprintf(" (%d ignored)", ignoredCount)))
			}
			content.WriteString("\n")
		}
	}
}

// renderBox renders a titled box
func renderBox(title, content string, width int) string {
	const (
		topLeft     = "┌"
		topRight    = "┐"
		bottomLeft  = "└"
		bottomRight = "┘"
		horizontal  = "─"
		vertical    = "│"
	)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatMauve)
	borderStyle := lipgloss.NewStyle().Foreground(styles.CatSurface1)

	innerWidth := width - 4
	if innerWidth < 10 {
		innerWidth = 10
	}

	var sb strings.Builder

	// Top border with title
	titleText := " " + title + " "
	titleLen := lipgloss.Width(titleText)
	rightPadding := innerWidth - titleLen - 1
	if rightPadding < 0 {
		rightPadding = 0
	}

	sb.WriteString(borderStyle.Render(topLeft + horizontal))
	sb.WriteString(titleStyle.Render(titleText))
	sb.WriteString(borderStyle.Render(strings.Repeat(horizontal, rightPadding) + topRight))
	sb.WriteString("\n")

	// Content lines
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		lineWidth := lipgloss.Width(line)
		padding := innerWidth - lineWidth
		if padding < 0 {
			padding = 0
		}
		sb.WriteString(borderStyle.Render(vertical))
		sb.WriteString(" ")
		sb.WriteString(line)
		sb.WriteString(strings.Repeat(" ", padding+1))
		sb.WriteString(borderStyle.Render(vertical))
		sb.WriteString("\n")
	}

	// Bottom border
	sb.WriteString(borderStyle.Render(bottomLeft + strings.Repeat(horizontal, innerWidth+2) + bottomRight))

	return sb.String()
}

func (m *DashboardModel) renderMachineSection() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.PrimaryColor)

	labelStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor).
		Width(12)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	var content strings.Builder
	content.WriteString(headerStyle.Render("Machine"))
	content.WriteString("\n")

	// Machine name with hostname
	content.WriteString(labelStyle.Render("Name:"))
	if m.hostname != "" {
		content.WriteString(valueStyle.Render(fmt.Sprintf("%s (%s)", m.machineName, m.hostname)))
	} else {
		content.WriteString(valueStyle.Render(m.machineName))
	}
	content.WriteString("\n")

	// Brewfile path (shortened)
	content.WriteString(labelStyle.Render("Brewfile:"))
	shortPath := shortenPath(m.brewfilePath)
	content.WriteString(valueStyle.Render(shortPath))

	return styles.BoxStyle.Render(content.String())
}

func (m *DashboardModel) renderPackagesSection() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.PrimaryColor)

	var content strings.Builder
	content.WriteString(headerStyle.Render("Packages"))
	content.WriteString("\n")

	if m.loading {
		content.WriteString(styles.DimmedStyle.Render("Loading..."))
	} else if m.err != nil {
		content.WriteString(styles.ErrorStyle.Render("Error: " + m.err.Error()))
	} else {
		// Package counts with icons
		counts := []struct {
			icon  string
			name  string
			count int
		}{
			{"🚰", "tap", m.packageCounts["tap"]},
			{"🍺", "brew", m.packageCounts["brew"]},
			{"📦", "cask", m.packageCounts["cask"]},
			{"💻", "vscode", m.packageCounts["vscode"]},
			{"✏️", "cursor", m.packageCounts["cursor"]},
			{"🚀", "antigravity", m.packageCounts["antigravity"]},
			{"🔷", "go", m.packageCounts["go"]},
			{"🍎", "mas", m.packageCounts["mas"]},
		}

		var line1, line2 []string
		for i, c := range counts {
			if c.count > 0 {
				style := styles.GetCategoryStyle(c.name)
				item := fmt.Sprintf("%s %s: %d", c.icon, c.name, c.count)
				if i < 4 {
					line1 = append(line1, style.Render(item))
				} else {
					line2 = append(line2, style.Render(item))
				}
			}
		}

		if len(line1) > 0 {
			content.WriteString(strings.Join(line1, "   "))
		}
		if len(line2) > 0 {
			content.WriteString("\n")
			content.WriteString(strings.Join(line2, "   "))
		}
		content.WriteString("\n")

		// Total
		totalStyle := lipgloss.NewStyle().Bold(true)
		content.WriteString(totalStyle.Render(fmt.Sprintf("Total: %d", m.totalPackages)))
	}

	return styles.BoxStyle.Render(content.String())
}

func (m *DashboardModel) renderStatusSection() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.PrimaryColor)

	labelStyle := lipgloss.NewStyle().
		Foreground(styles.MutedColor)

	var content strings.Builder
	content.WriteString(headerStyle.Render("Status"))
	content.WriteString("\n")

	// Last dump
	if !m.lastDump.IsZero() {
		content.WriteString(labelStyle.Render("Last dump: "))
		content.WriteString(formatTimeAgo(m.lastDump))
		content.WriteString("\n")
	}

	// Pending changes
	totalAdds := 0
	totalRemoves := 0
	for _, count := range m.pendingAddsByType {
		totalAdds += count
	}
	for _, count := range m.pendingRemovesByType {
		totalRemoves += count
	}
	if totalAdds > 0 || totalRemoves > 0 {
		content.WriteString(labelStyle.Render(fmt.Sprintf("Pending from %s: ", m.defaultSource)))
		if totalAdds > 0 {
			content.WriteString(styles.AddedStyle.Render(fmt.Sprintf("+%d", totalAdds)))
		}
		if totalRemoves > 0 {
			if totalAdds > 0 {
				content.WriteString(", ")
			}
			content.WriteString(styles.RemovedStyle.Render(fmt.Sprintf("-%d", totalRemoves)))
		}
		content.WriteString("\n")
	}

	// Ignored counts
	if m.ignoredCats > 0 || m.ignoredPkgs > 0 {
		content.WriteString(labelStyle.Render("Ignored: "))
		var parts []string
		if m.ignoredCats > 0 {
			parts = append(parts, fmt.Sprintf("%d categories", m.ignoredCats))
		}
		if m.ignoredPkgs > 0 {
			parts = append(parts, fmt.Sprintf("%d packages", m.ignoredPkgs))
		}
		content.WriteString(strings.Join(parts, ", "))
	}

	return styles.BoxStyle.Render(content.String())
}

func (m *DashboardModel) renderActionsSection() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.PrimaryColor)

	keyStyle := lipgloss.NewStyle().
		Foreground(styles.HighlightColor).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	var content strings.Builder
	content.WriteString(headerStyle.Render("Actions"))
	content.WriteString("\n")

	actions := []struct {
		key   string
		label string
	}{
		{"i", "Import"},
		{"s", "Sync"},
		{"d", "Diff"},
		{"D", "Dump"},
		{"l", "List"},
		{"g", "Ignore"},
		{"c", "Config"},
		{"h", "History"},
		{"p", "Profiles"},
		{"!", "Doctor"},
	}

	// Render in two rows
	var row1, row2 []string
	for i, a := range actions {
		item := fmt.Sprintf("[%s] %s", keyStyle.Render(a.key), labelStyle.Render(a.label))
		if i < 5 {
			row1 = append(row1, item)
		} else {
			row2 = append(row2, item)
		}
	}

	content.WriteString(strings.Join(row1, "   "))
	content.WriteString("\n")
	content.WriteString(strings.Join(row2, "   "))

	return styles.BoxStyle.Render(content.String())
}

// Helper functions

func shortenPath(path string) string {
	home, _ := filepath.Abs(filepath.Join("~"))
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	// Try to shorten with ~
	if expanded, err := filepath.Abs(path); err == nil {
		if homeDir, err := filepath.Abs(filepath.Join("~")); err == nil {
			if strings.HasPrefix(expanded, homeDir) {
				return "~" + expanded[len(homeDir):]
			}
		}
	}
	// Truncate long paths
	if len(path) > 50 {
		return "..." + path[len(path)-47:]
	}
	return path
}

func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}
