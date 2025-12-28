package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/installer"
	"github.com/asamgx/brewsync/internal/profile"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage and use profiles",
	Long: `Manage curated package groups (profiles).

Profiles are collections of packages for specific purposes,
like "core" tools or "dev-go" for Go development.

Subcommands:
  list     List available profiles
  show     Display profile contents
  install  Install packages from profile(s)
  create   Create a new profile
  edit     Edit a profile in $EDITOR
  delete   Delete a profile`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available profiles",
	RunE:  runProfileList,
}

var profileShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Display profile contents",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileShow,
}

var profileInstallCmd = &cobra.Command{
	Use:   "install [names...]",
	Short: "Install packages from profile(s)",
	Long: `Install packages from one or more profiles.

Examples:
  brewsync profile install core
  brewsync profile install core dev-go
  brewsync profile install core,dev-go,k8s`,
	Args: cobra.MinimumNArgs(1),
	RunE: runProfileInstall,
}

var (
	profileCreateDesc string
)

var profileCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileCreate,
}

var profileEditCmd = &cobra.Command{
	Use:   "edit [name]",
	Short: "Edit a profile in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileEdit,
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileDelete,
}

func init() {
	profileCreateCmd.Flags().StringVar(&profileCreateDesc, "description", "", "profile description")

	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileInstallCmd)
	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileEditCmd)
	profileCmd.AddCommand(profileDeleteCmd)
	rootCmd.AddCommand(profileCmd)
}

func runProfileList(cmd *cobra.Command, args []string) error {
	names, err := profile.List()
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	if len(names) == 0 {
		fmt.Println("No profiles found.")
		fmt.Println("Create one with 'brewsync profile create <name>'")
		return nil
	}

	fmt.Println("Available profiles:")
	for _, name := range names {
		p, err := profile.Load(name)
		if err != nil {
			fmt.Printf("  %s (error loading)\n", name)
			continue
		}

		desc := ""
		if p.Description != "" {
			desc = " - " + p.Description
		}
		fmt.Printf("  %s (%d packages)%s\n", name, p.Packages.Count(), desc)
	}

	return nil
}

func runProfileShow(cmd *cobra.Command, args []string) error {
	name := args[0]

	p, err := profile.Load(name)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}

	fmt.Printf("Profile: %s\n", p.Name)
	if p.Description != "" {
		fmt.Printf("Description: %s\n", p.Description)
	}
	fmt.Printf("Total packages: %d\n\n", p.Packages.Count())

	if len(p.Packages.Tap) > 0 {
		fmt.Printf("Taps (%d):\n", len(p.Packages.Tap))
		for _, name := range p.Packages.Tap {
			fmt.Printf("  %s\n", name)
		}
		fmt.Println()
	}

	if len(p.Packages.Brew) > 0 {
		fmt.Printf("Brews (%d):\n", len(p.Packages.Brew))
		for _, name := range p.Packages.Brew {
			fmt.Printf("  %s\n", name)
		}
		fmt.Println()
	}

	if len(p.Packages.Cask) > 0 {
		fmt.Printf("Casks (%d):\n", len(p.Packages.Cask))
		for _, name := range p.Packages.Cask {
			fmt.Printf("  %s\n", name)
		}
		fmt.Println()
	}

	if len(p.Packages.VSCode) > 0 {
		fmt.Printf("VSCode (%d):\n", len(p.Packages.VSCode))
		for _, name := range p.Packages.VSCode {
			fmt.Printf("  %s\n", name)
		}
		fmt.Println()
	}

	if len(p.Packages.Cursor) > 0 {
		fmt.Printf("Cursor (%d):\n", len(p.Packages.Cursor))
		for _, name := range p.Packages.Cursor {
			fmt.Printf("  %s\n", name)
		}
		fmt.Println()
	}

	if len(p.Packages.Go) > 0 {
		fmt.Printf("Go (%d):\n", len(p.Packages.Go))
		for _, name := range p.Packages.Go {
			fmt.Printf("  %s\n", name)
		}
		fmt.Println()
	}

	if len(p.Packages.Mas) > 0 {
		fmt.Printf("Mac App Store (%d):\n", len(p.Packages.Mas))
		for _, name := range p.Packages.Mas {
			fmt.Printf("  %s\n", name)
		}
	}

	return nil
}

func runProfileInstall(cmd *cobra.Command, args []string) error {
	// Parse profile names (handle comma-separated)
	var names []string
	for _, arg := range args {
		for _, name := range strings.Split(arg, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				names = append(names, name)
			}
		}
	}

	// Load all profiles
	profiles, err := profile.LoadMultiple(names)
	if err != nil {
		return err
	}

	// Merge packages from all profiles
	packages := profile.MergePackages(profiles)

	if len(packages) == 0 {
		printInfo("No packages in selected profiles")
		return nil
	}

	printInfo("Installing %d packages from %d profile(s)...", len(packages), len(profiles))

	if dryRun {
		fmt.Println("\nWould install:")
		for _, pkg := range packages {
			fmt.Printf("  %s:%s\n", pkg.Type, pkg.Name)
		}
		return nil
	}

	// Install packages
	mgr := installer.NewManager()
	var installed, failed int

	mgr.InstallMany(packages, func(pkg brewfile.Package, i, total int, err error) {
		if err != nil {
			printError("[%d/%d] Failed to install %s:%s: %v", i, total, pkg.Type, pkg.Name, err)
			failed++
		} else {
			printInfo("[%d/%d] Installed %s:%s", i, total, pkg.Type, pkg.Name)
			installed++
		}
	})

	fmt.Println()
	printInfo("Installed: %d, Failed: %d", installed, failed)

	return nil
}

func runProfileCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	if profile.Exists(name) {
		return fmt.Errorf("profile '%s' already exists", name)
	}

	p := &profile.Profile{
		Name:        name,
		Description: profileCreateDesc,
		Packages:    profile.Packages{},
	}

	if err := profile.Save(p); err != nil {
		return err
	}

	path, _ := profile.GetPath(name)
	printInfo("Created profile at %s", path)
	printInfo("Edit with 'brewsync profile edit %s'", name)

	return nil
}

func runProfileEdit(cmd *cobra.Command, args []string) error {
	name := args[0]

	path, err := profile.GetPath(name)
	if err != nil {
		return err
	}

	if !profile.Exists(name) {
		return fmt.Errorf("profile '%s' does not exist; create it first with 'brewsync profile create %s'", name, name)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	editorCmd := exec.Command(editor, path)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	return editorCmd.Run()
}

func runProfileDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !profile.Exists(name) {
		return fmt.Errorf("profile '%s' does not exist", name)
	}

	if !assumeYes {
		fmt.Printf("Delete profile '%s'? [y/N] ", name)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			return nil
		}
	}

	if err := profile.Delete(name); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	printInfo("Deleted profile '%s'", name)
	return nil
}
