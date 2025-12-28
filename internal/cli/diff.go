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
	diffFrom   string
	diffOnly   []string
	diffFormat string
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show differences between machines",
	Long: `Show differences between the current machine and a source machine.

Without arguments, compares with the default source machine.
Use --from to specify a different source machine.

Examples:
  brewsync diff                  # Compare with default source
  brewsync diff --from air       # Compare with specific machine
  brewsync diff --only brew,cask # Filter to specific types
  brewsync diff --format json    # Output as JSON`,
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().StringVar(&diffFrom, "from", "", "source machine to compare with")
	diffCmd.Flags().StringSliceVar(&diffOnly, "only", nil, "only include these package types")
	diffCmd.Flags().StringVar(&diffFormat, "format", "table", "output format: table, json")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine source machine
	source := diffFrom
	if source == "" {
		source = cfg.DefaultSource
	}
	if source == "" {
		return fmt.Errorf("no source machine specified and no default_source in config")
	}

	// Get current machine
	currentMachine := cfg.CurrentMachine
	if currentMachine == "" {
		return fmt.Errorf("current machine not detected; run 'brewsync config init'")
	}

	// Can't diff with self
	if source == currentMachine {
		return fmt.Errorf("cannot diff machine with itself")
	}

	// Get source machine config
	sourceMachine, ok := cfg.Machines[source]
	if !ok {
		return fmt.Errorf("source machine '%s' not found in config", source)
	}

	// Get current machine config
	current, ok := cfg.Machines[currentMachine]
	if !ok {
		return fmt.Errorf("current machine '%s' not found in config", currentMachine)
	}

	printInfo("Comparing %s -> %s", source, currentMachine)

	// Parse source Brewfile
	printVerbose("Parsing source Brewfile: %s", sourceMachine.Brewfile)
	sourcePackages, err := brewfile.Parse(sourceMachine.Brewfile)
	if err != nil {
		return fmt.Errorf("failed to parse source Brewfile: %w", err)
	}

	// Parse current Brewfile
	printVerbose("Parsing current Brewfile: %s", current.Brewfile)
	currentPackages, err := brewfile.Parse(current.Brewfile)
	if err != nil {
		if os.IsNotExist(err) {
			currentPackages = brewfile.Packages{}
			printWarning("Current Brewfile not found, assuming empty")
		} else {
			return fmt.Errorf("failed to parse current Brewfile: %w", err)
		}
	}

	// Compute diff
	var diff *brewfile.DiffResult
	if len(diffOnly) > 0 {
		types := parsePackageTypes(diffOnly)
		diff = brewfile.DiffByType(sourcePackages, currentPackages, types)
	} else {
		diff = brewfile.Diff(sourcePackages, currentPackages)
	}

	// Output results
	switch diffFormat {
	case "json":
		return outputDiffJSON(diff)
	default:
		return outputDiffTable(diff, source, currentMachine)
	}
}

func parsePackageTypes(types []string) []brewfile.PackageType {
	var result []brewfile.PackageType
	for _, t := range types {
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "tap":
			result = append(result, brewfile.TypeTap)
		case "brew":
			result = append(result, brewfile.TypeBrew)
		case "cask":
			result = append(result, brewfile.TypeCask)
		case "vscode":
			result = append(result, brewfile.TypeVSCode)
		case "cursor":
			result = append(result, brewfile.TypeCursor)
		case "go":
			result = append(result, brewfile.TypeGo)
		case "mas":
			result = append(result, brewfile.TypeMas)
		}
	}
	return result
}

