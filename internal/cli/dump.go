package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/config"
	"github.com/asamgx/brewsync/internal/exec"
	"github.com/asamgx/brewsync/internal/installer"
)

var (
	dumpCommit  bool
	dumpPush    bool
	dumpMessage string
)

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Update current machine's Brewfile from installed packages",
	Long: `Dump captures the current state of installed packages and writes them
to the machine's Brewfile. This includes:
- Homebrew taps, formulae, and casks
- VSCode extensions
- Cursor extensions
- Antigravity extensions
- Go tools
- Mac App Store apps

The Brewfile location is determined from the config for the current machine.`,
	RunE: runDump,
}

func init() {
	dumpCmd.Flags().BoolVar(&dumpCommit, "commit", false, "commit changes after dump")
	dumpCmd.Flags().BoolVar(&dumpPush, "push", false, "commit and push changes")
	dumpCmd.Flags().StringVarP(&dumpMessage, "message", "m", "", "custom commit message")
}

// dumpModel is the Bubble Tea model for the dump progress UI
type dumpModel struct {
	spinner        spinner.Model
	step           string
	completed      []string
	done           bool
	err            error
	packages       brewfile.Packages
	packagesByType map[brewfile.PackageType]int
}

type dumpStepMsg struct {
	step      string
	packages  brewfile.Packages
	countInfo string
}

type dumpCompleteMsg struct {
	packages brewfile.Packages
}

type dumpErrorMsg struct {
	err error
}

func newDumpModel() dumpModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(catMauve)

	return dumpModel{
		spinner:        s,
		completed:      []string{},
		packagesByType: make(map[brewfile.PackageType]int),
	}
}

func (m dumpModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m dumpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case dumpStepMsg:
		m.step = msg.step
		if msg.packages != nil {
			m.packages = append(m.packages, msg.packages...)
		}
		if msg.countInfo != "" {
			m.completed = append(m.completed, msg.countInfo)
		}
		return m, nil

	case dumpCompleteMsg:
		m.done = true
		m.packages = msg.packages
		return m, tea.Quit

	case dumpErrorMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m dumpModel) View() string {
	if m.err != nil {
		return styleError.Render(fmt.Sprintf("✗ Error: %v", m.err))
	}

	if m.done {
		return "" // Summary will be printed separately
	}

	var s strings.Builder

	// Show current step with spinner
	if m.step != "" {
		s.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), m.step))
	}

	// Show completed steps
	for _, info := range m.completed {
		s.WriteString(styleSuccess.Render("✓ ") + info + "\n")
	}

	return s.String()
}

func runDump(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current machine
	machine, ok := cfg.GetCurrentMachine()
	if !ok {
		return fmt.Errorf("current machine not configured (detected: %s)", cfg.CurrentMachine)
	}

	brewfilePath := machine.Brewfile
	if brewfilePath == "" {
		return fmt.Errorf("no Brewfile path configured for machine %s", cfg.CurrentMachine)
	}

	// Ensure directory exists
	dir := filepath.Dir(brewfilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// If quiet mode, run without animation
	if quiet {
		return runDumpQuiet(cfg, machine, brewfilePath)
	}

	// Run with animation
	return runDumpAnimated(cfg, machine, brewfilePath)
}

func runDumpQuiet(cfg *config.Config, machine config.Machine, brewfilePath string) error {
	allPackages, err := collectAllPackages(cfg, brewfilePath)
	if err != nil {
		return err
	}

	// Dry run
	if dryRun {
		printInfo("Dry run - would write %d packages to %s", len(allPackages), brewfilePath)
		return nil
	}

	// Write Brewfile
	writer := brewfile.NewWriter(allPackages)
	if err := writer.Write(brewfilePath); err != nil {
		return fmt.Errorf("failed to write Brewfile: %w", err)
	}

	printInfo("Wrote %d packages to %s", len(allPackages), brewfilePath)
	return nil
}

func runDumpAnimated(cfg *config.Config, machine config.Machine, brewfilePath string) error {
	// Create Bubble Tea program
	p := tea.NewProgram(newDumpModel())

	// Run collection in background
	go func() {
		allPackages, err := collectAllPackagesAnimated(cfg, brewfilePath, p)
		if err != nil {
			p.Send(dumpErrorMsg{err: err})
			return
		}
		p.Send(dumpCompleteMsg{packages: allPackages})
	}()

	// Run UI
	m, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run UI: %w", err)
	}

	// Check for errors
	model := m.(dumpModel)
	if model.err != nil {
		return model.err
	}

	allPackages := model.packages

	// Dry run
	if dryRun {
		printDumpSummary(cfg.CurrentMachine, brewfilePath, allPackages, true)
		return nil
	}

	// Write Brewfile
	writer := brewfile.NewWriter(allPackages)
	if err := writer.Write(brewfilePath); err != nil {
		return fmt.Errorf("failed to write Brewfile: %w", err)
	}

	// Print pretty summary
	printDumpSummary(cfg.CurrentMachine, brewfilePath, allPackages, false)

	// Handle commit and push
	if dumpCommit || dumpPush {
		if err := handleGitCommitAndPush(cfg, brewfilePath); err != nil {
			printWarning("Git commit/push failed: %v", err)
			return err
		}
	}

	return nil
}

