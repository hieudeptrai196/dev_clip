package platform

import "sync"

// FakePlatform is an in-memory Platform for tests on any OS.
type FakePlatform struct {
	mu         sync.Mutex
	events     PlatformEvents
	clip       *RawClip
	fgApp      AppInfo
	fgWindow   WindowRef
	pasteCalls []WindowRef
	cursorX    int
	cursorY    int
	hotkeyMod  uint
	hotkeyKey  uint
}

func NewFakePlatform() *FakePlatform { return &FakePlatform{} }

func (f *FakePlatform) Start(ev PlatformEvents) error { f.events = ev; return nil }
func (f *FakePlatform) Stop()                         {}

func (f *FakePlatform) ReadClipboard() (*RawClip, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.clip, nil
}
func (f *FakePlatform) WriteClipboard(c *RawClip) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clip = c
	return nil
}
func (f *FakePlatform) ForegroundApp() (AppInfo, error)    { return f.fgApp, nil }
func (f *FakePlatform) CaptureForegroundWindow() WindowRef { return f.fgWindow }
func (f *FakePlatform) SimulatePaste(t WindowRef) error {
	f.pasteCalls = append(f.pasteCalls, t)
	return nil
}
func (f *FakePlatform) CursorPos() (int, int)           { return f.cursorX, f.cursorY }
func (f *FakePlatform) RegisterHotkey(HotkeySpec) error { return nil }
func (f *FakePlatform) UpdateHotkey(mod, key uint) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hotkeyMod = mod
	f.hotkeyKey = key
	return nil
}

// --- test helpers ---
func (f *FakePlatform) SetClipboard(c *RawClip)          { f.WriteClipboard(c) }
func (f *FakePlatform) SetForegroundApp(a AppInfo)        { f.fgApp = a }
func (f *FakePlatform) SetForegroundWindow(w WindowRef)   { f.fgWindow = w }
func (f *FakePlatform) EmitClipboardChange()              { f.events.OnClipboardChange() }
func (f *FakePlatform) EmitHotkey(id int)                 { f.events.OnHotkey(id) }
func (f *FakePlatform) PasteCalls() []WindowRef           { return f.pasteCalls }
func (f *FakePlatform) EmitShowSettings()                 { f.events.OnShowSettings() }
func (f *FakePlatform) EmitQuitRequested()                { f.events.OnQuitRequested() }
