package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devclip/internal/clip"
	"devclip/internal/engine"
	"devclip/internal/platform"
	"devclip/internal/security"
	"devclip/internal/settings"
	"devclip/internal/snippet"
)

// App is the Wails-bound application surface.
type App struct {
	ctx      context.Context
	eng      *engine.Engine
	plat     platform.Platform
	snippets []snippet.Snippet
	cfg      *settings.Config
	cfgPath  string
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cfgPath = settings.ConfigPath()

	// Load unified config (settings + snippets).
	var err error
	a.cfg, err = settings.Load(a.cfgPath)
	if err != nil {
		runtime.LogError(a.ctx, "config load failed: "+err.Error())
		a.cfg = &settings.Config{Settings: settings.DefaultSettings()}
	}

	// Create sample config if no snippets exist and file is new.
	if len(a.cfg.Snippets) == 0 {
		a.cfg.Snippets = []settings.RawSnippet{
			{Name: "Console log", Content: "console.log('{{label}}:', {{value}});"},
			{Name: "MIT header", Content: "// Copyright (c) {{year}} {{author}}. MIT License."},
		}
		_ = settings.Save(a.cfgPath, a.cfg)
	}

	// Convert raw snippets to snippet.Snippet with IDs.
	a.snippets = make([]snippet.Snippet, 0, len(a.cfg.Snippets))
	for i, s := range a.cfg.Snippets {
		a.snippets = append(a.snippets, snippet.Snippet{
			ID:      uint64(i + 1),
			Name:    s.Name,
			Content: s.Content,
		})
	}

	a.plat = platform.NewPlatform()
	a.eng = engine.New(engine.Config{
		Platform: a.plat,
		Store:    clip.NewStore(a.cfg.Settings.MaxItems),
		Filter:   security.NewFilter(a.cfg.Settings.Blocklist),
		Capacity: a.cfg.Settings.MaxItems,
	})
	a.eng.OnChange(func() {
		runtime.EventsEmit(a.ctx, "clip:changed")
	})
	a.eng.OnHotkey(func() {
		x, y := a.plat.CursorPos()
		runtime.WindowSetPosition(a.ctx, x, y)
		runtime.WindowShow(a.ctx)
		runtime.EventsEmit(a.ctx, "popup:show")
	})
	a.eng.OnShowSettings(func() {
		x, y := a.plat.CursorPos()
		runtime.WindowSetPosition(a.ctx, x, y)
		runtime.WindowShow(a.ctx)
		runtime.EventsEmit(a.ctx, "settings:show")
	})
	a.eng.OnQuitRequested(func() {
		runtime.Quit(a.ctx)
	})

	if err := a.eng.Start(); err != nil {
		runtime.LogError(a.ctx, "engine start failed: "+err.Error())
	}

	// Warm up WebView2 so the FIRST Alt+V is not a cold start. Without this, the
	// first popup show races the webview load: JS event listeners aren't ready,
	// the "popup:show" grace period never applies, and the window flashes shut.
	// Showing the window far off-screen forces the webview + JS to initialize,
	// then we hide it again — invisible to the user.
	go func() {
		time.Sleep(300 * time.Millisecond)
		runtime.WindowSetPosition(a.ctx, -32000, -32000)
		runtime.WindowShow(a.ctx)
		time.Sleep(600 * time.Millisecond)
		runtime.WindowHide(a.ctx)
	}()
}

func (a *App) shutdown(ctx context.Context) {
	if a.plat != nil {
		a.plat.Stop()
	}
}

// History returns the current clipboard history, newest first (bound to JS).
func (a *App) History() []*clip.ClipItem { return a.eng.History() }

// PasteItem writes the selected item to the clipboard and pastes it into the
// previously focused window (bound to JS).
func (a *App) PasteItem(id uint64) error { return a.eng.PasteItem(id) }

// CursorPos returns the current cursor position (bound to JS).
func (a *App) CursorPos() []int {
	x, y := a.plat.CursorPos()
	return []int{x, y}
}

// Clear empties the history (bound to JS).
func (a *App) Clear() { a.eng.Clear() }

// TogglePin pins or unpins the item with the given id (bound to JS). Pinned
// items sort to the top and are not evicted by the history capacity.
func (a *App) TogglePin(id uint64) { a.eng.TogglePin(id) }

// ReorderPins sets the display order of pinned items to the given id sequence
// (bound to JS). Used by drag-and-drop reordering in the UI.
func (a *App) ReorderPins(ids []uint64) { a.eng.ReorderPins(ids) }

