package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/asamgx/brewsync/internal/config"
	"github.com/asamgx/brewsync/internal/tui/styles"
)

// IgnoreSection represents which section is focused
type IgnoreSection int

const (
	IgnoreSectionCategories IgnoreSection = iota
	IgnoreSectionPackages
)

// IgnoreScope represents global vs machine-specific
type IgnoreScope int

const (
	IgnoreScopeGlobal IgnoreScope = iota
	IgnoreScopeMachine
)

// ignoreItem represents an item in the ignore list
type ignoreItem struct {
	value    string
	isGlobal bool
}

// IgnoreModel is the model for the ignore management screen
type IgnoreModel struct {
	config  *config.Config
	width   int
	height  int
	section IgnoreSection
	scope   IgnoreScope
	cursor  int
	offset  int

	// Data
	categories []ignoreItem // Combined global + machine categories
	packages   []ignoreItem // Combined global + machine packages

	// Input mode
	inputMode    bool
	inputType    string // "category" or "package"
	textInput    textinput.Model
	inputScope   IgnoreScope
	showTypeMenu bool
	typeMenuIdx  int

	// Confirmation
	showConfirm   bool
	confirmAction string
	confirmItem   ignoreItem

	// Status
	loading       bool
	statusMessage string
	statusType    string // "success", "error"
}

// Available package types for adding
var packageTypes = []string{"tap", "brew", "cask", "vscode", "cursor", "antigravity", "go", "mas"}

// Available categories (same as package types)
var categoryTypes = []string{"tap", "brew", "cask", "vscode", "cursor", "antigravity", "go", "mas"}

// NewIgnoreModel creates a new ignore model
func NewIgnoreModel(cfg *config.Config) *IgnoreModel {
	ti := textinput.New()
	ti.Placeholder = "Enter value..."
	ti.CharLimit = 100
	ti.Width = 40

	return &IgnoreModel{
		config:    cfg,
		width:     80,
		height:    24,
		loading:   true,
		textInput: ti,
	}
}

type ignoreLoadedMsg struct {
	categories []ignoreItem
	packages   []ignoreItem
}

type ignoreActionMsg struct {
	success bool
	message string
}

// Init initializes the ignore model
func (m *IgnoreModel) Init() tea.Cmd {
	return m.loadIgnores
}

func (m *IgnoreModel) loadIgnores() tea.Msg {
	result := ignoreLoadedMsg{
		categories: []ignoreItem{},
		packages:   []ignoreItem{},
	}

	ignoreFile, err := config.LoadIgnoreFile()
	if err != nil {
		return result
	}

	// Get global categories
	for _, cat := range ignoreFile.Global.Categories {
		result.categories = append(result.categories, ignoreItem{value: cat, isGlobal: true})
	}

	// Get machine-specific categories
	if m.config != nil {
		if machineIgnore, ok := ignoreFile.Machines[m.config.CurrentMachine]; ok {
			for _, cat := range machineIgnore.Categories {
				result.categories = append(result.categories, ignoreItem{value: cat, isGlobal: false})
			}
		}
	}

	// Get global packages
	globalPkgs := flattenPackageList(&ignoreFile.Global.Packages)
	for _, pkg := range globalPkgs {
		result.packages = append(result.packages, ignoreItem{value: pkg, isGlobal: true})
	}

	// Get machine-specific packages
	if m.config != nil {
		if machineIgnore, ok := ignoreFile.Machines[m.config.CurrentMachine]; ok {
			machinePkgs := flattenPackageList(&machineIgnore.Packages)
			for _, pkg := range machinePkgs {
				result.packages = append(result.packages, ignoreItem{value: pkg, isGlobal: false})
			}
		}
	}

	return result
}

// flattenPackageList converts PackageIgnoreList to a flat list of "type:name" strings
func flattenPackageList(list *config.PackageIgnoreList) []string {
	var result []string
	for _, v := range list.Tap {
		result = append(result, "tap:"+v)
	}
	for _, v := range list.Brew {
		result = append(result, "brew:"+v)
	}
	for _, v := range list.Cask {
		result = append(result, "cask:"+v)
	}
	for _, v := range list.VSCode {
		result = append(result, "vscode:"+v)
	}
	for _, v := range list.Cursor {
		result = append(result, "cursor:"+v)
	}
	for _, v := range list.Antigravity {
		result = append(result, "antigravity:"+v)
	}
	for _, v := range list.Go {
		result = append(result, "go:"+v)
	}
	for _, v := range list.Mas {
		result = append(result, "mas:"+v)
	}
	return result
}

