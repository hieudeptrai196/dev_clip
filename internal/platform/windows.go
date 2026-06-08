//go:build windows

package platform

import (
	"errors"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsPlatform struct {
	events PlatformEvents
	hwnd   windows.Handle
}

// NewPlatform returns the Windows implementation.
func NewPlatform() Platform { return &windowsPlatform{} }

func (p *windowsPlatform) Start(ev PlatformEvents) error {
	p.events = ev
	ready := make(chan error, 1)
	go p.messageLoop(ready)
	return <-ready
}

func (p *windowsPlatform) Stop() { /* message loop ends on process exit */ }

func (p *windowsPlatform) messageLoop(ready chan<- error) {
	// CRITICAL: the window + GetMessage must stay on one OS thread.
	runtimeLockOSThread()

	className, _ := windows.UTF16PtrFromString("DevClipHiddenWindow")

	// wndProc must have signature func(uintptr, uintptr, uintptr, uintptr) uintptr
	// for windows.NewCallback.
	wndProcFn := func(hwnd, message, wParam, lParam uintptr) uintptr {
		return p.wndProc(hwnd, message, wParam, lParam)
	}
	wndProc := windows.NewCallback(wndProcFn)

	type wndClassEx struct {
		Size       uint32
		Style      uint32
		WndProc    uintptr
		ClsExtra   int32
		WndExtra   int32
		Instance   windows.Handle
		Icon       windows.Handle
		Cursor     windows.Handle
		Background windows.Handle
		MenuName   *uint16
		ClassName  *uint16
		IconSm     windows.Handle
	}
	wc := wndClassEx{
		Style:     0,
		WndProc:   wndProc,
		ClassName: className,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	if r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		ready <- err
		return
	}

	hwnd, _, err := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(className)), 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	)
	if hwnd == 0 {
		ready <- err
		return
	}
	p.hwnd = windows.Handle(hwnd)

	if r, _, err := procAddClipboardFormatListener.Call(hwnd); r == 0 {
		ready <- err
		return
	}

	// Register Alt+V as the paste hotkey.
	// Ctrl+Shift+V: avoids the Alt menu mnemonics (e.g. VS Code 'View' menu).
	if r, _, err := procRegisterHotKey.Call(hwnd, uintptr(HotkeyPasteID), modControl|modShift, vkV); r == 0 {
		ready <- err
		return
	}

	ready <- nil

	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 { // 0 = WM_QUIT, -1 = error
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// HotkeyPasteID must match engine.HotkeyPasteID. Defined here to avoid an
// import cycle (platform must not import engine).
const HotkeyPasteID = 1

func (p *windowsPlatform) wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch message {
	case wmClipboardUpdate:
		if p.events != nil {
			p.events.OnClipboardChange()
		}
		return 0
	case wmHotkey:
		if p.events != nil {
			p.events.OnHotkey(int(wParam))
		}
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

func (p *windowsPlatform) ReadClipboard() (*RawClip, error) {
	if err := openClipboardRetry(6); err != nil {
		return nil, err
	}
	defer procCloseClipboard.Call()

	if avail, _, _ := procIsClipboardFormatAvailable.Call(cfUnicodeText); avail != 0 {
		h, _, _ := procGetClipboardData.Call(cfUnicodeText)
		if h == 0 {
			return nil, errors.New("GetClipboardData returned null")
		}
		rawPtr, _, _ := syscall.SyscallN(procGlobalLock.Addr(), h)
		if rawPtr == 0 {
			return nil, errors.New("GlobalLock failed")
		}
		text := windows.UTF16PtrToString((*uint16)(win32PtrToUnsafe(rawPtr)))
		procGlobalUnlock.Call(h)
		return &RawClip{Kind: ClipText, Text: text}, nil
	}

	if avail, _, _ := procIsClipboardFormatAvailable.Call(cfDIB); avail != 0 {
		png, err := readDIBAsPNG()
		if err != nil {
			return nil, err
		}
		return &RawClip{Kind: ClipImage, Image: png}, nil
	}

	return nil, nil
}

func (p *windowsPlatform) WriteClipboard(c *RawClip) error {
	if c == nil {
		return nil
	}
	if err := openClipboardRetry(6); err != nil {
		return err
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()

	switch c.Kind {
	case ClipText:
		u16, err := windows.UTF16FromString(c.Text)
		if err != nil {
			return err
		}
		size := uintptr(len(u16) * 2)
		hMem, _, _ := syscall.SyscallN(procGlobalAlloc.Addr(), ghMemMoveable, size)
		if hMem == 0 {
			return errors.New("GlobalAlloc failed")
		}
		rawDst, _, _ := syscall.SyscallN(procGlobalLock.Addr(), hMem)
		copyU16(win32PtrToUnsafe(rawDst), u16)
		procGlobalUnlock.Call(hMem)
		if r, _, err := procSetClipboardData.Call(cfUnicodeText, hMem); r == 0 {
			return err
		}
		return nil
	case ClipImage:
		return writePNGAsDIB(c.Image)
	}
	return nil
}

func (p *windowsPlatform) ForegroundApp() (AppInfo, error) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return AppInfo{}, nil
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return AppInfo{}, nil
	}
	h, err := windows.OpenProcess(processQueryLimitedInformation, false, pid)
	if err != nil {
		return AppInfo{}, nil // fail soft: treat as unknown app
	}
	defer windows.CloseHandle(h)

	buf := make([]uint16, windows.MAX_PATH)
	n := uint32(len(buf))
	r, _, _ := procQueryFullProcessImageNameW.Call(
		uintptr(h), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	if r == 0 {
		return AppInfo{}, nil
	}
	return AppInfo{ExePath: windows.UTF16ToString(buf[:n])}, nil
}

func (p *windowsPlatform) CaptureForegroundWindow() WindowRef {
	hwnd, _, _ := procGetForegroundWindow.Call()
	return WindowRef(hwnd)
}

func (p *windowsPlatform) SimulatePaste(target WindowRef) error {
	if target != 0 {
		focusTargetWindow(windows.Handle(target))
		time.Sleep(40 * time.Millisecond) // let focus settle
	}
	inputs := []keyboardInput{
		makeKey(vkControl, false),
		makeKey(vkV, false),
		makeKey(vkV, true),
		makeKey(vkControl, true),
	}
	n, _, err := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
	if int(n) != len(inputs) {
		return err
	}
	return nil
}

// focusTargetWindow reliably brings `target` to the foreground and gives it
// keyboard focus. A plain SetForegroundWindow call from another process is
// usually blocked by Windows' foreground lock, so we temporarily attach our
// input thread to the target's thread (AttachThreadInput) — the standard trick
// for restoring focus to the editor the user was typing in before pasting.
func focusTargetWindow(target windows.Handle) {
	ourThread, _, _ := procGetCurrentThreadId.Call()

	var targetPID uint32
	targetThread, _, _ := procGetWindowThreadProcessId.Call(
		uintptr(target), uintptr(unsafe.Pointer(&targetPID)))

	if targetThread != 0 && targetThread != ourThread {
		procAttachThreadInput.Call(ourThread, targetThread, 1) // attach
		procBringWindowToTop.Call(uintptr(target))
		procSetForegroundWindow.Call(uintptr(target))
		procSetFocus.Call(uintptr(target))
		procAttachThreadInput.Call(ourThread, targetThread, 0) // detach
	} else {
		procSetForegroundWindow.Call(uintptr(target))
	}
}

func (p *windowsPlatform) CursorPos() (int, int) {
	x, y := getCursorPos() // physical pixels
	scale := dpiScaleAtCursor(int32(x), int32(y))
	if scale <= 0 {
		scale = 1.0
	}
	// Convert to logical (DIP) coordinates for Wails WindowSetPosition.
	return int(float64(x) / scale), int(float64(y) / scale)
}

func (p *windowsPlatform) RegisterHotkey(spec HotkeySpec) error {
	if r, _, err := procRegisterHotKey.Call(
		uintptr(p.hwnd), uintptr(spec.ID), uintptr(spec.Modifiers), uintptr(spec.KeyCode),
	); r == 0 {
		return err
	}
	return nil
}

func makeKey(vk uint16, up bool) keyboardInput {
	var in keyboardInput
	in.Type = inputKeyboard
	in.Ki.Vk = vk
	if up {
		in.Ki.Flags = keyEventKeyUp
	}
	return in
}

func copyU16(dst unsafe.Pointer, src []uint16) {
	d := unsafe.Slice((*uint16)(dst), len(src))
	copy(d, src)
}
