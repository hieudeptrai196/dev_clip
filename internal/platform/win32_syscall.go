//go:build windows

package platform

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazyDLL("user32.dll")
	kernel32 = windows.NewLazyDLL("kernel32.dll")
	shell32  = windows.NewLazyDLL("shell32.dll")

	procMonitorFromPoint = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW  = user32.NewProc("GetMonitorInfoW")

	procAddClipboardFormatListener = user32.NewProc("AddClipboardFormatListener")
	procRegisterClassExW           = user32.NewProc("RegisterClassExW")
	procCreateWindowExW            = user32.NewProc("CreateWindowExW")
	procDefWindowProcW             = user32.NewProc("DefWindowProcW")
	procGetMessageW                = user32.NewProc("GetMessageW")
	procTranslateMessage           = user32.NewProc("TranslateMessage")
	procDispatchMessageW           = user32.NewProc("DispatchMessageW")
	procRegisterHotKey             = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey           = user32.NewProc("UnregisterHotKey")
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
	procAttachThreadInput          = user32.NewProc("AttachThreadInput")
	procSetFocus                   = user32.NewProc("SetFocus")
	procBringWindowToTop           = user32.NewProc("BringWindowToTop")
	procPostMessageW               = user32.NewProc("PostMessageW")
	procCreatePopupMenu            = user32.NewProc("CreatePopupMenu")
	procAppendMenuW                = user32.NewProc("AppendMenuW")
	procTrackPopupMenuEx           = user32.NewProc("TrackPopupMenuEx")
	procDestroyMenu                = user32.NewProc("DestroyMenu")
	procLoadImageW                 = user32.NewProc("LoadImageW")
	procCreateIconFromResourceEx   = user32.NewProc("CreateIconFromResourceEx")
	procGetCurrentThreadId         = kernel32.NewProc("GetCurrentThreadId")
	procGetModuleHandleW           = kernel32.NewProc("GetModuleHandleW")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procGlobalAlloc                = kernel32.NewProc("GlobalAlloc")
	procGlobalLock                 = kernel32.NewProc("GlobalLock")
	procGlobalUnlock               = kernel32.NewProc("GlobalUnlock")

	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
)

// Win32 constants used by the platform layer.
const (
	cfUnicodeText = 13
	cfDIB         = 8

	wmClipboardUpdate = 0x031D
	wmHotkey          = 0x0312
	wmCommand         = 0x0111
	wmDestroy         = 0x0002
	wmApp             = 0x8000
	wmLButtonDblClk   = 0x0203
	wmRButtonUp       = 0x0205

	// Custom messages posted to the hidden window.
	wmTrayIcon      = wmApp + 100 // Shell_NotifyIcon callback message
	wmUpdateHotkey  = wmApp + 101 // request hotkey re-registration

	modAlt     = 0x0001
	modControl = 0x0002
	modShift   = 0x0004

	ghMemMoveable = 0x0002

	inputKeyboard = 1
	keyEventKeyUp = 0x0002
	vkControl     = 0x11
	vkV           = 0x56
	vkOEM3        = 0xC0 // the `~ key (backtick / grave), left of the 1 key

	processQueryLimitedInformation = 0x1000

	// Shell_NotifyIcon operations
	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002

	// NOTIFYICONDATA flags
	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	// Tray context menu item IDs
	idmShow     = 1001
	idmSettings = 1002
	idmQuit     = 1003

	// AppendMenu flags
	mfString    = 0x00000000
	mfSeparator = 0x00000800

	// TrackPopupMenu flags
	tpmRightAlign = 0x0008
	tpmBottomAlign = 0x0020
	tpmLeftButton  = 0x0000

	// LoadImage constants
	imageIcon      = 1
	lrLoadFromFile = 0x0010
	lrDefaultSize  = 0x0040
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

type rect struct{ Left, Top, Right, Bottom int32 }

type monitorInfo struct {
	cbSize    uint32
	rcMonitor rect
	rcWork    rect
	dwFlags   uint32
}

// cursorMonitorWorkOrigin returns the top-left of the work area (screen minus
// taskbar) of the monitor under the physical cursor point, in physical pixels.
//
// Wails positions the window via SetWindowPos at (workRect.Left+x, workRect.Top+y)
// of the window's monitor. To land the popup exactly at the absolute cursor
// position we pass cursorPhysical MINUS this work-area origin, so the two
// offsets cancel out (when the cursor and window share a monitor). This also
// correctly handles taskbars docked on the top/left.
func cursorMonitorWorkOrigin(physX, physY int32) (int32, int32) {
	const monitorDefaultToNearest = 0x00000002
	// POINT is passed by value: packed as two 32-bit ints into one 64-bit arg.
	pt := uintptr(uint32(physX)) | uintptr(uint32(physY))<<32
	hmon, _, _ := procMonitorFromPoint.Call(pt, monitorDefaultToNearest)
	if hmon == 0 {
		return 0, 0
	}
	var mi monitorInfo
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	if r, _, _ := procGetMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(&mi))); r == 0 {
		return 0, 0
	}
	return mi.rcWork.Left, mi.rcWork.Top
}

// notifyIconData mirrors the NOTIFYICONDATAW struct (V2 — szTip is 128 chars).
type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
}

// shellNotifyIcon wraps Shell_NotifyIconW.
func shellNotifyIcon(op uint32, nid *notifyIconData) error {
	nid.CbSize = uint32(unsafe.Sizeof(*nid))
	r, _, err := procShellNotifyIconW.Call(uintptr(op), uintptr(unsafe.Pointer(nid)))
	if r == 0 {
		return err
	}
	return nil
}

// setTipText copies a Go string into the fixed-size szTip field of notifyIconData.
func setTipText(nid *notifyIconData, tip string) {
	u16, _ := windows.UTF16FromString(tip)
	copy(nid.SzTip[:], u16)
}
