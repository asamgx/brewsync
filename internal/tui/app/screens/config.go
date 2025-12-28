package screens

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/asamgx/brewsync/internal/config"
	"github.com/asamgx/brewsync/internal/tui/styles"
)

// ConfigSection represents which section is focused
type ConfigSection int

const (
	ConfigSectionMachines ConfigSection = iota
	ConfigSectionGeneral
	ConfigSectionAutoDump
	ConfigSectionDump
	ConfigSectionOutput
)

// configItem represents an editable config item
type configItem struct {
	key         string
	label       string
	value       string
	itemType    string // "bool", "string", "select", "readonly"
	options     []string
	description string
}

// ConfigModel is the model for the config screen
type ConfigModel struct {
	config  *config.Config
	width   int
	height  int
	section ConfigSection
	cursor  int
	offset  int

	// Machine list
	machines     []string
	machineItems []configItem // Items for selected machine

	// Section items
	generalItems  []configItem
	autoDumpItems []configItem
	dumpItems     []configItem
	outputItems   []configItem

	// Edit mode
	editing      bool
	editingItem  *configItem
	textInput    textinput.Model
	selectIdx    int
	categoryEdit bool // Multi-select for categories

	// Machine editing
	editingMachine    bool
	machineEditField  int // 0=hostname, 1=brewfile, 2=description
	selectedMachine   string
	machineEditItems  []configItem
	addingMachine     bool
	newMachineName    string

	// Status
	statusMessage string
	statusType    string
	hasChanges    bool
}

// NewConfigModel creates a new config model
func NewConfigModel(cfg *config.Config) *ConfigModel {
	ti := textinput.New()
	ti.CharLimit = 200
	ti.Width = 50

	m := &ConfigModel{
		config:    cfg,
		width:     80,
		height:    24,
		textInput: ti,
	}

	m.loadMachines()
	m.buildItems()

	return m
}

func (m *ConfigModel) loadMachines() {
	m.machines = []string{}
	if m.config != nil {
		for name := range m.config.Machines {
			m.machines = append(m.machines, name)
		}
		sort.Strings(m.machines)
	}
}

func (m *ConfigModel) buildItems() {
	if m.config == nil {
		return
	}

	// General section items
	m.generalItems = []configItem{
		{
			key:         "current_machine",
			label:       "Current Machine",
			value:       m.config.CurrentMachine,
			itemType:    "select",
			options:     append([]string{"auto"}, m.machines...),
			description: "Machine to use (auto-detects by hostname)",
		},
		{
			key:         "default_source",
			label:       "Default Source",
			value:       m.config.DefaultSource,
			itemType:    "select",
			options:     m.machines,
			description: "Default machine to import/sync from",
		},
		{
			key:         "default_categories",
			label:       "Default Categories",
			value:       strings.Join(m.config.DefaultCategories, ", "),
			itemType:    "categories",
			options:     []string{"tap", "brew", "cask", "vscode", "cursor", "antigravity", "go", "mas"},
			description: "Package types to include by default",
		},
		{
			key:         "conflict_resolution",
			label:       "Conflict Resolution",
			value:       string(m.config.ConflictResolution),
			itemType:    "select",
			options:     []string{"ask", "skip", "source-wins", "current-wins"},
			description: "How to handle conflicts during sync",
		},
	}

	// Auto-dump section items
	m.autoDumpItems = []configItem{
		{
			key:         "auto_dump.enabled",
			label:       "Enabled",
			value:       boolToYesNo(m.config.AutoDump.Enabled),
			itemType:    "bool",
			description: "Enable automatic Brewfile dumps",
		},
		{
			key:         "auto_dump.after_install",
			label:       "After Install",
			value:       boolToYesNo(m.config.AutoDump.AfterInstall),
			itemType:    "bool",
			description: "Dump after installing packages",
		},
		{
			key:         "auto_dump.commit",
			label:       "Auto Commit",
			value:       boolToYesNo(m.config.AutoDump.Commit),
			itemType:    "bool",
			description: "Commit changes after dump",
		},
		{
			key:         "auto_dump.push",
			label:       "Auto Push",
			value:       boolToYesNo(m.config.AutoDump.Push),
			itemType:    "bool",
			description: "Push changes after commit",
		},
		{
			key:         "auto_dump.commit_message",
			label:       "Commit Message",
			value:       m.config.AutoDump.CommitMessage,
			itemType:    "string",
			description: "Message for auto-commits ({machine} placeholder)",
		},
	}

	// Dump section items
	m.dumpItems = []configItem{
		{
			key:         "dump.use_brew_bundle",
			label:       "Use Brew Bundle",
			value:       boolToYesNo(m.config.Dump.UseBrewBundle),
			itemType:    "bool",
			description: "Use 'brew bundle dump --describe' for better output",
		},
	}

	// Output section items
	m.outputItems = []configItem{
		{
			key:         "output.color",
			label:       "Color Output",
			value:       boolToYesNo(m.config.Output.Color),
			itemType:    "bool",
			description: "Enable colored terminal output",
		},
		{
			key:         "output.verbose",
			label:       "Verbose",
			value:       boolToYesNo(m.config.Output.Verbose),
			itemType:    "bool",
			description: "Show detailed output",
		},
		{
			key:         "output.show_descriptions",
			label:       "Show Descriptions",
			value:       boolToYesNo(m.config.Output.ShowDescriptions),
			itemType:    "bool",
			description: "Show package descriptions in lists",
		},
	}
}

