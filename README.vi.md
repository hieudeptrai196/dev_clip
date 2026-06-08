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

## 📸 Ảnh chụp màn hình

### Dashboard Web Cục Bộ
![Web Dashboard](docs/dashboard.png)

### Ứng dụng Desktop
| Lịch sử Clipboard | Cài đặt |
|---|---|
| ![Lịch sử Clipboard](docs/history.png) | ![Cài đặt](docs/settings.png) |


---

## ✨ Điểm nổi bật

- 🧑‍💻 **Nhận biết định dạng** — tự động nhận diện **JSON**, **SQL**, **JWT**, và **Timestamp** (nhãn gắn tự động).
- 🔑 **Giải mã JWT** — hiển thị Header & Payload trực quan dạng split-view kèm banner trạng thái hết hạn thời gian thực.
- 📅 **Chuyển đổi Epoch** — chuyển đổi nhanh Unix Timestamp sang ngày giờ địa phương và ngược lại.
- 🗜️ **JSON/SQL Minifier** — nén code JSON và truy vấn SQL thành một dòng duy nhất khi dán.
- 📊 **Dashboard Web cục bộ** — mở trang dashboard giao diện glassmorphic dark-mode hiển thị biểu đồ phân bố và tần suất sao chép.
- 📌 **Ghim & sắp xếp** — ghim item lên đầu và **kéo‑thả** để xếp lại thứ tự.
- ⌨️ **Hotkey tuỳ chỉnh + tìm kiếm** — đổi phím tắt toàn cục và lọc lịch sử khi gõ.
- 🔍 **Preview toàn màn hình** — mở full ảnh, text dài, hoặc token JWT đã giải mã trong popup lớn.

---

## 🆚 DevClip so với Windows + V

| | Windows + V | DevClip |
|---|:---:|:---:|
| Đối tượng | Người dùng phổ thông | **Lập trình viên** |
| Dung lượng lịch sử | — | **10–500 (tuỳ chỉnh)** |
| Hotkey | Cố định `Win+V` | **Đổi được** |
| Tìm kiếm | ❌ | ✅ |
| Format code (JSON/SQL) | ❌ | ✅ **(Pretty & Nén)** |
| Bộ giải mã JWT | ❌ | ✅ **(Split‑view & hạn token)** |
| Bộ đổi Timestamp | ❌ | ✅ **(Epoch ⇄ Ngày cục bộ)** |
| Dashboard Web thống kê | ❌ | ✅ **(Giao diện web trực quan)** |
| Biến đổi chuỗi | ❌ | ✅ (`UPPER`, `camel`, `snake`, v.v.) |
| Ghim | ✅ | ✅ + **kéo‑thả sắp xếp** |
| Preview toàn màn hình | ❌ | ✅ |
| Mã nguồn mở | ❌ | ✅ |

> DevClip không nhằm thay thế hoàn toàn Win + V — nó tối ưu cho luồng làm việc của **lập trình viên**.

---

## 🚀 Cài đặt

> Windows 10/11, 64‑bit.

**Cách 1 — Installer (khuyên dùng)**
1. Tải **`DevClip-amd64-installer.exe`** từ [Google Drive](https://drive.google.com/file/d/1sjutFfZesOxkNCqL9t3olw5J80enEV1h/view?usp=sharing).
2. Chạy file và làm theo các bước. App được cài kèm shortcut ở Start Menu.

**Cách 2 — Portable**
- Tải **`DevClip.exe`** từ [Google Drive](https://drive.google.com/file/d/1-v_gISTK4fH9cEbBL1x_nczpcadOvSx8/view?usp=sharing) và chạy thẳng — không cần cài.

> ℹ️ App **chưa ký số** nên lần đầu chạy Windows SmartScreen có thể cảnh báo. Bấm **More info → Run anyway**.

---

## 📖 Cách dùng

| Phím tắt / Hành động | Tác dụng |
|---|---|
| `Alt + V` | Mở popup tại con trỏ |
| `↑` / `↓` | Di chuyển lựa chọn |
| `Enter` | Dán item đang chọn |
| `Esc` | Đóng |
| **Nút Dashboard** 📊 | Click vào biểu tượng bên cạnh Settings Gear để mở Dashboard trên trình duyệt web |

1. **Copy** bất cứ thứ gì (text hoặc ảnh) như bình thường.
2. Bấm **`Alt + V`** — popup hiện ngay tại con trỏ kèm lịch sử.
3. Chọn một item rồi bấm **Enter** (hoặc click) để dán vào app bạn đang làm.
4. **Hover** vào item để ghim 📌 / xoá 🗑; bấm icon **con mắt** để xem preview toàn màn hình hoặc xem JWT giải mã.
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
- **Preview toàn màn hình** cho text dài, hình ảnh, và token JWT.

**Công cụ cho lập trình viên**
- **JSON / SQL Formatter**: Pretty-print (căn lề đẹp) và **Minify** (nén thành 1 dòng) khi dán.
- **JWT Decoder**: Giải mã token thành Header/Payload trong preview với banner hiển thị thời gian sống còn lại của token.
- **Timestamp Converter**: Đọc số copy và hiển thị Epoch timestamp, hỗ trợ dán dạng ngày giờ local, hoặc chuyển chuỗi ngày giờ local sang Epoch.
- **Local Web Dashboard**: Chạy web server nội bộ hiển thị các thống kê định dạng, biểu đồ cột thời gian copy, và nhật ký lịch sử.

**Tích hợp hệ thống**
- Icon ở **khay hệ thống** có menu + double‑click để mở.
- **Chỉ một bản chạy**, **thu nhỏ xuống tray** khi đóng, tuỳ chọn **khởi động cùng Windows**.

---

## ⭐ Ủng hộ

Nếu thấy DevClip hữu ích, cho mình xin một **star** ⭐ nhé — rất có ý nghĩa với mình!

Hoặc **mời mình một ly cà phê** ☕ bằng cách quét mã QR bên dưới:

<div align="center">
  <img src="docs/donate-qr.png" alt="Buy me a coffee" width="220" />
</div>

## 👤 Tác giả

Thực hiện bởi [**@hieudeptrai196**](https://github.com/hieudeptrai196).

Xây dựng bằng [Go](https://go.dev), [Wails](https://wails.io) và [React](https://react.dev).
