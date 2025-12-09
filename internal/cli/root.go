package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/andrew-sameh/brewsync/internal/config"
	"github.com/andrew-sameh/brewsync/internal/debug"
	"github.com/andrew-sameh/brewsync/internal/tui/app"
	"github.com/andrew-sameh/brewsync/pkg/version"
)

var (
	// Global flags
	cfgFile   string
	dryRun    bool
	verbose   bool
	quiet     bool
	noColor   bool
	assumeYes bool
)

// Catppuccin Mocha color palette
var (
	catRosewater = lipgloss.Color("#f5e0dc")
	catFlamingo  = lipgloss.Color("#f2cdcd")
	catPink      = lipgloss.Color("#f5c2e7")
	catMauve     = lipgloss.Color("#cba6f7")
	catRed       = lipgloss.Color("#f38ba8")
	catMaroon    = lipgloss.Color("#eba0ac")
	catPeach     = lipgloss.Color("#fab387")
	catYellow    = lipgloss.Color("#f9e2af")
	catGreen     = lipgloss.Color("#a6e3a1")
	catTeal      = lipgloss.Color("#94e2d5")
	catSky       = lipgloss.Color("#89dceb")
	catSapphire  = lipgloss.Color("#74c7ec")
	catBlue      = lipgloss.Color("#89b4fa")
	catLavender  = lipgloss.Color("#b4befe")
	catText      = lipgloss.Color("#cdd6f4")
	catSubtext1  = lipgloss.Color("#bac2de")
	catSubtext0  = lipgloss.Color("#a6adc8")
	catOverlay2  = lipgloss.Color("#9399b2")
	catOverlay1  = lipgloss.Color("#7f849c")
	catOverlay0  = lipgloss.Color("#6c7086")
	catSurface2  = lipgloss.Color("#585b70")
	catSurface1  = lipgloss.Color("#45475a")
	catSurface0  = lipgloss.Color("#313244")
	catBase      = lipgloss.Color("#1e1e2e")
	catMantle    = lipgloss.Color("#181825")
	catCrust     = lipgloss.Color("#11111b")
)

// Lipgloss styles using Catppuccin Mocha palette
var (
	styleSuccess = lipgloss.NewStyle().Foreground(catGreen).Bold(true)    // Green
	styleError   = lipgloss.NewStyle().Foreground(catRed).Bold(true)      // Red
	styleWarning = lipgloss.NewStyle().Foreground(catPeach).Bold(true)    // Peach
	styleBold    = lipgloss.NewStyle().Foreground(catLavender).Bold(true) // Lavender
	styleDim     = lipgloss.NewStyle().Foreground(catOverlay0)            // Overlay 0
	styleInfo    = lipgloss.NewStyle().Foreground(catSapphire)            // Sapphire
	styleMauve   = lipgloss.NewStyle().Foreground(catMauve).Bold(true)    // Mauve
	styleText    = lipgloss.NewStyle().Foreground(catText)                // Text
)

// rootCmd is the base command
var rootCmd = &cobra.Command{
	Use:   "brewsync",
	Short: "Sync Homebrew packages across macOS machines",
	Long: `BrewSync is a CLI tool to sync Homebrew packages, casks, taps,
VSCode/Cursor extensions, Go tools, and Mac App Store apps
across multiple macOS machines.

It uses a git-based dotfiles workflow where each machine has its own
Brewfile, and provides commands to import, sync, and diff packages
between machines.`,
	Version: version.Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for version command
		if cmd.Name() == "version" || cmd.Name() == "help" {
			return nil
		}

		// Set config path if provided
		if cfgFile != "" {
			config.SetConfigPath(cfgFile)
		}

		// Initialize config
		return config.Init()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Launch the full TUI when no subcommand is provided
		return runMainTUI()
	},
	SilenceUsage: true,
}

// runMainTUI launches the main TUI application
func runMainTUI() error {
	// Initialize debug logging (only if BREWSYNC_DEBUG=1 or BREWSYNC_DEBUG=true)
	debug.Init()
	defer debug.Close()

	debug.Log("runMainTUI: starting")

	var cfg *config.Config
	var err error

	// Check if config exists
	if config.Exists() {
		debug.Log("runMainTUI: config exists, loading...")
		cfg, err = config.Load()
		if err != nil {
			debug.Log("runMainTUI: config load error: %v", err)
			return fmt.Errorf("failed to load config: %w", err)
		}
		debug.Log("runMainTUI: config loaded, current machine: %s", cfg.CurrentMachine)
	} else {
		debug.Log("runMainTUI: config does not exist, will start setup wizard")
	}

	// Create and run the TUI
	debug.Log("runMainTUI: creating TUI model")
	model := app.New(cfg)

	debug.Log("runMainTUI: creating tea.Program")
	p := tea.NewProgram(model, tea.WithAltScreen())

	debug.Log("runMainTUI: running tea.Program")
	_, err = p.Run()

	debug.Log("runMainTUI: tea.Program finished, err=%v", err)
	return err
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/brewsync/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "preview without executing")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "detailed output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "minimal output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&assumeYes, "yes", "y", false, "skip confirmations")

	// Add subcommands
	rootCmd.AddCommand(dumpCmd)
}

// printInfo prints an info message (respects quiet flag)
func printInfo(format string, args ...interface{}) {
	if !quiet {
		fmt.Printf(format+"\n", args...)
	}
}

// printVerbose prints a verbose message (respects verbose flag)
func printVerbose(format string, args ...interface{}) {
	if verbose && !quiet {
		fmt.Printf(format+"\n", args...)
	}
}

// printError prints an error message
func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// printWarning prints a warning message
func printWarning(format string, args ...interface{}) {
	if !quiet {
		fmt.Fprintf(os.Stderr, "Warning: "+format+"\n", args...)
	}
}

// printStyled prints a styled string (respects noColor flag)
func printStyled(text string, style lipgloss.Style) {
	if noColor {
		fmt.Print(text)
	} else {
		fmt.Print(style.Render(text))
	}
}
