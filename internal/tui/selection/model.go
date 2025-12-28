package selection

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"

	"github.com/asamgx/brewsync/internal/brewfile"
)

// Category represents a package category tab
type Category string

const (
	CategoryAll         Category = "all"
	CategoryTap         Category = "tap"
	CategoryBrew        Category = "brew"
	CategoryCask        Category = "cask"
	CategoryVSCode      Category = "vscode"
	CategoryCursor      Category = "cursor"
	CategoryAntigravity Category = "antigravity"
	CategoryGo          Category = "go"
	CategoryMas         Category = "mas"
)

// AllCategories returns all available categories in order
func AllCategories() []Category {
	return []Category{
		CategoryAll,
		CategoryTap,
		CategoryBrew,
		CategoryCask,
		CategoryVSCode,
		CategoryCursor,
		CategoryAntigravity,
		CategoryGo,
		CategoryMas,
	}
}

// Item represents a selectable package item
type Item struct {
	Package  brewfile.Package
	Selected bool
	Ignored  bool
}

// FilterValue returns the value used for filtering
func (i Item) FilterValue() string {
	return i.Package.Name
}

// Model is the Bubble Tea model for package selection
type Model struct {
	title             string
	items             []Item
	cursor            int
	category          Category
	searching         bool
	searchText        textinput.Model
	filtered          []int // indices into items that match current filter
	keys              KeyMap
	help              help.Model
	showHelp          bool
	showIgnored       bool // Whether to show ignored items in the list
	width             int
	height            int
	cancelled         bool
	confirmed         bool
	ignoredCategories map[string]bool // Track categories marked for ignoring
}

// New creates a new selection model
func New(title string, packages brewfile.Packages) Model {
	items := make([]Item, len(packages))
	for i, pkg := range packages {
		items[i] = Item{
			Package:  pkg,
			Selected: false,
			Ignored:  false,
		}
	}

	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 50
	ti.Width = 30

	m := Model{
		title:             title,
		items:             items,
		cursor:            0,
		category:          CategoryAll,
		searching:         false,
		searchText:        ti,
		keys:              DefaultKeyMap(),
		help:              help.New(),
		showHelp:          false,
		width:             80,
		height:            24,
		ignoredCategories: make(map[string]bool),
	}

	m.updateFiltered()
	return m
}

// SetIgnored marks specific packages as ignored
func (m *Model) SetIgnored(ignored map[string]bool) {
	for i := range m.items {
		key := m.items[i].Package.ID()
		if ignored[key] {
			m.items[i].Ignored = true
		}
	}
}

// SetSelected marks specific packages as pre-selected
func (m *Model) SetSelected(selected map[string]bool) {
	for i := range m.items {
		key := m.items[i].Package.ID()
		if selected[key] {
			m.items[i].Selected = true
		}
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case tea.KeyMsg:
		// Handle search mode
		if m.searching {
			switch {
			case key.Matches(msg, m.keys.ClearSearch):
				m.searching = false
				m.searchText.SetValue("")
				m.updateFiltered()
				return m, nil
			case key.Matches(msg, m.keys.Confirm):
				m.searching = false
				return m, nil
			default:
				m.searchText, cmd = m.searchText.Update(msg)
				m.updateFiltered()
				m.cursor = 0
				return m, cmd
			}
		}

		// Normal mode keybindings
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.cancelled = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Confirm):
			m.confirmed = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Up):
			m.moveCursor(-1)

		case key.Matches(msg, m.keys.Down):
			m.moveCursor(1)

		case key.Matches(msg, m.keys.PageUp):
			m.moveCursor(-10)

		case key.Matches(msg, m.keys.PageDown):
			m.moveCursor(10)

		case key.Matches(msg, m.keys.Left):
			m.prevCategory()

		case key.Matches(msg, m.keys.Right):
			m.nextCategory()

		case key.Matches(msg, m.keys.Toggle):
			m.toggleCurrent()

		case key.Matches(msg, m.keys.SelectAll):
			m.selectAllVisible(true)

		case key.Matches(msg, m.keys.SelectNone):
			m.selectAllVisible(false)

		case key.Matches(msg, m.keys.Search):
			m.searching = true
			m.searchText.Focus()
			return m, textinput.Blink

		case key.Matches(msg, m.keys.Ignore):
			m.toggleIgnoreCurrent()

		case key.Matches(msg, m.keys.IgnoreCategory):
			m.toggleIgnoreCurrentCategory()

		case key.Matches(msg, m.keys.ToggleShowIgnored):
			m.showIgnored = !m.showIgnored
			m.updateFiltered()
			// Reset cursor if it's now out of bounds
			if m.cursor >= len(m.filtered) {
				m.cursor = len(m.filtered) - 1
				if m.cursor < 0 {
					m.cursor = 0
				}
			}

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp

		case key.Matches(msg, m.keys.TabAll):
			m.setCategory(CategoryAll)
		case key.Matches(msg, m.keys.TabTap):
			m.setCategory(CategoryTap)
		case key.Matches(msg, m.keys.TabBrew):
			m.setCategory(CategoryBrew)
		case key.Matches(msg, m.keys.TabCask):
			m.setCategory(CategoryCask)
		case key.Matches(msg, m.keys.TabVSCode):
			m.setCategory(CategoryVSCode)
		case key.Matches(msg, m.keys.TabCursor):
			m.setCategory(CategoryCursor)
		case key.Matches(msg, m.keys.TabAntigravity):
			m.setCategory(CategoryAntigravity)
		case key.Matches(msg, m.keys.TabGo):
			m.setCategory(CategoryGo)
		case key.Matches(msg, m.keys.TabMas):
			m.setCategory(CategoryMas)
		}
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	return m.render()
}

