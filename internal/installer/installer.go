package installer

import (
	"fmt"

	"github.com/asamgx/brewsync/internal/brewfile"
)

// Installer is the interface for package installers
type Installer interface {
	Install(pkg brewfile.Package) error
	Uninstall(pkg brewfile.Package) error
	List() (brewfile.Packages, error)
	IsAvailable() bool
}

// Manager orchestrates installations across different package types
type Manager struct {
	brew        *BrewInstaller
	vscode      *VSCodeInstaller
	cursor      *CursorInstaller
	antigravity *AntigravityInstaller
	mas         *MasInstaller
	go_         *GoToolsInstaller
}

// NewManager creates a new installation manager
func NewManager() *Manager {
	return &Manager{
		brew:        NewBrewInstaller(),
		vscode:      NewVSCodeInstaller(),
		cursor:      NewCursorInstaller(),
		antigravity: NewAntigravityInstaller(),
		mas:         NewMasInstaller(),
		go_:         NewGoToolsInstaller(),
	}
}

// Install installs a package using the appropriate installer
func (m *Manager) Install(pkg brewfile.Package) error {
	return m.InstallWithProgress(pkg, nil)
}

// InstallWithProgress installs a package and streams output to a callback
func (m *Manager) InstallWithProgress(pkg brewfile.Package, onOutput func(line string)) error {
	installer, err := m.getInstaller(pkg.Type)
	if err != nil {
		return err
	}

	if !installer.IsAvailable() {
		return fmt.Errorf("%s installer not available", pkg.Type)
	}

	// Use specialized method for brew packages that support streaming
	if pkg.Type == brewfile.TypeTap || pkg.Type == brewfile.TypeBrew || pkg.Type == brewfile.TypeCask {
		return m.brew.InstallWithProgress(pkg, onOutput)
	}

	// Other installers don't support streaming yet, use regular install
	return installer.Install(pkg)
}

// Uninstall removes a package using the appropriate installer
func (m *Manager) Uninstall(pkg brewfile.Package) error {
	installer, err := m.getInstaller(pkg.Type)
	if err != nil {
		return err
	}

	if !installer.IsAvailable() {
		return fmt.Errorf("%s installer not available", pkg.Type)
	}

	return installer.Uninstall(pkg)
}

// InstallMany installs multiple packages, returning errors for each failure
func (m *Manager) InstallMany(packages brewfile.Packages, onProgress func(pkg brewfile.Package, i, total int, err error)) error {
	return m.InstallManyWithOutput(packages, onProgress, nil)
}

// InstallManyWithOutput installs multiple packages with progress and output streaming
func (m *Manager) InstallManyWithOutput(
	packages brewfile.Packages,
	onProgress func(pkg brewfile.Package, i, total int, err error),
	onOutput func(pkg brewfile.Package, line string),
) error {
	var lastErr error
	total := len(packages)

	for i, pkg := range packages {
		var err error

		// If output callback is provided, use streaming install
		if onOutput != nil {
			err = m.InstallWithProgress(pkg, func(line string) {
				onOutput(pkg, line)
			})
		} else {
			err = m.Install(pkg)
		}

		if onProgress != nil {
			onProgress(pkg, i+1, total, err)
		}
		if err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// UninstallMany removes multiple packages
func (m *Manager) UninstallMany(packages brewfile.Packages, onProgress func(pkg brewfile.Package, i, total int, err error)) error {
	var lastErr error
	total := len(packages)

	for i, pkg := range packages {
		err := m.Uninstall(pkg)
		if onProgress != nil {
			onProgress(pkg, i+1, total, err)
		}
		if err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// ListAll returns all installed packages from all available installers
func (m *Manager) ListAll() (brewfile.Packages, error) {
	var all brewfile.Packages

	// Brew handles tap, brew, cask
	if m.brew.IsAvailable() {
		pkgs, err := m.brew.ListAll()
		if err != nil {
			return nil, fmt.Errorf("brew list failed: %w", err)
		}
		all = append(all, pkgs...)
	}

	// VSCode
	if m.vscode.IsAvailable() {
		pkgs, err := m.vscode.List()
		if err != nil {
			return nil, fmt.Errorf("vscode list failed: %w", err)
		}
		all = append(all, pkgs...)
	}

	// Cursor
	if m.cursor.IsAvailable() {
		pkgs, err := m.cursor.List()
		if err != nil {
			return nil, fmt.Errorf("cursor list failed: %w", err)
		}
		all = append(all, pkgs...)
	}

	// Go tools
	if m.go_.IsAvailable() {
		pkgs, err := m.go_.List()
		if err != nil {
			return nil, fmt.Errorf("go tools list failed: %w", err)
		}
		all = append(all, pkgs...)
	}

	// MAS
	if m.mas.IsAvailable() {
		pkgs, err := m.mas.List()
		if err != nil {
			return nil, fmt.Errorf("mas list failed: %w", err)
		}
		all = append(all, pkgs...)
	}

	return all, nil
}

// IsAvailable checks if the installer for a package type is available
func (m *Manager) IsAvailable(pkgType brewfile.PackageType) bool {
	installer, err := m.getInstaller(pkgType)
	if err != nil {
		return false
	}
	return installer.IsAvailable()
}

// getInstaller returns the appropriate installer for a package type
func (m *Manager) getInstaller(pkgType brewfile.PackageType) (Installer, error) {
	switch pkgType {
	case brewfile.TypeTap, brewfile.TypeBrew, brewfile.TypeCask:
		return m.brew, nil
	case brewfile.TypeVSCode:
		return m.vscode, nil
	case brewfile.TypeCursor:
		return m.cursor, nil
	case brewfile.TypeAntigravity:
		return m.antigravity, nil
	case brewfile.TypeMas:
		return m.mas, nil
	case brewfile.TypeGo:
		return m.go_, nil
	default:
		return nil, fmt.Errorf("unknown package type: %s", pkgType)
	}
}

// AvailableInstallers returns a map of installer types to availability
func (m *Manager) AvailableInstallers() map[string]bool {
	return map[string]bool{
		"brew":        m.brew.IsAvailable(),
		"vscode":      m.vscode.IsAvailable(),
		"cursor":      m.cursor.IsAvailable(),
		"antigravity": m.antigravity.IsAvailable(),
		"mas":         m.mas.IsAvailable(),
		"go":          m.go_.IsAvailable(),
	}
}
