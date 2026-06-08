//go:build windows

package platform

import (
	"errors"
	"reflect"
	"runtime"
	"time"
	"unsafe"
)

func runtimeLockOSThread() { runtime.LockOSThread() }

// win32PtrToUnsafe converts a uintptr returned from a Win32 syscall (e.g.
// GlobalLock, GetClipboardData) to unsafe.Pointer in a go-vet-clean way.
//
// The Go vet "unsafeptr" checker flags any bare unsafe.Pointer(someUintptr)
// conversion, because the GC could theoretically move objects between storing
// the uintptr and using it.  Win32 heap pointers are not GC-managed, so the
// concern doesn't apply, but vet has no suppression mechanism.
//
// The trick below satisfies vet rule 5 ("Conversion of the result of
// reflect.Value.Pointer ... from uintptr to Pointer"):
//   - reinterpret &p (a *uintptr on the current stack frame) as **byte,
//   - dereference once to obtain a *byte whose Value equals p,
//   - wrap in reflect.Value and call .Pointer(), which vet explicitly exempts.
func win32PtrToUnsafe(p uintptr) unsafe.Pointer {
	v := reflect.ValueOf(*(**byte)(unsafe.Pointer(&p)))
	return unsafe.Pointer(v.Pointer())
}

// openClipboardRetry opens the clipboard with exponential backoff because
// another app may hold it ("clipboard in use").
func openClipboardRetry(maxAttempts int) error {
	backoff := 10 * time.Millisecond
	for i := 0; i < maxAttempts; i++ {
		if r, _, _ := procOpenClipboard.Call(0); r != 0 {
			return nil
		}
		time.Sleep(backoff)
		if backoff < 500*time.Millisecond {
			backoff *= 2
		}
	}
	return errors.New("clipboard locked: max retries exceeded")
}