// Selected returns all selected (non-ignored) packages
func (m Model) Selected() brewfile.Packages {
	var selected brewfile.Packages
	for _, item := range m.items {
		if item.Selected && !item.Ignored {
			selected = append(selected, item.Package)
		}
	}
	return selected
}

// Ignored returns all packages marked as ignored
func (m Model) Ignored() brewfile.Packages {
	var ignored brewfile.Packages
	for _, item := range m.items {
		if item.Ignored {
			ignored = append(ignored, item.Package)
		}
	}
	return ignored
}

// IgnoredCategories returns the list of categories marked for ignoring
func (m Model) IgnoredCategories() []string {
	var categories []string
	for cat := range m.ignoredCategories {
		categories = append(categories, cat)
	}
	return categories
}

// Cancelled returns true if the user cancelled
func (m Model) Cancelled() bool {
	return m.cancelled
}

// Confirmed returns true if the user confirmed
func (m Model) Confirmed() bool {
	return m.confirmed
}

// updateFiltered updates the filtered list based on category, search, and showIgnored
func (m *Model) updateFiltered() {
	m.filtered = nil

	// First filter by category and ignored state
	var categoryFiltered []int
	for i, item := range m.items {
		// Skip ignored items if not showing them
		if item.Ignored && !m.showIgnored {
			continue
		}
		if m.category == CategoryAll || Category(item.Package.Type) == m.category {
			categoryFiltered = append(categoryFiltered, i)
		}
	}

	// Then filter by search
	searchTerm := strings.TrimSpace(m.searchText.Value())
	if searchTerm == "" {
		m.filtered = categoryFiltered
		return
	}

	// Build list of names for fuzzy search
	names := make([]string, len(categoryFiltered))
	for i, idx := range categoryFiltered {
		names[i] = m.items[idx].Package.Name
	}

	// Fuzzy search
	matches := fuzzy.Find(searchTerm, names)
	for _, match := range matches {
		m.filtered = append(m.filtered, categoryFiltered[match.Index])
	}
}

// moveCursor moves the cursor by delta positions
func (m *Model) moveCursor(delta int) {
	if len(m.filtered) == 0 {
		return
	}

	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
}

// toggleCurrent toggles the current item's selection
func (m *Model) toggleCurrent() {
	if len(m.filtered) == 0 {
		return
	}

	idx := m.filtered[m.cursor]
	if !m.items[idx].Ignored {
		m.items[idx].Selected = !m.items[idx].Selected
	}
}

// toggleIgnoreCurrent toggles the current item's ignored state
func (m *Model) toggleIgnoreCurrent() {
	if len(m.filtered) == 0 {
		return
	}

	idx := m.filtered[m.cursor]
	m.items[idx].Ignored = !m.items[idx].Ignored
	if m.items[idx].Ignored {
		m.items[idx].Selected = false
	}
}

// toggleIgnoreCurrentCategory toggles ignoring the entire current category
func (m *Model) toggleIgnoreCurrentCategory() {
	// Don't allow ignoring "all" category
	if m.category == CategoryAll {
		return
	}

	categoryType := string(m.category)

	// Toggle the category in ignored map
	if m.ignoredCategories[categoryType] {
		delete(m.ignoredCategories, categoryType)
	} else {
		m.ignoredCategories[categoryType] = true
	}

	// Update all packages of this category
	for i := range m.items {
		if string(m.items[i].Package.Type) == categoryType {
			m.items[i].Ignored = m.ignoredCategories[categoryType]
			if m.items[i].Ignored {
				m.items[i].Selected = false
			}
		}
	}
}

// selectAllVisible selects or deselects all visible items
func (m *Model) selectAllVisible(selected bool) {
	for _, idx := range m.filtered {
		if !m.items[idx].Ignored {
			m.items[idx].Selected = selected
		}
	}
}

// setCategory changes the current category
func (m *Model) setCategory(cat Category) {
	if m.category != cat {
		m.category = cat
		m.cursor = 0
		m.updateFiltered()
	}
}

// prevCategory moves to the previous category
func (m *Model) prevCategory() {
	categories := AllCategories()
	for i, cat := range categories {
		if cat == m.category {
			if i > 0 {
				m.setCategory(categories[i-1])
			}
			return
		}
	}
}

// nextCategory moves to the next category
func (m *Model) nextCategory() {
	categories := AllCategories()
	for i, cat := range categories {
		if cat == m.category {
			if i < len(categories)-1 {
				m.setCategory(categories[i+1])
			}
			return
		}
	}
}

// countByCategory returns counts of items by category
func (m *Model) countByCategory() map[Category]struct{ total, selected int } {
	counts := make(map[Category]struct{ total, selected int })

	for _, item := range m.items {
		cat := Category(item.Package.Type)
		c := counts[cat]
		c.total++
		if item.Selected && !item.Ignored {
			c.selected++
		}
		counts[cat] = c

		// Also count in "all"
		c = counts[CategoryAll]
		c.total++
		if item.Selected && !item.Ignored {
			c.selected++
		}
		counts[CategoryAll] = c
	}

	return counts
}
