# DevClip Windows Core (P0–P4) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a working Windows clipboard manager: capture text + image into an in-memory ring buffer, block sensitive apps, and paste a chosen item back into the previously focused window via Alt+V — all behind a `Platform` interface so macOS drops in later.

**Architecture:** Three layers — (1) `platform` (only layer touching Win32 syscalls, hidden behind the `Platform` interface), (2) `engine` + pure-Go core (`clip`, `security`) that is fully unit-tested against a fake platform, (3) Wails bridge + React UI. Data flows: Win32 `WM_CLIPBOARDUPDATE` → engine → `ClipStore` → emit to React; Alt+V `WM_HOTKEY` → popup → React calls back → engine triggers paste.

**Tech Stack:** Go 1.22+, Wails v2, `golang.org/x/sys/windows`, React + TypeScript (Wails default), `testify` for assertions.

**Scope note:** This plan delivers P0–P4 from the spec ([2026-06-08-devclip-design.md](../specs/2026-06-08-devclip-design.md)). P5 (Formatter), P6 (Snippet Vault), P7 (Polish), and the entire macOS platform ([macOS spec](../specs/2026-06-08-devclip-macos-design.md)) are separate plans written after this one ships.

---

## File Structure

```
devclip/
  go.mod
  wails.json
  main.go                       # Wails entry (generated, lightly edited)
  app.go                        # Wails App struct + bound methods (UI ↔ engine)
  internal/
    clip/
      item.go                   # ClipItem, ClipKind, hashing
      store.go                  # ClipStore: ring buffer + map, FIFO, dedup
      store_test.go
    security/
      filter.go                 # SecurityFilter: app blocklist
      filter_test.go
    platform/
      platform.go               # Platform + PlatformEvents interfaces, value types
      fake.go                   # FakePlatform for engine tests (all OS)
      windows.go                # //go:build windows — real Win32 impl
      win32_syscall.go          # //go:build windows — raw proc bindings
    engine/
      engine.go                 # wires platform events → store, exposes ops to app.go
      engine_test.go            # tests engine against FakePlatform
  frontend/                     # React/TS (Wails generated)
    src/
      App.tsx                   # history list + search + keyboard nav
      types.ts                  # mirror of ClipItem for TS
```

Each Go file has one responsibility. The pure-Go packages (`clip`, `security`, `engine`) never import `syscall`/`x/sys/windows` — they depend only on the `Platform` interface, so they are testable on any OS via `FakePlatform`.

---

## Task 1: Project scaffold + Wails skeleton

**Files:**
- Create: `go.mod`, `wails.json`, `main.go`, `app.go`, `frontend/*` (via Wails CLI)

- [ ] **Step 1: Install Wails CLI and verify**

Run:
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails doctor
```
Expected: report shows Go, WebView2, and npm present (no fatal issues).

- [ ] **Step 2: Scaffold the project**

Run from `D:/private_project/devcip`:
```bash
wails init -n devclip -t react-ts -d .
```
Expected: creates `main.go`, `app.go`, `wails.json`, `frontend/` with a React-TS template.

- [ ] **Step 3: Verify it builds and runs**

Run:
```bash
wails dev
```
Expected: a window opens showing the default Wails template. Close it.

- [ ] **Step 4: Add testify**

Run:
```bash
go get github.com/stretchr/testify@latest
```
Expected: `go.mod` now lists `github.com/stretchr/testify`.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: scaffold Wails v2 react-ts project"
```

---

## Task 2: Clip value types (`clip/item.go`)

**Files:**
- Create: `internal/clip/item.go`
- Test: covered via Task 3 (`store_test.go`) and Step 1 below

- [ ] **Step 1: Write the failing test**

Create `internal/clip/item_test.go`:
```go
package clip

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashTextIsStableAndDistinct(t *testing.T) {
	a := HashText("SELECT 1")
	b := HashText("SELECT 1")
	c := HashText("SELECT 2")
	assert.Equal(t, a, b, "same text must hash equal")
	assert.NotEqual(t, a, c, "different text must hash different")
}

func TestHashBytesDistinctFromText(t *testing.T) {
	img := HashBytes([]byte{1, 2, 3})
	assert.NotZero(t, img)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/clip/ -run TestHash -v`
Expected: FAIL — `undefined: HashText`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/clip/item.go`:
```go
package clip

import (
	"hash/fnv"
	"time"
)

type ClipKind int

const (
	KindText ClipKind = iota
	KindImage
)

// ClipItem is one entry in the in-memory history.
type ClipItem struct {
	ID        uint64    `json:"id"`
	Kind      ClipKind  `json:"kind"`
	Text      string    `json:"text"`  // empty if image
	Image     []byte    `json:"-"`     // PNG bytes; nil if text (not sent raw to UI)
	Preview   string    `json:"preview"` // short text or "image NxM" label for UI
	Hash      uint64    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	Pinned    bool      `json:"pinned"`
}