func collectAllPackages(cfg *config.Config, brewfilePath string) (brewfile.Packages, error) {
	var allPackages brewfile.Packages
	brewInst := installer.NewBrewInstaller()

	// Use brew bundle dump if configured (default), otherwise collect manually
	if cfg.Dump.UseBrewBundle && brewInst.IsAvailable() {
		// Create temp file for brew bundle dump
		tmpFile := brewfilePath + ".brewbundle.tmp"
		if err := brewInst.DumpToFile(tmpFile); err == nil {
			// Parse the brew bundle output (includes taps, formulae, casks with descriptions)
			if brewPkgs, err := brewfile.Parse(tmpFile); err == nil {
				allPackages = append(allPackages, brewPkgs...)
			}
			os.Remove(tmpFile)
		}
	} else if brewInst.IsAvailable() {
		// Manual collection
		if taps, err := brewInst.ListTaps(); err == nil {
			allPackages = append(allPackages, taps...)
		}
		if formulae, err := brewInst.ListFormulae(); err == nil {
			allPackages = append(allPackages, formulae...)
		}
		if casks, err := brewInst.ListCasks(); err == nil {
			allPackages = append(allPackages, casks...)
		}
	}

	// Collect extensions
	if vscodeInst := installer.NewVSCodeInstaller(); vscodeInst.IsAvailable() {
		if extensions, err := vscodeInst.List(); err == nil {
			allPackages = allPackages.AddUnique(extensions...)
		}
	}

	if cursorInst := installer.NewCursorInstaller(); cursorInst.IsAvailable() {
		if extensions, err := cursorInst.List(); err == nil {
			allPackages = allPackages.AddUnique(extensions...)
		}
	}

	if antigravityInst := installer.NewAntigravityInstaller(); antigravityInst.IsAvailable() {
		if extensions, err := antigravityInst.List(); err == nil {
			allPackages = allPackages.AddUnique(extensions...)
		}
	}

	if goInst := installer.NewGoToolsInstaller(); goInst.IsAvailable() {
		if tools, err := goInst.List(); err == nil {
			allPackages = allPackages.AddUnique(tools...)
		}
	}

	if masInst := installer.NewMasInstaller(); masInst.IsAvailable() {
		if apps, err := masInst.List(); err == nil {
			allPackages = allPackages.AddUnique(apps...)
		}
	}

	return allPackages, nil
}

