package app

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/andrew-sameh/brewsync/internal/config"
	"github.com/andrew-sameh/brewsync/internal/debug"
	"github.com/andrew-sameh/brewsync/internal/tui/app/components"
	"github.com/andrew-sameh/brewsync/internal/tui/app/screens"
)

// Screen represents which screen is currently active
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenImport
	ScreenSync
	ScreenDiff
	ScreenDump
	ScreenList
	ScreenIgnore
	ScreenConfig
	ScreenHistory
	ScreenProfile
	ScreenDoctor
	ScreenSetup
)

// Model is the main TUI model that manages all screens
type Model struct {
	screen     Screen
	prevScreen Screen
	config     *config.Config
	width      int
	height     int

	// Layout components
	sidebar components.SidebarModel
	header  components.HeaderModel
	footer  components.FooterModel
	layout  Layout

	// Screen models
	dashboard *screens.DashboardModel
	list      *screens.ListModel
	diff      *screens.DiffModel
	doctor    *screens.DoctorModel
	dump      *screens.DumpModel
	importM   *screens.ImportModel
	syncM     *screens.SyncModel
	ignore    *screens.IgnoreModel
	history   *screens.HistoryModel
	profile   *screens.ProfileModel
	configM   *screens.ConfigModel
	setup     *screens.SetupModel

	// State
	statusMessage string
	statusType    string // info, success, error, warning
	needsSetup    bool
	showIgnored   bool // Global toggle to show/hide ignored items (default: false)

	keys KeyMap
	help help.Model
}

// New creates a new main TUI model
func New(cfg *config.Config) Model {
	debug.Log("App.New: creating model, config=%v", cfg != nil)
	needsSetup := cfg == nil || len(cfg.Machines) == 0
	debug.Log("App.New: needsSetup=%v", needsSetup)

	// Default dimensions
	width := 80
	height := 24

	// Get machine name for header
	machineName := "unknown"
	if cfg != nil && cfg.CurrentMachine != "" {
		machineName = cfg.CurrentMachine
	}

	// Initialize layout components
	sidebar := components.NewSidebar(components.DefaultMenuItems(), SidebarWidth)
	header := components.NewHeader(machineName, width)
	footer := components.NewFooter(width)
	footer.SetKeybindings(components.DashboardKeybindings())
	layout := NewLayout(width, height)

	m := Model{
		screen:     ScreenDashboard,
		prevScreen: ScreenDashboard,
		config:     cfg,
		width:      width,
		height:     height,
		sidebar:    sidebar,
		header:     header,
		footer:     footer,
		layout:     layout,
		keys:       DefaultKeyMap(),
		help:       help.New(),
		needsSetup: needsSetup,
	}

	// Create initial screen models here (not in Init) because Init has a value receiver
	if needsSetup {
		m.screen = ScreenSetup
		m.setup = screens.NewSetupModel()
		debug.Log("App.New: created setup model")
	} else {
		m.dashboard = screens.NewDashboardModel(cfg)
		debug.Log("App.New: created dashboard model")
	}

	return m
}

// Init initializes the model and returns initial commands
func (m Model) Init() tea.Cmd {
	debug.Log("App.Init: initializing, needsSetup=%v", m.needsSetup)
	var cmds []tea.Cmd

	if m.needsSetup && m.setup != nil {
		debug.Log("App.Init: calling setup.Init()")
		cmds = append(cmds, m.setup.Init())
	} else if m.dashboard != nil {
		debug.Log("App.Init: calling dashboard.Init()")
		cmd := m.dashboard.Init()
		debug.Log("App.Init: dashboard.Init() returned cmd=%v", cmd != nil)
		cmds = append(cmds, cmd)
	}

	debug.Log("App.Init: returning %d commands", len(cmds))
	return tea.Batch(cmds...)
}