func HashText(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

func HashBytes(b []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(b)
	return h.Sum64()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/clip/ -run TestHash -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/clip/item.go internal/clip/item_test.go
git commit -m "feat(clip): add ClipItem types and content hashing"
```

---

## Task 3: ClipStore ring buffer (`clip/store.go`)

**Files:**
- Create: `internal/clip/store.go`
- Test: `internal/clip/store_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/clip/store_test.go`:
```go
package clip

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func textItem(s string) *ClipItem {
	return &ClipItem{Kind: KindText, Text: s, Preview: s, Hash: HashText(s)}
}

func TestPushAndListNewestFirst(t *testing.T) {
	s := NewStore(3)
	s.Push(textItem("a"))
	s.Push(textItem("b"))
	items := s.List()
	require.Len(t, items, 2)
	assert.Equal(t, "b", items[0].Text, "newest first")
	assert.Equal(t, "a", items[1].Text)
}

func TestPushAssignsMonotonicIDs(t *testing.T) {
	s := NewStore(3)
	s.Push(textItem("a"))
	s.Push(textItem("b"))
	items := s.List()
	assert.Equal(t, uint64(2), items[0].ID)
	assert.Equal(t, uint64(1), items[1].ID)
}

func TestRingBufferEvictsOldestFIFO(t *testing.T) {
	s := NewStore(3)
	for _, v := range []string{"a", "b", "c", "d"} {
		s.Push(textItem(v))
	}
	items := s.List()
	require.Len(t, items, 3)
	assert.Equal(t, []string{"d", "c", "b"}, []string{items[0].Text, items[1].Text, items[2].Text})
}

func TestDedupMovesExistingToTop(t *testing.T) {
	s := NewStore(3)
	s.Push(textItem("a"))
	s.Push(textItem("b"))
	s.Push(textItem("a")) // duplicate of first
	items := s.List()
	require.Len(t, items, 2, "no new item added")
	assert.Equal(t, "a", items[0].Text, "duplicate moved to top")
}

func TestEvictedItemDataIsCleared(t *testing.T) {
	s := NewStore(1)
	first := textItem("a")
	s.Push(first)
	s.Push(textItem("b"))
	assert.Empty(t, first.Text, "evicted item's Text must be cleared for GC")
	assert.Nil(t, first.Image)
}

func TestConcurrentPushIsSafe(t *testing.T) {
	s := NewStore(100)
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 50; j++ {
				s.Push(textItem(fmt.Sprintf("g%d-%d", n, j)))
			}
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	assert.Len(t, s.List(), 100)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/clip/ -run TestPush -v`
Expected: FAIL — `undefined: NewStore`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/clip/store.go`:
```go
package clip

import (
	"sync"
	"sync/atomic"
	"time"
)

// ClipStore is a fixed-size in-memory FIFO ring buffer with O(1) dedup.
type ClipStore struct {
	mu     sync.RWMutex
	buf    []*ClipItem
	head   int // next write index
	count  int
	byHash map[uint64]*ClipItem
	nextID uint64
}

func NewStore(capacity int) *ClipStore {
	if capacity < 1 {
		capacity = 1
	}
	return &ClipStore{
		buf:    make([]*ClipItem, capacity),
		byHash: make(map[uint64]*ClipItem, capacity),
	}
}

// Push inserts item. Duplicate (same Hash) refreshes the existing item to top
// instead of adding a new one. Evicted items are cleared to release memory.
func (s *ClipStore) Push(item *ClipItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.byHash[item.Hash]; ok {
		existing.CreatedAt = time.Now()
		existing.ID = atomic.AddUint64(&s.nextID, 1) // new ID => sorts to top
		return
	}

	if old := s.buf[s.head]; old != nil {
		delete(s.byHash, old.Hash)
		old.Text = ""
		old.Image = nil
		s.buf[s.head] = nil
	}

	item.ID = atomic.AddUint64(&s.nextID, 1)
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	s.buf[s.head] = item
	s.byHash[item.Hash] = item
	s.head = (s.head + 1) % len(s.buf)
	if s.count < len(s.buf) {
		s.count++
	}
}

// List returns a snapshot of items, newest (highest ID) first.
func (s *ClipStore) List() []*ClipItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ClipItem, 0, s.count)
	for _, it := range s.buf {
		if it != nil {
			out = append(out, it)
		}
	}
	// newest first by ID (descending)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].ID > out[i].ID {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// Get returns the item with the given ID, or nil.
func (s *ClipStore) Get(id uint64) *ClipItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, it := range s.buf {
		if it != nil && it.ID == id {
			return it
		}
	}
	return nil
}

// Clear removes all items and releases memory.
func (s *ClipStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.buf {
		if it := s.buf[i]; it != nil {
			it.Text = ""
			it.Image = nil
		}
		s.buf[i] = nil
	}
	s.byHash = make(map[uint64]*ClipItem, len(s.buf))
	s.head = 0
	s.count = 0
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/clip/ -race -v`
Expected: PASS (including `TestConcurrentPushIsSafe` under `-race`).

- [ ] **Step 5: Commit**

```bash
git add internal/clip/store.go internal/clip/store_test.go
git commit -m "feat(clip): add fixed-size FIFO ring buffer store with dedup"
```

---

## Task 4: SecurityFilter (`security/filter.go`)

**Files:**
- Create: `internal/security/filter.go`
- Test: `internal/security/filter_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/security/filter_test.go`:
```go
package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlocksKnownSensitiveAppByExeName(t *testing.T) {
	f := NewFilter([]string{"1Password.exe", "Bitwarden.exe"})
	assert.True(t, f.IsBlocked(`C:\Program Files\1Password\1Password.exe`))
	assert.True(t, f.IsBlocked(`D:\apps\Bitwarden.exe`))
}

func TestBlockingIsCaseInsensitive(t *testing.T) {
	f := NewFilter([]string{"1Password.exe"})
	assert.True(t, f.IsBlocked(`C:\x\1PASSWORD.EXE`))
}

func TestAllowsNormalApp(t *testing.T) {
	f := NewFilter([]string{"1Password.exe"})
	assert.False(t, f.IsBlocked(`C:\Windows\notepad.exe`))
}

func TestEmptyPathIsNotBlocked(t *testing.T) {
	f := NewFilter([]string{"1Password.exe"})
	assert.False(t, f.IsBlocked(""))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/security/ -v`
Expected: FAIL — `undefined: NewFilter`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/security/filter.go`:
```go
package security

import (
	"path/filepath"
	"strings"
)

// SecurityFilter decides whether a clipboard capture from a given source
// application should be dropped (e.g. password managers).
type SecurityFilter struct {
	blocked map[string]struct{} // lowercased exe base names
}

func NewFilter(exeNames []string) *SecurityFilter {
	m := make(map[string]struct{}, len(exeNames))
	for _, n := range exeNames {
		m[strings.ToLower(n)] = struct{}{}
	}
	return &SecurityFilter{blocked: m}
}

// IsBlocked reports whether the given full exe path belongs to a blocked app.
func (f *SecurityFilter) IsBlocked(exePath string) bool {
	if exePath == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(exePath))
	_, ok := f.blocked[base]
	return ok
}

// DefaultBlocklist returns the built-in sensitive apps.
func DefaultBlocklist() []string {
	return []string{"1Password.exe", "Bitwarden.exe", "KeePass.exe", "KeePassXC.exe"}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/security/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/security/
git commit -m "feat(security): add app-based clipboard blocklist filter"
```

---

## Task 5: Platform interface + FakePlatform (`platform/platform.go`, `platform/fake.go`)

**Files:**
- Create: `internal/platform/platform.go`
- Create: `internal/platform/fake.go`
- Test: `internal/platform/fake_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/platform/fake_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/ -v`
Expected: FAIL — `undefined: NewFakePlatform`.

- [ ] **Step 3: Write the interface and value types**

Create `internal/platform/platform.go`:
```go
package platform

// RawClipKind is the raw payload kind read from / written to the OS clipboard.
type RawClipKind int

const (
	ClipText RawClipKind = iota
	ClipImage
)

// RawClip is platform-neutral clipboard content.
type RawClip struct {
	Kind  RawClipKind
	Text  string
	Image []byte // PNG bytes when Kind == ClipImage
}

// AppInfo identifies the foreground application.
// Windows fills ExePath; macOS (later) fills BundleID.
type AppInfo struct {
	ExePath  string
	BundleID string
}

// WindowRef is an opaque handle to the paste-target window (HWND on Windows).
type WindowRef uintptr

// HotkeySpec describes a global hotkey to register.
type HotkeySpec struct {
	ID        int
	Modifiers uint // platform-specific modifier bitmask
	KeyCode   uint // virtual key code
}

// PlatformEvents receives callbacks from the platform layer.
type PlatformEvents interface {
	OnClipboardChange()
	OnHotkey(id int)
}

// EventsFunc is a func-based adapter implementing PlatformEvents.
type EventsFunc struct {
	ClipboardChange func()
	Hotkey          func(id int)
}

func (e EventsFunc) OnClipboardChange() {
	if e.ClipboardChange != nil {
		e.ClipboardChange()
	}
}
func (e EventsFunc) OnHotkey(id int) {
	if e.Hotkey != nil {
		e.Hotkey(id)
	}
}

// Platform is the single boundary to OS-specific clipboard/input APIs.
type Platform interface {
	Start(ev PlatformEvents) error
	Stop()
	ReadClipboard() (*RawClip, error)
	WriteClipboard(*RawClip) error
	ForegroundApp() (AppInfo, error)
	CaptureForegroundWindow() WindowRef // remember paste target before showing popup
	SimulatePaste(target WindowRef) error
	CursorPos() (x, y int)
	RegisterHotkey(spec HotkeySpec) error
}
```

- [ ] **Step 4: Write the fake**

Create `internal/platform/fake.go`:
```go
package platform

import "sync"

// FakePlatform is an in-memory Platform for tests on any OS.
type FakePlatform struct {
	mu         sync.Mutex
	events     PlatformEvents
	clip       *RawClip
	fgApp      AppInfo
	fgWindow   WindowRef
	pasteCalls []WindowRef
	cursorX    int
	cursorY    int
}

func NewFakePlatform() *FakePlatform { return &FakePlatform{} }

func (f *FakePlatform) Start(ev PlatformEvents) error { f.events = ev; return nil }
func (f *FakePlatform) Stop()                         {}

func (f *FakePlatform) ReadClipboard() (*RawClip, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.clip, nil
}
func (f *FakePlatform) WriteClipboard(c *RawClip) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clip = c
	return nil
}
func (f *FakePlatform) ForegroundApp() (AppInfo, error)   { return f.fgApp, nil }
func (f *FakePlatform) CaptureForegroundWindow() WindowRef { return f.fgWindow }
func (f *FakePlatform) SimulatePaste(t WindowRef) error {
	f.pasteCalls = append(f.pasteCalls, t)
	return nil
}
func (f *FakePlatform) CursorPos() (int, int)             { return f.cursorX, f.cursorY }
func (f *FakePlatform) RegisterHotkey(HotkeySpec) error   { return nil }

// --- test helpers ---
func (f *FakePlatform) SetClipboard(c *RawClip)       { f.WriteClipboard(c) }
func (f *FakePlatform) SetForegroundApp(a AppInfo)    { f.fgApp = a }
func (f *FakePlatform) SetForegroundWindow(w WindowRef) { f.fgWindow = w }
func (f *FakePlatform) EmitClipboardChange()          { f.events.OnClipboardChange() }
func (f *FakePlatform) EmitHotkey(id int)             { f.events.OnHotkey(id) }
func (f *FakePlatform) PasteCalls() []WindowRef       { return f.pasteCalls }
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/platform/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/platform/platform.go internal/platform/fake.go internal/platform/fake_test.go
git commit -m "feat(platform): add Platform interface and FakePlatform"
```

---

## Task 6: Engine wiring (`engine/engine.go`)

**Files:**
- Create: `internal/engine/engine.go`
- Test: `internal/engine/engine_test.go`

The engine subscribes to platform events, applies the security filter, converts `RawClip` → `ClipItem`, pushes to the store, and notifies a UI callback. On hotkey it captures the paste target. On paste request it writes the item to the clipboard, simulates paste, then restores the previous clipboard after a delay.

- [ ] **Step 1: Write the failing test**

Create `internal/engine/engine_test.go`:
```go
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
	// existing user clipboard we must restore
	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipText, Text: "original"})
	fp.EmitClipboardChange()
	original := e.History()[0]

	// add the item we want to paste
	fp.SetClipboard(&platform.RawClip{Kind: platform.ClipText, Text: "snippet"})
	fp.EmitClipboardChange()
	target := e.History()[0]

	fp.SetForegroundWindow(platform.WindowRef(99))
	fp.EmitHotkey(HotkeyPasteID)

	require.NoError(t, e.PasteItem(target.ID))

	// clipboard had "snippet" set, paste simulated to target window
	calls := fp.PasteCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, platform.WindowRef(99), calls[0])

	// after restore delay, clipboard returns to the original
	time.Sleep(20 * time.Millisecond)
	got, _ := fp.ReadClipboard()
	assert.Equal(t, "original", got.Text)
	_ = original
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -v`
Expected: FAIL — `undefined: New` / `undefined: Engine`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/engine/engine.go`:
```go
package engine

import (
	"fmt"
	"sync"
	"time"

	"devclip/internal/clip"
	"devclip/internal/platform"
	"devclip/internal/security"
)

const HotkeyPasteID = 1

type Config struct {
	Platform     platform.Platform
	Store        *clip.ClipStore
	Filter       *security.SecurityFilter
	Capacity     int
	RestoreDelay time.Duration
}

type Engine struct {
	cfg        Config
	mu         sync.Mutex
	pasteTgt   platform.WindowRef
	onChange   func()
	lastSelfWrite uint64 // hash we wrote ourselves, to ignore on next change
}

func New(cfg Config) *Engine {
	if cfg.RestoreDelay == 0 {
		cfg.RestoreDelay = 200 * time.Millisecond
	}
	return &Engine{cfg: cfg}
}

func (e *Engine) Start() error {
	return e.cfg.Platform.Start(platform.EventsFunc{
		ClipboardChange: e.handleClipboardChange,
		Hotkey:          e.handleHotkey,
	})
}

func (e *Engine) OnChange(fn func()) { e.onChange = fn }

func (e *Engine) handleClipboardChange() {
	// Drop captures coming from blocked apps.
	if app, err := e.cfg.Platform.ForegroundApp(); err == nil {
		if e.cfg.Filter.IsBlocked(app.ExePath) {
			return
		}
	}

	raw, err := e.cfg.Platform.ReadClipboard()
	if err != nil || raw == nil {
		return
	}

	var item *clip.ClipItem
	switch raw.Kind {
	case platform.ClipText:
		if raw.Text == "" {
			return
		}
		item = &clip.ClipItem{
			Kind:    clip.KindText,
			Text:    raw.Text,
			Preview: preview(raw.Text),
			Hash:    clip.HashText(raw.Text),
		}
	case platform.ClipImage:
		if len(raw.Image) == 0 {
			return
		}
		item = &clip.ClipItem{
			Kind:    clip.KindImage,
			Image:   raw.Image,
			Preview: fmt.Sprintf("image (%d bytes)", len(raw.Image)),
			Hash:    clip.HashBytes(raw.Image),
		}
	default:
		return
	}

	// Ignore the change we caused ourselves during paste-restore.
	e.mu.Lock()
	if item.Hash == e.lastSelfWrite {
		e.lastSelfWrite = 0
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()

	e.cfg.Store.Push(item)
	if e.onChange != nil {
		e.onChange()
	}
}

func (e *Engine) handleHotkey(id int) {
	if id != HotkeyPasteID {
		return
	}
	e.mu.Lock()
	e.pasteTgt = e.cfg.Platform.CaptureForegroundWindow()
	e.mu.Unlock()
}

func (e *Engine) PasteTarget() platform.WindowRef {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.pasteTgt
}

// PasteItem writes the given history item to the clipboard, simulates paste
// into the remembered target window, then restores the previous clipboard.
func (e *Engine) PasteItem(id uint64) error {
	it := e.cfg.Store.Get(id)
	if it == nil {
		return fmt.Errorf("item %d not found", id)
	}

	prev, _ := e.cfg.Platform.ReadClipboard()

	out := &platform.RawClip{}
	switch it.Kind {
	case clip.KindText:
		out.Kind = platform.ClipText
		out.Text = it.Text
		e.mu.Lock()
		e.lastSelfWrite = clip.HashText(it.Text)
		e.mu.Unlock()
	case clip.KindImage:
		out.Kind = platform.ClipImage
		out.Image = it.Image
	}
	if err := e.cfg.Platform.WriteClipboard(out); err != nil {
		return err
	}

	e.mu.Lock()
	target := e.pasteTgt
	e.mu.Unlock()
	if err := e.cfg.Platform.SimulatePaste(target); err != nil {
		return err
	}

	if prev != nil {
		go func() {
			time.Sleep(e.cfg.RestoreDelay)
			_ = e.cfg.Platform.WriteClipboard(prev)
		}()
	}
	return nil
}

func (e *Engine) History() []*clip.ClipItem { return e.cfg.Store.List() }
func (e *Engine) Clear()                    { e.cfg.Store.Clear() }

func preview(s string) string {
	const max = 120
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine/ -race -v`
Expected: PASS (all five tests).