func collectAllPackagesAnimated(cfg *config.Config, brewfilePath string, p *tea.Program) (brewfile.Packages, error) {
	var allPackages brewfile.Packages
	brewInst := installer.NewBrewInstaller()

	// Homebrew packages
	p.Send(dumpStepMsg{step: "Collecting Homebrew packages..."})
	time.Sleep(100 * time.Millisecond) // Brief pause for UI update

	if cfg.Dump.UseBrewBundle && brewInst.IsAvailable() {
		tmpFile := brewfilePath + ".brewbundle.tmp"
		if err := brewInst.DumpToFile(tmpFile); err == nil {
			if brewPkgs, err := brewfile.Parse(tmpFile); err == nil {
				allPackages = append(allPackages, brewPkgs...)
				byType := brewPkgs.ByType()
				taps := len(byType[brewfile.TypeTap])
				formulae := len(byType[brewfile.TypeBrew])
				casks := len(byType[brewfile.TypeCask])
				info := fmt.Sprintf("Homebrew: %d packages (taps: %d, formulae: %d, casks: %d)",
					len(brewPkgs), taps, formulae, casks)
				p.Send(dumpStepMsg{countInfo: info})
			}
			os.Remove(tmpFile)
		}
	} else if brewInst.IsAvailable() {
		var brewCount int
		if taps, err := brewInst.ListTaps(); err == nil {
			allPackages = append(allPackages, taps...)
			brewCount += len(taps)
		}
		if formulae, err := brewInst.ListFormulae(); err == nil {
			allPackages = append(allPackages, formulae...)
			brewCount += len(formulae)
		}
		if casks, err := brewInst.ListCasks(); err == nil {
			allPackages = append(allPackages, casks...)
			brewCount += len(casks)
		}
		if brewCount > 0 {
			p.Send(dumpStepMsg{countInfo: fmt.Sprintf("Homebrew: %d packages", brewCount)})
		}
	}

	// VSCode extensions
	if vscodeInst := installer.NewVSCodeInstaller(); vscodeInst.IsAvailable() {
		p.Send(dumpStepMsg{step: "Collecting VSCode extensions..."})
		time.Sleep(100 * time.Millisecond)
		if extensions, err := vscodeInst.List(); err == nil {
			beforeCount := len(allPackages)
			allPackages = allPackages.AddUnique(extensions...)
			addedCount := len(allPackages) - beforeCount
			p.Send(dumpStepMsg{countInfo: fmt.Sprintf("VSCode: %d extensions (%d new)", len(extensions), addedCount)})
		}
	}

	// Cursor extensions
	if cursorInst := installer.NewCursorInstaller(); cursorInst.IsAvailable() {
		p.Send(dumpStepMsg{step: "Collecting Cursor extensions..."})
		time.Sleep(100 * time.Millisecond)
		if extensions, err := cursorInst.List(); err == nil {
			beforeCount := len(allPackages)
			allPackages = allPackages.AddUnique(extensions...)
			addedCount := len(allPackages) - beforeCount
			p.Send(dumpStepMsg{countInfo: fmt.Sprintf("Cursor: %d extensions (%d new)", len(extensions), addedCount)})
		}
	}

	// Antigravity extensions
	if antigravityInst := installer.NewAntigravityInstaller(); antigravityInst.IsAvailable() {
		p.Send(dumpStepMsg{step: "Collecting Antigravity extensions..."})
		time.Sleep(100 * time.Millisecond)
		if extensions, err := antigravityInst.List(); err == nil {
			beforeCount := len(allPackages)
			allPackages = allPackages.AddUnique(extensions...)
			addedCount := len(allPackages) - beforeCount
			p.Send(dumpStepMsg{countInfo: fmt.Sprintf("Antigravity: %d extensions (%d new)", len(extensions), addedCount)})
		}
	}

	// Go tools
	if goInst := installer.NewGoToolsInstaller(); goInst.IsAvailable() {
		p.Send(dumpStepMsg{step: "Collecting Go tools..."})
		time.Sleep(100 * time.Millisecond)
		if tools, err := goInst.List(); err == nil {
			beforeCount := len(allPackages)
			allPackages = allPackages.AddUnique(tools...)
			addedCount := len(allPackages) - beforeCount
			p.Send(dumpStepMsg{countInfo: fmt.Sprintf("Go: %d tools (%d new)", len(tools), addedCount)})
		}
	}

	// Mac App Store apps
	if masInst := installer.NewMasInstaller(); masInst.IsAvailable() {
		p.Send(dumpStepMsg{step: "Collecting Mac App Store apps..."})
		time.Sleep(100 * time.Millisecond)
		if apps, err := masInst.List(); err == nil {
			beforeCount := len(allPackages)
			allPackages = allPackages.AddUnique(apps...)
			addedCount := len(allPackages) - beforeCount
			p.Send(dumpStepMsg{countInfo: fmt.Sprintf("Mac App Store: %d apps (%d new)", len(apps), addedCount)})
		}
	}

	return allPackages, nil
}

