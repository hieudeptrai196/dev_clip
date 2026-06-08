# DevClip — Design Spec

**Date:** 2026-06-08
**Status:** Approved (v1)

## 1. Tóm tắt

DevClip là một Clipboard Manager chạy ngầm trên Windows, chuyên cho lập trình viên,
thay thế tính năng Win + V mặc định. Toàn bộ lịch sử clipboard chỉ lưu trên RAM
(in-memory, session-based), không ghi xuống ổ cứng. Khi thoát app, dữ liệu bốc hơi.

- **Backend:** Go (Golang)
- **Frontend:** React + TypeScript qua **Wails v2**
- **UI runtime:** WebView2 (có sẵn trong Windows 10/11)
- **Win32 binding:** `golang.org/x/sys/windows` + `LazyDLL.NewProc` cho các hàm hiếm

## 2. Quyết định nền (đã chốt)

| Quyết định | Lựa chọn | Lý do |
|---|---|---|
| Framework | Wails v2 (không phải v3 alpha) | Ổn định cho production MVP |
| Phạm vi MVP | Làm đủ theo từng tầng (layer-by-layer) | Mỗi phase vẫn phải test được độc lập |
| Image support | Có trong MVP | Đúng spec gốc sớm |
| Focus model khi paste | Nhớ HWND đích + `SetForegroundWindow` rồi `SendInput` | Robust, không cần keyboard hook |
| Restore clipboard sau paste | Có, restore clipboard cũ sau ~200ms | UX tốt, không phá clipboard user đang giữ |
| Detect password trong terminal | **CẮT BỎ** | Không khả thi tin cậy (không phân biệt được pass vs command). Security filter chỉ block theo app |
| Testing | TDD cho core Go logic, manual checklist cho Win32 | Win32/clipboard khó test tự động |
| Cross-platform | Tách `Platform` interface ngay từ MVP Windows | macOS (làm sau) là drop-in, không phải viết lại Core. Spec macOS riêng: [2026-06-08-devclip-macos-design.md](2026-06-08-devclip-macos-design.md) |

## 3. Kiến trúc (3 lớp)

```
┌──────────────────────────────────────────────────────────┐
│                    DevClip.exe (Go)                       │
│                                                           │
│  Lớp 1: Win32 Event Layer (1 goroutine, LockOSThread)     │
│    - Hidden window + message loop                         │
│    - WM_CLIPBOARDUPDATE, WM_HOTKEY                        │
│    - Lớp DUY NHẤT chạm syscall Win32                      │
│                          │                                │
│                          ▼                                │
│  Lớp 2: Core Engine (thuần Go, unit-testable)             │
│    - ClipStore      (ring buffer 100 + map[hash])         │
│    - SecurityFilter (blocklist theo app)                  │
│    - Formatter      (JSON / SQL / string transform, lazy) │
│    - SnippetVault   (load config.json + placeholder)      │
│    - Clipboard      (open/read/write + retry/backoff)     │
│    - Paster         (SendInput Ctrl+V + restore clip)     │
│                          │                                │
│                          ▼                                │
│  Lớp 3: Wails Bridge (bound methods + EventsEmit)         │
│    - React/TS Frontend (WebView2)                         │
└──────────────────────────────────────────────────────────┘
        (Không có Database / File I/O cho history)
```

### Nguyên tắc tách lớp
- **Lớp 1** là nơi duy nhất gọi syscall. Toàn bộ Lớp 1 ẩn sau **interface `Platform`** (xem dưới) → Lớp 2 mock được khi test, và macOS sau này chỉ cần viết thêm một implementation.
- **Lớp 2** thuần Go, không import syscall trực tiếp → TDD được toàn bộ.
- **Lớp 3** chỉ là cầu nối, không chứa business logic.

### Platform interface (cross-platform từ ngày đầu)

Lớp 1 hiện thực interface này; Core Engine chỉ nói chuyện qua đây, không biết Windows hay macOS:

