package installer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/asamgx/brewsync/internal/exec"
)

// GoToolsInstaller handles Go tools
type GoToolsInstaller struct {
	runner *exec.Runner
}

// NewGoToolsInstaller creates a new Go tools installer
func NewGoToolsInstaller() *GoToolsInstaller {
	return &GoToolsInstaller{
		runner: exec.Default,
	}
}

// List returns all installed Go tools from GOPATH/bin or GOBIN
func (g *GoToolsInstaller) List() (brewfile.Packages, error) {
	binDir := g.getBinDir()
	if binDir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(binDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var packages brewfile.Packages
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Skip non-executable files
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Mode()&0111 == 0 {
			continue
		}

		name := entry.Name()
		// Try to get the full module path from go version -m
		modulePath := g.getModulePath(filepath.Join(binDir, name))
		if modulePath != "" {
			packages = append(packages, brewfile.NewPackage(brewfile.TypeGo, modulePath))
		} else {
			// Fallback to just the binary name (not ideal but better than nothing)
			packages = append(packages, brewfile.NewPackage(brewfile.TypeGo, name))
		}
	}
	return packages, nil
}

// getBinDir returns the Go bin directory
func (g *GoToolsInstaller) getBinDir() string {
	// Check GOBIN first
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		return gobin
	}

	// Then GOPATH/bin
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		// Default GOPATH
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		gopath = filepath.Join(home, "go")
	}

	return filepath.Join(gopath, "bin")
}

// getModulePath tries to get the module path for a binary using go version -m
func (g *GoToolsInstaller) getModulePath(binPath string) string {
	output, err := g.runner.Run("go", "version", "-m", binPath)
	if err != nil {
		return ""
	}

	// Parse output to find the path line
	// Format: binary: path\n\tpath\tmodule/path\tversion\n...
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "path") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}

// Install installs a Go tool
func (g *GoToolsInstaller) Install(pkg brewfile.Package) error {
	// Add @latest if no version specified
	name := pkg.Name
	if !strings.Contains(name, "@") {
		name += "@latest"
	}
	_, err := g.runner.Run("go", "install", name)
	return err
}

// Uninstall removes a Go tool binary
func (g *GoToolsInstaller) Uninstall(pkg brewfile.Package) error {
	binDir := g.getBinDir()
	if binDir == "" {
		return nil
	}

	// Extract binary name from module path
	binName := filepath.Base(pkg.Name)
	binPath := filepath.Join(binDir, binName)

	return os.Remove(binPath)
}

// IsAvailable checks if go is available
func (g *GoToolsInstaller) IsAvailable() bool {
	return g.runner.Exists("go")
}
