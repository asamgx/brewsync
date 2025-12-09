package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	enabled   bool
	logFile   *os.File
	mu        sync.Mutex
	initOnce  sync.Once
)

// Init initializes the debug logger based on BREWSYNC_DEBUG env var
func Init() {
	initOnce.Do(func() {
		if os.Getenv("BREWSYNC_DEBUG") == "1" || os.Getenv("BREWSYNC_DEBUG") == "true" {
			enabled = true

			// Create log file in temp directory
			logPath := filepath.Join(os.TempDir(), "brewsync-debug.log")
			var err error
			logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create debug log: %v\n", err)
				enabled = false
				return
			}

			Log("Debug logging enabled, writing to: %s", logPath)
		}
	})
}

// Enabled returns whether debug mode is enabled
func Enabled() bool {
	return enabled
}

// Log writes a debug message if debug mode is enabled
func Log(format string, args ...interface{}) {
	if !enabled || logFile == nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(logFile, "[%s] %s\n", timestamp, msg)
	logFile.Sync() // Flush immediately for debugging
}

// LogError logs an error message
func LogError(context string, err error) {
	if err != nil {
		Log("ERROR [%s]: %v", context, err)
	}
}

// Close closes the debug log file
func Close() {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// GetLogPath returns the path to the debug log file
func GetLogPath() string {
	return filepath.Join(os.TempDir(), "brewsync-debug.log")
}
