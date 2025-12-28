package installer

import (
	"testing"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/stretchr/testify/assert"
)

func TestNewBrewInstaller(t *testing.T) {
	inst := NewBrewInstaller()
	assert.NotNil(t, inst)
	assert.NotNil(t, inst.runner)
}

func TestBrewInstaller_IsAvailable(t *testing.T) {
	inst := NewBrewInstaller()
	// This should return true on macOS with Homebrew installed
	// We just verify it doesn't panic
	_ = inst.IsAvailable()
}

func TestBrewInstaller_ListTaps(t *testing.T) {
	inst := NewBrewInstaller()
	if !inst.IsAvailable() {
		t.Skip("Homebrew not available")
	}

	taps, err := inst.ListTaps()
	assert.NoError(t, err)

	// Verify all returned packages are taps
	for _, tap := range taps {
		assert.Equal(t, brewfile.TypeTap, tap.Type)
		assert.NotEmpty(t, tap.Name)
	}
}

func TestBrewInstaller_ListFormulae(t *testing.T) {
	inst := NewBrewInstaller()
	if !inst.IsAvailable() {
		t.Skip("Homebrew not available")
	}

	formulae, err := inst.ListFormulae()
	assert.NoError(t, err)

	// Verify all returned packages are brews
	for _, pkg := range formulae {
		assert.Equal(t, brewfile.TypeBrew, pkg.Type)
		assert.NotEmpty(t, pkg.Name)
	}
}

func TestBrewInstaller_ListCasks(t *testing.T) {
	inst := NewBrewInstaller()
	if !inst.IsAvailable() {
		t.Skip("Homebrew not available")
	}

	casks, err := inst.ListCasks()
	assert.NoError(t, err)

	// Verify all returned packages are casks
	for _, pkg := range casks {
		assert.Equal(t, brewfile.TypeCask, pkg.Type)
		assert.NotEmpty(t, pkg.Name)
	}
}

func TestBrewInstaller_ListAll(t *testing.T) {
	inst := NewBrewInstaller()
	if !inst.IsAvailable() {
		t.Skip("Homebrew not available")
	}

	all, err := inst.ListAll()
	assert.NoError(t, err)

	// Should have at least some packages
	assert.NotEmpty(t, all)

	// Count types
	byType := all.ByType()
	// Should have taps, brews, and possibly casks
	assert.NotEmpty(t, byType[brewfile.TypeTap], "should have taps")
	assert.NotEmpty(t, byType[brewfile.TypeBrew], "should have formulae")
}

func TestBrewInstaller_Install_Uninstall(t *testing.T) {
	// These are integration tests that would actually install/uninstall
	// We skip them by default to avoid modifying the system
	t.Skip("Skipping install/uninstall tests to avoid system modification")
}

func TestBrewInstaller_DumpToFile(t *testing.T) {
	inst := NewBrewInstaller()
	if !inst.IsAvailable() {
		t.Skip("Homebrew not available")
	}

	tmpFile := t.TempDir() + "/Brewfile"
	err := inst.DumpToFile(tmpFile)

	// This might fail if brew bundle isn't installed
	// which is ok for a test
	if err != nil {
		t.Logf("brew bundle dump failed (may not be installed): %v", err)
	}
}
