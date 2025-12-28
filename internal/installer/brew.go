package installer

import (
	"strings"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/exec"
)

// BrewInstaller handles Homebrew formulae and casks
type BrewInstaller struct {
	runner *exec.Runner
}

// NewBrewInstaller creates a new Homebrew installer
func NewBrewInstaller() *BrewInstaller {
	return &BrewInstaller{
		runner: exec.Default,
	}
}

// ListTaps returns all installed taps
func (b *BrewInstaller) ListTaps() (brewfile.Packages, error) {
	lines, err := b.runner.RunLines("brew", "tap")
	if err != nil {
		return nil, err
	}

	var packages brewfile.Packages
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			packages = append(packages, brewfile.NewPackage(brewfile.TypeTap, line))
		}
	}
	return packages, nil
}

// ListFormulae returns all installed formulae (without descriptions)
// Use 'brew bundle dump --describe' via DumpToFile for descriptions
func (b *BrewInstaller) ListFormulae() (brewfile.Packages, error) {
	lines, err := b.runner.RunLines("brew", "list", "--formula", "-1")
	if err != nil {
		return nil, err
	}

	var packages brewfile.Packages
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			packages = append(packages, brewfile.NewPackage(brewfile.TypeBrew, line))
		}
	}
	return packages, nil
}

// ListCasks returns all installed casks (without descriptions)
// Use 'brew bundle dump --describe' via DumpToFile for descriptions
func (b *BrewInstaller) ListCasks() (brewfile.Packages, error) {
	lines, err := b.runner.RunLines("brew", "list", "--cask", "-1")
	if err != nil {
		return nil, err
	}

	var packages brewfile.Packages
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			packages = append(packages, brewfile.NewPackage(brewfile.TypeCask, line))
		}
	}
	return packages, nil
}

// ListAll returns all taps, formulae, and casks
func (b *BrewInstaller) ListAll() (brewfile.Packages, error) {
	var all brewfile.Packages

	taps, err := b.ListTaps()
	if err != nil {
		return nil, err
	}
	all = append(all, taps...)

	formulae, err := b.ListFormulae()
	if err != nil {
		return nil, err
	}
	all = append(all, formulae...)

	casks, err := b.ListCasks()
	if err != nil {
		return nil, err
	}
	all = append(all, casks...)

	return all, nil
}

// List returns all installed packages (alias for ListAll)
func (b *BrewInstaller) List() (brewfile.Packages, error) {
	return b.ListAll()
}

// Install installs a package
func (b *BrewInstaller) Install(pkg brewfile.Package) error {
	return b.InstallWithProgress(pkg, nil)
}

// InstallWithProgress installs a package and streams output to a callback
func (b *BrewInstaller) InstallWithProgress(pkg brewfile.Package, onOutput func(line string)) error {
	var args []string

	switch pkg.Type {
	case brewfile.TypeTap:
		args = []string{"tap", pkg.Name}
	case brewfile.TypeBrew:
		args = []string{"install", pkg.Name}
	case brewfile.TypeCask:
		args = []string{"install", "--cask", pkg.Name}
	default:
		return nil
	}

	// If no callback provided, use the regular Run method
	if onOutput == nil {
		_, err := b.runner.Run("brew", args...)
		return err
	}

	// Use streaming method with callback
	return b.runner.RunWithOutput("brew", args, onOutput)
}

// Uninstall removes a package
func (b *BrewInstaller) Uninstall(pkg brewfile.Package) error {
	switch pkg.Type {
	case brewfile.TypeTap:
		_, err := b.runner.Run("brew", "untap", pkg.Name)
		return err
	case brewfile.TypeBrew:
		_, err := b.runner.Run("brew", "uninstall", pkg.Name)
		return err
	case brewfile.TypeCask:
		_, err := b.runner.Run("brew", "uninstall", "--cask", pkg.Name)
		return err
	default:
		return nil
	}
}

// IsAvailable checks if brew is available
func (b *BrewInstaller) IsAvailable() bool {
	return b.runner.Exists("brew")
}

// DumpToFile runs brew bundle dump to a file with descriptions
// This uses 'brew bundle dump --describe' which automatically includes
// package descriptions as comments in the output Brewfile
func (b *BrewInstaller) DumpToFile(path string) error {
	_, err := b.runner.Run("brew", "bundle", "dump", "--force", "--describe", "--file="+path)
	return err
}
