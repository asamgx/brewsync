package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/asamgx/brewsync/internal/config"
)

var ignoreCmd = &cobra.Command{
	Use:   "ignore",
	Short: "Manage ignore lists",
	Long: `Manage packages and categories that should be ignored during sync/import.

Ignores are stored in a separate ignore.yaml file with two layers:
1. Categories - Ignore entire package types (e.g., all mas packages)
2. Packages - Ignore specific packages within non-ignored categories

Subcommands:
  category  Manage ignored categories (tap, brew, cask, vscode, cursor, antigravity, go, mas)
  add       Add a package to ignore list
  remove    Remove a package from ignore list
  list      Show all ignored categories and packages
  path      Show ignore file path
  init      Create default ignore.yaml file`,
}

var (
	ignoreMachine string
	ignoreGlobal  bool
)

// Category commands
var ignoreCategoryCmd = &cobra.Command{
	Use:   "category",
	Short: "Manage ignored categories",
	Long:  "Manage entire package type ignores (e.g., ignore ALL mas packages)",
}

var ignoreCategoryAddCmd = &cobra.Command{
	Use:   "add [category]",
	Short: "Add a category to ignore list",
	Long: `Ignore an entire package category.

Valid categories: tap, brew, cask, vscode, cursor, antigravity, go, mas

Examples:
  brewsync ignore category add mas                 # Ignore all Mac App Store apps globally
  brewsync ignore category add go --machine mini   # Ignore all Go tools on mini`,
	Args: cobra.ExactArgs(1),
	RunE: runIgnoreCategoryAdd,
}

var ignoreCategoryRemoveCmd = &cobra.Command{
	Use:   "remove [category]",
	Short: "Remove a category from ignore list",
	Args:  cobra.ExactArgs(1),
	RunE:  runIgnoreCategoryRemove,
}

var ignoreCategoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all ignored categories",
	RunE:  runIgnoreCategoryList,
}

// Package commands
var ignoreAddCmd = &cobra.Command{
	Use:   "add [type:name]",
	Short: "Add a package to ignore list",
	Long: `Add a specific package to the ignore list.

Format: type:name
Examples:
  brewsync ignore add cask:bluestacks              # Add to current machine
  brewsync ignore add brew:postgresql --global     # Add globally
  brewsync ignore add vscode:ext --machine mini    # Add to specific machine`,
	Args: cobra.ExactArgs(1),
	RunE: runIgnoreAdd,
}

var ignoreRemoveCmd = &cobra.Command{
	Use:   "remove [type:name]",
	Short: "Remove a package from ignore list",
	Args:  cobra.ExactArgs(1),
	RunE:  runIgnoreRemove,
}

var ignoreListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all ignored categories and packages",
	RunE:  runIgnoreList,
}

var ignorePathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show ignore file path",
	RunE:  runIgnorePath,
}

var ignoreInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default ignore.yaml file",
	RunE:  runIgnoreInit,
}

func init() {
	// Category subcommand flags
	ignoreCategoryAddCmd.Flags().StringVar(&ignoreMachine, "machine", "", "add to specific machine")
	ignoreCategoryAddCmd.Flags().BoolVar(&ignoreGlobal, "global", false, "add globally (default if no machine)")
	ignoreCategoryRemoveCmd.Flags().StringVar(&ignoreMachine, "machine", "", "remove from specific machine")
	ignoreCategoryRemoveCmd.Flags().BoolVar(&ignoreGlobal, "global", false, "remove from global")
	ignoreCategoryListCmd.Flags().StringVar(&ignoreMachine, "machine", "", "show only for specific machine")

	ignoreCategoryCmd.AddCommand(ignoreCategoryAddCmd)
	ignoreCategoryCmd.AddCommand(ignoreCategoryRemoveCmd)
	ignoreCategoryCmd.AddCommand(ignoreCategoryListCmd)

	// Package command flags
	ignoreAddCmd.Flags().StringVar(&ignoreMachine, "machine", "", "add to specific machine's ignore list")
	ignoreAddCmd.Flags().BoolVar(&ignoreGlobal, "global", false, "add to global ignore list")
	ignoreRemoveCmd.Flags().StringVar(&ignoreMachine, "machine", "", "remove from specific machine's ignore list")
	ignoreRemoveCmd.Flags().BoolVar(&ignoreGlobal, "global", false, "remove from global ignore list")
	ignoreListCmd.Flags().StringVar(&ignoreMachine, "machine", "", "show only for specific machine")

	ignoreCmd.AddCommand(ignoreCategoryCmd)
	ignoreCmd.AddCommand(ignoreAddCmd)
	ignoreCmd.AddCommand(ignoreRemoveCmd)
	ignoreCmd.AddCommand(ignoreListCmd)
	ignoreCmd.AddCommand(ignorePathCmd)
	ignoreCmd.AddCommand(ignoreInitCmd)
	rootCmd.AddCommand(ignoreCmd)
}