- [ ] **Step 5: Commit**

```bash
git add internal/engine/
git commit -m "feat(engine): wire platform events to store with security filter and paste+restore"
```

---

## Task 7: Win32 syscall bindings (`platform/win32_syscall.go`)

**Files:**
- Create: `internal/platform/win32_syscall.go` (build tag `windows`)

This file holds the raw `LazyDLL`/`LazyProc` wrappers. No business logic. Manual verification only (covered in Task 9 checklist).

- [ ] **Step 1: Write the bindings**

Create `internal/platform/win32_syscall.go`:
```go
//go:build windows

package platform

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazyDLL("user32.dll")
	kernel32 = windows.NewLazyDLL("kernel32.dll")

	procAddClipboardFormatListener = user32.NewProc("AddClipboardFormatListener")
	procRegisterClassExW           = user32.NewProc("RegisterClassExW")
	procCreateWindowExW            = user32.NewProc("CreateWindowExW")
	procDefWindowProcW             = user32.NewProc("DefWindowProcW")
	procGetMessageW                = user32.NewProc("GetMessageW")
	procTranslateMessage           = user32.NewProc("TranslateMessage")
	procDispatchMessageW           = user32.NewProc("DispatchMessageW")
	procRegisterHotKey             = user32.NewProc("RegisterHotKey")
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
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procGlobalAlloc                = kernel32.NewProc("GlobalAlloc")
	procGlobalLock                 = kernel32.NewProc("GlobalLock")
	procGlobalUnlock               = kernel32.NewProc("GlobalUnlock")
)

// Win32 constants used by the platform layer.
const (
	cfUnicodeText = 13
	cfDIB         = 8

	wmClipboardUpdate = 0x031D
	wmHotkey          = 0x0312

	modAlt = 0x0001

	ghMemMoveable = 0x0002

	inputKeyboard   = 1
	keyEventKeyUp   = 0x0002
	vkControl       = 0x11
	vkV             = 0x56

	processQueryLimitedInformation = 0x1000
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
// The union is sized for the largest member; keyboard fits within it.
type keyboardInput struct {
	Type uint32
	Ki   struct {
		Vk          uint16
		Scan        uint16
		Flags       uint32
		Time        uint32
		ExtraInfo   uintptr
	}
	_ [8]byte // padding so struct size matches the MOUSEINPUT union on amd64
}

func getCursorPos() (int, int) {
	var p point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return int(p.X), int(p.Y)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/platform/`
