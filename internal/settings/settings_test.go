package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()

	assert.Equal(t, uint(1), s.HotkeyMod, "HotkeyMod should be MOD_ALT (1)")
	assert.Equal(t, uint(0x56), s.HotkeyKey, "HotkeyKey should be 'V' (0x56)")
	assert.Equal(t, "Alt+V", s.HotkeyLabel)
	assert.Equal(t, []string{"1Password.exe", "Bitwarden.exe", "KeePass.exe", "KeePassXC.exe"}, s.Blocklist)
	assert.Equal(t, 100, s.MaxItems)
	assert.False(t, s.AutoStart)
}

func TestLoadNonExistent(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does", "not", "exist.json"))
	require.NoError(t, err)

	assert.Empty(t, cfg.Snippets)
	assert.Equal(t, DefaultSettings(), cfg.Settings)
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	original := &Config{
		Snippets: []RawSnippet{
			{Name: "greet", Content: "Hello {{name}}"},
			{Name: "bye", Content: "Goodbye"},
		},
		Settings: Settings{
			HotkeyMod:   2, // MOD_CTRL
			HotkeyKey:   0x43,
			HotkeyLabel: "Ctrl+C",
			Blocklist:   []string{"secret.exe"},
			MaxItems:    200,
			AutoStart:   true,
		},
	}

	require.NoError(t, Save(path, original))

	loaded, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, original.Snippets, loaded.Snippets)
	assert.Equal(t, original.Settings, loaded.Settings)
}

func TestLoadBackwardCompat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Legacy config: only snippets, no settings field at all.
	legacy := `{
  "snippets": [
    {"name": "sig", "content": "-- Alice"}
  ]
}`
	require.NoError(t, os.WriteFile(path, []byte(legacy), 0644))

	cfg, err := Load(path)
	require.NoError(t, err)

	// Snippets must be preserved.
	require.Len(t, cfg.Snippets, 1)
	assert.Equal(t, "sig", cfg.Snippets[0].Name)
	assert.Equal(t, "-- Alice", cfg.Snippets[0].Content)

	// Settings should be filled in with defaults.
	assert.Equal(t, DefaultSettings(), cfg.Settings)
}

func TestLoadClampsMaxItems(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected int
	}{
		{
			name:     "zero becomes default (100)",
			json:     `{"snippets":[],"settings":{"max_items":0}}`,
			expected: 100, // zero → mergeDefaults sets 100, then clamp is a no-op
		},
		{
			name:     "below minimum clamped to 10",
			json:     `{"snippets":[],"settings":{"max_items":5}}`,
			expected: 10,
		},
		{
			name:     "above maximum clamped to 500",
			json:     `{"snippets":[],"settings":{"max_items":1000}}`,
			expected: 500,
		},
		{
			name:     "valid value unchanged",
			json:     `{"snippets":[],"settings":{"max_items":250}}`,
			expected: 250,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			require.NoError(t, os.WriteFile(path, []byte(tc.json), 0644))

			cfg, err := Load(path)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, cfg.Settings.MaxItems)
		})
	}
}