```go
type Platform interface {
    Start(ev PlatformEvents) error   // win: message loop (push); mac: poll timer + event tap
    Stop()
    ReadClipboard() (*RawClip, error)
    WriteClipboard(*RawClip) error
    ForegroundApp() (AppInfo, error) // win: exe path; mac: bundleID
    SimulatePaste(target WindowRef) error // win: Ctrl+V; mac: Cmd+V
    CursorPos() (x, y int)
    RegisterHotkey(spec HotkeySpec) error
}

type PlatformEvents interface {
    OnClipboardChange()  // win gọi từ WM_CLIPBOARDUPDATE; mac gọi khi changeCount đổi
    OnHotkey(id int)
}
```

- Khác biệt **push (Windows event) vs poll (macOS changeCount)** bị giấu sau `Start()`; cả hai chỉ phát `OnClipboardChange()`.
- File layout: `platform/platform.go` (interface), `platform/windows.go` (`//go:build windows`), `platform/darwin.go` (`//go:build darwin`, làm sau).
- Implementation Windows là phần được build/test trong MVP này. macOS có spec + plan riêng.

## 4. Cấu trúc dữ liệu

```go
type ClipKind int
const (
    KindText ClipKind = iota
    KindImage
    KindJSON
    KindSQL
)

type ClipItem struct {
    ID        uint64
    Kind      ClipKind
    Text      string    // nil-able nếu là image
    Image     []byte    // PNG-encoded; nil nếu là text
    Hash      uint64    // FNV/xxHash để dedup O(1)
    CreatedAt time.Time
    Pinned    bool
}

type ClipStore struct {
    mu      sync.RWMutex
    buf     [100]*ClipItem       // ring buffer cố định
    head    int
    count   int
    byHash  map[uint64]*ClipItem // dedup + lookup O(1)
    nextID  uint64
}
```

- **Ring buffer** cố định 100 item, FIFO. Khi đầy, item cũ nhất bị ghi đè.
- Item bị đẩy ra: `delete(byHash)`, `nil` hóa `Image`/`Text`, `buf[slot] = nil` → đủ điều kiện GC.
- **Dedup:** nếu hash trùng item đang có → update `CreatedAt` (move-to-top), không thêm bản mới.
- **Snippet pinned KHÔNG nằm trong ring buffer** — ở `SnippetVault` riêng, không bị FIFO đẩy ra.

## 5. Luồng dữ liệu

### Copy (capture)
1. `WM_CLIPBOARDUPDATE` tới hidden window (đã `AddClipboardFormatListener`).
2. `SecurityFilter` check TRƯỚC: `GetForegroundWindow` → PID → exe name → nếu ∈ blocklist thì DROP (không vào RAM).
3. Mở clipboard có retry/backoff: đọc `CF_UNICODETEXT` (text) hoặc `CF_DIBV5` (ảnh → encode PNG).
4. `ClipStore.Push(item)` (dedup theo hash).
5. `runtime.EventsEmit("clip:new", item)` → React prepend vào list.

### Paste
1. `WM_HOTKEY` (Alt+V).
2. **Nhớ HWND foreground hiện tại** (app đích) + `GetCursorPos`.
3. Hiện popup tại cursor (activate bình thường → gõ search được).
4. User ↑/↓ chọn, Enter.
5. Lưu clipboard cũ → set item lên clipboard (retry) → `SetForegroundWindow(target)` → `SendInput(Ctrl down, V down, V up, Ctrl up)`.
6. Sau ~200ms: restore clipboard cũ.
7. (Snippet có `{{var}}`: trước bước 5, hiện form hỏi giá trị → render template → mới paste.)

## 6. Win32 API cần dùng

