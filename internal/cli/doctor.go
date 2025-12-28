package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/asamgx/brewsync/internal/config"
	"github.com/asamgx/brewsync/internal/exec"
	"github.com/asamgx/brewsync/pkg/version"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate setup and diagnose issues",
	Long: `Check your BrewSync configuration and environment for potential issues.

Validates:
  - BrewSync version
  - Config file exists and is valid
  - Ignore file exists
  - Current machine is detected
  - Brewfile paths exist
  - Required CLI tools are available (brew, code, cursor, antigravity, mas, go)`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

type checkResult struct {
	name    string
	ok      bool
	message string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	var results []checkResult

	// Check version
	results = append(results, checkVersion())

	// Check config file
	results = append(results, checkConfigFile())

	// Check ignore file
	results = append(results, checkIgnoreFile())

	// Load config for further checks
	cfg, err := config.Get()
	if err != nil {
		results = append(results, checkResult{
			name:    "Config loading",
			ok:      false,
			message: fmt.Sprintf("Failed to load config: %v", err),
		})
		printResults(results)
		return nil
	}

	// Check current machine
	results = append(results, checkCurrentMachine(cfg))

	// Check Brewfile paths
	results = append(results, checkBrewfilePaths(cfg)...)

	// Check default source
	if cfg.DefaultSource != "" {
		results = append(results, checkDefaultSource(cfg))
	}

	// Check CLI tools
	results = append(results, checkCLITools()...)

	printResults(results)
	return nil
}

func checkVersion() checkResult {
	return checkResult{
		name:    "BrewSync version",
		ok:      true,
		message: version.Full(),
	}
}

func checkConfigFile() checkResult {
	if config.Exists() {
		path, _ := config.ConfigPath()
		return checkResult{
			name:    "Config file",
			ok:      true,
			message: fmt.Sprintf("Found at %s", path),
		}
	}
	return checkResult{
		name:    "Config file",
		ok:      false,
		message: "Not found. Run 'brewsync config init' to create one.",
	}
}

func checkIgnoreFile() checkResult {
	ignorePath := config.IgnorePath()
	if _, err := os.Stat(ignorePath); os.IsNotExist(err) {
		return checkResult{
			name:    "Ignore file",
			ok:      true, // It's optional
			message: fmt.Sprintf("Not found (optional). Run 'brewsync ignore init' to create one."),
		}
	} else if err != nil {
		return checkResult{
			name:    "Ignore file",
			ok:      false,
			message: fmt.Sprintf("Error: %v", err),
		}
	}
	return checkResult{
		name:    "Ignore file",
		ok:      true,
		message: fmt.Sprintf("Found at %s", ignorePath),
	}
}

func checkCurrentMachine(cfg *config.Config) checkResult {
	if cfg.CurrentMachine == "" {
		return checkResult{
			name:    "Current machine",
			ok:      false,
			message: "Not detected. Check hostname configuration.",
		}
	}

	if _, ok := cfg.Machines[cfg.CurrentMachine]; !ok {
		return checkResult{
			name:    "Current machine",
			ok:      false,
			message: fmt.Sprintf("'%s' not found in machines config", cfg.CurrentMachine),
		}
	}

	return checkResult{
		name:    "Current machine",
		ok:      true,
		message: cfg.CurrentMachine,
	}
}

func checkBrewfilePaths(cfg *config.Config) []checkResult {
	var results []checkResult

	for name, machine := range cfg.Machines {
		if machine.Brewfile == "" {
			results = append(results, checkResult{
				name:    fmt.Sprintf("Brewfile (%s)", name),
				ok:      false,
				message: "Path not configured",
			})
			continue
		}

		if _, err := os.Stat(machine.Brewfile); os.IsNotExist(err) {
			// Only warn if it's the current machine
			if name == cfg.CurrentMachine {
				results = append(results, checkResult{
					name:    fmt.Sprintf("Brewfile (%s)", name),
					ok:      false,
					message: fmt.Sprintf("Not found at %s", machine.Brewfile),
				})
			}
		} else if err != nil {
			results = append(results, checkResult{
				name:    fmt.Sprintf("Brewfile (%s)", name),
				ok:      false,
				message: fmt.Sprintf("Error: %v", err),
			})
		} else {
			results = append(results, checkResult{
				name:    fmt.Sprintf("Brewfile (%s)", name),
				ok:      true,
				message: machine.Brewfile,
			})
		}
	}

	return results
}

func checkDefaultSource(cfg *config.Config) checkResult {
	if _, ok := cfg.Machines[cfg.DefaultSource]; !ok {
		return checkResult{
			name:    "Default source",
			ok:      false,
			message: fmt.Sprintf("'%s' not found in machines config", cfg.DefaultSource),
		}
	}

	return checkResult{
		name:    "Default source",
		ok:      true,
		message: cfg.DefaultSource,
	}
}

