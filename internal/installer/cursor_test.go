package installer

import (
	"testing"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/stretchr/testify/assert"
)

func TestNewCursorInstaller(t *testing.T) {
	inst := NewCursorInstaller()
	assert.NotNil(t, inst)
	assert.NotNil(t, inst.runner)
	assert.Equal(t, "cursor", inst.command)
}

func TestCursorInstaller_IsAvailable(t *testing.T) {
	inst := NewCursorInstaller()
	// Just verify it doesn't panic
	_ = inst.IsAvailable()
}

func TestCursorInstaller_List(t *testing.T) {
	inst := NewCursorInstaller()
	if !inst.IsAvailable() {
		t.Skip("Cursor CLI not available")
	}

	extensions, err := inst.List()
	assert.NoError(t, err)

	// Verify all returned packages are cursor type
	for _, ext := range extensions {
		assert.Equal(t, brewfile.TypeCursor, ext.Type)
		assert.NotEmpty(t, ext.Name)
	}
}

func TestCursorInstaller_Install_Uninstall(t *testing.T) {
	t.Skip("Skipping install/uninstall tests to avoid system modification")
}
