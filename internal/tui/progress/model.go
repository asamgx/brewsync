package progress

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/tui/styles"
)

// InstallResult represents the result of installing a package
type InstallResult struct {
	Package brewfile.Package
	Error   error
}

// InstallFunc is the function that installs a package
type InstallFunc func(pkg brewfile.Package) error

// InstallWithOutputFunc is the function that installs a package with output streaming
type InstallWithOutputFunc func(pkg brewfile.Package, onOutput func(line string)) error

// InstallMsg is sent when a package installation completes
type InstallMsg struct {
	Package brewfile.Package
	Index   int
	Total   int
	Error   error
}

// OutputLineMsg is sent when a line of output is received from the installer
type OutputLineMsg struct {
	Package brewfile.Package
	Line    string
}

// DoneMsg is sent when all installations are complete
type DoneMsg struct {
	Installed int
	Failed    int
	Results   []InstallResult
}

// Model is the progress UI model
type Model struct {
	title           string
	packages        brewfile.Packages
	current         int
	spinner         spinner.Model
	progress        progress.Model
	results         []InstallResult
	installed       int
	failed          int
	done            bool
	width           int
	height          int
	installFn       InstallFunc
	installOutputFn InstallWithOutputFunc
	outputLines     []string          // Recent output lines
	maxOutputLines  int               // Max lines to keep
	currentPkg      *brewfile.Package // Current package being installed
}

// New creates a new progress model
func New(title string, packages brewfile.Packages, installFn InstallFunc) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	p := progress.New(progress.WithDefaultGradient())

	return Model{
		title:          title,
		packages:       packages,
		current:        0,
		spinner:        s,
		progress:       p,
		results:        make([]InstallResult, 0, len(packages)),
		installFn:      installFn,
		outputLines:    make([]string, 0),
		maxOutputLines: 10,
		width:          80,
		height:         24,
	}
}

// NewWithOutput creates a new progress model with output streaming support
func NewWithOutput(title string, packages brewfile.Packages, installOutputFn InstallWithOutputFunc) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	p := progress.New(progress.WithDefaultGradient())

	return Model{
		title:           title,
		packages:        packages,
		current:         0,
		spinner:         s,
		progress:        p,
		results:         make([]InstallResult, 0, len(packages)),
		installOutputFn: installOutputFn,
		outputLines:     make([]string, 0),
		maxOutputLines:  10,
		width:           80,
		height:          24,
	}
}

// Init starts the installation process
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.installNext(),
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 10
		if m.progress.Width > 60 {
			m.progress.Width = 60
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case OutputLineMsg:
		// Add line to output buffer
		m.outputLines = append(m.outputLines, msg.Line)

		// Keep only the last N lines
		if len(m.outputLines) > m.maxOutputLines {
			m.outputLines = m.outputLines[len(m.outputLines)-m.maxOutputLines:]
		}
		return m, nil

	case InstallMsg:
		result := InstallResult{
			Package: msg.Package,
			Error:   msg.Error,
		}
		m.results = append(m.results, result)

		if msg.Error != nil {
			m.failed++
		} else {
			m.installed++
		}

		m.current = msg.Index + 1
		m.currentPkg = nil
		m.outputLines = []string{} // Clear output for next package

		if m.current >= len(m.packages) {
			m.done = true
			return m, tea.Quit
		}

		return m, m.installNext()

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	return m, nil
}

// installNext returns a command to install the next package
func (m *Model) installNext() tea.Cmd {
	if m.current >= len(m.packages) {
		return nil
	}

	pkg := m.packages[m.current]
	m.currentPkg = &pkg
	idx := m.current
	total := len(m.packages)

	// Use streaming install if available
	if m.installOutputFn != nil {
		return m.streamingInstall(pkg, idx, total)
	}

	// Fallback to regular install
	return func() tea.Msg {
		err := m.installFn(pkg)
		return InstallMsg{
			Package: pkg,
			Index:   idx,
			Total:   total,
			Error:   err,
		}
	}
}

// streamingInstall performs installation with output capture (for future use)
func (m *Model) streamingInstall(pkg brewfile.Package, idx, total int) tea.Cmd {
	return func() tea.Msg {
		// Call the install function with output streaming
		// Output lines are captured but not shown in real-time yet
		// This keeps the door open for future enhancement
		err := m.installOutputFn(pkg, func(line string) {
			// Output callback - could be used for logging or future real-time display
			// For now, we just let it flow through
		})

		return InstallMsg{
			Package: pkg,
			Index:   idx,
			Total:   total,
			Error:   err,
		}
	}
}

// View renders the progress UI
func (m Model) View() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.TitleStyle.Render(m.title))
	b.WriteString("\n\n")

	// Progress bar
	percent := float64(m.current) / float64(len(m.packages))
	b.WriteString(m.progress.ViewAs(percent))
	b.WriteString("\n\n")

	// Current status
	if !m.done && m.current < len(m.packages) {
		pkg := m.packages[m.current]
		b.WriteString(m.spinner.View())
		b.WriteString(" Installing ")
		b.WriteString(styles.GetCategoryStyle(string(pkg.Type)).Render(string(pkg.Type)))
		b.WriteString(": ")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render(pkg.Name))
		b.WriteString(fmt.Sprintf(" (%d/%d)", m.current+1, len(m.packages)))
	}
	b.WriteString("\n\n")

	// Recent results (last 5)
	b.WriteString("Recent:\n")
	start := len(m.results) - 5
	if start < 0 {
		start = 0
	}
	for i := start; i < len(m.results); i++ {
		result := m.results[i]
		if result.Error != nil {
			b.WriteString(styles.CrossStyle.String())
			b.WriteString(" ")
			b.WriteString(styles.ErrorStyle.Render(
				fmt.Sprintf("%s:%s - %v", result.Package.Type, result.Package.Name, result.Error)))
		} else {
			b.WriteString(styles.CheckmarkStyle.String())
			b.WriteString(" ")
			b.WriteString(styles.AddedStyle.Render(
				fmt.Sprintf("%s:%s", result.Package.Type, result.Package.Name)))
		}
		b.WriteString("\n")
	}

	// Summary
	b.WriteString("\n")
	summary := fmt.Sprintf("Installed: %s | Failed: %s",
		styles.AddedStyle.Render(fmt.Sprintf("%d", m.installed)),
		styles.ErrorStyle.Render(fmt.Sprintf("%d", m.failed)))
	b.WriteString(summary)

	if m.done {
		b.WriteString("\n\n")
		b.WriteString(styles.DimmedStyle.Render("Press q to exit"))
	}

	return b.String()
}

// Results returns the installation results
func (m Model) Results() []InstallResult {
	return m.results
}

// Installed returns the number of successfully installed packages
func (m Model) Installed() int {
	return m.installed
}

// Failed returns the number of failed installations
func (m Model) Failed() int {
	return m.failed
}

// Done returns true if all installations are complete
func (m Model) Done() bool {
	return m.done
}