Expected: builds with no errors on Windows.

- [ ] **Step 3: Commit**

```bash
git add internal/platform/win32_syscall.go
git commit -m "feat(platform): add raw Win32 syscall bindings"
```

---

## Task 8: Windows Platform implementation (`platform/windows.go`)

**Files:**
- Create: `internal/platform/windows.go` (build tag `windows`)

Implements `Platform` using the bindings from Task 7. The message loop runs on a locked OS thread and dispatches `WM_CLIPBOARDUPDATE` / `WM_HOTKEY` to the engine callbacks.

- [ ] **Step 1: Write the implementation**

Create `internal/platform/windows.go`:
```go
//go:build windows

package platform

import (
	"errors"
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
	wndProc := windows.NewCallback(p.wndProc)

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
	if r, _, err := procRegisterHotKey.Call(hwnd, uintptr(HotkeyPasteID), modAlt, vkV); r == 0 {
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

func (p *windowsPlatform) wndProc(hwnd windows.Handle, message uint32, wParam, lParam uintptr) uintptr {
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
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(message), wParam, lParam)
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
		ptr, _, _ := procGlobalLock.Call(h)
		if ptr == 0 {
			return nil, errors.New("GlobalLock failed")
		}
		text := windows.UTF16PtrToString((*uint16)(unsafe.Pointer(ptr)))
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
		hMem, _, _ := procGlobalAlloc.Call(ghMemMoveable, size)
		if hMem == 0 {
			return errors.New("GlobalAlloc failed")
		}
		dst, _, _ := procGlobalLock.Call(hMem)
		copyU16(dst, u16)
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
		procSetForegroundWindow.Call(uintptr(target))
		time.Sleep(30 * time.Millisecond) // let focus settle
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

func (p *windowsPlatform) CursorPos() (int, int) { return getCursorPos() }

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

func copyU16(dst uintptr, src []uint16) {
	d := unsafe.Slice((*uint16)(unsafe.Pointer(dst)), len(src))
	copy(d, src)
}
```

