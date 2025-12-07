package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/andrew-sameh/brewsync/internal/brewfile"
	"github.com/andrew-sameh/brewsync/internal/config"
	"github.com/andrew-sameh/brewsync/internal/history"
	"github.com/andrew-sameh/brewsync/internal/installer"
)

var (
	syncFrom    string
	syncOnly    string
	syncApply   bool
	syncPreview bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync current machine to match source exactly",
	Long: `Make current machine match source exactly (adds AND removes packages).

Unlike import, sync will both install missing packages and remove packages
that exist on current but not on source. This makes the machines identical.

By default, sync shows a preview. Use --apply to execute changes.

Examples:
  brewsync sync                    # Preview mode (dry-run)
  brewsync sync --preview          # Explicit preview
  brewsync sync --apply            # Execute changes
  brewsync sync --from air         # Sync from specific machine
  brewsync sync --only brew        # Only sync brews
  brewsync sync --apply --dry-run  # Preview even with --apply`,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().StringVar(&syncFrom, "from", "", "source machine to sync from")
	syncCmd.Flags().StringVar(&syncOnly, "only", "", "only sync these package types (comma-separated)")
	syncCmd.Flags().BoolVar(&syncApply, "apply", false, "apply changes (default is preview only)")
	syncCmd.Flags().BoolVar(&syncPreview, "preview", false, "show preview (default behavior)")

	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
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

	// Determine source machine
	source := cfg.DefaultSource
	if syncFrom != "" {
		source = syncFrom
	}

	if source == currentMachine {
		return fmt.Errorf("cannot sync from current machine '%s'", source)
	}

	if _, ok := cfg.Machines[source]; !ok {
		return fmt.Errorf("unknown source machine: %s", source)
	}

	printInfo("Syncing %s to match %s", currentMachine, source)

	// Load both Brewfiles
	currentBrewfile := cfg.Machines[currentMachine].Brewfile
	currentPkgs, err := brewfile.Parse(currentBrewfile)
	if err != nil {
		currentPkgs = brewfile.Packages{}
	}

	sourceBrewfile := cfg.Machines[source].Brewfile
	sourcePkgs, err := brewfile.Parse(sourceBrewfile)
	if err != nil {
		return fmt.Errorf("failed to parse source Brewfile: %w", err)
	}

	// Compute diff
	diff := brewfile.Diff(sourcePkgs, currentPkgs)
	additions := diff.Additions
	removals := diff.Removals

	// Filter by category if specified
	if syncOnly != "" {
		categories := parseCategories(syncOnly)
		additions = filterByCategories(additions, categories, true)
		removals = filterByCategories(removals, categories, true)
	}

	// Filter ignored packages from additions
	ignored := cfg.GetIgnoredPackages(currentMachine)
	ignoredMap := make(map[string]bool)
	for _, pkg := range ignored {
		ignoredMap[pkg] = true
	}

	var filteredAdditions brewfile.Packages
	for _, pkg := range additions {
		if !ignoredMap[pkg.ID()] {
			filteredAdditions = append(filteredAdditions, pkg)
		}
	}
	additions = filteredAdditions

	// Get machine-specific packages for current machine (protected from removal)
	machineSpecific := cfg.GetMachineSpecificPackages()
	protectedPkgs := make(map[string]bool)
	if specific, ok := machineSpecific[currentMachine]; ok {
		for _, pkg := range specific {
			protectedPkgs[pkg] = true
		}
	}

	// Also mark ignored packages as protected from removal
	for pkg := range ignoredMap {
		protectedPkgs[pkg] = true
	}

	// Filter protected packages from removals
	var filteredRemovals brewfile.Packages
	var protectedList brewfile.Packages
	for _, pkg := range removals {
		if protectedPkgs[pkg.ID()] {
			protectedList = append(protectedList, pkg)
		} else {
			filteredRemovals = append(filteredRemovals, pkg)
		}
	}
	removals = filteredRemovals

	// Check if there's anything to do
	if len(additions) == 0 && len(removals) == 0 {
		printInfo("Already in sync - no changes needed")
		return nil
	}

	// Display preview
	fmt.Println()
	fmt.Printf("Sync Preview: %s → %s\n", source, currentMachine)
	fmt.Println(strings.Repeat("─", 50))

	if len(additions) > 0 {
		fmt.Printf("\n%s TO BE INSTALLED (+%d)\n", colorGreen("▶"), len(additions))
		grouped := groupByType(additions)
		for pkgType, pkgs := range grouped {
			names := getPkgNames(pkgs)
			if len(names) > 5 {
				fmt.Printf("  %s: %s (+%d more)\n", pkgType, strings.Join(names[:5], ", "), len(names)-5)
			} else {
				fmt.Printf("  %s: %s\n", pkgType, strings.Join(names, ", "))
			}
		}
	}

	if len(removals) > 0 {
		fmt.Printf("\n%s TO BE REMOVED (-%d)\n", colorRed("▶"), len(removals))
		grouped := groupByType(removals)
		for pkgType, pkgs := range grouped {
			names := getPkgNames(pkgs)
			if len(names) > 5 {
				fmt.Printf("  %s: %s (+%d more)\n", pkgType, strings.Join(names[:5], ", "), len(names)-5)
			} else {
				fmt.Printf("  %s: %s\n", pkgType, strings.Join(names, ", "))
			}
		}
	}

	if len(protectedList) > 0 {
		fmt.Printf("\n%s PROTECTED (machine-specific/ignored, won't be removed: %d)\n", colorYellow("▶"), len(protectedList))
		grouped := groupByType(protectedList)
		for pkgType, pkgs := range grouped {
			names := getPkgNames(pkgs)
			fmt.Printf("  %s: %s\n", pkgType, strings.Join(names, ", "))
		}
	}

	fmt.Println()

	// If preview mode or dry-run, stop here
	if !syncApply || dryRun {
		if dryRun {
			printInfo("Dry-run mode - no changes made")
		} else {
			printInfo("Run 'brewsync sync --apply' to execute these changes")
		}
		return nil
	}

	// Confirm before applying
	if !assumeYes {
		fmt.Printf("Apply these changes? [y/N] ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			printInfo("Sync cancelled")
			return nil
		}
	}

	// Apply changes
	mgr := installer.NewManager()
	var installedCount, removedCount, failedCount int

	// Install additions first
	if len(additions) > 0 {
		printInfo("Installing %d packages...", len(additions))
		mgr.InstallMany(additions, func(pkg brewfile.Package, i, total int, err error) {
			if err != nil {
				printError("[%d/%d] Failed to install %s:%s: %v", i, total, pkg.Type, pkg.Name, err)
				failedCount++
			} else {
				printInfo("[%d/%d] Installed %s:%s", i, total, pkg.Type, pkg.Name)
				installedCount++
			}
		})
	}

	// Then remove packages
	if len(removals) > 0 {
		printInfo("Removing %d packages...", len(removals))
		mgr.UninstallMany(removals, func(pkg brewfile.Package, i, total int, err error) {
			if err != nil {
				printError("[%d/%d] Failed to remove %s:%s: %v", i, total, pkg.Type, pkg.Name, err)
				failedCount++
			} else {
				printInfo("[%d/%d] Removed %s:%s", i, total, pkg.Type, pkg.Name)
				removedCount++
			}
		})
	}

	fmt.Println()
	printInfo("Sync complete: +%d installed, -%d removed, %d failed",
		installedCount, removedCount, failedCount)

	// Log to history
	history.LogSync(currentMachine, source, installedCount, removedCount)

	// Auto-dump if enabled and changes were made
	if (installedCount > 0 || removedCount > 0) && cfg.AutoDump.Enabled && cfg.AutoDump.AfterInstall {
		printInfo("Auto-dumping Brewfile...")

		// Set flags for commit/push based on config
		oldCommit := dumpCommit
		oldPush := dumpPush
		oldMessage := dumpMessage

		dumpCommit = cfg.AutoDump.Commit
		dumpPush = cfg.AutoDump.Push
		if cfg.AutoDump.CommitMessage != "" {
			dumpMessage = strings.ReplaceAll(cfg.AutoDump.CommitMessage, "{machine}", currentMachine)
		}

		err := runDump(nil, []string{})

		// Restore flags
		dumpCommit = oldCommit
		dumpPush = oldPush
		dumpMessage = oldMessage

		if err != nil {
			printWarning("Auto-dump failed: %v", err)
		}
	}

	return nil
}

// groupByType groups packages by their type
func groupByType(pkgs brewfile.Packages) map[brewfile.PackageType]brewfile.Packages {
	result := make(map[brewfile.PackageType]brewfile.Packages)
	for _, pkg := range pkgs {
		result[pkg.Type] = append(result[pkg.Type], pkg)
	}
	return result
}

// getPkgNames extracts package names as a slice
func getPkgNames(pkgs brewfile.Packages) []string {
	names := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		names[i] = pkg.Name
	}
	return names
}

// Color helpers for non-TUI output
func colorGreen(s string) string {
	if noColor {
		return s
	}
	return "\033[32m" + s + "\033[0m"
}

func colorRed(s string) string {
	if noColor {
		return s
	}
	return "\033[31m" + s + "\033[0m"
}

func colorYellow(s string) string {
	if noColor {
		return s
	}
	return "\033[33m" + s + "\033[0m"
}
