package installer

import (
	"strings"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/exec"
)

// VSCodeInstaller handles VSCode extensions
type VSCodeInstaller struct {
	runner  *exec.Runner
	command string
}

// NewVSCodeInstaller creates a new VSCode installer
func NewVSCodeInstaller() *VSCodeInstaller {
	return &VSCodeInstaller{
		runner:  exec.Default,
		command: "code",
	}
}

// List returns all installed VSCode extensions
func (v *VSCodeInstaller) List() (brewfile.Packages, error) {
	lines, err := v.runner.RunLines(v.command, "--list-extensions")
	if err != nil {
		return nil, err
	}

	var packages brewfile.Packages
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			packages = append(packages, brewfile.NewPackage(brewfile.TypeVSCode, line))
		}
	}
	return packages, nil
}

// Install installs a VSCode extension
func (v *VSCodeInstaller) Install(pkg brewfile.Package) error {
	_, err := v.runner.Run(v.command, "--install-extension", pkg.Name)
	return err
}

// Uninstall removes a VSCode extension
func (v *VSCodeInstaller) Uninstall(pkg brewfile.Package) error {
	_, err := v.runner.Run(v.command, "--uninstall-extension", pkg.Name)
	return err
}

// IsAvailable checks if code CLI is available
func (v *VSCodeInstaller) IsAvailable() bool {
	return v.runner.Exists(v.command)
}
