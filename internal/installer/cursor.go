package installer

import (
	"strings"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/exec"
)

// CursorInstaller handles Cursor extensions
type CursorInstaller struct {
	runner  *exec.Runner
	command string
}

// NewCursorInstaller creates a new Cursor installer
func NewCursorInstaller() *CursorInstaller {
	return &CursorInstaller{
		runner:  exec.Default,
		command: "cursor",
	}
}

// List returns all installed Cursor extensions
func (c *CursorInstaller) List() (brewfile.Packages, error) {
	lines, err := c.runner.RunLines(c.command, "--list-extensions")
	if err != nil {
		return nil, err
	}

	var packages brewfile.Packages
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			packages = append(packages, brewfile.NewPackage(brewfile.TypeCursor, line))
		}
	}
	return packages, nil
}

// Install installs a Cursor extension
func (c *CursorInstaller) Install(pkg brewfile.Package) error {
	_, err := c.runner.Run(c.command, "--install-extension", pkg.Name)
	return err
}

// Uninstall removes a Cursor extension
func (c *CursorInstaller) Uninstall(pkg brewfile.Package) error {
	_, err := c.runner.Run(c.command, "--uninstall-extension", pkg.Name)
	return err
}

// IsAvailable checks if cursor CLI is available
func (c *CursorInstaller) IsAvailable() bool {
	return c.runner.Exists(c.command)
}