func runIgnoreCategoryAdd(cmd *cobra.Command, args []string) error {
	category := args[0]

	// Validate category
	validCategories := map[string]bool{
		"tap": true, "brew": true, "cask": true,
		"vscode": true, "cursor": true, "antigravity": true,
		"go": true, "mas": true,
	}
	if !validCategories[category] {
		return fmt.Errorf("invalid category '%s'; valid categories: tap, brew, cask, vscode, cursor, antigravity, go, mas", category)
	}

	// Determine machine
	machine := ignoreMachine
	global := ignoreGlobal || machine == ""

	if !global {
		// If machine not specified and not global, use current machine
		if machine == "" {
			cfg, err := config.Get()
			if err == nil && cfg.CurrentMachine != "" {
				machine = cfg.CurrentMachine
			}
		}
	}

	// Add category ignore
	if err := config.AddCategoryIgnore(machine, category, global); err != nil {
		return fmt.Errorf("failed to add category ignore: %w", err)
	}

	if global || machine == "" {
		printInfo("Added category '%s' to global ignore list", category)
	} else {
		printInfo("Added category '%s' to %s ignore list", category, machine)
	}

	return nil
}

func runIgnoreCategoryRemove(cmd *cobra.Command, args []string) error {
	category := args[0]

	machine := ignoreMachine
	global := ignoreGlobal || machine == ""

	if !global && machine == "" {
		cfg, err := config.Get()
		if err == nil && cfg.CurrentMachine != "" {
			machine = cfg.CurrentMachine
		}
	}

	if err := config.RemoveCategoryIgnore(machine, category, global); err != nil {
		return fmt.Errorf("failed to remove category ignore: %w", err)
	}

	if global || machine == "" {
		printInfo("Removed category '%s' from global ignore list", category)
	} else {
		printInfo("Removed category '%s' from %s ignore list", category, machine)
	}

	return nil
}

func runIgnoreCategoryList(cmd *cobra.Command, args []string) error {
	ignoreFile, err := config.LoadIgnoreFile()
	if err != nil {
		return fmt.Errorf("failed to load ignore file: %w", err)
	}

	hasEntries := false

	// Show global ignored categories
	if ignoreMachine == "" && len(ignoreFile.Global.Categories) > 0 {
		fmt.Println("Global ignored categories:")
		for _, cat := range ignoreFile.Global.Categories {
			fmt.Printf("  %s\n", cat)
		}
		hasEntries = true
	}

	// Show machine-specific ignored categories
	for machine, ignoreConfig := range ignoreFile.Machines {
		if ignoreMachine != "" && machine != ignoreMachine {
			continue
		}

		if len(ignoreConfig.Categories) > 0 {
			if hasEntries {
				fmt.Println()
			}
			fmt.Printf("Machine '%s' ignored categories:\n", machine)
			for _, cat := range ignoreConfig.Categories {
				fmt.Printf("  %s\n", cat)
			}
			hasEntries = true
		}
	}

	if !hasEntries {
		fmt.Println("No categories are being ignored.")
	}

	return nil
}

func runIgnoreAdd(cmd *cobra.Command, args []string) error {
	pkgID := args[0]

	machine := ignoreMachine
	global := ignoreGlobal || machine == ""

	if !global && machine == "" {
		cfg, err := config.Get()
		if err == nil && cfg.CurrentMachine != "" {
			machine = cfg.CurrentMachine
		}
	}

	if err := config.AddPackageIgnore(machine, pkgID, global); err != nil {
		return fmt.Errorf("failed to add package ignore: %w", err)
	}

	if global || machine == "" {
		printInfo("Added %s to global ignore list", pkgID)
	} else {
		printInfo("Added %s to %s ignore list", pkgID, machine)
	}

	return nil
}