| Nhóm | Hàm | DLL |
|---|---|---|
| Listener | `AddClipboardFormatListener`, `RemoveClipboardFormatListener` | user32 |
| Clipboard | `OpenClipboard`, `CloseClipboard`, `GetClipboardData`, `SetClipboardData`, `EmptyClipboard`, `IsClipboardFormatAvailable` | user32 |
| Hotkey | `RegisterHotKey`, `UnregisterHotKey` | user32 |
| Giả lập phím | `SendInput` | user32 |
| Active window | `GetForegroundWindow`, `GetWindowThreadProcessId`, `SetForegroundWindow` | user32 |
| Process name | `OpenProcess`, `QueryFullProcessImageNameW` | kernel32 |
| Cursor | `GetCursorPos` | user32 |
| Overlay (P7) | `SetWindowLongPtr`, `SetWindowPos` | user32 |
| Message loop | `CreateWindowEx`, `RegisterClassEx`, `GetMessage`, `TranslateMessage`, `DispatchMessage` | user32 |

**Threading:** message loop PHẢI `runtime.LockOSThread()` ở đầu goroutine — window handle gắn chặt với thread tạo ra nó.

## 7. Quản lý RAM

- Ring buffer cố định 100 → không phình theo thời gian.
- `nil` hóa `Image`/`Text` tường minh khi item bị đẩy ra.
- `debug.SetMemoryLimit(N)` (soft cap) để GC tự siết khi gần trần.
- `debug.FreeOSMemory()` gọi **có chọn lọc**: chỉ khi giải phóng ảnh nặng (> ngưỡng ~5MB) hoặc clear all — KHÔNG gọi mỗi lần copy (stop-the-world).
- Luôn `defer CloseClipboard()` để tránh leak handle ở tầng OS.

## 8. Xử lý lỗi & rủi ro

| Rủi ro | Giải pháp |
|---|---|
| Clipboard bị khóa ("in use") | Retry + exponential backoff (10ms → 20 → 40... trần 500ms, ~6 lần). Fail mềm: bỏ qua lần đó, log warning, KHÔNG crash loop |
| `SendInput` bị app admin chặn (UIPI) | Cảnh báo user; cân nhắc tùy chọn chạy elevated (kèm cảnh báo bảo mật) |
| Antivirus false-positive (giống keylogger) | Code signing bắt buộc khi distribute |
| CF_DIBV5 parsing phức tạp | Cân nhắc dùng `golang.design/x/clipboard` cho phần đọc/ghi; listener + filter tự viết |
| Memory "tưởng leak" (GC giữ heap) | `FreeOSMemory()` có chọn lọc + `SetMemoryLimit` |
| Popup cướp focus | MVP: activate + nhớ HWND đích + `SetForegroundWindow` trước `SendInput` |

## 9. Phase breakdown

- **P0 — Spike:** Wails v2 scaffold + hidden window + message loop + `RegisterHotKey(Alt+V)` log ra console. *(khử rủi ro lớn nhất trước)*
- **P1 — Capture (text):** `AddClipboardFormatListener` + đọc CF_UNICODETEXT + `ClipStore` + emit lên UI list.
- **P2 — Image:** đọc CF_DIBV5 → PNG → thumbnail + memory hardening (`FreeOSMemory` có chọn lọc).
- **P3 — Security filter:** foreground app → exe name → blocklist drop. Blocklist mặc định: 1Password, Bitwarden, KeePass.
- **P4 — Paste:** popup tại cursor + nav ↑/↓/Enter + `SendInput(Ctrl+V)` + restore clipboard cũ sau ~200ms.
- **P5 — Formatter:** JSON detect/pretty/tree, SQL format (uppercase keyword), string transformer (Upper/Lower/camelCase/snake_case/kebab).
- **P6 — Snippet Vault:** load `config.json` read-only lúc khởi động + tab riêng + placeholder `{{var}}` injection popup.
- **P7 — Polish:** system tray, settings UI (đổi hotkey/blocklist/max items), single-instance guard, auto-start (registry Run key), code signing, installer.

## 10. Search

- MVP: substring search (case-insensitive) trên text content, lọc realtime khi gõ trong popup.
- Fuzzy search để dành cho P7 nếu cần (YAGNI cho MVP).

## 11. Out of scope (v1)

- Detect password đang gõ trong terminal (cắt bỏ).
- Sync/cloud (trái triết lý in-memory).
- Persist history xuống disk (trái triết lý).
- Multi-monitor edge cases tinh vi (xử lý cơ bản ở P4).