// Update handles messages
func (m *IgnoreModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case ignoreLoadedMsg:
		m.loading = false
		m.categories = msg.categories
		m.packages = msg.packages
		return m, nil

	case ignoreActionMsg:
		if msg.success {
			m.statusMessage = msg.message
			m.statusType = "success"
		} else {
			m.statusMessage = msg.message
			m.statusType = "error"
		}
		// Reload data
		return m, m.loadIgnores

	case tea.KeyMsg:
		// Clear status message on any key
		m.statusMessage = ""

		// Handle confirmation dialog
		if m.showConfirm {
			return m.handleConfirmInput(msg)
		}

		// Handle type menu selection
		if m.showTypeMenu {
			return m.handleTypeMenuInput(msg)
		}

		// Handle text input mode
		if m.inputMode {
			return m.handleTextInput(msg)
		}

		// Normal navigation
		return m.handleNormalInput(msg)
	}

	return m, nil
}

func (m *IgnoreModel) handleConfirmInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.showConfirm = false
		return m, m.executeDelete()
	case "n", "N", "esc":
		m.showConfirm = false
	}
	return m, nil
}

func (m *IgnoreModel) handleTypeMenuInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	types := categoryTypes
	if m.inputType == "package" {
		types = packageTypes
	}

	switch msg.String() {
	case "up", "k":
		if m.typeMenuIdx > 0 {
			m.typeMenuIdx--
		}
	case "down", "j":
		if m.typeMenuIdx < len(types)-1 {
			m.typeMenuIdx++
		}
	case "enter":
		m.showTypeMenu = false
		if m.inputType == "category" {
			// For category, the type IS the value
			m.textInput.SetValue(types[m.typeMenuIdx])
			m.inputMode = true
			m.textInput.Focus()
		} else {
			// For package, prepend type to input
			m.textInput.SetValue(types[m.typeMenuIdx] + ":")
			m.inputMode = true
			m.textInput.Focus()
		}
	case "esc":
		m.showTypeMenu = false
		m.inputType = ""
	}
	return m, nil
}

