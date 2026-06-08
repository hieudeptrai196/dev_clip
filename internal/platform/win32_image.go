//go:build windows

package platform

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// readDIBAsPNG reads CF_DIB from the (already open) clipboard and encodes PNG.
func readDIBAsPNG() ([]byte, error) {
	h, _, _ := syscall.SyscallN(procGetClipboardData.Addr(), cfDIB)
	if h == 0 {
		return nil, errors.New("no DIB data")
	}
	rawPtr, _, _ := syscall.SyscallN(procGlobalLock.Addr(), h)
	if rawPtr == 0 {
		return nil, errors.New("GlobalLock failed on DIB")
	}
	defer procGlobalUnlock.Call(h) //nolint:errcheck

	img, err := dibToImage(win32PtrToUnsafe(rawPtr))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// dibToImage parses a BITMAPINFOHEADER + pixel data into an image.Image.
// Supports the common 24/32-bit bottom-up cases used by screenshots.
func dibToImage(ptr unsafe.Pointer) (image.Image, error) {
	type bitmapInfoHeader struct {
		Size          uint32
		Width         int32
		Height        int32
		Planes        uint16
		BitCount      uint16
		Compression   uint32
		SizeImage     uint32
		XPelsPerMeter int32
		YPelsPerMeter int32
		ClrUsed       uint32
		ClrImportant  uint32
	}
	bih := (*bitmapInfoHeader)(ptr)
	if bih.BitCount != 24 && bih.BitCount != 32 {
		return nil, errors.New("unsupported DIB bit depth")
	}
	w := int(bih.Width)
	h := int(bih.Height)
	bottomUp := h > 0
	if h < 0 {
		h = -h
	}
	bytesPP := int(bih.BitCount) / 8
	stride := ((w*bytesPP + 3) / 4) * 4
	// Advance pointer past the header (unsafe.Add is rule-3-safe: starting
	// from an existing unsafe.Pointer).
	pixPtr := unsafe.Add(ptr, bih.Size)
	pixels := unsafe.Slice((*byte)(pixPtr), stride*h)

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		srcY := y
		if bottomUp {
			srcY = h - 1 - y
		}
		for x := 0; x < w; x++ {
			i := srcY*stride + x*bytesPP
			b, g, r := pixels[i], pixels[i+1], pixels[i+2]
			a := byte(255)
			if bytesPP == 4 {
				a = pixels[i+3]
				if a == 0 {
					a = 255 // many DIBs leave alpha zeroed
				}
			}
			o := img.PixOffset(x, y)
			img.Pix[o+0] = r
			img.Pix[o+1] = g
			img.Pix[o+2] = b
			img.Pix[o+3] = a
		}
	}
	return img, nil
}

// writePNGAsDIB decodes PNG and writes a CF_DIB to the (already open) clipboard.
func writePNGAsDIB(pngBytes []byte) error {
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	stride := w * 4
	dibSize := 40 + stride*h

	hMem, _, _ := syscall.SyscallN(procGlobalAlloc.Addr(), ghMemMoveable, uintptr(dibSize))
	if hMem == 0 {
		return errors.New("GlobalAlloc failed for DIB")
	}
	rawPtr, _, _ := syscall.SyscallN(procGlobalLock.Addr(), hMem)
	header := unsafe.Slice((*byte)(win32PtrToUnsafe(rawPtr)), dibSize)

	putU32(header[0:], 40)
	putI32(header[4:], int32(w))
	putI32(header[8:], int32(h)) // positive => bottom-up
	putU16(header[12:], 1)
	putU16(header[14:], 32)
	// remaining header fields left zero (BI_RGB)

	off := 40
	for y := 0; y < h; y++ {
		srcY := h - 1 - y // bottom-up
		for x := 0; x < w; x++ {
			r, g, bb, a := img.At(b.Min.X+x, b.Min.Y+srcY).RGBA()
			header[off+0] = byte(bb >> 8)
			header[off+1] = byte(g >> 8)
			header[off+2] = byte(r >> 8)
			header[off+3] = byte(a >> 8)
			off += 4
		}
	}
	procGlobalUnlock.Call(hMem)
	if r, _, err := procSetClipboardData.Call(cfDIB, hMem); r == 0 {
		return err
	}
	return nil
}

func putU16(b []byte, v uint16) { b[0] = byte(v); b[1] = byte(v >> 8) }
func putU32(b []byte, v uint32) {
	b[0] = byte(v); b[1] = byte(v >> 8); b[2] = byte(v >> 16); b[3] = byte(v >> 24)
}
func putI32(b []byte, v int32) { putU32(b, uint32(v)) }

var _ = windows.MAX_PATH // keep windows import used