func (m *ConfigModel) buildMachineEditItems() {
	if m.selectedMachine == "" || m.config == nil {
		m.machineEditItems = nil
		return
	}

	machine, ok := m.config.Machines[m.selectedMachine]
	if !ok {
		m.machineEditItems = nil
		return
	}

	m.machineEditItems = []configItem{
		{
			key:         "hostname",
			label:       "Hostname",
			value:       machine.Hostname,
			itemType:    "string",
			description: "System hostname for auto-detection",
		},
		{
			key:         "brewfile",
			label:       "Brewfile Path",
			value:       machine.Brewfile,
			itemType:    "string",
			description: "Path to the Brewfile",
		},
		{
			key:         "description",
			label:       "Description",
			value:       machine.Description,
			itemType:    "string",
			description: "Optional description",
		},
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
		// Clear status on any key
		m.statusMessage = ""

		// Handle text input mode
		if m.editing && m.editingItem != nil && m.editingItem.itemType == "string" {
			return m.handleTextInput(msg)
		}

		// Handle select mode
		if m.editing && m.editingItem != nil && (m.editingItem.itemType == "select" || m.editingItem.itemType == "categories") {
			return m.handleSelectInput(msg)
		}

		// Handle machine editing
		if m.editingMachine {
			return m.handleMachineEdit(msg)
		}

		// Handle adding new machine
		if m.addingMachine {
			return m.handleAddMachine(msg)
		}

		// Normal mode
		return m.handleNormalInput(msg)
	}

	return m, nil
}

func (m *ConfigModel) handleTextInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		value := m.textInput.Value()
		m.applyChange(m.editingItem.key, value)
		m.editing = false
		m.editingItem = nil
		m.textInput.SetValue("")
		return m, nil
	case "esc":
		m.editing = false
		m.editingItem = nil
		m.textInput.SetValue("")
		return m, nil
	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

func (m *ConfigModel) handleSelectInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectIdx > 0 {
			m.selectIdx--
		}
	case "down", "j":
		if m.selectIdx < len(m.editingItem.options)-1 {
			m.selectIdx++
		}
	case "enter":
		if m.editingItem.itemType == "categories" {
			// Toggle category
			m.toggleCategory(m.editingItem.options[m.selectIdx])
		} else {
			// Select option
			m.applyChange(m.editingItem.key, m.editingItem.options[m.selectIdx])
			m.editing = false
			m.editingItem = nil
		}
	case "esc":
		m.editing = false
		m.editingItem = nil
	case " ":
		if m.editingItem.itemType == "categories" {
			m.toggleCategory(m.editingItem.options[m.selectIdx])
		}
	}
	return m, nil
}

