package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakePlatformDeliversClipboardEvent(t *testing.T) {
	f := NewFakePlatform()
	var changes int
	events := EventsFunc{
		ClipboardChange: func() { changes++ },
		Hotkey:          func(id int) {},
	}
	require.NoError(t, f.Start(events))

	f.SetClipboard(&RawClip{Kind: ClipText, Text: "hello"})
	f.EmitClipboardChange()

	assert.Equal(t, 1, changes)
	got, err := f.ReadClipboard()
	require.NoError(t, err)
	assert.Equal(t, "hello", got.Text)
}

func TestFakePlatformHotkeyAndForegroundApp(t *testing.T) {
	f := NewFakePlatform()
	var firedID int
	require.NoError(t, f.Start(EventsFunc{Hotkey: func(id int) { firedID = id }}))
	f.SetForegroundApp(AppInfo{ExePath: `C:\x\1Password.exe`})
	f.EmitHotkey(42)

	assert.Equal(t, 42, firedID)
	app, err := f.ForegroundApp()
	require.NoError(t, err)
	assert.Equal(t, `C:\x\1Password.exe`, app.ExePath)
}
