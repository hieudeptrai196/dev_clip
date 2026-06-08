package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"devclip/internal/clip"
	"devclip/internal/platform"
	"devclip/internal/security"
)

func newTestEngine(t *testing.T) (*Engine, *platform.FakePlatform) {
	t.Helper()
	fp := platform.NewFakePlatform()
	e := New(Config{
		Platform: fp,
		Store:    clip.NewStore(10),
		Filter:   security.NewFilter([]string{"1Password.exe"}),
		Capacity: 10,
		RestoreDelay: 5 * time.Millisecond,
	})
	require.NoError(t, e.Start())
	return e, fp
}

func TestCopiedTextEntersStoreAndNotifiesUI(t *testing.T) {
	e, fp := newTestEngine(t)
	var notified int
	e.OnChange(func() { notified++ })

	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipText, Text: "hello"})
	fp.EmitClipboardChange()

	items := e.History()
	require.Len(t, items, 1)
	assert.Equal(t, "hello", items[0].Text)
	assert.GreaterOrEqual(t, notified, 1)
}

func TestCopyFromBlockedAppIsDropped(t *testing.T) {
	e, fp := newTestEngine(t)
	fp.SetForegroundApp(platform.AppInfo{ExePath: `C:\x\1Password.exe`})
	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipText, Text: "secret"})
	fp.EmitClipboardChange()

	assert.Empty(t, e.History(), "blocked app capture must not be stored")
}

func TestHotkeyCapturesPasteTarget(t *testing.T) {
	e, fp := newTestEngine(t)
	fp.SetForegroundWindow(platform.WindowRef(1234))
	fp.EmitHotkey(HotkeyPasteID)
	assert.Equal(t, platform.WindowRef(1234), e.PasteTarget())
}

func TestPasteItemWritesClipboardSimulatesAndRestores(t *testing.T) {
	e, fp := newTestEngine(t)

	// "snippet" is an older history item the user will paste.
	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipText, Text: "snippet"})
	fp.EmitClipboardChange()
	snippet := e.History()[0]

	// "original" is what the user currently has on the clipboard and must be
	// restored after the paste.
	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipText, Text: "original"})
	fp.EmitClipboardChange()

	fp.SetForegroundWindow(platform.WindowRef(99))
	fp.EmitHotkey(HotkeyPasteID)

	require.NoError(t, e.PasteItem(snippet.ID))

	// During paste the clipboard was set to the pasted item and paste was
	// simulated into the captured target window.
	calls := fp.PasteCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, platform.WindowRef(99), calls[0])

	// After the restore delay, the user's previous clipboard is restored.
	time.Sleep(20 * time.Millisecond)
	got, _ := fp.ReadClipboard()
	assert.Equal(t, "original", got.Text)
}

func TestImageCopyStoredWithPreviewLabel(t *testing.T) {
	e, fp := newTestEngine(t)
	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipImage, Image: []byte("PNGDATA")})
	fp.EmitClipboardChange()

	items := e.History()
	require.Len(t, items, 1)
	assert.Equal(t, clip.KindImage, items[0].Kind)
	assert.Contains(t, items[0].Preview, "image")
}