func (m *ConfigModel) toggleCategory(cat string) {
	// Check if category is currently enabled
	enabled := false
	for _, c := range m.config.DefaultCategories {
		if c == cat {
			enabled = true
			break
		}
	}

	if enabled {
		// Remove category
		var newCats []string
		for _, c := range m.config.DefaultCategories {
			if c != cat {
				newCats = append(newCats, c)
			}
		}
		m.config.DefaultCategories = newCats
	} else {
		// Add category
		m.config.DefaultCategories = append(m.config.DefaultCategories, cat)
	}

	m.hasChanges = true
	m.buildItems() // Refresh display
}

func (m *ConfigModel) handleMachineEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.editingMachine = false
		m.selectedMachine = ""
		m.machineEditItems = nil
		m.cursor = 0
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.machineEditItems)-1 {
			m.cursor++
		}
	case "enter", " ":
		if m.cursor < len(m.machineEditItems) {
			item := &m.machineEditItems[m.cursor]
			m.editing = true
			m.editingItem = item
			m.textInput.SetValue(item.value)
			m.textInput.Focus()
		}
	case "s":
		// Save machine changes
		m.saveMachineChanges()
		m.editingMachine = false
		m.selectedMachine = ""
		m.machineEditItems = nil
		m.cursor = 0
	case "d", "x":
		// Delete machine (with confirmation)
		if m.selectedMachine != m.config.CurrentMachine {
			delete(m.config.Machines, m.selectedMachine)
			m.hasChanges = true
			m.loadMachines()
			m.buildItems()
			m.editingMachine = false
			m.selectedMachine = ""
			m.machineEditItems = nil
			m.cursor = 0
			m.statusMessage = "Machine deleted"
			m.statusType = "success"
		} else {
			m.statusMessage = "Cannot delete current machine"
			m.statusType = "error"
		}
	}
	return m, nil
}

func (m *ConfigModel) saveMachineChanges() {
	if m.selectedMachine == "" || len(m.machineEditItems) < 3 {
		return
	}

	machine := config.Machine{
		Hostname:    m.machineEditItems[0].value,
		Brewfile:    m.machineEditItems[1].value,
		Description: m.machineEditItems[2].value,
	}

	m.config.Machines[m.selectedMachine] = machine
	m.hasChanges = true
	m.buildItems()
	m.statusMessage = "Machine updated"
	m.statusType = "success"
}

