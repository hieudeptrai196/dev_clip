//go:build windows

package platform

import (
	"fmt"

	"golang.org/x/sys/windows"
)

const mutexName = "Global\\DevClipSingleInstance"

// EnsureSingleInstance creates a named mutex to prevent multiple instances.
// Returns a release function to call on shutdown, or an error if another
// instance is already running.
func EnsureSingleInstance() (release func(), err error) {
	name, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return nil, fmt.Errorf("utf16 conversion: %w", err)
	}

	h, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		// ERROR_ALREADY_EXISTS means another instance holds the mutex
		if h != 0 {
			windows.CloseHandle(h)
		}
		return nil, fmt.Errorf("DevClip is already running")
	}

	return func() { windows.CloseHandle(h) }, nil
}
