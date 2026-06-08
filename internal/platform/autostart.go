//go:build windows

package platform

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

const (
	autoStartRegKey  = `Software\Microsoft\Windows\CurrentVersion\Run`
	autoStartAppName = "DevClip"
)

// SetAutoStart enables or disables auto-start with Windows by creating or
// deleting a registry entry in HKCU\Software\Microsoft\Windows\CurrentVersion\Run.
func SetAutoStart(enable bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, autoStartRegKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if enable {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		return k.SetStringValue(autoStartAppName, exe)
	}

	// Ignore error if value doesn't exist
	_ = k.DeleteValue(autoStartAppName)
	return nil
}

// IsAutoStart checks if DevClip is configured to start with Windows.
func IsAutoStart() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, autoStartRegKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	_, _, err = k.GetStringValue(autoStartAppName)
	return err == nil
}