func (m *ConfigModel) handleAddMachine(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.textInput.Value())
		if name != "" {
			if _, exists := m.config.Machines[name]; exists {
				m.statusMessage = "Machine already exists"
				m.statusType = "error"
			} else {
				m.config.Machines[name] = config.Machine{
					Hostname:    "",
					Brewfile:    "",
					Description: "",
				}
				m.hasChanges = true
				m.loadMachines()
				m.buildItems()

				// Enter edit mode for the new machine
				m.selectedMachine = name
				m.buildMachineEditItems()
				m.editingMachine = true
				m.cursor = 0

				m.statusMessage = "Machine created - please fill in details"
				m.statusType = "success"
			}
		}
		m.addingMachine = false
		m.textInput.SetValue("")
	case "esc":
		m.addingMachine = false
		m.textInput.SetValue("")
	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *ConfigModel) handleNormalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		if m.hasChanges {
			// Prompt to save? For now just go back
		}
		return m, func() tea.Msg { return Navigate("dashboard") }

	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		m.section = (m.section + 1) % 5
		m.cursor = 0
		m.offset = 0

	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
		if m.section == 0 {
			m.section = 4
		} else {
			m.section--
		}
		m.cursor = 0
		m.offset = 0

	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.cursor > 0 {
			m.cursor--
			m.adjustOffset()
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		maxItems := m.getMaxItems()
		if m.cursor < maxItems-1 {
			m.cursor++
			m.adjustOffset()
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
		return m.activateCurrentItem()

	case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
		// Add new machine (only in machines section)
		if m.section == ConfigSectionMachines {
			m.addingMachine = true
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Enter machine name..."
			m.textInput.Focus()
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
		// Save config
		if m.hasChanges {
			if err := config.Save(m.config); err != nil {
				m.statusMessage = fmt.Sprintf("Save failed: %v", err)
				m.statusType = "error"
			} else {
				m.statusMessage = "Config saved"
				m.statusType = "success"
				m.hasChanges = false
			}
		}
	}

	return m, nil
}

func (m *ConfigModel) activateCurrentItem() (tea.Model, tea.Cmd) {
	var item *configItem

	switch m.section {
	case ConfigSectionMachines:
		if m.cursor < len(m.machines) {
			// Edit machine
			m.selectedMachine = m.machines[m.cursor]
			m.buildMachineEditItems()
			m.editingMachine = true
			m.cursor = 0
			return m, nil
		}
	case ConfigSectionGeneral:
		if m.cursor < len(m.generalItems) {
			item = &m.generalItems[m.cursor]
		}
	case ConfigSectionAutoDump:
		if m.cursor < len(m.autoDumpItems) {
			item = &m.autoDumpItems[m.cursor]
		}
	case ConfigSectionDump:
		if m.cursor < len(m.dumpItems) {
			item = &m.dumpItems[m.cursor]
		}
	case ConfigSectionOutput:
		if m.cursor < len(m.outputItems) {
			item = &m.outputItems[m.cursor]
		}
	}

	if item == nil {
		return m, nil
	}

	switch item.itemType {
	case "bool":
		// Toggle boolean
		newValue := "No"
		if item.value == "No" {
			newValue = "Yes"
		}
		m.applyChange(item.key, newValue)
	case "string":
		m.editing = true
		m.editingItem = item
		m.textInput.SetValue(item.value)
		m.textInput.Placeholder = item.description
		m.textInput.Focus()
	case "select", "categories":
		m.editing = true
		m.editingItem = item
		// Find current selection index
		m.selectIdx = 0
		for i, opt := range item.options {
			if opt == item.value {
				m.selectIdx = i
				break
			}
		}
	}

	return m, nil
}

func (m *ConfigModel) applyChange(key, value string) {
	if m.config == nil {
		return
	}

	switch key {
	case "current_machine":
		m.config.CurrentMachine = value
	case "default_source":
		m.config.DefaultSource = value
	case "conflict_resolution":
		m.config.ConflictResolution = config.ConflictResolution(value)
	case "auto_dump.enabled":
		m.config.AutoDump.Enabled = value == "Yes"
	case "auto_dump.after_install":
		m.config.AutoDump.AfterInstall = value == "Yes"
	case "auto_dump.commit":
		m.config.AutoDump.Commit = value == "Yes"
	case "auto_dump.push":
		m.config.AutoDump.Push = value == "Yes"
	case "auto_dump.commit_message":
		m.config.AutoDump.CommitMessage = value
	case "dump.use_brew_bundle":
		m.config.Dump.UseBrewBundle = value == "Yes"
	case "output.color":
		m.config.Output.Color = value == "Yes"
	case "output.verbose":
		m.config.Output.Verbose = value == "Yes"
	case "output.show_descriptions":
		m.config.Output.ShowDescriptions = value == "Yes"
	// Machine edit fields
	case "hostname":
		if m.selectedMachine != "" && len(m.machineEditItems) > 0 {
			m.machineEditItems[0].value = value
		}
	case "brewfile":
		if m.selectedMachine != "" && len(m.machineEditItems) > 1 {
			m.machineEditItems[1].value = value
		}
	case "description":
		if m.selectedMachine != "" && len(m.machineEditItems) > 2 {
			m.machineEditItems[2].value = value
		}
	}

	m.hasChanges = true
	m.buildItems()
}

func (m *ConfigModel) getMaxItems() int {
	switch m.section {
	case ConfigSectionMachines:
		return len(m.machines)
	case ConfigSectionGeneral:
		return len(m.generalItems)
	case ConfigSectionAutoDump:
		return len(m.autoDumpItems)
	case ConfigSectionDump:
		return len(m.dumpItems)
	case ConfigSectionOutput:
		return len(m.outputItems)
	}
	return 0
}

func (m *ConfigModel) getVisibleHeight() int {
	h := m.height - 12
	if h < 5 {
		h = 5
	}
	return h
}

func (m *ConfigModel) adjustOffset() {
	visibleHeight := m.getVisibleHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visibleHeight {
		m.offset = m.cursor - visibleHeight + 1
	}
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

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatMauve)
	title := "Configuration"
	if m.hasChanges {
		title += " [modified]"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Handle special modes
	if m.addingMachine {
		return m.renderAddMachine()
	}

	if m.editingMachine {
		return m.renderMachineEdit()
	}

	if m.editing && m.editingItem != nil {
		if m.editingItem.itemType == "select" || m.editingItem.itemType == "categories" {
			return m.renderSelectMode()
		}
		if m.editingItem.itemType == "string" {
			return m.renderTextInputMode()
		}
	}

	// Section tabs
	sections := []string{"Machines", "General", "Auto-Dump", "Dump", "Output"}
	var tabs []string
	for i, s := range sections {
		if ConfigSection(i) == m.section {
			tabs = append(tabs, lipgloss.NewStyle().Bold(true).Foreground(styles.CatMauve).Underline(true).Render(s))
		} else {
			tabs = append(tabs, styles.DimmedStyle.Render(s))
		}
	}
	b.WriteString(strings.Join(tabs, " │ "))
	b.WriteString("\n")
	b.WriteString(styles.DimmedStyle.Render(strings.Repeat("─", width-4)))
	b.WriteString("\n\n")

	// Render current section
	switch m.section {
	case ConfigSectionMachines:
		b.WriteString(m.renderMachinesSection())
	case ConfigSectionGeneral:
		b.WriteString(m.renderItemsSection(m.generalItems))
	case ConfigSectionAutoDump:
		b.WriteString(m.renderItemsSection(m.autoDumpItems))
	case ConfigSectionDump:
		b.WriteString(m.renderItemsSection(m.dumpItems))
	case ConfigSectionOutput:
		b.WriteString(m.renderItemsSection(m.outputItems))
	}

	// Status message
	if m.statusMessage != "" {
		b.WriteString("\n")
		if m.statusType == "success" {
			b.WriteString(styles.SelectedStyle.Render("✓ " + m.statusMessage))
		} else {
			b.WriteString(styles.ErrorStyle.Render("✗ " + m.statusMessage))
		}
	}

	// Help line
	b.WriteString("\n\n")
	helpStyle := lipgloss.NewStyle().Foreground(styles.CatSubtext0)
	b.WriteString(helpStyle.Render("Tab"))
	b.WriteString(styles.DimmedStyle.Render(":sections • "))
	b.WriteString(helpStyle.Render("Enter/Space"))
	b.WriteString(styles.DimmedStyle.Render(":edit • "))
	if m.section == ConfigSectionMachines {
		b.WriteString(helpStyle.Render("a"))
		b.WriteString(styles.DimmedStyle.Render(":add • "))
	}
	if m.hasChanges {
		b.WriteString(helpStyle.Render("s"))
		b.WriteString(styles.DimmedStyle.Render(":save • "))
	}
	b.WriteString(helpStyle.Render("esc"))
	b.WriteString(styles.DimmedStyle.Render(":back"))

	return b.String()
}

func (m *ConfigModel) renderMachinesSection() string {
	var b strings.Builder

	if len(m.machines) == 0 {
		b.WriteString(styles.DimmedStyle.Render("  No machines configured"))
		b.WriteString("\n")
		return b.String()
	}

	for i, name := range m.machines {
		machine, _ := m.config.Machines[name]
		isCursor := i == m.cursor

		prefix := "  "
		if isCursor {
			prefix = styles.CursorStyle.Render("> ")
		}

		// Machine name
		nameStyle := lipgloss.NewStyle().Foreground(styles.CatText)
		if isCursor {
			nameStyle = nameStyle.Foreground(styles.CatMauve).Bold(true)
		}

		// Current indicator
		current := ""
		if name == m.config.CurrentMachine {
			current = styles.SelectedStyle.Render(" (current)")
		}

		// Default source indicator
		defaultSrc := ""
		if name == m.config.DefaultSource {
			defaultSrc = lipgloss.NewStyle().Foreground(styles.CatBlue).Render(" [default source]")
		}

		b.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, nameStyle.Render(name), current, defaultSrc))

		// Details (only for cursor item)
		if isCursor {
			detailStyle := styles.DimmedStyle
			b.WriteString(detailStyle.Render(fmt.Sprintf("      Hostname: %s", machine.Hostname)) + "\n")
			b.WriteString(detailStyle.Render(fmt.Sprintf("      Brewfile: %s", machine.Brewfile)) + "\n")
			if machine.Description != "" {
				b.WriteString(detailStyle.Render(fmt.Sprintf("      Description: %s", machine.Description)) + "\n")
			}
		}
	}

	return b.String()
}

