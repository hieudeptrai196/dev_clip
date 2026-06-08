//go:build windows

package platform

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazyDLL("user32.dll")
	kernel32 = windows.NewLazyDLL("kernel32.dll")
	shcore   = windows.NewLazyDLL("shcore.dll")

	procMonitorFromPoint = user32.NewProc("MonitorFromPoint")
	procGetDpiForMonitor = shcore.NewProc("GetDpiForMonitor")

	procAddClipboardFormatListener = user32.NewProc("AddClipboardFormatListener")
	procRegisterClassExW           = user32.NewProc("RegisterClassExW")
	procCreateWindowExW            = user32.NewProc("CreateWindowExW")
	procDefWindowProcW             = user32.NewProc("DefWindowProcW")
	procGetMessageW                = user32.NewProc("GetMessageW")
	procTranslateMessage           = user32.NewProc("TranslateMessage")
	procDispatchMessageW           = user32.NewProc("DispatchMessageW")
	procRegisterHotKey             = user32.NewProc("RegisterHotKey")
	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procSetClipboardData           = user32.NewProc("SetClipboardData")
	procEmptyClipboard             = user32.NewProc("EmptyClipboard")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procGetForegroundWindow        = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow        = user32.NewProc("SetForegroundWindow")
	procGetWindowThreadProcessId   = user32.NewProc("GetWindowThreadProcessId")
	procGetCursorPos               = user32.NewProc("GetCursorPos")
	procSendInput                  = user32.NewProc("SendInput")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procGlobalAlloc                = kernel32.NewProc("GlobalAlloc")
	procGlobalLock                 = kernel32.NewProc("GlobalLock")
	procGlobalUnlock               = kernel32.NewProc("GlobalUnlock")
)

// Win32 constants used by the platform layer.
const (
	cfUnicodeText = 13
	cfDIB         = 8

	wmClipboardUpdate = 0x031D
	wmHotkey          = 0x0312

	modAlt = 0x0001

	ghMemMoveable = 0x0002

	inputKeyboard = 1
	keyEventKeyUp = 0x0002
	vkControl     = 0x11
	vkV           = 0x56

	processQueryLimitedInformation = 0x1000
)

type point struct{ X, Y int32 }

type msg struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

// keyboardInput mirrors the Win32 INPUT struct for keyboard events.
// On amd64 sizeof(INPUT) is 40 bytes: Type(4) + 4-byte align pad + a union
// sized to the largest member (MOUSEINPUT = 32 bytes). SendInput rejects the
// call (returns 0) unless cbSize == 40, so the struct MUST be 40 bytes with
// the KEYBDINPUT fields placed at the union offset (Ki starts at offset 8).
// Layout: Type(4) + pad(4) + Ki(24) + trailing pad(8) = 40 bytes.
type keyboardInput struct {
	Type uint32
	_    uint32 // align Ki (the union) onto the 8-byte boundary at offset 8
	Ki   struct {
		Vk        uint16
		Scan      uint16
		Flags     uint32
		Time      uint32
		ExtraInfo uintptr
	}
	_ [8]byte // pad so total size == sizeof(INPUT) (40) for SendInput's cbSize
}

// Compile-time guard: SendInput rejects the call unless cbSize == sizeof(INPUT)
// (40 on amd64). If keyboardInput is ever made smaller than 40 bytes, this line
// fails to compile (negative constant overflows uint) instead of failing silently
// at runtime with paste not working.
const _ uint = uint(unsafe.Sizeof(keyboardInput{})) - 40

func getCursorPos() (int, int) {
	var p point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return int(p.X), int(p.Y)
}

// dpiScaleAtCursor returns the effective DPI scale (1.0 == 100%) of the monitor
// under the given physical cursor point. Wails' WindowSetPosition expects
// logical (DIP) coordinates, while GetCursorPos returns physical pixels, so we
// must divide physical coords by this scale to place the popup at the cursor on
// scaled displays.
func dpiScaleAtCursor(physX, physY int32) float64 {
	const monitorDefaultToNearest = 0x00000002
	const mdtEffectiveDpi = 0
	// POINT is passed by value: packed as two 32-bit ints into one 64-bit arg.
	pt := uintptr(uint32(physX)) | uintptr(uint32(physY))<<32
	hmon, _, _ := procMonitorFromPoint.Call(pt, monitorDefaultToNearest)
	if hmon == 0 {
		return 1.0
	}
	var dpiX, dpiY uint32
	ret, _, _ := procGetDpiForMonitor.Call(
		hmon, mdtEffectiveDpi,
		uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY)),
	)
	if ret != 0 || dpiX == 0 { // S_OK == 0
		return 1.0
	}
	return float64(dpiX) / 96.0
}