// Update handles messages and routes them to the active screen
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	debug.Log("App.Update: received msg type %T, screen=%d", msg, m.screen)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		debug.Log("App.Update: window resize %dx%d", msg.Width, msg.Height)
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width

		// Update layout dimensions
		m.layout.SetSize(msg.Width, msg.Height)
		m.header.SetWidth(msg.Width)
		m.footer.SetWidth(msg.Width)
		m.sidebar.SetSize(SidebarWidth, m.layout.SidebarHeight())

		// Propagate to active screen
		return m.propagateResize(msg)

	case tea.KeyMsg:
		// Setup screen handles its own input
		if m.screen == ScreenSetup {
			return m.routeToScreen(msg)
		}

		// ctrl+c always quits
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// q always quits
		if msg.String() == "q" {
			return m, tea.Quit
		}

		// Esc returns to dashboard (except when already on dashboard)
		if msg.String() == "esc" {
			if m.screen != ScreenDashboard {
				return m.navigateToScreen(ScreenDashboard)
			}
			return m, nil
		}

		// 'h' toggles showing ignored items
		if msg.String() == "h" {
			m.showIgnored = !m.showIgnored
			m.header.SetShowIgnored(m.showIgnored)
			// Broadcast to current screen
			return m.routeToScreen(screens.ShowIgnoredMsg{Show: m.showIgnored})
		}

		// Global screen hotkeys (1-9, 0, !)
		if screen := m.getScreenFromShortcut(msg.String()); screen >= 0 {
			return m.navigateToScreen(screen)
		}

		// All other keys go to active screen
		return m.routeToScreen(msg)

	case screens.NavigateMsg:
		return m.handleNavigation(msg)

	case screens.SetupCompleteMsg:
		// Setup completed, reload config and go to dashboard
		cfg, err := config.Load()
		if err == nil {
			m.config = cfg
			m.needsSetup = false
			m.screen = ScreenDashboard
			m.dashboard = screens.NewDashboardModel(m.config)
			// Update header with machine name
			if cfg.CurrentMachine != "" {
				m.header.SetMachine(cfg.CurrentMachine)
			}
			m.sidebar.SetActive(int(ScreenDashboard))
			return m, m.dashboard.Init()
		}

	case screens.StatusMsg:
		m.statusMessage = msg.Message
		m.statusType = msg.Type
	}

	// Route to active screen
	return m.routeToScreen(msg)
}

// getScreenFromShortcut returns the screen for a shortcut key, or -1 if not a shortcut
func (m Model) getScreenFromShortcut(key string) Screen {
	switch key {
	case "1":
		return ScreenDashboard
	case "2":
		return ScreenImport
	case "3":
		return ScreenSync
	case "4":
		return ScreenDiff
	case "5":
		return ScreenDump
	case "6":
		return ScreenList
	case "7":
		return ScreenIgnore
	case "8":
		return ScreenConfig
	case "9":
		return ScreenHistory
	case "0":
		return ScreenProfile
	case "!":
		return ScreenDoctor
	}
	return -1
}


// navigateToScreen navigates to the specified screen
func (m Model) navigateToScreen(screen Screen) (tea.Model, tea.Cmd) {
	m.prevScreen = m.screen
	m.screen = screen
	m.sidebar.SetActive(int(screen))

	// Update footer keybindings based on screen
	m.updateFooterKeybindings()

	switch screen {
	case ScreenDashboard:
		if m.dashboard == nil {
			m.dashboard = screens.NewDashboardModel(m.config)
		}
		return m, m.dashboard.Init()

	case ScreenImport:
		m.importM = screens.NewImportModel(m.config)
		return m, m.importM.Init()

	case ScreenSync:
		m.syncM = screens.NewSyncModel(m.config)
		return m, m.syncM.Init()

	case ScreenDiff:
		m.diff = screens.NewDiffModel(m.config)
		return m, m.diff.Init()

	case ScreenDump:
		m.dump = screens.NewDumpModel(m.config)
		return m, m.dump.Init()

	case ScreenList:
		m.list = screens.NewListModel(m.config)
		return m, m.list.Init()

	case ScreenIgnore:
		m.ignore = screens.NewIgnoreModel(m.config)
		return m, m.ignore.Init()

	case ScreenConfig:
		m.configM = screens.NewConfigModel(m.config)
		return m, m.configM.Init()

	case ScreenHistory:
		m.history = screens.NewHistoryModel(m.config)
		return m, m.history.Init()

	case ScreenProfile:
		m.profile = screens.NewProfileModel(m.config)
		return m, m.profile.Init()

	case ScreenDoctor:
		m.doctor = screens.NewDoctorModel(m.config)
		return m, m.doctor.Init()
	}

	return m, nil
}