func (m *ConfigModel) renderItemsSection(items []configItem) string {
	var b strings.Builder

	labelWidth := 20
	for _, item := range items {
		if len(item.label) > labelWidth-2 {
			labelWidth = len(item.label) + 2
		}
	}

	for i, item := range items {
		isCursor := i == m.cursor

		prefix := "  "
		if isCursor {
			prefix = styles.CursorStyle.Render("> ")
		}

		// Label - pad manually instead of using Width()
		labelStyle := lipgloss.NewStyle().Foreground(styles.CatSubtext0)
		if isCursor {
			labelStyle = labelStyle.Foreground(styles.CatText)
		}
		label := item.label
		if len(label) < labelWidth {
			label = label + strings.Repeat(" ", labelWidth-len(label))
		}

		// Value
		valueStyle := lipgloss.NewStyle().Foreground(styles.CatText)
		if isCursor {
			valueStyle = valueStyle.Foreground(styles.CatMauve).Bold(true)
		}

		// Type indicator
		typeIndicator := ""
		switch item.itemType {
		case "bool":
			if item.value == "Yes" {
				typeIndicator = styles.SelectedStyle.Render(" ●")
			} else {
				typeIndicator = styles.DimmedStyle.Render(" ○")
			}
		case "select":
			typeIndicator = styles.DimmedStyle.Render(" ▾")
		case "categories":
			typeIndicator = styles.DimmedStyle.Render(" [...]")
		}

		b.WriteString(fmt.Sprintf("%s%s%s%s\n",
			prefix,
			labelStyle.Render(label),
			valueStyle.Render(item.value),
			typeIndicator,
		))

		// Description for current item - on its own line with proper indentation
		if isCursor && item.description != "" {
			b.WriteString(fmt.Sprintf("      %s\n", styles.DimmedStyle.Render(item.description)))
		}
	}

	return b.String()
}

