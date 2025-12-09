package screens

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/andrew-sameh/brewsync/internal/config"
	"github.com/andrew-sameh/brewsync/internal/tui/styles"
)

// Check represents a single health check
type Check struct {
	Name     string
	Status   string // pass, fail, warn
	Message  string
	Optional bool
}

// DoctorModel is the model for the doctor screen
type DoctorModel struct {
	config  *config.Config
	width   int
	height  int
	checks  []Check
	loading bool
}

// NewDoctorModel creates a new doctor model
func NewDoctorModel(cfg *config.Config) *DoctorModel {
	return &DoctorModel{
		config:  cfg,
		width:   80,
		height:  24,
		loading: true,
	}
}

type doctorDoneMsg struct {
	checks []Check
}

// Init initializes the doctor model and runs checks
func (m *DoctorModel) Init() tea.Cmd {
	return func() tea.Msg {
		var checks []Check

		// Config checks
		checks = append(checks, Check{
			Name:   "Config file exists",
			Status: boolToStatus(config.Exists()),
		})

		if m.config != nil {
			checks = append(checks, Check{
				Name:   "Config loads successfully",
				Status: "pass",
			})

			// Machine checks
			if m.config.CurrentMachine != "" {
				checks = append(checks, Check{
					Name:    "Current machine detected",
					Status:  "pass",
					Message: m.config.CurrentMachine,
				})
			} else {
				checks = append(checks, Check{
					Name:   "Current machine detected",
					Status: "fail",
				})
			}
		}

		// Tool checks
		tools := []struct {
			name     string
			cmd      string
			optional bool
		}{
			{"Homebrew", "brew", false},
			{"brew bundle", "brew", false},
			{"VSCode CLI", "code", true},
			{"Cursor CLI", "cursor", true},
			{"Mac App Store CLI", "mas", true},
			{"Go", "go", true},
		}

		for _, tool := range tools {
			_, err := exec.LookPath(tool.cmd)
			status := "pass"
			if err != nil {
				if tool.optional {
					status = "warn"
				} else {
					status = "fail"
				}
			}
			checks = append(checks, Check{
				Name:     tool.name,
				Status:   status,
				Optional: tool.optional,
			})
		}

		return doctorDoneMsg{checks: checks}
	}
}

func boolToStatus(b bool) string {
	if b {
		return "pass"
	}
	return "fail"
}

// Update handles messages
func (m *DoctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case doctorDoneMsg:
		m.loading = false
		m.checks = msg.checks
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "b"))):
			return m, func() tea.Msg { return Navigate("dashboard") }
		}
	}

	return m, nil
}

// SetSize updates the doctor dimensions
func (m *DoctorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the doctor screen (legacy)
func (m *DoctorModel) View() string {
	return m.ViewContent(m.width, m.height)
}

// ViewContent renders just the content area (for use in layout)
func (m *DoctorModel) ViewContent(width, height int) string {
	var b strings.Builder

	if m.loading {
		b.WriteString(styles.DimmedStyle.Render("Running checks..."))
		return b.String()
	}

	// Results
	var passCount, failCount, warnCount int
	for _, check := range m.checks {
		var icon string
		var lineStyle lipgloss.Style
		switch check.Status {
		case "pass":
			icon = "✓"
			lineStyle = lipgloss.NewStyle().Foreground(styles.CatGreen)
			passCount++
		case "fail":
			icon = "✗"
			lineStyle = lipgloss.NewStyle().Foreground(styles.CatRed)
			failCount++
		case "warn":
			icon = "⚠"
			lineStyle = lipgloss.NewStyle().Foreground(styles.CatPeach)
			warnCount++
		}

		line := icon + " " + check.Name
		if check.Message != "" {
			line += " (" + check.Message + ")"
		}
		if check.Optional {
			line += lipgloss.NewStyle().Foreground(styles.MutedColor).Render(" (optional)")
		}

		b.WriteString(lineStyle.Render(line))
		b.WriteString("\n")
	}

	// Summary
	b.WriteString("\n")
	summaryLine := lipgloss.NewStyle().Foreground(styles.CatGreen).Render("✓ "+itoa(passCount)+" passed") + "  " +
		lipgloss.NewStyle().Foreground(styles.CatPeach).Render("⚠ "+itoa(warnCount)+" warnings") + "  " +
		lipgloss.NewStyle().Foreground(styles.CatRed).Render("✗ "+itoa(failCount)+" failed")
	b.WriteString(summaryLine)

	return b.String()
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
