//go:build windows

package platform

import (
	_ "embed"
	"encoding/binary"
	"unsafe"
)

// trayIconICO is the app icon, embedded so the tray can build an HICON from it
// directly — independent of the exe's resource IDs (which LoadImage's
// resource lookup was failing to match, leaving the tray/hidden-icons blank).
//
//go:embed trayicon.ico
var trayIconICO []byte

// loadEmbeddedTrayIcon builds a small HICON from the embedded .ico, picking the
// frame closest to 16x16. CreateIconFromResourceEx decodes both PNG- and
// BMP-encoded frames, so this works regardless of how the icon was generated.
// Returns 0 on failure (caller falls back to the resource/file loader).
func loadEmbeddedTrayIcon() uintptr {
	const desired = 16
	ico := trayIconICO
	if len(ico) < 6 {
		return 0
	}
	count := int(binary.LittleEndian.Uint16(ico[4:6]))
	if count == 0 || len(ico) < 6+count*16 {
		return 0
	}

	bestOff, bestLen, bestScore := 0, 0, 1<<30
	for i := 0; i < count; i++ {
		e := 6 + i*16
		w := int(ico[e])
		if w == 0 {
			w = 256
		}
		size := int(binary.LittleEndian.Uint32(ico[e+8 : e+12]))
		off := int(binary.LittleEndian.Uint32(ico[e+12 : e+16]))
		if size == 0 || off < 0 || off+size > len(ico) {
			continue
		}
		// Prefer the frame nearest 16px, penalising frames smaller than desired
		// (upscaling a tiny frame looks worse than downscaling a larger one).
		score := w - desired
		if score < 0 {
			score = -score + 1000
		}
		if score < bestScore {
			bestScore, bestOff, bestLen = score, off, size
		}
	}
	if bestLen == 0 {
		return 0
	}

	h, _, _ := procCreateIconFromResourceEx.Call(
		uintptr(unsafe.Pointer(&ico[bestOff])),
		uintptr(bestLen),
		1,          // fIcon = TRUE
		0x00030000, // dwVersion (3.0)
		uintptr(desired),
		uintptr(desired),
		0, // LR_DEFAULTCOLOR
	)
	return h
}
