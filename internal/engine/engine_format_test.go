package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"devclip/internal/platform"
)

func TestJSONTextDetectedAsJSON(t *testing.T) {
	e, fp := newTestEngine(t)

	jsonText := `{"name":"Alice","age":30}`
	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipText, Text: jsonText})
	fp.EmitClipboardChange()

	items := e.History()
	require.Len(t, items, 1)
	assert.Equal(t, "json", items[0].Format)
}

func TestFormatItemReturnsIndentedJSON(t *testing.T) {
	e, fp := newTestEngine(t)

	jsonText := `{"name":"Alice","age":30}`
	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipText, Text: jsonText})
	fp.EmitClipboardChange()

	items := e.History()
	require.Len(t, items, 1)

	result := e.FormatItem(items[0].ID)
	assert.Contains(t, result, "\n")
	assert.Contains(t, result, `"name"`)
	assert.Contains(t, result, `"Alice"`)
}

func TestFormatItemReturnsEmptyForImage(t *testing.T) {
	e, fp := newTestEngine(t)

	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipImage, Image: []byte("PNGDATA")})
	fp.EmitClipboardChange()

	items := e.History()
	require.Len(t, items, 1)

	result := e.FormatItem(items[0].ID)
	assert.Equal(t, "", result)
}

func TestPasteTransformedUppercase(t *testing.T) {
	e, fp := newTestEngine(t)

	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipText, Text: "hello world"})
	fp.EmitClipboardChange()

	items := e.History()
	require.Len(t, items, 1)

	fp.SetForegroundWindow(platform.WindowRef(42))
	fp.EmitHotkey(HotkeyPasteID)

	require.NoError(t, e.PasteTransformed(items[0].ID, "upper"))

	// Clipboard should now contain the uppercased text.
	got, err := fp.ReadClipboard()
	require.NoError(t, err)
	assert.Equal(t, "HELLO WORLD", got.Text)

	// Paste was simulated into the target window.
	calls := fp.PasteCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, platform.WindowRef(42), calls[0])

	// After restore delay the clipboard is restored to the original.
	time.Sleep(20 * time.Millisecond)
	restored, _ := fp.ReadClipboard()
	// The original clipboard before PasteTransformed was "hello world"
	// (last item written before we called PasteTransformed).
	// The important thing is that "HELLO WORLD" is no longer on the clipboard.
	assert.NotEqual(t, "HELLO WORLD", restored.Text)
}

func TestPlainTextDetectedAsPlain(t *testing.T) {
	e, fp := newTestEngine(t)

	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipText, Text: "just some text"})
	fp.EmitClipboardChange()

	items := e.History()
	require.Len(t, items, 1)
	assert.Equal(t, "plain", items[0].Format)
}

func TestSQLTextDetectedAsSQL(t *testing.T) {
	e, fp := newTestEngine(t)

	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipText, Text: "select * from users where id = 1"})
	fp.EmitClipboardChange()

	items := e.History()
	require.Len(t, items, 1)
	assert.Equal(t, "sql", items[0].Format)
}

func TestFormatItemSQLReturnsFormattedSQL(t *testing.T) {
	e, fp := newTestEngine(t)

	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipText, Text: "select * from users where id = 1"})
	fp.EmitClipboardChange()

	items := e.History()
	require.Len(t, items, 1)

	result := e.FormatItem(items[0].ID)
	assert.True(t, strings.Contains(result, "SELECT"), "SQL keywords should be uppercased")
}