- [ ] **Step 2: Add the OS-thread lock helper and clipboard retry**

Create `internal/platform/win32_helpers.go`:
```go
//go:build windows

package platform

import (
	"errors"
	"runtime"
	"time"
)

func runtimeLockOSThread() { runtime.LockOSThread() }

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
```

- [ ] **Step 3: Add image DIB↔PNG conversion stubs**

Create `internal/platform/win32_image.go`:
```go
//go:build windows

package platform

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"unsafe"

	"golang.org/x/sys/windows"
)

// readDIBAsPNG reads CF_DIB from the (already open) clipboard and encodes PNG.
func readDIBAsPNG() ([]byte, error) {
	h, _, _ := procGetClipboardData.Call(cfDIB)
	if h == 0 {
		return nil, errors.New("no DIB data")
	}
	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return nil, errors.New("GlobalLock failed on DIB")
	}
	defer procGlobalUnlock.Call(h)

	img, err := dibToImage(ptr)
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
func dibToImage(ptr uintptr) (image.Image, error) {
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
	bih := (*bitmapInfoHeader)(unsafe.Pointer(ptr))
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
	pixels := unsafe.Slice((*byte)(unsafe.Pointer(ptr+uintptr(bih.Size))), stride*h)

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

	hMem, _, _ := procGlobalAlloc.Call(ghMemMoveable, uintptr(dibSize))
	if hMem == 0 {
		return errors.New("GlobalAlloc failed for DIB")
	}
	dst, _, _ := procGlobalLock.Call(hMem)
	header := unsafe.Slice((*byte)(unsafe.Pointer(dst)), dibSize)

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

var _ = windows.MAX_PATH // keep windows import if unused elsewhere
```

