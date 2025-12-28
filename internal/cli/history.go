package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/asamgx/brewsync/internal/history"
)

var (
	historyLimit  int
	historyDetail bool
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View operation history",
	Long: `View recent BrewSync operations.

Shows a log of dump, import, sync, and other operations
performed by BrewSync.`,
	RunE: runHistory,
}

func init() {
	historyCmd.Flags().IntVarP(&historyLimit, "limit", "n", 10, "number of entries to show")
	historyCmd.Flags().BoolVar(&historyDetail, "detail", false, "show detailed information")
	rootCmd.AddCommand(historyCmd)
}

func runHistory(cmd *cobra.Command, args []string) error {
	entries, err := history.Read(historyLimit)
	if err != nil {
		return fmt.Errorf("failed to read history: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No history entries found.")
		return nil
	}

	fmt.Printf("Recent operations (showing %d):\n\n", len(entries))

	for _, entry := range entries {
		fmt.Println(entry.Format(historyDetail))
	}

	return nil
}
