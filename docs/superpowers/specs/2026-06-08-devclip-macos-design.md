# DevClip — macOS Design Spec

**Date:** 2026-06-08
**Status:** Approved (v1) — triển khai SAU khi bản Windows hoàn tất
**Spec gốc (Windows):** [2026-06-08-devclip-design.md](2026-06-08-devclip-design.md)

## 1. Phạm vi & nguyên tắc

Đây là spec cho phần platform macOS của DevClip. **Core Engine (Lớp 2) và toàn bộ
frontend React/TS được tái sử dụng 100%** — macOS chỉ là một implementation mới của
interface `Platform` đã định nghĩa trong spec Windows.

- **UI runtime:** Wails v2 trên macOS dùng **WKWebView** → frontend React giữ nguyên.
- **Gọi Cocoa từ Go:** **cgo + Objective-C** (Wails đã dùng cgo trên mac), hoặc `purego` + objc runtime. Cần Xcode command-line tools.
- **Điều kiện tiên quyết:** bản Windows xong, `Platform` interface đã ổn định.

## 2. Khác biệt cốt lõi so với Windows

| Hạng mục | Windows | macOS | Tác động |
|---|---|---|---|
| Bắt clipboard | `AddClipboardFormatListener` (push, có event) | **KHÔNG có event** — poll `NSPasteboard.changeCount` trên timer (~250–500ms) | Khác biệt kiến trúc lớn nhất |
| Phím dán | `Ctrl+V` | `Cmd+V` | `SimulatePaste` theo platform |
| Global hotkey | `RegisterHotKey` (không cần quyền) | Carbon `RegisterEventHotKey` **hoặc** `CGEventTap` | CGEventTap cần quyền Accessibility |
| Active app | exe path qua PID | `NSWorkspace.frontmostApplication.bundleIdentifier` | Blocklist dùng bundle ID |
| Giả lập phím | `SendInput` (không cần quyền) | `CGEventPost` (Cmd+V) — **cần quyền Accessibility** | User phải cấp quyền thủ công |
| Overlay không cướp focus | `WS_EX_NOACTIVATE` | **NSPanel** `.nonactivatingPanel` — Wails v2 không expose tốt, **Wails v3-alpha mới có** | Điểm khó nhất; có thể cần v3 |
| Đọc OS API từ Go | `syscall`/`LazyDLL` thuần | cgo + Objective-C | Build phức tạp hơn |
| Ảnh | CF_DIBV5 | `NSPasteboardTypePNG` / `NSPasteboardTypeTIFF` | |
| Ký & phân phối | Authenticode | **Developer ID + notarization**; **không sandbox được** (CGEventPost bị chặn) → khó lên Mac App Store, phân phối qua `.dmg` | |

## 3. Hai thách thức "đắt" nhất của macOS

### 3.1. Polling thay vì event
macOS không có notification khi clipboard đổi. Phải chạy timer poll
`NSPasteboard.general.changeCount`; nếu khác lần trước → đọc nội dung mới.

- Timer interval ~250–500ms (cân bằng độ trễ vs CPU khi idle).
- Có thể giảm tần suất poll khi app không ở foreground để tiết kiệm pin.
- Implementation gọi `OnClipboardChange()` của `PlatformEvents` → Core Engine xử lý
  giống hệt đường Windows.

### 3.2. Quyền Accessibility (Accessibility / Input Monitoring)
Cần cho cả `CGEventTap` (nghe global hotkey) lẫn `CGEventPost` (giả lập Cmd+V).

- Check `AXIsProcessTrusted()` lúc khởi động.
- Nếu chưa được cấp: hiện màn **onboarding** hướng dẫn user mở
  *System Settings → Privacy & Security → Accessibility* và bật DevClip.
- Gọi `AXIsProcessTrustedWithOptions` với prompt để hệ thống tự mở dialog.
- App phải xử lý graceful khi quyền bị thu hồi giữa chừng (paste/hotkey fail mềm).

> Lưu ý: hotkey có thể dùng Carbon `RegisterEventHotKey` (KHÔNG cần Accessibility)
> để giảm phụ thuộc quyền cho riêng phần hotkey; nhưng `CGEventPost` để dán thì
> bắt buộc phải có quyền. Cân nhắc dùng `RegisterEventHotKey` cho hotkey + chỉ
> cần Accessibility cho paste.

