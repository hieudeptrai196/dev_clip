<div align="center">

<img src="build/appicon.png" alt="DevClip logo" width="120" />

# DevClip

**Clipboard manager chạy ngầm dành cho lập trình viên — bản `Win + V` thông minh hơn.**

[English](README.md) · Tiếng Việt

![Platform](https://img.shields.io/badge/platform-Windows-0078d4)
![Built with](https://img.shields.io/badge/built%20with-Go%20%2B%20Wails%20%2B%20React-22b24c)
![Status](https://img.shields.io/badge/status-active-success)

</div>

> Copy bất cứ thứ gì, bấm **Alt + V** rồi dán lại — kèm các "siêu năng lực" cho dev như format JSON/SQL, biến đổi chuỗi, ghim và preview toàn màn hình.

<!-- TODO: thay bằng GIF demo thật -->
<!-- ![demo](docs/demo.gif) -->

---

## ✨ Điểm nổi bật

- 🧑‍💻 **Hiểu định dạng** — tự nhận diện **JSON** và **SQL**, pretty‑print khi dán.
- 🔤 **Biến đổi chuỗi** — `UPPER`, `lower`, `camelCase`, `snake_case`, `kebab-case`, **Base64**.
- 📌 **Ghim & sắp xếp** — ghim item lên đầu và **kéo‑thả** để xếp lại thứ tự.
- ⌨️ **Hotkey tuỳ chỉnh + tìm kiếm** — đổi phím tắt toàn cục và lọc lịch sử khi gõ.
- 🔍 **Preview toàn màn hình** — mở full ảnh hoặc text dài trong popup lớn.
- 🗔 **Nằm ở khay hệ thống** — dán thẳng vào cửa sổ bạn đang làm việc.

---

## 🆚 DevClip so với Windows + V

| | Windows + V | DevClip |
|---|:---:|:---:|
| Đối tượng | Người dùng phổ thông | **Lập trình viên** |
| Dung lượng lịch sử | — | **10–500 (tuỳ chỉnh)** |
| Hotkey | Cố định `Win+V` | **Đổi được** |
| Tìm kiếm | ❌ | ✅ |
| Format code (JSON/SQL) | ❌ | ✅ |
| Biến đổi chuỗi | ❌ | ✅ |
| Ghim | ✅ | ✅ + **kéo‑thả sắp xếp** |
| Preview toàn màn hình | ❌ | ✅ |
| Mã nguồn mở | ❌ | ✅ |

> DevClip không nhằm thay thế hoàn toàn Win + V — nó tối ưu cho luồng làm việc của **lập trình viên**.

---

## 🚀 Cài đặt

> Windows 10/11, 64‑bit.

**Cách 1 — Installer (khuyên dùng)**
1. Tải **`DevClip-amd64-installer.exe`** ở [bản phát hành mới nhất](https://github.com/hieudeptrai196/dev_clip/releases/latest).
2. Chạy file và làm theo các bước. App được cài kèm shortcut ở Start Menu.

**Cách 2 — Portable**
- Tải **`DevClip.exe`** ở [bản phát hành mới nhất](https://github.com/hieudeptrai196/dev_clip/releases/latest) và chạy thẳng — không cần cài.

> ℹ️ App **chưa ký số** nên lần đầu chạy Windows SmartScreen có thể cảnh báo. Bấm **More info → Run anyway**.

---

## 📖 Cách dùng

| Phím tắt | Tác dụng |
|---|---|
| `Alt + V` | Mở popup tại con trỏ |
| `↑` / `↓` | Di chuyển lựa chọn |
| `Enter` | Dán item đang chọn |
| `Esc` | Đóng |

1. **Copy** bất cứ thứ gì (text hoặc ảnh) như bình thường.
2. Bấm **`Alt + V`** — popup hiện ngay tại con trỏ kèm lịch sử.
3. Chọn một item rồi bấm **Enter** (hoặc click) để dán vào app bạn đang làm.
4. **Hover** vào item để ghim 📌 / xoá 🗑; bấm icon **con mắt** để xem preview toàn màn hình.
5. **Icon ở khay hệ thống** có menu chuột phải (Show / Settings / Quit); double‑click để mở popup.
6. Vào **⚙ Settings** để đổi hotkey, dung lượng lịch sử, hoặc bật khởi động cùng Windows.

---

## 🎯 Tính năng

**Bắt clipboard**
- Tự động bắt **text và ảnh** khi copy.
- Lịch sử giới hạn (10–500) có **chống trùng** — copy lại item cũ thì nó nhảy lên đầu.
- Thumbnail ảnh tải khi cần.

**Popup & dán**
- Hotkey toàn cục mở popup **ngay tại con trỏ**; tìm kiếm tức thì, không phân biệt hoa thường.
- Dán vào **cửa sổ đang làm việc trước đó** và **khôi phục clipboard cũ** sau khi dán.
- Cửa sổ frameless, luôn nổi trên cùng, trong mờ; **kéo để di chuyển**.

**Thao tác item**
- **Ghim** lên đầu (không bị giới hạn lịch sử đẩy ra) và **kéo‑thả sắp xếp** item đã ghim.
- **Xoá** từng item hoặc **Clear all**.
- **Preview toàn màn hình** cho text dài và ảnh.

**Formatter (cho dev)**
- Nhận diện **JSON / SQL** và gắn badge.
- Pretty‑print **JSON** (thụt lề) và **SQL** (viết hoa từ khoá, xuống dòng theo mệnh đề).
- Biến đổi & dán: `UPPER`, `lower`, `camelCase`, `snake_case`, `kebab-case`, **Base64**.

**Tích hợp hệ thống**
- Icon ở **khay hệ thống** có menu + double‑click để mở.
- **Chỉ một bản chạy**, **thu nhỏ xuống tray** khi đóng, tuỳ chọn **khởi động cùng Windows**.

---

## 👤 Tác giả

Thực hiện bởi [**@hieudeptrai196**](https://github.com/hieudeptrai196).

Xây dựng bằng [Go](https://go.dev), [Wails](https://wails.io) và [React](https://react.dev).