func runIgnoreRemove(cmd *cobra.Command, args []string) error {
	pkgID := args[0]

	machine := ignoreMachine
	global := ignoreGlobal || machine == ""

	if !global && machine == "" {
		cfg, err := config.Get()
		if err == nil && cfg.CurrentMachine != "" {
			machine = cfg.CurrentMachine
		}
	}

	if err := config.RemovePackageIgnore(machine, pkgID, global); err != nil {
		return fmt.Errorf("failed to remove package ignore: %w", err)
	}

	if global || machine == "" {
		printInfo("Removed %s from global ignore list", pkgID)
	} else {
		printInfo("Removed %s from %s ignore list", pkgID, machine)
	}

	return nil
}

func runIgnoreList(cmd *cobra.Command, args []string) error {
	ignoreFile, err := config.LoadIgnoreFile()
	if err != nil {
		return fmt.Errorf("failed to load ignore file: %w", err)
	}

	hasEntries := false

	// Show global ignores
	if ignoreMachine == "" {
		if len(ignoreFile.Global.Categories) > 0 || hasPackages(ignoreFile.Global.Packages) {
			fmt.Println("Global ignores:")

			if len(ignoreFile.Global.Categories) > 0 {
				fmt.Println("  Categories:")
				for _, cat := range ignoreFile.Global.Categories {
					fmt.Printf("    - %s\n", cat)
				}
			}

			pkgs := listPackages(ignoreFile.Global.Packages)
			if len(pkgs) > 0 {
				fmt.Println("  Packages:")
				for _, pkg := range pkgs {
					fmt.Printf("    - %s\n", pkg)
				}
			}

			hasEntries = true
		}
	}

	// Show machine-specific ignores
	for machine, ignoreConfig := range ignoreFile.Machines {
		if ignoreMachine != "" && machine != ignoreMachine {
			continue
		}

		if len(ignoreConfig.Categories) > 0 || hasPackages(ignoreConfig.Packages) {
			if hasEntries {
				fmt.Println()
			}
			fmt.Printf("Machine '%s' ignores:\n", machine)

			if len(ignoreConfig.Categories) > 0 {
				fmt.Println("  Categories:")
				for _, cat := range ignoreConfig.Categories {
					fmt.Printf("    - %s\n", cat)
				}
			}

			pkgs := listPackages(ignoreConfig.Packages)
			if len(pkgs) > 0 {
				fmt.Println("  Packages:")
				for _, pkg := range pkgs {
					fmt.Printf("    - %s\n", pkg)
				}
			}

			hasEntries = true
		}
	}

	if !hasEntries {
		fmt.Println("No packages or categories are being ignored.")
		fmt.Printf("\nIgnore file location: %s\n", config.IgnorePath())
	}

	return nil
}

func runIgnorePath(cmd *cobra.Command, args []string) error {
	fmt.Println(config.IgnorePath())
	return nil
}

func runIgnoreInit(cmd *cobra.Command, args []string) error {
	if err := config.CreateDefaultIgnoreFile(); err != nil {
		return fmt.Errorf("failed to create ignore file: %w", err)
	}

	path := config.IgnorePath()
	printInfo("Created ignore file at: %s", path)

	return nil
}

// Helper functions

func hasPackages(list config.PackageIgnoreList) bool {
	return len(list.Tap) > 0 || len(list.Brew) > 0 || len(list.Cask) > 0 ||
		len(list.VSCode) > 0 || len(list.Cursor) > 0 || len(list.Antigravity) > 0 ||
		len(list.Go) > 0 || len(list.Mas) > 0
}

func listPackages(list config.PackageIgnoreList) []string {
	var pkgs []string

	for _, name := range list.Tap {
		pkgs = append(pkgs, "tap:"+name)
	}
	for _, name := range list.Brew {
		pkgs = append(pkgs, "brew:"+name)
	}
	for _, name := range list.Cask {
		pkgs = append(pkgs, "cask:"+name)
	}
	for _, name := range list.VSCode {
		pkgs = append(pkgs, "vscode:"+name)
	}
	for _, name := range list.Cursor {
		pkgs = append(pkgs, "cursor:"+name)
	}
	for _, name := range list.Antigravity {
		pkgs = append(pkgs, "antigravity:"+name)
	}
	for _, name := range list.Go {
		pkgs = append(pkgs, "go:"+name)
	}
	for _, name := range list.Mas {
		pkgs = append(pkgs, "mas:"+name)
	}

	return pkgs
}
