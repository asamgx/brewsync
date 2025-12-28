package history

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/asamgx/brewsync/internal/config"
)

// Operation represents the type of operation logged
type Operation string

const (
	OpDump      Operation = "dump"
	OpImport    Operation = "import"
	OpSync      Operation = "sync"
	OpIgnore    Operation = "ignore"
	OpProfile   Operation = "profile"
	OpInstall   Operation = "install"
	OpUninstall Operation = "uninstall"
)

// Entry represents a single history log entry
type Entry struct {
	Timestamp time.Time
	Operation Operation
	Machine   string
	Details   string
	Summary   string
}

// Log appends an entry to the history log
func Log(op Operation, machine, details, summary string) error {
	path, err := config.HistoryPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := config.EnsureDir(); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	entry := formatEntry(Entry{
		Timestamp: time.Now(),
		Operation: op,
		Machine:   machine,
		Details:   details,
		Summary:   summary,
	})

	if _, err := f.WriteString(entry + "\n"); err != nil {
		return fmt.Errorf("failed to write history entry: %w", err)
	}

	return nil
}

// LogDump logs a dump operation
func LogDump(machine string, counts map[string]int, committed bool) error {
	var parts []string
	for typ, count := range counts {
		parts = append(parts, fmt.Sprintf("%s:%d", typ, count))
	}
	details := strings.Join(parts, ",")

	summary := "dumped"
	if committed {
		summary = "committed"
	}

	return Log(OpDump, machine, details, summary)
}

// LogImport logs an import operation
func LogImport(machine, source string, added []string) error {
	details := fmt.Sprintf("←%s;+%s", source, strings.Join(added, ","))
	summary := fmt.Sprintf("%d packages", len(added))
	return Log(OpImport, machine, details, summary)
}

// LogSync logs a sync operation
func LogSync(machine, source string, added, removed int) error {
	details := fmt.Sprintf("←%s;+%d,-%d", source, added, removed)
	summary := "applied"
	return Log(OpSync, machine, details, summary)
}

// LogInstall logs a single package install operation
func LogInstall(machine, pkgID string, success bool) error {
	summary := "installed"
	if !success {
		summary = "failed"
	}
	return Log(OpInstall, machine, pkgID, summary)
}

// LogUninstall logs a single package uninstall operation
func LogUninstall(machine, pkgID string, success bool) error {
	summary := "uninstalled"
	if !success {
		summary = "failed"
	}
	return Log(OpUninstall, machine, pkgID, summary)
}

// Read returns the most recent history entries
func Read(limit int) ([]Entry, error) {
	path, err := config.HistoryPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		entry, err := parseEntry(line)
		if err != nil {
			continue // Skip malformed entries
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading history file: %w", err)
	}

	// Return most recent entries (last N)
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	// Reverse to show most recent first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, nil
}

// formatEntry formats an entry for the log file
// Format: timestamp|operation|machine|details|summary
func formatEntry(e Entry) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s",
		e.Timestamp.Format(time.RFC3339),
		e.Operation,
		e.Machine,
		e.Details,
		e.Summary,
	)
}

// parseEntry parses a log line into an Entry
func parseEntry(line string) (Entry, error) {
	parts := strings.SplitN(line, "|", 5)
	if len(parts) < 5 {
		return Entry{}, fmt.Errorf("invalid entry format")
	}

	timestamp, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return Entry{}, fmt.Errorf("invalid timestamp: %w", err)
	}

	return Entry{
		Timestamp: timestamp,
		Operation: Operation(parts[1]),
		Machine:   parts[2],
		Details:   parts[3],
		Summary:   parts[4],
	}, nil
}

// Clear removes all history entries
func Clear() error {
	path, err := config.HistoryPath()
	if err != nil {
		return err
	}

	return os.Remove(path)
}

// FormatEntry returns a human-readable representation of an entry
func (e Entry) Format(detailed bool) string {
	timeStr := e.Timestamp.Format("2006-01-02 15:04")

	if detailed {
		return fmt.Sprintf("%s  %-8s  %-10s  %s  (%s)",
			timeStr,
			e.Operation,
			e.Machine,
			e.Details,
			e.Summary,
		)
	}

	return fmt.Sprintf("%s  %-8s  %s  %s",
		timeStr,
		e.Operation,
		e.Machine,
		e.Summary,
	)
}

// ParseCounts parses a counts string like "tap:6,brew:85"
func ParseCounts(s string) map[string]int {
	counts := make(map[string]int)
	parts := strings.Split(s, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			if count, err := strconv.Atoi(kv[1]); err == nil {
				counts[kv[0]] = count
			}
		}
	}
	return counts
}
