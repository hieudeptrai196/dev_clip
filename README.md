<div align="center">

<img src="build/appicon.png" alt="DevClip logo" width="120" />

# DevClip

**A background clipboard manager built for developers — as simple as `Win + V`, just smarter.**

<img src="docs/banner.png" alt="DevClip — Copy anything. Paste it smarter." width="100%" />

English · [Tiếng Việt](README.vi.md)

![Platform](https://img.shields.io/badge/platform-Windows-0078d4)
![Built with](https://img.shields.io/badge/built%20with-Go%20%2B%20Wails%20%2B%20React-22b24c)
![Status](https://img.shields.io/badge/status-active-success)

</div>

> Nothing to set up and nothing to learn: copy anything, press **Alt + V**, and paste it back. The developer super‑powers — JSON/SQL formatting, case transforms, JWT decoding, pinning and fullscreen preview — are there the moment you need them.

## 📸 Screenshots

### Local Web Dashboard
![Web Dashboard](docs/dashboard.png)

### Desktop App
| Clipboard History | Settings |
|---|---|
| ![Clipboard History](docs/history.png) | ![Settings](docs/settings.png) |


---

## ✨ Highlights

- 🧑‍💻 **Format‑aware** — auto‑detects **JSON**, **SQL**, **JWT**, and **Timestamps** with automatic badge labeling.
- 🔑 **JWT Decoder** — decodes tokens into pretty‑printed Header & Payload split-view alongside a live expiration status banner.
- 📅 **Epoch Converter** — instantly converts Unix timestamps to local dates, and local dates back to epoch values.
- 🗜️ **JSON/SQL Minifier** — compresses JSON and SQL snippets into a single line on paste.
- 📊 **Local Web Dashboard** — opens a premium glassmorphic web dashboard showing session stats and real-time activity/format charts.
- 📌 **Pin & reorder** — pin items to the top and **drag‑and‑drop** to arrange them.
- ⌨️ **Custom hotkey + search** — change the global shortcut and filter history as you type.
- 🔍 **Fullscreen preview** — open the full image, long text, or decoded JWT in a large popup.

---

## 🆚 How DevClip compares

DevClip stands on three pillars the alternatives don't combine:

- 🔒 **Private by design** — history lives only in RAM, evaporates when you quit, and sensitive apps (password managers) are auto‑ignored. Win + V, Raycast and CopyQ all persist history to disk.
- 🧑‍💻 **Built for code** — auto‑detects and decodes JSON, SQL, JWT and timestamps, plus case transforms and minify. The others are general‑purpose.
- 🆓 **Free, open & light** — no subscription, no account, a small Go + Wails footprint.
- ⚡ **Familiar & zero‑setup** — it works just like Win + V: press the hotkey, pick, paste. No config files, no concepts to learn, none of CopyQ's complexity. The dev tools are there when you need them and out of the way when you don't.

| | Windows + V | Raycast | CopyQ | **DevClip** |
|---|:---:|:---:|:---:|:---:|
| Platform | Windows | macOS + Windows *(beta)* | Win / Mac / Linux | Windows |
| Price / license | Built‑in | Freemium, Pro $8+/mo | Free, open‑source | **Free, open‑source** |
| Focus | General users | General launcher | Power clipboard | **Developer clipboard** |
| History storage | Disk | Encrypted disk | Disk | **RAM only** ⭐ |
| Clears on exit | ❌ | ❌ | ❌ | **✅** ⭐ |
| Auto‑ignore password managers | ❌ | ❌ | ❌ | **✅** ⭐ |
| JSON / SQL format + minify | ❌ | ❌ | scripting | **✅** ⭐ |
| JWT decoder | ❌ | ❌ | ❌ | **✅** ⭐ |
| Epoch ⇄ date converter | ❌ | ❌ | ❌ | **✅** ⭐ |
| String transforms | ❌ | via extensions | via scripting | **✅ built‑in** |
| Search | ❌ | ✅ | ✅ | ✅ |
| Pin / reorder | ✅ | ✅ | ✅ (tabs) | ✅ + **drag** |
| Custom hotkey | ❌ | ✅ | ✅ | ✅ |
| Footprint | OS built‑in | Heavier | Light (Qt) | **Light (Go + Wails)** |
| Setup / learning curve | None | Low | **Steep** | **None — works like Win + V** |

<sub>⭐ = unique to DevClip among these tools.</sub>

**Which should you pick?**

- **Raycast** — an all‑in‑one launcher with AI, best on macOS.
- **CopyQ** — cross‑platform with deep scripting and a long‑term archive.
- **DevClip** — a private, developer‑focused clipboard for Windows with zero setup.

---

## 🚀 Installation

> Windows 10/11, 64‑bit.

**[⬇️ Download from Google Drive](https://drive.google.com/drive/folders/11JG4dXV7Vn4KnnRNvwhoQ0_kt8oleWjn?usp=sharing)** — the folder contains both:

- **`DevClip-amd64-installer.exe`** — installer (recommended): run it and follow the steps; it adds a Start‑menu shortcut.
- **`DevClip.exe`** — portable: run it directly, no install needed.

> ℹ️ The app isn’t code‑signed yet, so Windows SmartScreen may warn on first launch. Click **More info → Run anyway**.

---

## 📖 Usage

| Shortcut / Action | Action |
|---|---|
| `Alt + V` | Open the popup at the cursor |
| `↑` / `↓` | Move selection |
| `Enter` | Paste the selected item |
| `Esc` | Close |
| **Dashboard Button** 📊 | Click the icon next to settings gear to open the local web dashboard |

1. **Copy** anything (text or image) as usual.
2. Press **`Alt + V`** — the popup appears at your cursor with the history.
3. Pick an item and press **Enter** (or click) to paste it into the app you were in.
4. **Hover** an item to pin 📌 / delete 🗑; click the **eye** icon for a fullscreen preview / JWT split-view.
5. The **tray icon** has a right‑click menu (Show / Settings / Quit); double‑click it to open the popup.
6. Open **⚙ Settings** to change the hotkey, history size, or enable start‑with‑Windows.

---

## 🎯 Features

**Clipboard capture**
- Automatically captures copied **text and images**.
- A capped history (10–500) with **de‑duplication** — re‑copying an item bumps it to the top.
- Image thumbnails loaded on demand.

**Popup & paste**
- Global hotkey opens the popup **at the cursor**; instant case‑insensitive search.
- Pastes into the **previously focused window** and **restores your previous clipboard** afterwards.
- Frameless, always‑on‑top, translucent window; **drag to move**.

**Item actions**
- **Pin** to top (survives history limits) and **drag‑to‑reorder** pinned items.
- **Delete** a single item or **Clear all**.
- **Fullscreen preview** of long text, images, and decoded JWT tokens.

**Developer Tools**
- **JSON / SQL Formatter**: Pretty-print (indents) and **Minify** (compresses to 1 line) on paste.
- **JWT Decoder**: Decodes tokens into beautiful Header/Payload panels with live expiration banner showing remaining/elapsed time.
- **Timestamp Converter**: Displays numeric copy values as Epoch timestamps, enabling one-click paste as local Date, or converting local Dates to Epochs.
- **Local Web Dashboard**: Bounds an HTTP server to a free local port, serving interactive format distributions, hourly copy line charts, and active clip logs.

**System integration**
- **System‑tray** icon with menu + double‑click to open.
- **Single instance** (bypassed for Wails CLI bindings), **minimize‑to‑tray** on close, optional **start with Windows**.

---

## ⭐ Support

If you find DevClip useful, please consider giving it a **star** ⭐ — it really helps!

Or **buy me a coffee** ☕ by scanning the QR code below:

<div align="center">
  <img src="docs/donate-qr.png" alt="Buy me a coffee" width="220" />
</div>

## 👤 Author

Made by [**@hieudeptrai196**](https://github.com/hieudeptrai196).

Built with [Go](https://go.dev), [Wails](https://wails.io) and [React](https://react.dev).