// updateFooterKeybindings updates footer based on current screen
func (m *Model) updateFooterKeybindings() {
	switch m.screen {
	case ScreenDashboard:
		m.footer.SetKeybindings(components.DashboardKeybindings())
	case ScreenList:
		m.footer.SetKeybindings(components.ListKeybindings())
	case ScreenImport:
		m.footer.SetKeybindings(components.ImportKeybindings())
	case ScreenSync:
		m.footer.SetKeybindings(components.SyncKeybindings())
	case ScreenDiff:
		m.footer.SetKeybindings(components.DiffKeybindings())
	case ScreenDump:
		m.footer.SetKeybindings(components.DumpKeybindings())
	case ScreenIgnore:
		m.footer.SetKeybindings(components.IgnoreKeybindings())
	default:
		m.footer.SetKeybindings(components.ContentKeybindings())
	}
}

// View renders the active screen
func (m Model) View() string {
	// Setup screen uses full screen without sidebar
	if m.screen == ScreenSetup {
		if m.setup != nil {
			return m.layout.RenderSimple(m.setup.View())
		}
		return "Loading setup..."
	}

	// Build header content
	headerContent := "  " + m.header.SimpleHeader()

	// Build sidebar content
	sidebarContent := m.sidebar.View()

	// Build main content
	contentStr := m.renderContent()

	// Build footer content
	footerContent := "  " + m.footer.View()

	// Render full layout
	return m.layout.Render(headerContent, sidebarContent, contentStr, footerContent)
}

// renderContent renders the current screen's content
func (m Model) renderContent() string {
	width := m.layout.ContentWidth()
	height := m.layout.ContentHeight()

	switch m.screen {
	case ScreenDashboard:
		if m.dashboard != nil {
			return m.dashboard.ViewContent(width, height)
		}
	case ScreenList:
		if m.list != nil {
			return m.list.ViewContent(width, height)
		}
	case ScreenDiff:
		if m.diff != nil {
			return m.diff.ViewContent(width, height)
		}
	case ScreenDoctor:
		if m.doctor != nil {
			return m.doctor.ViewContent(width, height)
		}
	case ScreenDump:
		if m.dump != nil {
			return m.dump.ViewContent(width, height)
		}
	case ScreenImport:
		if m.importM != nil {
			return m.importM.ViewContent(width, height)
		}
	case ScreenSync:
		if m.syncM != nil {
			return m.syncM.ViewContent(width, height)
		}
	case ScreenIgnore:
		if m.ignore != nil {
			return m.ignore.ViewContent(width, height)
		}
	case ScreenHistory:
		if m.history != nil {
			return m.history.ViewContent(width, height)
		}
	case ScreenProfile:
		if m.profile != nil {
			return m.profile.ViewContent(width, height)
		}
	case ScreenConfig:
		if m.configM != nil {
			return m.configM.ViewContent(width, height)
		}
	}

	return "Loading..."
}

// routeToScreen routes messages to the active screen
func (m Model) routeToScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.screen {
	case ScreenSetup:
		if m.setup != nil {
			newSetup, cmd := m.setup.Update(msg)
			m.setup = newSetup.(*screens.SetupModel)
			return m, cmd
		}

	case ScreenDashboard:
		if m.dashboard != nil {
			newDash, cmd := m.dashboard.Update(msg)
			m.dashboard = newDash.(*screens.DashboardModel)
			return m, cmd
		}

	case ScreenList:
		if m.list != nil {
			newList, cmd := m.list.Update(msg)
			m.list = newList.(*screens.ListModel)
			return m, cmd
		}

	case ScreenDiff:
		if m.diff != nil {
			newDiff, cmd := m.diff.Update(msg)
			m.diff = newDiff.(*screens.DiffModel)
			return m, cmd
		}

	case ScreenDoctor:
		if m.doctor != nil {
			newDoctor, cmd := m.doctor.Update(msg)
			m.doctor = newDoctor.(*screens.DoctorModel)
			return m, cmd
		}

	case ScreenDump:
		if m.dump != nil {
			newDump, cmd := m.dump.Update(msg)
			m.dump = newDump.(*screens.DumpModel)
			return m, cmd
		}

	case ScreenImport:
		if m.importM != nil {
			newImport, cmd := m.importM.Update(msg)
			m.importM = newImport.(*screens.ImportModel)
			return m, cmd
		}

	case ScreenSync:
		if m.syncM != nil {
			newSync, cmd := m.syncM.Update(msg)
			m.syncM = newSync.(*screens.SyncModel)
			return m, cmd
		}

	case ScreenIgnore:
		if m.ignore != nil {
			newIgnore, cmd := m.ignore.Update(msg)
			m.ignore = newIgnore.(*screens.IgnoreModel)
			return m, cmd
		}

	case ScreenHistory:
		if m.history != nil {
			newHistory, cmd := m.history.Update(msg)
			m.history = newHistory.(*screens.HistoryModel)
			return m, cmd
		}

	case ScreenProfile:
		if m.profile != nil {
			newProfile, cmd := m.profile.Update(msg)
			m.profile = newProfile.(*screens.ProfileModel)
			return m, cmd
		}

	case ScreenConfig:
		if m.configM != nil {
			newConfig, cmd := m.configM.Update(msg)
			m.configM = newConfig.(*screens.ConfigModel)
			return m, cmd
		}
	}

	return m, cmd
}