- [ ] **Step 4: Verify it compiles**

Run: `go build ./...`
Expected: builds on Windows with no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/platform/windows.go internal/platform/win32_helpers.go internal/platform/win32_image.go
git commit -m "feat(platform): implement Windows clipboard, hotkey, paste, image conversion"
```

---

## Task 9: Wails bridge — bound methods (`app.go`)

**Files:**
- Modify: `app.go`
- Modify: `main.go`

- [ ] **Step 1: Replace `app.go` with engine wiring**

Replace the generated `app.go` contents:
```go
package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devclip/internal/clip"
	"devclip/internal/engine"
	"devclip/internal/platform"
	"devclip/internal/security"
)

// App is the Wails-bound application surface.
type App struct {
	ctx context.Context
	eng *engine.Engine
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.eng = engine.New(engine.Config{
		Platform: platform.NewPlatform(),
		Store:    clip.NewStore(100),
		Filter:   security.NewFilter(security.DefaultBlocklist()),
		Capacity: 100,
	})
	a.eng.OnChange(func() {
		runtime.EventsEmit(a.ctx, "clip:changed")
	})
	if err := a.eng.Start(); err != nil {
		runtime.LogError(a.ctx, "engine start failed: "+err.Error())
	}
}

// History returns the current clipboard history, newest first (bound to JS).
func (a *App) History() []*clip.ClipItem { return a.eng.History() }