func outputDiffJSON(diff *brewfile.DiffResult) error {
	output := map[string]interface{}{
		"additions": packageNames(diff.Additions),
		"removals":  packageNames(diff.Removals),
		"common":    len(diff.Common),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func packageNames(pkgs brewfile.Packages) map[string][]string {
	result := make(map[string][]string)
	for _, pkg := range pkgs {
		typeName := string(pkg.Type)
		result[typeName] = append(result[typeName], pkg.Name)
	}
	return result
}

func outputDiffTable(diff *brewfile.DiffResult, source, current string) error {
	cfg, _ := config.Get()

	const tableWidth = 80

	// Header
	headerText := fmt.Sprintf("%s → %s", source, current)
	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(catOverlay0).
		Padding(0, 2).
		Width(tableWidth).
		Align(lipgloss.Center).
		Foreground(catLavender).
		Bold(true)

	fmt.Println()
	fmt.Println(headerBox.Render(headerText))
	fmt.Println()

	if diff.IsEmpty() {
		// No differences box
		noDiffBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(catGreen).
			Padding(0, 2).
			Width(tableWidth).
			Align(lipgloss.Center).
			Foreground(catGreen).
			Bold(true)

		successIcon := lipgloss.NewStyle().
			Foreground(catGreen).
			Bold(true).
			Render("✓")

		msg := lipgloss.NewStyle().
			Foreground(catGreen).
			Render("No differences found - machines are in sync!")

		content := lipgloss.JoinHorizontal(lipgloss.Left, successIcon, " ", msg)
		fmt.Println(noDiffBox.Render(content))
		fmt.Println()
		return nil
	}

	// Get ignored packages for current machine
	ignoredIDs := make(map[string]bool)
	for _, id := range cfg.GetIgnoredPackages(current) {
		ignoredIDs[id] = true
	}

	// Column width (split the table in half with some margin)
	colWidth := (tableWidth - 6) / 2 // 6 = padding + divider

	// Group packages by type
	additionsByType := diff.Additions.ByType()
	removalsByType := diff.Removals.ByType()

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

	// Build rows for each category
	for _, pkgType := range typeOrder {
		additions := additionsByType[pkgType]
		removals := removalsByType[pkgType]

		// Skip if no changes for this type
		if len(additions) == 0 && len(removals) == 0 {
			continue
		}

		info := typeInfo[pkgType]

		// Category header (spans both columns)
		categoryHeader := lipgloss.NewStyle().
			Foreground(info.color).
			Bold(true).
			Render(fmt.Sprintf("%s %s", info.icon, pkgType))

		allRows = append(allRows, categoryHeader)

		// Separator
		separator := lipgloss.NewStyle().
			Foreground(catOverlay0).
			Render(strings.Repeat("─", tableWidth-4))
		allRows = append(allRows, separator)

		// Build left column (additions) for this type
		var leftLines []string
		if len(additions) > 0 {
			leftHeader := lipgloss.NewStyle().
				Foreground(catGreen).
				Bold(true).
				Render(fmt.Sprintf("➕ To Install (%d)", len(additions)))
			leftLines = append(leftLines, leftHeader)

			for _, pkg := range additions {
				prefix := lipgloss.NewStyle().
					Foreground(catGreen).
					Bold(true).
					Render("+")

				pkgName := lipgloss.NewStyle().
					Foreground(catText).
					Render(pkg.Name)

				ignored := ignoredIDs != nil && ignoredIDs[pkg.ID()]
				if ignored {
					ignoredTag := lipgloss.NewStyle().
						Foreground(catOverlay1).
						Italic(true).
						Render("(ignored)")
					leftLines = append(leftLines, fmt.Sprintf("  %s %s %s", prefix, pkgName, ignoredTag))
				} else {
					leftLines = append(leftLines, fmt.Sprintf("  %s %s", prefix, pkgName))
				}
			}
		}

		// Build right column (removals) for this type
		var rightLines []string
		if len(removals) > 0 {
			rightHeader := lipgloss.NewStyle().
				Foreground(catRed).
				Bold(true).
				Render(fmt.Sprintf("➖ To Remove (%d)", len(removals)))
			rightLines = append(rightLines, rightHeader)

			for _, pkg := range removals {
				prefix := lipgloss.NewStyle().
					Foreground(catRed).
					Bold(true).
					Render("-")

				pkgName := lipgloss.NewStyle().
					Foreground(catText).
					Render(pkg.Name)

				ignored := ignoredIDs != nil && ignoredIDs[pkg.ID()]
				if ignored {
					ignoredTag := lipgloss.NewStyle().
						Foreground(catOverlay1).
						Italic(true).
						Render("(ignored)")
					rightLines = append(rightLines, fmt.Sprintf("  %s %s %s", prefix, pkgName, ignoredTag))
				} else {
					rightLines = append(rightLines, fmt.Sprintf("  %s %s", prefix, pkgName))
				}
			}
		}

		// Equalize line counts
		maxLines := len(leftLines)
		if len(rightLines) > maxLines {
			maxLines = len(rightLines)
		}

		// Pad shorter column with empty lines
		for len(leftLines) < maxLines {
			leftLines = append(leftLines, "")
		}
		for len(rightLines) < maxLines {
			rightLines = append(rightLines, "")
		}

		// Create side-by-side rows
		for i := 0; i < maxLines; i++ {
			leftCol := lipgloss.NewStyle().
				Width(colWidth).
				Render(leftLines[i])

			rightCol := lipgloss.NewStyle().
				Width(colWidth).
				Render(rightLines[i])

			divider := lipgloss.NewStyle().
				Foreground(catOverlay0).
				Render("│")

			row := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, divider, rightCol)
			allRows = append(allRows, row)
		}

		// Add spacing between categories
		allRows = append(allRows, "")
	}

	// Content box
	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(catOverlay0).
		Padding(1, 2).
		Width(tableWidth)

	content := strings.Join(allRows, "\n")
	fmt.Println(contentBox.Render(content))
	fmt.Println()

	// Summary box
	summaryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(catBlue).
		Padding(0, 2).
		Width(tableWidth).
		Align(lipgloss.Center).
		Foreground(catBlue)

	summaryText := diff.Summary()
	fmt.Println(summaryBox.Render(summaryText))
	fmt.Println()

	return nil
}

func formatPackagesByType(pkgs brewfile.Packages, prefix string, prefixColor lipgloss.Color, ignoredIDs map[string]bool) []string {
	var lines []string

	byType := pkgs.ByType()
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

	for _, t := range typeOrder {
		typePkgs := byType[t]
		if len(typePkgs) == 0 {
			continue
		}

		info := typeInfo[t]

		// Type header
		typeHeader := lipgloss.NewStyle().
			Foreground(info.color).
			Bold(true).
			Render(fmt.Sprintf("%s %s (%d)", info.icon, t, len(typePkgs)))

		lines = append(lines, typeHeader)

		// Packages
		for _, pkg := range typePkgs {
			prefixStyled := lipgloss.NewStyle().
				Foreground(prefixColor).
				Bold(true).
				Render(prefix)

			pkgName := lipgloss.NewStyle().
				Foreground(catText).
				Render(pkg.Name)

			// Check if ignored
			ignored := ignoredIDs != nil && ignoredIDs[pkg.ID()]

			var line string
			if ignored {
				ignoredTag := lipgloss.NewStyle().
					Foreground(catOverlay1).
					Italic(true).
					Render("(ignored)")
				line = fmt.Sprintf("  %s %s %s", prefixStyled, pkgName, ignoredTag)
			} else {
				line = fmt.Sprintf("  %s %s", prefixStyled, pkgName)
			}

			lines = append(lines, line)
		}
	}

	return lines
}

func printPackagesByType(pkgs brewfile.Packages, prefix string) {
	// Legacy function - kept for compatibility
	lines := formatPackagesByType(pkgs, prefix, catText, nil)
	for _, line := range lines {
		fmt.Println(line)
	}
}