func (m *IgnoreModel) handleTextInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		value := strings.TrimSpace(m.textInput.Value())
		if value != "" {
			m.inputMode = false
			return m, m.executeAdd(value)
		}
	case "esc":
		m.inputMode = false
		m.textInput.SetValue("")
	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *IgnoreModel) handleNormalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		return m, func() tea.Msg { return Navigate("dashboard") }

	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		// Switch between categories and packages
		if m.section == IgnoreSectionCategories {
			m.section = IgnoreSectionPackages
		} else {
			m.section = IgnoreSectionCategories
		}
		m.cursor = 0
		m.offset = 0

	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
		// Switch scope between global and machine
		if m.scope == IgnoreScopeGlobal {
			m.scope = IgnoreScopeMachine
		} else {
			m.scope = IgnoreScopeGlobal
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.cursor > 0 {
			m.cursor--
			m.adjustOffset()
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		maxItems := m.getCurrentListLen()
		if m.cursor < maxItems-1 {
			m.cursor++
			m.adjustOffset()
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
		m.cursor = 0
		m.offset = 0

	case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
		maxItems := m.getCurrentListLen()
		if maxItems > 0 {
			m.cursor = maxItems - 1
			m.adjustOffset()
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
		// Add new ignore
		m.inputScope = m.scope
		if m.section == IgnoreSectionCategories {
			m.inputType = "category"
		} else {
			m.inputType = "package"
		}
		m.showTypeMenu = true
		m.typeMenuIdx = 0

	case key.Matches(msg, key.NewBinding(key.WithKeys("d", "x"))):
		// Delete current item
		item := m.getCurrentItem()
		if item != nil {
			m.confirmItem = *item
			m.confirmAction = "delete"
			m.showConfirm = true
		}
	}

	return m, nil
}

func (m *IgnoreModel) executeAdd(value string) tea.Cmd {
	return func() tea.Msg {
		var err error
		machine := ""
		if m.config != nil {
			machine = m.config.CurrentMachine
		}
		global := m.inputScope == IgnoreScopeGlobal

		if m.inputType == "category" {
			err = config.AddCategoryIgnore(machine, value, global)
		} else {
			err = config.AddPackageIgnore(machine, value, global)
		}

		m.textInput.SetValue("")
		m.inputType = ""

		if err != nil {
			return ignoreActionMsg{success: false, message: fmt.Sprintf("Failed to add: %v", err)}
		}
		return ignoreActionMsg{success: true, message: fmt.Sprintf("Added %s", value)}
	}
}

func (m *IgnoreModel) executeDelete() tea.Cmd {
	return func() tea.Msg {
		var err error
		machine := ""
		if m.config != nil {
			machine = m.config.CurrentMachine
		}
		global := m.confirmItem.isGlobal

		if m.section == IgnoreSectionCategories {
			err = config.RemoveCategoryIgnore(machine, m.confirmItem.value, global)
		} else {
			err = config.RemovePackageIgnore(machine, m.confirmItem.value, global)
		}

		if err != nil {
			return ignoreActionMsg{success: false, message: fmt.Sprintf("Failed to remove: %v", err)}
		}
		return ignoreActionMsg{success: true, message: fmt.Sprintf("Removed %s", m.confirmItem.value)}
	}
}

func (m *IgnoreModel) getCurrentListLen() int {
	if m.section == IgnoreSectionCategories {
		return len(m.categories)
	}
	return len(m.packages)
}

func (m *IgnoreModel) getCurrentItem() *ignoreItem {
	if m.section == IgnoreSectionCategories {
		if m.cursor < len(m.categories) {
			return &m.categories[m.cursor]
		}
	} else {
		if m.cursor < len(m.packages) {
			return &m.packages[m.cursor]
		}
	}
	return nil
}

func (m *IgnoreModel) getVisibleHeight() int {
	h := m.height - 10 // Account for headers, status, help
	if h < 5 {
		h = 5
	}
	return h
}

func (m *IgnoreModel) adjustOffset() {
	visibleHeight := m.getVisibleHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visibleHeight {
		m.offset = m.cursor - visibleHeight + 1
	}
}

// SetSize updates the ignore dimensions
func (m *IgnoreModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the ignore screen (legacy)
func (m *IgnoreModel) View() string {
	return m.ViewContent(m.width, m.height)
}

// ViewContent renders just the content area (for use in layout)
func (m *IgnoreModel) ViewContent(width, height int) string {
	var b strings.Builder

	if m.loading {
		b.WriteString(styles.DimmedStyle.Render("Loading..."))
		return b.String()
	}

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatMauve)
	machine := "unknown"
	if m.config != nil {
		machine = m.config.CurrentMachine
	}
	b.WriteString(titleStyle.Render(fmt.Sprintf("Ignore Management - %s", machine)))
	b.WriteString("\n\n")

	// Scope indicator
	scopeStyle := lipgloss.NewStyle().Foreground(styles.CatSubtext0)
	globalStyle := scopeStyle
	machineStyle := scopeStyle
	if m.scope == IgnoreScopeGlobal {
		globalStyle = lipgloss.NewStyle().Foreground(styles.CatGreen).Bold(true)
	} else {
		machineStyle = lipgloss.NewStyle().Foreground(styles.CatGreen).Bold(true)
	}
	b.WriteString(globalStyle.Render("[G]lobal"))
	b.WriteString(" | ")
	b.WriteString(machineStyle.Render("[M]achine"))
	b.WriteString(styles.DimmedStyle.Render("  (Shift+Tab to switch)"))
	b.WriteString("\n\n")

	// Calculate column widths
	colWidth := (width - 4) / 2
	if colWidth < 25 {
		colWidth = 25
	}
	visibleHeight := m.getVisibleHeight()

	// Build categories column
	catLines := m.renderSection(
		"CATEGORIES",
		m.categories,
		colWidth,
		visibleHeight,
		m.section == IgnoreSectionCategories,
	)

	// Build packages column
	pkgLines := m.renderSection(
		"PACKAGES",
		m.packages,
		colWidth,
		visibleHeight,
		m.section == IgnoreSectionPackages,
	)

	// Combine columns
	maxLines := len(catLines)
	if len(pkgLines) > maxLines {
		maxLines = len(pkgLines)
	}

	for i := 0; i < maxLines; i++ {
		left := ""
		right := ""
		if i < len(catLines) {
			left = catLines[i]
		}
		if i < len(pkgLines) {
			right = pkgLines[i]
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

	// Status message
	if m.statusMessage != "" {
		b.WriteString("\n")
		if m.statusType == "success" {
			b.WriteString(styles.SelectedStyle.Render("✓ " + m.statusMessage))
		} else {
			b.WriteString(styles.ErrorStyle.Render("✗ " + m.statusMessage))
		}
		b.WriteString("\n")
	}

	// Input mode overlay
	if m.showTypeMenu {
		b.WriteString("\n")
		b.WriteString(m.renderTypeMenu())
	} else if m.inputMode {
		b.WriteString("\n")
		b.WriteString(m.renderTextInput())
	} else if m.showConfirm {
		b.WriteString("\n")
		b.WriteString(m.renderConfirmDialog())
	} else {
		// Help line
		b.WriteString("\n")
		helpStyle := lipgloss.NewStyle().Foreground(styles.CatSubtext0)
		b.WriteString(helpStyle.Render("Tab"))
		b.WriteString(styles.DimmedStyle.Render(":switch section • "))
		b.WriteString(helpStyle.Render("a"))
		b.WriteString(styles.DimmedStyle.Render(":add • "))
		b.WriteString(helpStyle.Render("d/x"))
		b.WriteString(styles.DimmedStyle.Render(":delete • "))
		b.WriteString(helpStyle.Render("j/k"))
		b.WriteString(styles.DimmedStyle.Render(":navigate • "))
		b.WriteString(helpStyle.Render("esc"))
		b.WriteString(styles.DimmedStyle.Render(":back"))
	}

	return b.String()
}

func (m *IgnoreModel) renderSection(title string, items []ignoreItem, width, visibleHeight int, focused bool) []string {
	var lines []string

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatText)
	if focused {
		headerStyle = headerStyle.Foreground(styles.CatMauve).Underline(true)
	}
	lines = append(lines, headerStyle.Render(fmt.Sprintf("%s (%d)", title, len(items))))
	lines = append(lines, styles.DimmedStyle.Render(strings.Repeat("─", width-2)))

	if len(items) == 0 {
		lines = append(lines, styles.DimmedStyle.Render("  (none)"))
		return lines
	}

	// Determine offset for this section
	offset := 0
	cursor := -1
	if focused {
		offset = m.offset
		cursor = m.cursor
	}

	// Calculate visible range
	endIdx := offset + visibleHeight
	if endIdx > len(items) {
		endIdx = len(items)
	}

	for i := offset; i < endIdx; i++ {
		item := items[i]
		isCursor := focused && i == cursor

		prefix := "  "
		if isCursor {
			prefix = styles.CursorStyle.Render("> ")
		}

		// Scope indicator
		scopeLabel := ""
		if item.isGlobal {
			scopeLabel = styles.DimmedStyle.Render(" (global)")
		} else {
			scopeLabel = lipgloss.NewStyle().Foreground(styles.CatBlue).Render(" (machine)")
		}

		// Value styling
		valueStyle := lipgloss.NewStyle().Foreground(styles.CatText)
		if isCursor {
			valueStyle = lipgloss.NewStyle().Foreground(styles.CatMauve).Bold(true)
		}

		// Truncate if needed
		maxLen := width - 15
		value := item.value
		if len(value) > maxLen {
			value = value[:maxLen-3] + "..."
		}

		line := prefix + valueStyle.Render(value) + scopeLabel
		lines = append(lines, line)
	}

	// Scroll indicator
	if len(items) > visibleHeight && focused {
		scrollInfo := fmt.Sprintf(" %d/%d ", cursor+1, len(items))
		lines = append(lines, styles.DimmedStyle.Render(scrollInfo))
	}

	return lines
}

func (m *IgnoreModel) renderTypeMenu() string {
	var b strings.Builder

	types := categoryTypes
	label := "Select category type:"
	if m.inputType == "package" {
		types = packageTypes
		label = "Select package type:"
	}

	scopeLabel := "global"
	if m.inputScope == IgnoreScopeMachine {
		scopeLabel = "machine"
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatYellow)
	b.WriteString(headerStyle.Render(fmt.Sprintf("%s (%s)", label, scopeLabel)))
	b.WriteString("\n")

	for i, t := range types {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(styles.CatText)
		if i == m.typeMenuIdx {
			prefix = styles.CursorStyle.Render("> ")
			style = style.Foreground(styles.CatMauve).Bold(true)
		}
		b.WriteString(prefix + style.Render(t) + "\n")
	}

	b.WriteString("\n" + styles.DimmedStyle.Render("Enter:select • Esc:cancel"))

	return b.String()
}

func (m *IgnoreModel) renderTextInput() string {
	var b strings.Builder

	scopeLabel := "global"
	if m.inputScope == IgnoreScopeMachine {
		scopeLabel = "machine"
	}

	label := fmt.Sprintf("Add %s (%s):", m.inputType, scopeLabel)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatYellow)
	b.WriteString(headerStyle.Render(label))
	b.WriteString("\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n")
	b.WriteString(styles.DimmedStyle.Render("Enter:confirm • Esc:cancel"))

	return b.String()
}

func (m *IgnoreModel) renderConfirmDialog() string {
	var b strings.Builder

	scopeLabel := "global"
	if !m.confirmItem.isGlobal {
		scopeLabel = "machine"
	}

	confirmStyle := lipgloss.NewStyle().
		Background(styles.CatSurface0).
		Foreground(styles.CatYellow).
		Padding(0, 1).
		Bold(true)

	b.WriteString(confirmStyle.Render(fmt.Sprintf("Delete '%s' (%s)? (y/n)", m.confirmItem.value, scopeLabel)))

	return b.String()
}