// PasteItem writes the selected item to the clipboard and pastes it into the
// previously focused window (bound to JS).
func (a *App) PasteItem(id uint64) error { return a.eng.PasteItem(id) }

// ShowPopupAtCursor reports the cursor position so the frontend can place the
// popup window (bound to JS).
func (a *App) CursorPos() []int {
	x, y := platform.NewPlatform().CursorPos()
	return []int{x, y}
}

// Clear empties the history (bound to JS).
func (a *App) Clear() { a.eng.Clear() }
```

- [ ] **Step 2: Confirm `main.go` binds the App**

Ensure `main.go` (generated) contains, inside `wails.Run`'s `Bind`:
```go
Bind: []interface{}{
    app,
},
OnStartup: app.startup,
```
If the generated template uses a different startup hook name, wire `app.startup` to `OnStartup`.

- [ ] **Step 3: Verify it builds**

Run: `go build ./...`
Expected: builds. Wails will generate TS bindings on `wails dev`.

- [ ] **Step 4: Commit**

```bash
git add app.go main.go
git commit -m "feat(app): bind engine history/paste/cursor to Wails frontend"
```

---

## Task 10: React UI — history list + keyboard nav (`frontend/src/App.tsx`)

**Files:**
- Modify: `frontend/src/App.tsx`
- Create: `frontend/src/types.ts`

- [ ] **Step 1: Add the TS type**

Create `frontend/src/types.ts`:
```ts
export interface ClipItem {
  id: number;
  kind: number; // 0 = text, 1 = image
  text: string;
  preview: string;
  createdAt: string;
  pinned: boolean;
}
```

- [ ] **Step 2: Write the UI**

Replace `frontend/src/App.tsx`:
```tsx
import { useEffect, useState, useCallback } from "react";
import { History, PasteItem } from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import type { ClipItem } from "./types";

function App() {
  const [items, setItems] = useState<ClipItem[]>([]);
  const [query, setQuery] = useState("");
  const [sel, setSel] = useState(0);

  const refresh = useCallback(async () => {
    const list = (await History()) as unknown as ClipItem[];
    setItems(list ?? []);
  }, []);

  useEffect(() => {
    refresh();
    const off = EventsOn("clip:changed", refresh);
    return () => off();
  }, [refresh]);

  const filtered = items.filter((it) =>
    it.preview.toLowerCase().includes(query.toLowerCase())
  );

  const onKeyDown = async (e: React.KeyboardEvent) => {
    if (e.key === "ArrowDown") {
      setSel((s) => Math.min(s + 1, filtered.length - 1));
    } else if (e.key === "ArrowUp") {
      setSel((s) => Math.max(s - 1, 0));
    } else if (e.key === "Enter" && filtered[sel]) {
      await PasteItem(filtered[sel].id);
    }
  };

  return (
    <div className="container" onKeyDown={onKeyDown} tabIndex={0}>
      <input
        autoFocus
        placeholder="Search clipboard…"
        value={query}
        onChange={(e) => {
          setQuery(e.target.value);
          setSel(0);
        }}
      />
      <ul>
        {filtered.map((it, i) => (
          <li
            key={it.id}
            className={i === sel ? "selected" : ""}
            onClick={() => PasteItem(it.id)}
          >
            {it.kind === 1 ? `🖼 ${it.preview}` : it.preview}
          </li>
        ))}
      </ul>
    </div>
  );
}

