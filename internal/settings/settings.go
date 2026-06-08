package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Settings holds user-configurable preferences for DevClip.
type Settings struct {
	HotkeyMod   uint     `json:"hotkey_mod"`   // modifier bitmask: MOD_ALT=1, MOD_CTRL=2, MOD_SHIFT=4
	HotkeyKey   uint     `json:"hotkey_key"`   // virtual key code (e.g. 0x56 = 'V')
	HotkeyLabel string   `json:"hotkey_label"` // human-readable label, e.g. "Alt+V"
	Blocklist   []string `json:"blocklist"`    // exe names to ignore, e.g. ["1Password.exe"]
	MaxItems    int      `json:"max_items"`    // ring buffer capacity (10-500, default 100)
	AutoStart   bool     `json:"auto_start"`   // whether to launch at Windows startup
}

// RawSnippet mirrors the JSON shape used by the existing snippet config.
type RawSnippet struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// Config is the top-level JSON structure for DevClip's config file.
// It combines the legacy snippets array with the new settings object.
type Config struct {
	Snippets []RawSnippet `json:"snippets"`
	Settings Settings     `json:"settings"`
}

// DefaultSettings returns a Settings populated with sensible defaults.
func DefaultSettings() Settings {
	return Settings{
		HotkeyMod:   1,    // MOD_ALT
		HotkeyKey:   0x56, // 'V'
		HotkeyLabel: "Alt+V",
		Blocklist:   []string{"1Password.exe", "Bitwarden.exe", "KeePass.exe", "KeePassXC.exe"},
		MaxItems:    100,
		AutoStart:   false,
	}
}

// Load reads the config file at path and returns a Config.
//
// If the file does not exist, a Config with empty snippets and DefaultSettings
// is returned (no error). If the file exists but contains no "settings" field,
// defaults are merged in so legacy configs keep working.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := &Config{
				Snippets: []RawSnippet{},
				Settings: DefaultSettings(),
			}
			return cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	mergeDefaults(&cfg)
	clampMaxItems(&cfg)

	return &cfg, nil
}

// mergeDefaults fills in default values for settings fields that were absent
// in the JSON (i.e. left at their zero value). This preserves backward
// compatibility with config files that only contain a "snippets" array.
func mergeDefaults(cfg *Config) {
	defaults := DefaultSettings()

	if cfg.Settings.HotkeyMod == 0 {
		cfg.Settings.HotkeyMod = defaults.HotkeyMod
	}
	if cfg.Settings.HotkeyKey == 0 {
		cfg.Settings.HotkeyKey = defaults.HotkeyKey
	}
	if cfg.Settings.HotkeyLabel == "" {
		cfg.Settings.HotkeyLabel = defaults.HotkeyLabel
	}
	if cfg.Settings.Blocklist == nil {
		cfg.Settings.Blocklist = defaults.Blocklist
	}
	if cfg.Settings.MaxItems == 0 {
		cfg.Settings.MaxItems = defaults.MaxItems
	}
	// AutoStart is a bool; false is a valid user choice, so we don't override it.
}

// clampMaxItems constrains MaxItems to the valid range [10, 500].
func clampMaxItems(cfg *Config) {
	if cfg.Settings.MaxItems < 10 {
		cfg.Settings.MaxItems = 10
	}
	if cfg.Settings.MaxItems > 500 {
		cfg.Settings.MaxItems = 500
	}
}

// Save writes cfg to path as pretty-printed JSON, creating parent directories
// as needed.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// ConfigPath returns the default config file path:
//
//	<UserConfigDir>/DevClip/config.json
//
// If the user config directory cannot be determined, it falls back to
// ./config.json in the current working directory.
func ConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(dir, "DevClip", "config.json")
}
