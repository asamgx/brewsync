package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/stretchr/testify/assert"
)

func TestNewGoToolsInstaller(t *testing.T) {
	inst := NewGoToolsInstaller()
	assert.NotNil(t, inst)
	assert.NotNil(t, inst.runner)
}

func TestGoToolsInstaller_IsAvailable(t *testing.T) {
	inst := NewGoToolsInstaller()
	// Should be true since we're running Go tests
	assert.True(t, inst.IsAvailable())
}

func TestGoToolsInstaller_getBinDir(t *testing.T) {
	inst := NewGoToolsInstaller()

	t.Run("with GOBIN set", func(t *testing.T) {
		origGoBin := os.Getenv("GOBIN")
		defer os.Setenv("GOBIN", origGoBin)

		os.Setenv("GOBIN", "/custom/gobin")
		binDir := inst.getBinDir()
		assert.Equal(t, "/custom/gobin", binDir)
	})

	t.Run("with GOPATH set", func(t *testing.T) {
		origGoBin := os.Getenv("GOBIN")
		origGoPath := os.Getenv("GOPATH")
		defer func() {
			os.Setenv("GOBIN", origGoBin)
			os.Setenv("GOPATH", origGoPath)
		}()

		os.Unsetenv("GOBIN")
		os.Setenv("GOPATH", "/custom/gopath")
		binDir := inst.getBinDir()
		assert.Equal(t, "/custom/gopath/bin", binDir)
	})

	t.Run("default GOPATH", func(t *testing.T) {
		origGoBin := os.Getenv("GOBIN")
		origGoPath := os.Getenv("GOPATH")
		defer func() {
			os.Setenv("GOBIN", origGoBin)
			os.Setenv("GOPATH", origGoPath)
		}()

		os.Unsetenv("GOBIN")
		os.Unsetenv("GOPATH")
		binDir := inst.getBinDir()

		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, "go", "bin")
		assert.Equal(t, expected, binDir)
	})
}

func TestGoToolsInstaller_List(t *testing.T) {
	inst := NewGoToolsInstaller()
	if !inst.IsAvailable() {
		t.Skip("Go not available")
	}

	tools, err := inst.List()
	// This might return empty if no tools installed, which is OK
	assert.NoError(t, err)

	// Verify all returned packages are go type
	for _, tool := range tools {
		assert.Equal(t, brewfile.TypeGo, tool.Type)
		assert.NotEmpty(t, tool.Name)
	}
}

func TestGoToolsInstaller_Install_Uninstall(t *testing.T) {
	t.Skip("Skipping install/uninstall tests to avoid system modification")
}

func TestGoToolsInstaller_getModulePath(t *testing.T) {
	inst := NewGoToolsInstaller()

	// Test with non-existent binary
	path := inst.getModulePath("/nonexistent/binary")
	assert.Empty(t, path)
}
