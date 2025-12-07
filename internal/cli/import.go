package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/andrew-sameh/brewsync/internal/brewfile"
	"github.com/andrew-sameh/brewsync/internal/config"
	"github.com/andrew-sameh/brewsync/internal/history"
	"github.com/andrew-sameh/brewsync/internal/installer"
	"github.com/andrew-sameh/brewsync/internal/tui/progress"
	"github.com/andrew-sameh/brewsync/internal/tui/selection"
)

var (
	importFrom                  string
	importOnly                  string
	importSkip                  string
	importIncludeMachineSpecific bool
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import packages from another machine",
	Long: `Import (install) missing packages from another machine.

The import command shows packages that exist on the source machine but not
on the current machine, and lets you select which ones to install.

Examples:
  brewsync import                      # From default source, interactive
  brewsync import --from air           # From specific machine
  brewsync import --from mini,air      # Union of multiple machines
  brewsync import --only brew,cask     # Filter categories
  brewsync import --skip vscode        # Exclude categories
  brewsync import --yes                # Install all without prompts
  brewsync import --dry-run            # Show what would be installed`,
	RunE: runImport,
}

func init() {
	importCmd.Flags().StringVar(&importFrom, "from", "", "source machine(s) to import from (comma-separated)")
	importCmd.Flags().StringVar(&importOnly, "only", "", "only import these package types (comma-separated)")
	importCmd.Flags().StringVar(&importSkip, "skip", "", "skip these package types (comma-separated)")
	importCmd.Flags().BoolVar(&importIncludeMachineSpecific, "include-machine-specific", false, "include machine-specific packages")

	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Detect current machine
	currentMachine := cfg.CurrentMachine
	if currentMachine == "" {
		return fmt.Errorf("could not detect current machine; run 'brewsync config init' first")
	}

	// Determine source machines
	sources := []string{cfg.DefaultSource}
	if importFrom != "" {
		sources = strings.Split(importFrom, ",")
		for i := range sources {
			sources[i] = strings.TrimSpace(sources[i])
		}
	}

	// Validate source machines
	for _, source := range sources {
		if source == currentMachine {
			return fmt.Errorf("cannot import from current machine '%s'", source)
		}
		if _, ok := cfg.Machines[source]; !ok {
			return fmt.Errorf("unknown source machine: %s", source)
		}
	}

	printInfo("Importing to %s from %s", currentMachine, strings.Join(sources, ", "))

	// Load current machine's Brewfile
	currentBrewfile := cfg.Machines[currentMachine].Brewfile
	currentPkgs, err := brewfile.Parse(currentBrewfile)
	if err != nil {
		// If file doesn't exist, start with empty
		currentPkgs = brewfile.Packages{}
	}

	// Load and merge source Brewfiles
	var sourcePkgs brewfile.Packages
	seen := make(map[string]bool)

	for _, source := range sources {
		sourceBrewfile := cfg.Machines[source].Brewfile
		pkgs, err := brewfile.Parse(sourceBrewfile)
		if err != nil {
			printWarning("Failed to parse %s's Brewfile: %v", source, err)
			continue
		}

		for _, pkg := range pkgs {
			key := pkg.ID()
			if !seen[key] {
				seen[key] = true
				sourcePkgs = append(sourcePkgs, pkg)
			}
		}
	}

	// Compute diff (what's in source but not in current)
	diff := brewfile.Diff(sourcePkgs, currentPkgs)
	missing := diff.Additions

	if len(missing) == 0 {
		printInfo("No new packages to import")
		return nil
	}

	// Filter by category
	if importOnly != "" {
		categories := parseCategories(importOnly)
		missing = filterByCategories(missing, categories, true)
	}
	if importSkip != "" {
		categories := parseCategories(importSkip)
		missing = filterByCategories(missing, categories, false)
	}

	// Filter ignored packages
	ignored := cfg.GetIgnoredPackages(currentMachine)
	ignoredMap := make(map[string]bool)
	for _, pkg := range ignored {
		ignoredMap[pkg] = true
	}

	var filtered brewfile.Packages
	for _, pkg := range missing {
		if !ignoredMap[pkg.ID()] {
			filtered = append(filtered, pkg)
		}
	}
	missing = filtered

	// Filter machine-specific packages (unless opted in)
	if !importIncludeMachineSpecific {
		machineSpecific := cfg.GetMachineSpecificPackages()
		var nonSpecific brewfile.Packages
		for _, pkg := range missing {
			isSpecific := false
			for machine, pkgs := range machineSpecific {
				if machine == currentMachine {
					continue
				}
				for _, specific := range pkgs {
					if pkg.ID() == specific {
						isSpecific = true
						break
					}
				}
				if isSpecific {
					break
				}
			}
			if !isSpecific {
				nonSpecific = append(nonSpecific, pkg)
			}
		}
		missing = nonSpecific
	}

	if len(missing) == 0 {
		printInfo("No new packages to import (after filters)")
		return nil
	}

	printInfo("Found %d packages to import", len(missing))

	// Dry run - just show what would be imported
	if dryRun {
		fmt.Println("\nWould import:")
		for _, pkg := range missing {
			fmt.Printf("  %s:%s\n", pkg.Type, pkg.Name)
		}
		return nil
	}

	// Interactive or auto mode
	var toInstall brewfile.Packages

	if assumeYes {
		// Install all without prompts
		toInstall = missing
	} else {
		// Interactive selection
		title := fmt.Sprintf("Import from %s - Select packages to install", strings.Join(sources, ", "))
		model := selection.New(title, missing)
		model.SetIgnored(ignoredMap)

		// Pre-select all by default
		preselected := make(map[string]bool)
		for _, pkg := range missing {
			preselected[pkg.ID()] = true
		}
		model.SetSelected(preselected)

		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		m := finalModel.(selection.Model)
		if m.Cancelled() {
			printInfo("Import cancelled")
			return nil
		}

		toInstall = m.Selected()

		// Handle newly ignored categories
		ignoredCategories := m.IgnoredCategories()
		if len(ignoredCategories) > 0 {
			printInfo("Adding %d categories to ignore list", len(ignoredCategories))
			for _, category := range ignoredCategories {
				if err := config.AddCategoryIgnore(currentMachine, category, false); err != nil {
					printWarning("Failed to ignore category %s: %v", category, err)
				}
			}
		}

		// Handle newly ignored packages
		newlyIgnored := m.Ignored()
		if len(newlyIgnored) > 0 {
			printInfo("Adding %d packages to ignore list", len(newlyIgnored))
			for _, pkg := range newlyIgnored {
				pkgID := string(pkg.Type) + ":" + pkg.Name
				if err := config.AddPackageIgnore(currentMachine, pkgID, false); err != nil {
					printWarning("Failed to ignore %s: %v", pkg.Name, err)
				}
			}
		}
	}

	if len(toInstall) == 0 {
		printInfo("No packages selected for installation")
		return nil
	}

	printInfo("Installing %d packages...", len(toInstall))

	// Install packages
	mgr := installer.NewManager()

	if assumeYes {
		// Non-interactive progress
		var installed, failed int
		mgr.InstallMany(toInstall, func(pkg brewfile.Package, i, total int, err error) {
			if err != nil {
				printError("[%d/%d] Failed: %s:%s - %v", i, total, pkg.Type, pkg.Name, err)
				failed++
			} else {
				printInfo("[%d/%d] Installed: %s:%s", i, total, pkg.Type, pkg.Name)
				installed++
			}
		})

		fmt.Println()
		printInfo("Installed: %d, Failed: %d", installed, failed)

		// Log to history
		var pkgNames []string
		for _, pkg := range toInstall {
			pkgNames = append(pkgNames, pkg.ID())
		}
		history.LogImport(currentMachine, strings.Join(sources, ","), pkgNames)
	} else {
		// Interactive progress UI with streaming support
		title := "Installing packages"
		progressModel := progress.NewWithOutput(title, toInstall, func(pkg brewfile.Package, onOutput func(line string)) error {
			return mgr.InstallWithProgress(pkg, onOutput)
		})

		p := tea.NewProgram(progressModel, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("progress TUI error: %w", err)
		}

		m := finalModel.(progress.Model)
		printInfo("Installed: %d, Failed: %d", m.Installed(), m.Failed())

		// Log to history
		var pkgNames []string
		for _, pkg := range toInstall {
			pkgNames = append(pkgNames, pkg.ID())
		}
		history.LogImport(currentMachine, strings.Join(sources, ","), pkgNames)
	}

	return nil
}

// parseCategories parses a comma-separated list of category names
func parseCategories(s string) []brewfile.PackageType {
	var result []brewfile.PackageType
	for _, c := range strings.Split(s, ",") {
		c = strings.TrimSpace(c)
		if c != "" {
			result = append(result, brewfile.PackageType(c))
		}
	}
	return result
}

// filterByCategories filters packages by category
// If include is true, only include packages matching categories
// If include is false, exclude packages matching categories
func filterByCategories(pkgs brewfile.Packages, categories []brewfile.PackageType, include bool) brewfile.Packages {
	categorySet := make(map[brewfile.PackageType]bool)
	for _, c := range categories {
		categorySet[c] = true
	}

	var result brewfile.Packages
	for _, pkg := range pkgs {
		inSet := categorySet[pkg.Type]
		if include && inSet {
			result = append(result, pkg)
		} else if !include && !inSet {
			result = append(result, pkg)
		}
	}
	return result
}
