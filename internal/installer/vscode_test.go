package installer

import (
	"testing"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/stretchr/testify/assert"
)

func TestNewVSCodeInstaller(t *testing.T) {
	inst := NewVSCodeInstaller()
	assert.NotNil(t, inst)
	assert.NotNil(t, inst.runner)
	assert.Equal(t, "code", inst.command)
}

func TestVSCodeInstaller_IsAvailable(t *testing.T) {
	inst := NewVSCodeInstaller()
	// Just verify it doesn't panic
	_ = inst.IsAvailable()
}

func TestVSCodeInstaller_List(t *testing.T) {
	inst := NewVSCodeInstaller()
	if !inst.IsAvailable() {
		t.Skip("VSCode CLI not available")
	}

	extensions, err := inst.List()
	assert.NoError(t, err)

	// Verify all returned packages are vscode type
	for _, ext := range extensions {
		assert.Equal(t, brewfile.TypeVSCode, ext.Type)
		assert.NotEmpty(t, ext.Name)
	}
}

func TestVSCodeInstaller_Install_Uninstall(t *testing.T) {
	t.Skip("Skipping install/uninstall tests to avoid system modification")
}