## 4. Implementation `platform/darwin.go`

Hiện thực các method của `Platform`:

| Method | macOS API |
|---|---|
| `Start` | Khởi tạo timer poll `changeCount` + đăng ký hotkey; phát events |
| `Stop` | Dừng timer, gỡ event tap / hotkey |
| `ReadClipboard` | `NSPasteboard.general` đọc `string` / `NSPasteboardTypePNG` |
| `WriteClipboard` | `clearContents` + `setData:forType:` / `setString:` |
| `ForegroundApp` | `NSWorkspace.shared.frontmostApplication.bundleIdentifier` |
| `SimulatePaste` | `CGEventCreateKeyboardEvent` Cmd+V + `CGEventPost` |
| `CursorPos` | `NSEvent.mouseLocation` (lưu ý gốc toạ độ macOS ở góc dưới-trái) |
| `RegisterHotkey` | Carbon `RegisterEventHotKey` (khuyến nghị) hoặc `CGEventTap` |

**Restore clipboard sau paste:** giống Windows — lưu nội dung pasteboard cũ
(theo type), set item, post Cmd+V, sau ~200ms restore. Lưu ý `changeCount` sẽ
nhảy khi ta tự ghi → cần đánh dấu để poll loop không tự bắt lại item vừa paste.

## 5. Blocklist mặc định (macOS)

Theo bundle ID thay vì exe name:
- `com.1password.1password`
- `com.bitwarden.desktop`
- `com.apple.keychainaccess`
- (terminal là tùy chọn user tự thêm — KHÔNG cố đoán đang gõ password, giống Windows)

## 6. Overlay popup trên macOS

- Cần **NSPanel** với style mask `.nonactivatingPanel` để nổi mà không activate app.
- Wails v2 không expose NSPanel tốt → các lựa chọn:
  - **(A)** Dùng mô hình "nhớ app đích + kích hoạt app đích trước khi `CGEventPost`" (giống cách Windows đã chọn) — có thể chấp nhận window thường.
  - **(B)** Nâng phần macOS lên **Wails v3-alpha** (có hỗ trợ NSPanel) — đánh giá độ ổn định tại thời điểm làm.
- Quyết định cuối chốt ở phase M2 sau khi đánh giá Wails v3 lúc đó.

## 7. Phase macOS

- **M0 — Spike:** Wails build macOS + cgo/objc "hello" + poll `changeCount` log ra console + flow xin quyền Accessibility (`AXIsProcessTrusted`). *(khử rủi ro lớn nhất: cgo + quyền)*
- **M1 — Platform impl:** hiện thực đầy đủ `darwin/platform.go` (poll clipboard text+image, foreground app theo bundleID, `CGEventPost` Cmd+V, `RegisterEventHotKey`). Core Engine tái dùng 100%. Test bằng manual checklist.
- **M2 — Overlay & permission UX:** NSPanel nonactivating (đánh giá Wails v3) + màn onboarding cấp quyền Accessibility + xử lý thu hồi quyền giữa chừng.
- **M3 — Distribution:** Developer ID signing + notarization + đóng gói `.dmg` + (tùy chọn) auto-update.

## 8. Rủi ro riêng của macOS

| Rủi ro | Giải pháp |
|---|---|
| User không cấp Accessibility → app vô dụng | Onboarding rõ ràng + check `AXIsProcessTrusted` + nút "Mở System Settings" |
| Poll loop tự bắt lại item vừa paste | Đánh dấu `changeCount` của lần tự ghi, bỏ qua |
| cgo/objc linker error khi build | Set `CGO_LDFLAGS="-framework AppKit -framework Carbon ..."` |
| Toạ độ chuột gốc khác Windows (góc dưới-trái) | Quy đổi khi định vị popup |
| Notarization bị Apple từ chối | Tuân thủ hardened runtime + entitlement tối thiểu |
| Wails v2 thiếu NSPanel | Phương án (A) hoặc nâng v3 ở M2 |

## 9. Out of scope (macOS v1)

- Mac App Store (sandbox chặn CGEventPost).
- iCloud sync (trái triết lý in-memory).
- Hỗ trợ macOS quá cũ (target macOS 12+ trở lên).
