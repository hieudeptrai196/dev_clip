package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devclip/internal/clip"
	"devclip/internal/engine"
	"devclip/internal/platform"
	"devclip/internal/security"
	"devclip/internal/snippet"
)

// App is the Wails-bound application surface.
type App struct {
	ctx      context.Context
	eng      *engine.Engine
	snippets []snippet.Snippet
}

func NewApp() *App { return &App{} }

// snippetConfigPath returns <UserConfigDir>/DevClip/config.json,
// falling back to "./config.json" on error.
func snippetConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "DevClip", "config.json")
}

// sampleConfig is written on first run so the user has a template to edit.
var sampleConfig = map[string]interface{}{
	"snippets": []map[string]string{
		{"name": "Console log", "content": "console.log('{{label}}:', {{value}});"},
		{"name": "MIT header", "content": "// Copyright (c) {{year}} {{author}}. MIT License."},
	},
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.eng = engine.New(engine.Config{
		Platform: platform.NewPlatform(),
		Store:    clip.NewStore(100),
		Filter:   security.NewFilter(security.DefaultBlocklist()),
		Capacity: 100,
	})
	a.eng.OnChange(func() {
		runtime.EventsEmit(a.ctx, "clip:changed")
	})
	a.eng.OnHotkey(func() {
		x, y := platform.NewPlatform().CursorPos()
		runtime.WindowSetPosition(a.ctx, x, y)
		runtime.WindowShow(a.ctx)
		runtime.EventsEmit(a.ctx, "popup:show")
	})
	if err := a.eng.Start(); err != nil {
		runtime.LogError(a.ctx, "engine start failed: "+err.Error())
	}

	// Load snippets from config file, creating sample if missing.
	cfgPath := snippetConfigPath()
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(filepath.Dir(cfgPath), 0755); mkErr == nil {
			if data, jsonErr := json.MarshalIndent(sampleConfig, "", "  "); jsonErr == nil {
				_ = os.WriteFile(cfgPath, data, 0644)
			}
		}
	}
	var loadErr error
	a.snippets, loadErr = snippet.Load(cfgPath)
	if loadErr != nil {
		runtime.LogError(a.ctx, "snippet load failed: "+loadErr.Error())
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

// History returns the current clipboard history, newest first (bound to JS).
func (a *App) History() []*clip.ClipItem { return a.eng.History() }

// PasteItem writes the selected item to the clipboard and pastes it into the
// previously focused window (bound to JS).
func (a *App) PasteItem(id uint64) error { return a.eng.PasteItem(id) }

// CursorPos returns the current cursor position (bound to JS).
func (a *App) CursorPos() []int {
	x, y := platform.NewPlatform().CursorPos()
	return []int{x, y}
}

// Clear empties the history (bound to JS).
func (a *App) Clear() { a.eng.Clear() }

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