// handleNavigation handles screen navigation messages
func (m Model) handleNavigation(msg screens.NavigateMsg) (tea.Model, tea.Cmd) {
	m.prevScreen = m.screen

	switch msg.Target {
	case "dashboard":
		return m.navigateToScreen(ScreenDashboard)
	case "import":
		return m.navigateToScreen(ScreenImport)
	case "sync":
		return m.navigateToScreen(ScreenSync)
	case "diff":
		return m.navigateToScreen(ScreenDiff)
	case "dump":
		return m.navigateToScreen(ScreenDump)
	case "list":
		return m.navigateToScreen(ScreenList)
	case "ignore":
		return m.navigateToScreen(ScreenIgnore)
	case "config":
		return m.navigateToScreen(ScreenConfig)
	case "history":
		return m.navigateToScreen(ScreenHistory)
	case "profile":
		return m.navigateToScreen(ScreenProfile)
	case "doctor":
		return m.navigateToScreen(ScreenDoctor)
	}

	return m, nil
}

// goBack returns to the previous screen (or dashboard)
func (m Model) goBack() (tea.Model, tea.Cmd) {
	return m.navigateToScreen(ScreenDashboard)
}

// propagateResize sends resize message to active screen
func (m Model) propagateResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	// Create content-sized message for screens
	contentMsg := tea.WindowSizeMsg{
		Width:  m.layout.ContentWidth(),
		Height: m.layout.ContentHeight(),
	}

	switch m.screen {
	case ScreenSetup:
		if m.setup != nil {
			newSetup, _ := m.setup.Update(msg)
			m.setup = newSetup.(*screens.SetupModel)
		}
	case ScreenDashboard:
		if m.dashboard != nil {
			newDash, _ := m.dashboard.Update(contentMsg)
			m.dashboard = newDash.(*screens.DashboardModel)
		}
	case ScreenList:
		if m.list != nil {
			newList, _ := m.list.Update(contentMsg)
			m.list = newList.(*screens.ListModel)
		}
	case ScreenDiff:
		if m.diff != nil {
			newDiff, _ := m.diff.Update(contentMsg)
			m.diff = newDiff.(*screens.DiffModel)
		}
	case ScreenDoctor:
		if m.doctor != nil {
			newDoctor, _ := m.doctor.Update(contentMsg)
			m.doctor = newDoctor.(*screens.DoctorModel)
		}
	case ScreenDump:
		if m.dump != nil {
			newDump, _ := m.dump.Update(contentMsg)
			m.dump = newDump.(*screens.DumpModel)
		}
	case ScreenImport:
		if m.importM != nil {
			newImport, _ := m.importM.Update(contentMsg)
			m.importM = newImport.(*screens.ImportModel)
		}
	case ScreenSync:
		if m.syncM != nil {
			newSync, _ := m.syncM.Update(contentMsg)
			m.syncM = newSync.(*screens.SyncModel)
		}
	case ScreenIgnore:
		if m.ignore != nil {
			newIgnore, _ := m.ignore.Update(contentMsg)
			m.ignore = newIgnore.(*screens.IgnoreModel)
		}
	case ScreenHistory:
		if m.history != nil {
			newHistory, _ := m.history.Update(contentMsg)
			m.history = newHistory.(*screens.HistoryModel)
		}
	case ScreenProfile:
		if m.profile != nil {
			newProfile, _ := m.profile.Update(contentMsg)
			m.profile = newProfile.(*screens.ProfileModel)
		}
	case ScreenConfig:
		if m.configM != nil {
			newConfig, _ := m.configM.Update(contentMsg)
			m.configM = newConfig.(*screens.ConfigModel)
		}
	}

	return m, nil
}
