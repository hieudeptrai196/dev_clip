package engine

import (
	"fmt"
	"sync"
	"time"

	"devclip/internal/clip"
	"devclip/internal/format"
	"devclip/internal/platform"
	"devclip/internal/security"
)

const HotkeyPasteID = 1

type Config struct {
	Platform     platform.Platform
	Store        *clip.ClipStore
	Filter       *security.SecurityFilter
	Capacity     int
	RestoreDelay time.Duration
}

type Engine struct {
	cfg           Config
	mu            sync.Mutex
	pasteTgt      platform.WindowRef
	onChange      func()
	onHotkey      func()
	lastSelfWrite uint64 // hash we wrote ourselves, to ignore on next change
}

func New(cfg Config) *Engine {
	if cfg.RestoreDelay == 0 {
		cfg.RestoreDelay = 200 * time.Millisecond
	}
	return &Engine{cfg: cfg}
}

func (e *Engine) Start() error {
	return e.cfg.Platform.Start(platform.EventsFunc{
		ClipboardChange: e.handleClipboardChange,
		Hotkey:          e.handleHotkey,
	})
}

func (e *Engine) OnChange(fn func()) { e.onChange = fn }
func (e *Engine) OnHotkey(fn func()) { e.onHotkey = fn }

func (e *Engine) handleClipboardChange() {
	// Drop captures coming from blocked apps.
	if app, err := e.cfg.Platform.ForegroundApp(); err == nil {
		if e.cfg.Filter.IsBlocked(app.ExePath) {
			return
		}
	}

	raw, err := e.cfg.Platform.ReadClipboard()
	if err != nil || raw == nil {
		return
	}

	var item *clip.ClipItem
	switch raw.Kind {
	case platform.ClipText:
		if raw.Text == "" {
			return
		}
		fmtLabel := "plain"
		switch format.Detect(raw.Text) {
		case format.JSON:
			fmtLabel = "json"
		case format.SQL:
			fmtLabel = "sql"
		}
		item = &clip.ClipItem{
			Kind:    clip.KindText,
			Text:    raw.Text,
			Preview: preview(raw.Text),
			Hash:    clip.HashText(raw.Text),
			Format:  fmtLabel,
		}
	case platform.ClipImage:
		if len(raw.Image) == 0 {
			return
		}
		item = &clip.ClipItem{
			Kind:    clip.KindImage,
			Image:   raw.Image,
			Preview: fmt.Sprintf("image (%d bytes)", len(raw.Image)),
			Hash:    clip.HashBytes(raw.Image),
		}
	default:
		return
	}

	// Ignore the change we caused ourselves during paste-restore.
	e.mu.Lock()
	if item.Hash == e.lastSelfWrite {
		e.lastSelfWrite = 0
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()

	e.cfg.Store.Push(item)
	if e.onChange != nil {
		e.onChange()
	}
}

func (e *Engine) handleHotkey(id int) {
	if id != HotkeyPasteID {
		return
	}
	e.mu.Lock()
	e.pasteTgt = e.cfg.Platform.CaptureForegroundWindow()
	e.mu.Unlock()
	if e.onHotkey != nil {
		e.onHotkey()
	}
}

func (e *Engine) PasteTarget() platform.WindowRef {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.pasteTgt
}

// PasteItem writes the given history item to the clipboard, simulates paste
// into the remembered target window, then restores the previous clipboard.
func (e *Engine) PasteItem(id uint64) error {
	it := e.cfg.Store.Get(id)
	if it == nil {
		return fmt.Errorf("item %d not found", id)
	}

	prev, _ := e.cfg.Platform.ReadClipboard() // save current clipboard to restore later

	out := &platform.RawClip{}
	switch it.Kind {
	case clip.KindText:
		out.Kind = platform.ClipText
		out.Text = it.Text
		e.mu.Lock()
		e.lastSelfWrite = clip.HashText(it.Text)
		e.mu.Unlock()
	case clip.KindImage:
		out.Kind = platform.ClipImage
		out.Image = it.Image
	}
	if err := e.cfg.Platform.WriteClipboard(out); err != nil {
		return err
	}

	e.mu.Lock()
	target := e.pasteTgt
	e.mu.Unlock()
	if err := e.cfg.Platform.SimulatePaste(target); err != nil {
		return err
	}

	if prev != nil {
		go func() {
			time.Sleep(e.cfg.RestoreDelay)
			_ = e.cfg.Platform.WriteClipboard(prev)
		}()
	}
	return nil
}

// PasteText pastes an arbitrary string into the captured target window.
func (e *Engine) PasteText(text string) error { return e.pasteText(text) }

// pasteText writes text to the clipboard, simulates paste, then restores the
// previous clipboard contents after RestoreDelay.
func (e *Engine) pasteText(text string) error {
	prev, _ := e.cfg.Platform.ReadClipboard()
	e.mu.Lock()
	e.lastSelfWrite = clip.HashText(text)
	e.mu.Unlock()
	if err := e.cfg.Platform.WriteClipboard(&platform.RawClip{Kind: platform.ClipText, Text: text}); err != nil {
		return err
	}
	e.mu.Lock()
	target := e.pasteTgt
	e.mu.Unlock()
	if err := e.cfg.Platform.SimulatePaste(target); err != nil {
		return err
	}
	if prev != nil {
		go func() {
			time.Sleep(e.cfg.RestoreDelay)
			_ = e.cfg.Platform.WriteClipboard(prev)
		}()
	}
	return nil
}

// FormatItem returns a pretty-printed string for the given text item:
// indented JSON for "json" items, formatted SQL for "sql" items, or the
// original text for "plain" items. Returns "" if the item is not found or is
// an image item.
func (e *Engine) FormatItem(id uint64) string {
	it := e.cfg.Store.Get(id)
	if it == nil || it.Kind != clip.KindText {
		return ""
	}
	switch it.Format {
	case "json":
		s, err := format.PrettyJSON(it.Text)
		if err != nil {
			return it.Text
		}
		return s
	case "sql":
		return format.FormatSQL(it.Text)
	default:
		return it.Text
	}
}

// PasteTransformed transforms the item's text using format.Transform then
// pastes it via pasteText. Returns an error if the item is not found or is an
// image.
func (e *Engine) PasteTransformed(id uint64, op string) error {
	it := e.cfg.Store.Get(id)
	if it == nil {
		return fmt.Errorf("item %d not found", id)
	}
	if it.Kind != clip.KindText {
		return fmt.Errorf("item %d is not a text item", id)
	}
	return e.pasteText(format.Transform(it.Text, op))
}

// PasteFormatted pastes the pretty-printed version of the given text item.
// Returns an error if the item is not found or is an image.
func (e *Engine) PasteFormatted(id uint64) error {
	s := e.FormatItem(id)
	if s == "" {
		it := e.cfg.Store.Get(id)
		if it == nil {
			return fmt.Errorf("item %d not found", id)
		}
		return fmt.Errorf("item %d is not a text item", id)
	}
	return e.pasteText(s)
}

func (e *Engine) History() []*clip.ClipItem { return e.cfg.Store.List() }
func (e *Engine) Clear()                    { e.cfg.Store.Clear() }

// ItemImagePNG returns the PNG bytes of an image item, or nil if not an image.
func (e *Engine) ItemImagePNG(id uint64) []byte {
	it := e.cfg.Store.Get(id)
	if it == nil {
		return nil
	}
	return it.Image
}

func preview(s string) string {
	const max = 120
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
