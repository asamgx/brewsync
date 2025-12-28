package installer

import (
	"testing"

	"github.com/asamgx/brewsync/internal/brewfile"
	"github.com/stretchr/testify/assert"
)

func TestNewMasInstaller(t *testing.T) {
	inst := NewMasInstaller()
	assert.NotNil(t, inst)
	assert.NotNil(t, inst.runner)
}

func TestMasInstaller_IsAvailable(t *testing.T) {
	inst := NewMasInstaller()
	// Just verify it doesn't panic
	_ = inst.IsAvailable()
}

func TestMasInstaller_List(t *testing.T) {
	inst := NewMasInstaller()
	if !inst.IsAvailable() {
		t.Skip("mas CLI not available")
	}

	apps, err := inst.List()
	assert.NoError(t, err)

	// Verify all returned packages are mas type
	for _, app := range apps {
		assert.Equal(t, brewfile.TypeMas, app.Type)
		assert.NotEmpty(t, app.Name)
		// mas packages should have an id option
		assert.NotEmpty(t, app.Options["id"])
	}
}

func TestMasListPattern(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string // [id, name] or nil if no match
	}{
		{"497799835 Xcode (15.2)", []string{"497799835", "Xcode"}},
		{"1480933944 Vimari (2.2)", []string{"1480933944", "Vimari"}},
		{"899247664 TestFlight (3.3.0)", []string{"899247664", "TestFlight"}},
		{"1274495053 Microsoft To Do (2.99)", []string{"1274495053", "Microsoft To Do"}},
		{"invalid line", nil},
		{"", nil},
		{"123 NoVersion", nil},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			matches := masListPattern.FindStringSubmatch(tc.input)
			if tc.expected == nil {
				assert.Nil(t, matches)
			} else {
				assert.NotNil(t, matches)
				assert.Equal(t, tc.expected[0], matches[1]) // id
				assert.Equal(t, tc.expected[1], matches[2]) // name
			}
		})
	}
}

func TestMasInstaller_Install_Uninstall(t *testing.T) {
	t.Skip("Skipping install/uninstall tests to avoid system modification")
}

func TestMasInstaller_Uninstall_NotSupported(t *testing.T) {
	inst := NewMasInstaller()
	pkg := brewfile.NewPackage(brewfile.TypeMas, "123")

	// Uninstall should return nil (not supported)
	err := inst.Uninstall(pkg)
	assert.NoError(t, err)
}
