package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devclip/internal/clip"
	"devclip/internal/engine"
	"devclip/internal/platform"
	"devclip/internal/security"
)

// App is the Wails-bound application surface.
type App struct {
	ctx context.Context
	eng *engine.Engine
}

func NewApp() *App { return &App{} }

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

// Hide hides the popup window (bound to JS).
func (a *App) Hide() { runtime.WindowHide(a.ctx) }