func (m *ConfigModel) renderMachineEdit() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatMauve)
	b.WriteString(titleStyle.Render(fmt.Sprintf("Edit Machine: %s", m.selectedMachine)))
	b.WriteString("\n\n")

	// If editing a text field
	if m.editing && m.editingItem != nil && m.editingItem.itemType == "string" {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.CatYellow).Render(m.editingItem.label + ":"))
		b.WriteString("\n")
		b.WriteString(m.textInput.View())
		b.WriteString("\n\n")
		b.WriteString(styles.DimmedStyle.Render("Enter:confirm • Esc:cancel"))
		return b.String()
	}

	for i, item := range m.machineEditItems {
		isCursor := i == m.cursor

		prefix := "  "
		if isCursor {
			prefix = styles.CursorStyle.Render("> ")
		}

		labelStyle := lipgloss.NewStyle().Foreground(styles.CatSubtext0).Width(15)
		valueStyle := lipgloss.NewStyle().Foreground(styles.CatText)
		if isCursor {
			labelStyle = labelStyle.Foreground(styles.CatText)
			valueStyle = valueStyle.Foreground(styles.CatMauve).Bold(true)
		}

		value := item.value
		if value == "" {
			value = styles.DimmedStyle.Render("(not set)")
		} else {
			value = valueStyle.Render(value)
		}

		b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, labelStyle.Render(item.label), value))
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(styles.CatSubtext0)
	b.WriteString(helpStyle.Render("Enter"))
	b.WriteString(styles.DimmedStyle.Render(":edit • "))
	b.WriteString(helpStyle.Render("s"))
	b.WriteString(styles.DimmedStyle.Render(":save • "))
	b.WriteString(helpStyle.Render("d"))
	b.WriteString(styles.DimmedStyle.Render(":delete • "))
	b.WriteString(helpStyle.Render("esc"))
	b.WriteString(styles.DimmedStyle.Render(":back"))

	return b.String()
}