// Thumbnail returns a base64 PNG data URL for an image item (bound to JS),
// or an empty string if the item is not an image. Loaded lazily by the UI so
// History() stays small.
func (a *App) Thumbnail(id uint64) string {
	png := a.eng.ItemImagePNG(id)
	if len(png) == 0 {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

// FormatItem returns the pretty-printed text for a JSON or SQL item, or the
// original text for a plain item (bound to JS). Returns "" for image items.
func (a *App) FormatItem(id uint64) string { return a.eng.FormatItem(id) }

// PasteTransformed transforms the item text with the given op then pastes it
// (bound to JS). Valid ops: "upper","lower","camel","snake","kebab","base64encode".
func (a *App) PasteTransformed(id uint64, op string) error {
	return a.eng.PasteTransformed(id, op)
}

// PasteFormatted pastes the pretty-printed version of the item (bound to JS).
func (a *App) PasteFormatted(id uint64) error { return a.eng.PasteFormatted(id) }

// Hide hides the popup window (bound to JS).
func (a *App) Hide() { runtime.WindowHide(a.ctx) }

// Snippets returns the loaded snippet list (bound to JS).
func (a *App) Snippets() []snippet.Snippet { return a.snippets }

// SnippetPlaceholders returns the placeholder names for the given snippet (bound to JS).
func (a *App) SnippetPlaceholders(id uint64) []string {
	for _, s := range a.snippets {
		if s.ID == id {
			return snippet.Placeholders(s.Content)
		}
	}
	return nil
}

// PasteSnippet renders a snippet with the given values and pastes it (bound to JS).
func (a *App) PasteSnippet(id uint64, values map[string]string) error {
	for _, s := range a.snippets {
		if s.ID == id {
			return a.eng.PasteText(snippet.Render(s.Content, values))
		}
	}
	return fmt.Errorf("snippet %d not found", id)
}

// --- Settings-related bound methods ---

// GetSettings returns the current settings for the frontend (bound to JS).
func (a *App) GetSettings() settings.Settings {
	return a.cfg.Settings
}

// SaveSettings persists new settings and applies changes live (bound to JS).
func (a *App) SaveSettings(s settings.Settings) error {
	old := a.cfg.Settings

	// Validate max_items range.
	if s.MaxItems < 10 {
		s.MaxItems = 10
	} else if s.MaxItems > 500 {
		s.MaxItems = 500
	}

	// Apply hotkey change if needed.
	if s.HotkeyMod != old.HotkeyMod || s.HotkeyKey != old.HotkeyKey {
		if err := a.eng.UpdateHotkey(s.HotkeyMod, s.HotkeyKey); err != nil {
			return fmt.Errorf("hotkey update failed: %w", err)
		}
	}

	// Apply blocklist change if needed.
	if !stringSliceEqual(s.Blocklist, old.Blocklist) {
		a.eng.UpdateFilter(security.NewFilter(s.Blocklist))
	}

	// Apply auto-start change if needed.
	if s.AutoStart != old.AutoStart {
		if err := platform.SetAutoStart(s.AutoStart); err != nil {
			return fmt.Errorf("auto-start failed: %w", err)
		}
	}

	// Persist.
	a.cfg.Settings = s
	if err := settings.Save(a.cfgPath, a.cfg); err != nil {
		return fmt.Errorf("save config failed: %w", err)
	}

	return nil
}

// TestHotkey checks if a hotkey combo can be registered (bound to JS).
// Returns true if the hotkey is available.
func (a *App) TestHotkey(mod, key uint) bool {
	// We can't test without actually registering, so just validate inputs.
	return mod > 0 && key > 0
}

// GetBlocklist returns the current blocklist (bound to JS).
func (a *App) GetBlocklist() []string {
	return a.cfg.Settings.Blocklist
}

// AddToBlocklist adds an exe name to the blocklist and persists (bound to JS).
func (a *App) AddToBlocklist(exe string) error {
	if exe == "" {
		return fmt.Errorf("empty exe name")
	}
	// Check duplicate.
	for _, b := range a.cfg.Settings.Blocklist {
		if b == exe {
			return nil // already present
		}
	}
	a.cfg.Settings.Blocklist = append(a.cfg.Settings.Blocklist, exe)
	a.eng.UpdateFilter(security.NewFilter(a.cfg.Settings.Blocklist))
	return settings.Save(a.cfgPath, a.cfg)
}

// RemoveFromBlocklist removes an exe name from the blocklist and persists (bound to JS).
func (a *App) RemoveFromBlocklist(exe string) error {
	bl := a.cfg.Settings.Blocklist
	for i, b := range bl {
		if b == exe {
			a.cfg.Settings.Blocklist = append(bl[:i], bl[i+1:]...)
			a.eng.UpdateFilter(security.NewFilter(a.cfg.Settings.Blocklist))
			return settings.Save(a.cfgPath, a.cfg)
		}
	}
	return nil
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
