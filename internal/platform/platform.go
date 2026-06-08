package platform

// RawClipKind is the raw payload kind read from / written to the OS clipboard.
type RawClipKind int

const (
	ClipText RawClipKind = iota
	ClipImage
)

// RawClip is platform-neutral clipboard content.
type RawClip struct {
	Kind  RawClipKind
	Text  string
	Image []byte // PNG bytes when Kind == ClipImage
}

// AppInfo identifies the foreground application.
// Windows fills ExePath; macOS (later) fills BundleID.
type AppInfo struct {
	ExePath  string
	BundleID string
}

// WindowRef is an opaque handle to the paste-target window (HWND on Windows).
type WindowRef uintptr

// HotkeySpec describes a global hotkey to register.
type HotkeySpec struct {
	ID        int
	Modifiers uint // platform-specific modifier bitmask
	KeyCode   uint // virtual key code
}

// PlatformEvents receives callbacks from the platform layer.
type PlatformEvents interface {
	OnClipboardChange()
	OnHotkey(id int)
	OnShowSettings()  // tray "Settings" clicked
	OnQuitRequested() // tray "Quit" clicked
}

// EventsFunc is a func-based adapter implementing PlatformEvents.
type EventsFunc struct {
	ClipboardChange func()
	Hotkey          func(id int)
	ShowSettings    func()
	QuitRequested   func()
}

func (e EventsFunc) OnClipboardChange() {
	if e.ClipboardChange != nil {
		e.ClipboardChange()
	}
}
func (e EventsFunc) OnHotkey(id int) {
	if e.Hotkey != nil {
		e.Hotkey(id)
	}
}
func (e EventsFunc) OnShowSettings() {
	if e.ShowSettings != nil {
		e.ShowSettings()
	}
}
func (e EventsFunc) OnQuitRequested() {
	if e.QuitRequested != nil {
		e.QuitRequested()
	}
}

// Platform is the single boundary to OS-specific clipboard/input APIs.
type Platform interface {
	Start(ev PlatformEvents) error
	Stop()
	ReadClipboard() (*RawClip, error)
	WriteClipboard(*RawClip) error
	ForegroundApp() (AppInfo, error)
	CaptureForegroundWindow() WindowRef // remember paste target before showing popup
	SimulatePaste(target WindowRef) error
	CursorPos() (x, y int)
	RegisterHotkey(spec HotkeySpec) error
	UpdateHotkey(mod, key uint) error // re-register the paste hotkey with new combo
}