func (m *ConfigModel) renderAddMachine() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatYellow)
	b.WriteString(titleStyle.Render("Add New Machine"))
	b.WriteString("\n\n")
	b.WriteString("Machine Name:\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")
	b.WriteString(styles.DimmedStyle.Render("Enter:create • Esc:cancel"))

	return b.String()
}

func (m *ConfigModel) renderSelectMode() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatYellow)
	b.WriteString(titleStyle.Render(m.editingItem.label))
	b.WriteString("\n")
	b.WriteString(styles.DimmedStyle.Render(m.editingItem.description))
	b.WriteString("\n\n")

	isCategories := m.editingItem.itemType == "categories"

	for i, opt := range m.editingItem.options {
		isCursor := i == m.selectIdx

		prefix := "  "
		if isCursor {
			prefix = styles.CursorStyle.Render("> ")
		}

		style := lipgloss.NewStyle().Foreground(styles.CatText)
		if isCursor {
			style = style.Foreground(styles.CatMauve).Bold(true)
		}

		// For categories, show checkbox
		checkbox := ""
		if isCategories {
			enabled := false
			for _, c := range m.config.DefaultCategories {
				if c == opt {
					enabled = true
					break
				}
			}
			if enabled {
				checkbox = styles.SelectedStyle.Render("[✓] ")
			} else {
				checkbox = styles.DimmedStyle.Render("[ ] ")
			}
		}

		// For select, show current indicator
		current := ""
		if !isCategories && opt == m.editingItem.value {
			current = styles.SelectedStyle.Render(" ●")
		}

		b.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, checkbox, style.Render(opt), current))
	}

	b.WriteString("\n")
	if isCategories {
		b.WriteString(styles.DimmedStyle.Render("Space/Enter:toggle • Esc:done"))
	} else {
		b.WriteString(styles.DimmedStyle.Render("Enter:select • Esc:cancel"))
	}

	return b.String()
}

func (m *ConfigModel) renderTextInputMode() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.CatYellow)
	b.WriteString(titleStyle.Render(m.editingItem.label))
	b.WriteString("\n")
	b.WriteString(styles.DimmedStyle.Render(m.editingItem.description))
	b.WriteString("\n\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")
	b.WriteString(styles.DimmedStyle.Render("Enter:confirm • Esc:cancel"))

	return b.String()
}

func boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
