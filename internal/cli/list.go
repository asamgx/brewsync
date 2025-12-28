package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/config"
)

var (
	listFrom   string
	listOnly   []string
	listFormat string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List packages in a Brewfile",
	Long: `List packages in a machine's Brewfile.

Without arguments, lists packages from the current machine.
Use --from to list from a different machine.

Examples:
  brewsync list                  # Current machine
  brewsync list --from mini      # Another machine
  brewsync list --only brew      # Filter by type
  brewsync list --format json    # JSON output`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringVar(&listFrom, "from", "", "machine to list packages from")
	listCmd.Flags().StringSliceVar(&listOnly, "only", nil, "only include these package types")
	listCmd.Flags().StringVar(&listFormat, "format", "table", "output format: table, json")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine which machine to list
	machineName := listFrom
	if machineName == "" {
		machineName = cfg.CurrentMachine
	}
	if machineName == "" {
		return fmt.Errorf("no machine specified and current machine not detected")
	}

	// Get machine config
	machine, ok := cfg.Machines[machineName]
	if !ok {
		return fmt.Errorf("machine '%s' not found in config", machineName)
	}

	printVerbose("Reading Brewfile: %s", machine.Brewfile)

	// Parse Brewfile
	packages, err := brewfile.Parse(machine.Brewfile)
	if err != nil {
		if os.IsNotExist(err) {
			printInfo("No Brewfile found at %s", machine.Brewfile)
			return nil
		}
		return fmt.Errorf("failed to parse Brewfile: %w", err)
	}

	// Filter by type if specified
	if len(listOnly) > 0 {
		types := parsePackageTypes(listOnly)
		packages = packages.Filter(types...)
	}

	// Output results
	switch listFormat {
	case "json":
		return outputListJSON(packages, machineName)
	default:
		return outputListTable(packages, machineName)
	}
}

func outputListJSON(packages brewfile.Packages, machine string) error {
	output := map[string]interface{}{
		"machine":  machine,
		"packages": packageNames(packages),
		"counts":   packageCounts(packages),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func packageCounts(pkgs brewfile.Packages) map[string]int {
	counts := make(map[string]int)
	for _, pkg := range pkgs {
		counts[string(pkg.Type)]++
	}
	return counts
}

func outputListTable(packages brewfile.Packages, machine string) error {
	if len(packages) == 0 {
		printInfo("No packages found for %s", machine)
		return nil
	}

	const tableWidth = 80

	// Header box
	headerText := fmt.Sprintf("Packages for %s", machine)
	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(catOverlay0).
		Padding(0, 2).
		Width(tableWidth).
		Align(lipgloss.Center).
		Foreground(catLavender).
		Bold(true)

	// Total count
	totalText := lipgloss.NewStyle().
		Foreground(catSubtext0).
		Render(fmt.Sprintf("Total: %d packages", len(packages)))

	header := lipgloss.JoinVertical(lipgloss.Center, headerText, totalText)

	fmt.Println()
	fmt.Println(headerBox.Render(header))
	fmt.Println()

	// Package groups
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

	// Type icons/colors
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

	var allRows []string

	for _, t := range typeOrder {
		typePkgs := byType[t]
		if len(typePkgs) == 0 {
			continue
		}

		info := typeInfo[t]

		// Category header with icon
		categoryHeader := lipgloss.NewStyle().
			Foreground(info.color).
			Bold(true).
			Render(fmt.Sprintf("%s %s (%d)", info.icon, t, len(typePkgs)))

		allRows = append(allRows, categoryHeader)

		// Separator
		separator := lipgloss.NewStyle().
			Foreground(catOverlay0).
			Render(strings.Repeat("─", tableWidth-4))
		allRows = append(allRows, separator)

		// Package list
		for _, pkg := range typePkgs {
			bullet := lipgloss.NewStyle().
				Foreground(catOverlay1).
				Render("•")

			pkgName := lipgloss.NewStyle().
				Foreground(catText).
				Render(pkg.Name)

			// Add description if available
			var row string
			if pkg.Description != "" {
				desc := lipgloss.NewStyle().
					Foreground(catSubtext0).
					Italic(true).
					Render(pkg.Description)
				row = fmt.Sprintf("  %s %s — %s", bullet, pkgName, desc)
			} else {
				row = fmt.Sprintf("  %s %s", bullet, pkgName)
			}

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

	return nil
}