func checkCLITools() []checkResult {
	var results []checkResult

	tools := []struct {
		name     string
		command  string
		required bool
	}{
		{"Homebrew", "brew", true},
		{"brew bundle", "brew", true}, // Will check bundle separately
		{"VSCode CLI", "code", false},
		{"Cursor CLI", "cursor", false},
		{"Antigravity CLI", "agy", false},
		{"Mac App Store CLI", "mas", false},
		{"Go", "go", false},
	}

	for _, tool := range tools {
		if tool.name == "brew bundle" {
			// Special check for brew bundle
			_, err := exec.Run("brew", "bundle", "--help")
			if err != nil {
				results = append(results, checkResult{
					name:    tool.name,
					ok:      false,
					message: "Not available. Run 'brew tap homebrew/bundle'",
				})
			} else {
				results = append(results, checkResult{
					name:    tool.name,
					ok:      true,
					message: "Available",
				})
			}
			continue
		}

		if exec.Exists(tool.command) {
			results = append(results, checkResult{
				name:    tool.name,
				ok:      true,
				message: "Installed",
			})
		} else {
			msg := "Not found"
			if !tool.required {
				msg += fmt.Sprintf(" (%s packages won't sync)", tool.name)
			}
			results = append(results, checkResult{
				name:    tool.name,
				ok:      !tool.required,
				message: msg,
			})
		}
	}

	return results
}

func printResults(results []checkResult) {
	const tableWidth = 80

	// Header box
	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(catOverlay0).
		Padding(0, 2).
		Width(tableWidth).
		Align(lipgloss.Center).
		Foreground(catLavender).
		Bold(true)

	fmt.Println()
	fmt.Println(headerBox.Render("BrewSync Environment Diagnostics"))
	fmt.Println()

	// Group results by category
	categories := []struct {
		title string
		start int
		end   int
	}{
		{"Version & Configuration", 0, 3},
		{"Machine Setup", 3, 7},
		{"CLI Tools", 7, len(results)},
	}

	// Build content for each category
	var allRows []string

	for _, cat := range categories {
		if cat.start >= len(results) {
			continue
		}

		// Category header
		categoryHeader := lipgloss.NewStyle().
			Foreground(catMauve).
			Bold(true).
			Padding(0, 1).
			Render("◆ " + cat.title)

		allRows = append(allRows, categoryHeader)

		// Separator
		separator := lipgloss.NewStyle().
			Foreground(catOverlay0).
			Render(strings.Repeat("─", tableWidth-4))
		allRows = append(allRows, separator)

		// Print results in this category
		end := cat.end
		if end > len(results) {
			end = len(results)
		}

		for i := cat.start; i < end; i++ {
			r := results[i]

			// Status icon with color
			var statusIcon string
			var statusColor lipgloss.Color
			if r.ok {
				statusIcon = "✓"
				statusColor = catGreen
			} else {
				statusIcon = "✗"
				statusColor = catRed
			}

			status := lipgloss.NewStyle().
				Foreground(statusColor).
				Bold(true).
				Width(3).
				Render(statusIcon)

			// Check name
			name := lipgloss.NewStyle().
				Foreground(catText).
				Bold(true).
				Width(28).
				Render(r.name)

			// Message
			message := lipgloss.NewStyle().
				Foreground(catSubtext0).
				Width(tableWidth - 35).
				Render(r.message)

			row := lipgloss.JoinHorizontal(lipgloss.Left, status, " ", name, " ", message)
			allRows = append(allRows, row)
		}

		// Add spacing between categories
		allRows = append(allRows, "")
	}

	// Main content box
	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(catOverlay0).
		Padding(1, 2).
		Width(tableWidth)

	content := strings.Join(allRows, "\n")
	fmt.Println(contentBox.Render(content))
	fmt.Println()

	// Summary
	var failures int
	for _, r := range results {
		if !r.ok {
			failures++
		}
	}

	// Summary box
	summaryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(catOverlay0).
		Padding(0, 2).
		Width(tableWidth).
		Align(lipgloss.Center)

	var summaryContent string
	if failures == 0 {
		successIcon := lipgloss.NewStyle().
			Foreground(catGreen).
			Bold(true).
			Render("✓")

		successMsg := lipgloss.NewStyle().
			Foreground(catGreen).
			Bold(true).
			Render("All checks passed! Your setup is ready to go!")

		summaryContent = lipgloss.JoinHorizontal(lipgloss.Left, successIcon, " ", successMsg)
		summaryBox = summaryBox.BorderForeground(catGreen)
	} else {
		errorIcon := lipgloss.NewStyle().
			Foreground(catRed).
			Bold(true).
			Render("✗")

		errorMsg := lipgloss.NewStyle().
			Foreground(catRed).
			Bold(true).
			Render(fmt.Sprintf("Found %d issue(s) that need attention", failures))

		summaryContent = lipgloss.JoinHorizontal(lipgloss.Left, errorIcon, " ", errorMsg)
		summaryBox = summaryBox.BorderForeground(catRed)
	}

	fmt.Println(summaryBox.Render(summaryContent))
	fmt.Println()
}