export default App;
```

- [ ] **Step 3: Run the app and verify end-to-end (manual)**

Run: `wails dev`

Manual checklist:
- Copy text in Notepad → it appears at top of the DevClip list. ✅
- Copy a different text → list updates, newest first. ✅
- Copy the same text again → it moves to top, no duplicate. ✅
- Copy a screenshot (PrtSc into Paint, then Ctrl+C) → an `🖼 image (...)` row appears. ✅
- Type in Notepad, press **Alt+V** → DevClip popup; ↓/↑ to select; **Enter** → text pastes into Notepad. ✅
- After paste, copy something new in Notepad → original clipboard restore did not pollute history with the pasted item. ✅
- Copy inside 1Password (if installed) → nothing is added to history. ✅

- [ ] **Step 4: Commit**

```bash
git add frontend/src/App.tsx frontend/src/types.ts
git commit -m "feat(ui): clipboard history list with search and keyboard paste"
```

---

## Task 11: Popup positioning + hide-after-paste (wire Alt+V to window)

**Files:**
- Modify: `app.go`
- Modify: `frontend/src/App.tsx`

Currently Alt+V captures the paste target in the engine, but the Wails window does not yet show/move to the cursor. Wire the window show + positioning.

- [ ] **Step 1: Emit a popup event on hotkey**

In `internal/engine/engine.go`, extend `handleHotkey` to also notify the UI. Add a callback field and setter:

```go
// add to Engine struct:
//   onHotkey func()

func (e *Engine) OnHotkey(fn func()) { e.onHotkey = fn }
```
And at the end of `handleHotkey` (after capturing the target):
```go
	if e.onHotkey != nil {
		e.onHotkey()
	}
```

- [ ] **Step 2: Show + position the window in `app.go`**

In `startup`, after creating the engine:
```go
	a.eng.OnHotkey(func() {
		x, y := platform.NewPlatform().CursorPos()
		runtime.WindowSetPosition(a.ctx, x, y)
		runtime.WindowShow(a.ctx)
		runtime.EventsEmit(a.ctx, "popup:show")
	})
```
And add a bound method to hide after paste:
```go
func (a *App) Hide() { runtime.WindowHide(a.ctx) }
```

- [ ] **Step 3: Hide window after paste in the UI**

In `frontend/src/App.tsx`, import `Hide` and call it after `PasteItem`:
```tsx
import { History, PasteItem, Hide } from "../wailsjs/go/main/App";
// ...
} else if (e.key === "Enter" && filtered[sel]) {
  await PasteItem(filtered[sel].id);
  await Hide();
}
```
Also hide on `Escape`:
```tsx
} else if (e.key === "Escape") {
  await Hide();
}
```

- [ ] **Step 4: Configure window as hidden frameless on startup**

In `main.go` `wails.Run` options, set:
```go
		Width:             480,
		Height:            560,
		Frameless:         true,
		StartHidden:       true,
		AlwaysOnTop:       true,
		DisableResize:     true,
```

- [ ] **Step 5: Run and verify (manual)**

Run: `wails dev`
Manual checklist:
- App starts hidden (no window, only the dev console). ✅
- Press **Alt+V** anywhere → popup appears near the cursor, on top. ✅
- **Enter** pastes and the popup hides. ✅
- **Esc** hides the popup without pasting. ✅

- [ ] **Step 6: Commit**

```bash
git add internal/engine/engine.go app.go frontend/src/App.tsx main.go
git commit -m "feat: show frameless popup at cursor on Alt+V, hide after paste/Esc"
```

---

## Task 12: Final regression pass + full test run

- [ ] **Step 1: Run all Go tests**

Run: `go test ./... -race`
Expected: all packages PASS.

- [ ] **Step 2: Run `go vet`**

Run: `go vet ./...`
Expected: no findings.

- [ ] **Step 3: Build a release binary**

Run: `wails build`
Expected: produces `build/bin/devclip.exe`.

- [ ] **Step 4: Full manual smoke test of the built binary**

Run `build/bin/devclip.exe` and repeat the Task 10 Step 3 + Task 11 Step 5 checklists against the production build.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: P0-P4 Windows core complete — capture, store, filter, paste"
```

---

## Self-Review Notes (coverage vs spec)

- **P0 spike** (Wails + message loop + hotkey): Tasks 1, 7, 8.
- **P1 capture text** → store → emit: Tasks 2, 3, 6, 9, 10.
- **P2 image** (CF_DIB → PNG): Task 8 Step 3, verified in Task 10 Step 3.
- **P3 security filter**: Task 4, wired in Task 6, verified in Task 10 Step 3.
- **P4 paste + restore clipboard + popup at cursor**: Tasks 6, 11.
- **Platform interface for macOS drop-in**: Task 5 (interface), Task 8 (Windows impl). macOS impl = separate plan.
- **Memory hardening** (`debug.FreeOSMemory` after heavy image eviction): deferred to P5+ plan; ring buffer eviction clearing (Task 3) already prevents unbounded growth.

## Out of scope (separate plans)
- P5 Formatter (JSON tree / SQL format / string transform).
- P6 Snippet Vault + placeholder injection.
- P7 Polish (tray, settings, single-instance, auto-start, code signing).
- macOS platform (M0–M3) — see [macOS spec](../specs/2026-06-08-devclip-macos-design.md).
