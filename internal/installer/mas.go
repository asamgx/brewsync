package installer

import (
	"regexp"
	"strings"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/exec"
)

// MasInstaller handles Mac App Store apps
type MasInstaller struct {
	runner *exec.Runner
}

// NewMasInstaller creates a new Mac App Store installer
func NewMasInstaller() *MasInstaller {
	return &MasInstaller{
		runner: exec.Default,
	}
}

// masListPattern matches "123456789 App Name (1.0.0)"
var masListPattern = regexp.MustCompile(`^(\d+)\s+(.+?)\s+\([\d.]+\)$`)

// List returns all installed Mac App Store apps
func (m *MasInstaller) List() (brewfile.Packages, error) {
	lines, err := m.runner.RunLines("mas", "list")
	if err != nil {
		return nil, err
	}

	var packages brewfile.Packages
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := masListPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		id := matches[1]
		name := matches[2]

		pkg := brewfile.NewPackage(brewfile.TypeMas, id)
		pkg.FullName = name
		pkg = pkg.WithOption("id", id)
		packages = append(packages, pkg)
	}
	return packages, nil
}

// Install installs a Mac App Store app by ID
func (m *MasInstaller) Install(pkg brewfile.Package) error {
	id := pkg.Name
	if idOpt, ok := pkg.Options["id"]; ok {
		id = idOpt
	}
	_, err := m.runner.Run("mas", "install", id)
	return err
}

// Uninstall is not supported for Mac App Store apps
func (m *MasInstaller) Uninstall(pkg brewfile.Package) error {
	// mas doesn't support uninstall, need to use the App Store or manual deletion
	return nil
}

// IsAvailable checks if mas CLI is available
func (m *MasInstaller) IsAvailable() bool {
	return m.runner.Exists("mas")
}