func printDumpSummary(machineName, brewfilePath string, packages brewfile.Packages, isDryRun bool) {
	const tableWidth = 80

	var allLines []string

	// Header
	headerIcon := "📊"
	if isDryRun {
		headerIcon = "🔍"
	}
	headerText := fmt.Sprintf("%s Dump %s - %s", headerIcon,
		map[bool]string{true: "Preview", false: "Complete"}[isDryRun], machineName)
	header := lipgloss.NewStyle().
		Foreground(catLavender).
		Bold(true).
		Render(headerText)
	allLines = append(allLines, header)

	// Separator
	separator := lipgloss.NewStyle().
		Foreground(catOverlay0).
		Render(strings.Repeat("─", tableWidth-4))
	allLines = append(allLines, separator, "")

	// Package counts by type
	byType := packages.ByType()
	typeOrder := []brewfile.PackageType{
		brewfile.TypeTap,
		brewfile.TypeBrew,
		brewfile.TypeCask,
		brewfile.TypeVSCode,
		brewfile.TypeCursor,
		brewfile.TypeAntigravity,
		brewfile.TypeGo,
		brewfile.TypeMas,
	}

	typeInfo := map[brewfile.PackageType]struct {
		icon  string
		color lipgloss.Color
	}{
		brewfile.TypeTap:         {"🚰", catTeal},
		brewfile.TypeBrew:        {"🍺", catYellow},
		brewfile.TypeCask:        {"📦", catPeach},
		brewfile.TypeVSCode:      {"💻", catBlue},
		brewfile.TypeCursor:      {"✏️ ", catMauve},
		brewfile.TypeAntigravity: {"🚀", catPink},
		brewfile.TypeGo:          {"🔷", catSapphire},
		brewfile.TypeMas:         {"🍎", catRed},
	}

	for _, t := range typeOrder {
		if pkgs, ok := byType[t]; ok && len(pkgs) > 0 {
			info := typeInfo[t]
			icon := lipgloss.NewStyle().Foreground(info.color).Render(info.icon)
			label := lipgloss.NewStyle().Foreground(catText).Bold(true).Render(string(t))
			count := lipgloss.NewStyle().Foreground(catGreen).Render(fmt.Sprintf("%d", len(pkgs)))
			allLines = append(allLines, fmt.Sprintf("%s %s: %s", icon, label, count))
		}
	}

	allLines = append(allLines, "")

	// File path
	fileIcon := lipgloss.NewStyle().Foreground(catMauve).Render("📄")
	fileLabel := lipgloss.NewStyle().Foreground(catSubtext0).Render("Brewfile:")
	filePath := lipgloss.NewStyle().Foreground(catText).Render(brewfilePath)
	allLines = append(allLines, fmt.Sprintf("%s %s", fileIcon, fileLabel))
	allLines = append(allLines, fmt.Sprintf("   %s", filePath))

	allLines = append(allLines, "")

	// Total
	totalIcon := "✅"
	if isDryRun {
		totalIcon = "👁️ "
	}
	totalText := lipgloss.NewStyle().
		Foreground(catGreen).
		Bold(true).
		Render(fmt.Sprintf("%s Total: %d packages", totalIcon, len(packages)))
	allLines = append(allLines, totalText)

	// Summary box
	summaryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(catGreen).
		Padding(1, 2).
		Width(tableWidth)

	fmt.Println()
	fmt.Println(summaryBox.Render(strings.Join(allLines, "\n")))
	fmt.Println()
}

func handleGitCommitAndPush(cfg *config.Config, brewfilePath string) error {
	runner := exec.NewRunner()
	dir := filepath.Dir(brewfilePath)

	// Check if it's a git repo
	if _, err := runner.Run("git", "-C", dir, "rev-parse", "--git-dir"); err != nil {
		return fmt.Errorf("not a git repository: %s", dir)
	}

	// Add the Brewfile
	fileName := filepath.Base(brewfilePath)
	if _, err := runner.Run("git", "-C", dir, "add", fileName); err != nil {
		return fmt.Errorf("failed to git add: %w", err)
	}

	// Check if there are changes to commit
	status, err := runner.Run("git", "-C", dir, "status", "--porcelain", fileName)
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if strings.TrimSpace(status) == "" {
		printInfo("No changes to commit")
		return nil
	}

	// Prepare commit message
	commitMsg := dumpMessage
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("brewsync: update %s Brewfile", cfg.CurrentMachine)
	}

	// Commit
	if _, err := runner.Run("git", "-C", dir, "commit", "-m", commitMsg); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	printInfo("✓ Committed changes: %s", commitMsg)

	// Push if requested
	if dumpPush {
		printInfo("Pushing to remote...")
		if _, err := runner.Run("git", "-C", dir, "push"); err != nil {
			return fmt.Errorf("failed to push: %w", err)
		}
		printInfo("✓ Pushed to remote")
	}

	return nil
}
