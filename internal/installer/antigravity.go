package installer

import (
	"strings"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/exec"
)

// AntigravityInstaller handles Antigravity editor extensions
type AntigravityInstaller struct {
	runner *exec.Runner
}

// NewAntigravityInstaller creates a new Antigravity installer
func NewAntigravityInstaller() *AntigravityInstaller {
	return &AntigravityInstaller{
		runner: exec.Default,
	}
}

// List returns all installed Antigravity extensions
func (a *AntigravityInstaller) List() (brewfile.Packages, error) {
	lines, err := a.runner.RunLines("agy", "--list-extensions")
	if err != nil {
		return nil, err
	}

	var packages brewfile.Packages
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			packages = append(packages, brewfile.NewPackage(brewfile.TypeAntigravity, line))
		}
	}
	return packages, nil
}

// Install installs an Antigravity extension
func (a *AntigravityInstaller) Install(pkg brewfile.Package) error {
	if pkg.Type != brewfile.TypeAntigravity {
		return nil
	}
	_, err := a.runner.Run("agy", "--install-extension", pkg.Name)
	return err
}

// Uninstall removes an Antigravity extension
func (a *AntigravityInstaller) Uninstall(pkg brewfile.Package) error {
	if pkg.Type != brewfile.TypeAntigravity {
		return nil
	}
	_, err := a.runner.Run("agy", "--uninstall-extension", pkg.Name)
	return err
}

// IsAvailable checks if agy CLI is available
func (a *AntigravityInstaller) IsAvailable() bool {
	return a.runner.Exists("agy")
}
