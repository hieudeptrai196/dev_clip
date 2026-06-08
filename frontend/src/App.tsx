import { useEffect, useRef, useState, useCallback } from "react";
import { History, PasteItem, Hide, Thumbnail, FormatItem, PasteTransformed, PasteFormatted, Snippets, SnippetPlaceholders, PasteSnippet, GetSettings, SaveSettings, TogglePin, ReorderPins, DeleteItem, ClearAll } from "../wailsjs/go/main/App";
import { EventsOn, WindowFullscreen, WindowUnfullscreen, BrowserOpenURL } from "../wailsjs/runtime/runtime";
import type { ClipItem, AppSettings } from "./types";
import { snippet } from "../wailsjs/go/models";
import { Icon } from "./Icon";
import "./App.css";

const MODIFIERS = [
  { label: "Alt", value: 1 },
  { label: "Ctrl", value: 2 },
  { label: "Shift", value: 4 },
  { label: "Ctrl+Alt", value: 3 },
  { label: "Ctrl+Shift", value: 6 },
  { label: "Alt+Shift", value: 5 },
  { label: "Ctrl+Alt+Shift", value: 7 },
];

const KEYS = [
  ...Array.from({ length: 26 }, (_, i) => ({
    label: String.fromCharCode(65 + i),
    value: 65 + i,
  })),
  ...Array.from({ length: 10 }, (_, i) => ({
    label: String.fromCharCode(48 + i),
    value: 48 + i,
  })),
  ...Array.from({ length: 12 }, (_, i) => ({
    label: `F${i + 1}`,
    value: 0x70 + i,
  })),
  { label: "`", value: 0xC0 },
  { label: "Space", value: 0x20 },
];

function getModifierLabel(mod: number): string {
  const parts: string[] = [];
  if (mod & 2) parts.push("Ctrl");
  if (mod & 1) parts.push("Alt");
  if (mod & 4) parts.push("Shift");
  return parts.join("+");
}

function getKeyLabel(key: number): string {
  if (key >= 0x41 && key <= 0x5A) {
    return String.fromCharCode(key);
  }
  if (key >= 0x30 && key <= 0x39) {
    return String.fromCharCode(key);
  }
  if (key >= 0x70 && key <= 0x7B) {
    return "F" + (key - 0x70 + 1);
  }
  if (key === 0xC0) return "`";
  if (key === 0x20) return "Space";
  return "0x" + key.toString(16).toUpperCase();
}

function getHotkeyLabel(mod: number, key: number): string {
  const modLabel = getModifierLabel(mod);
  const keyLabel = getKeyLabel(key);
  return modLabel ? `${modLabel}+${keyLabel}` : keyLabel;
}

// Transform ops shown as action buttons for text items.
const TRANSFORM_OPS: { label: string; op: string }[] = [
  { label: "Aa", op: "upper" },
  { label: "aa", op: "lower" },
  { label: "camel", op: "camel" },
  { label: "snake", op: "snake" },
  { label: "kebab", op: "kebab" },
  { label: "b64", op: "base64encode" },
];

function App() {
  // ── Tab state ──
  const [tab, setTab] = useState<"recent" | "snippets" | "settings">("recent");

  // ── Recent tab state ──
  const [items, setItems] = useState<ClipItem[]>([]);
  const [query, setQuery] = useState("");
  const [sel, setSel] = useState(0);
  const [thumbs, setThumbs] = useState<Record<number, string>>({});
  const searchRef = useRef<HTMLInputElement>(null);
  const itemRefs = useRef<(HTMLLIElement | null)[]>([]);
  // Drag-to-reorder state for pinned items (holds the id being dragged).
  const [dragId, setDragId] = useState<number | null>(null);
  // Item whose full content is shown in the large preview popup (null = closed).
  const [previewItem, setPreviewItem] = useState<ClipItem | null>(null);
  // True while the preview has put the window into fullscreen (so we know to
  // restore it). A ref so the blur handler reads the live value.
  const fullscreenRef = useRef(false);

  // ── Snippets tab state ──
  const [snippetList, setSnippetList] = useState<snippet.Snippet[]>([]);
  const [snippetSel, setSnippetSel] = useState(0);
  const snippetRefs = useRef<(HTMLLIElement | null)[]>([]);
  // Placeholder form state
  const [phSnippet, setPhSnippet] = useState<snippet.Snippet | null>(null);
  const [phValues, setPhValues] = useState<Record<string, string>>({});
  const [phNames, setPhNames] = useState<string[]>([]);
  const firstPhRef = useRef<HTMLInputElement>(null);

  // ── Settings tab state ──
  const [settings, setSettings] = useState<AppSettings | null>(null);
  const [hotkeyMod, setHotkeyMod] = useState<number>(1);
  const [hotkeyKey, setHotkeyKey] = useState<number>(0x56);
  const [maxItems, setMaxItems] = useState<number>(100);
  const [autoStart, setAutoStart] = useState<boolean>(false);
  // Unified save status. Rendered in a fixed-height bar so it never shifts
  // layout (the old appear/disappear alerts caused a one-frame flicker).
  const [saveStatus, setSaveStatus] = useState<{ kind: "idle" | "saving" | "success" | "error"; msg: string }>({ kind: "idle", msg: "" });
  const statusTimer = useRef<number>(0);

  // Suppress the blur-to-hide for a short grace period right after the popup is
  // shown, otherwise the window fires a transient `blur` during the show
  // transition and the popup closes itself immediately on Alt+V.
  const ignoreBlurUntil = useRef(0);

  const refresh = useCallback(async () => {
    const list = (await History()) as unknown as ClipItem[];
    setItems(list ?? []);
  }, []);

  const loadSettings = useCallback(async () => {
    try {
      const s = await GetSettings();
      if (s) {
        setSettings(s);
        setHotkeyMod(s.hotkey_mod);
        setHotkeyKey(s.hotkey_key);
        setMaxItems(s.max_items);
        setAutoStart(s.auto_start);
        setSaveStatus({ kind: "idle", msg: "" });
      }
    } catch (err: any) {
      setSaveStatus({ kind: "error", msg: "Failed to load settings: " + (err.message || err) });
    }
  }, []);

  // Persist a settings change with smooth loading -> saved feedback (no flicker).
  const runSave = useCallback(async (updated: AppSettings, successMsg: string) => {
    window.clearTimeout(statusTimer.current);
    setSaveStatus({ kind: "saving", msg: "Saving…" });
    try {
      await SaveSettings(updated);
      setSettings(updated);
      setSaveStatus({ kind: "success", msg: successMsg });
      statusTimer.current = window.setTimeout(
        () => setSaveStatus({ kind: "idle", msg: "" }),
        1800
      );
    } catch (err: any) {
      setSaveStatus({ kind: "error", msg: err.message || String(err) });
    }
  }, []);

  const loadSnippets = useCallback(async () => {
    const list = await Snippets() as unknown as snippet.Snippet[];
    setSnippetList(list ?? []);
    setSnippetSel(0);
  }, []);

  useEffect(() => {
    refresh();
    const offClip = EventsOn("clip:changed", refresh);
    const offShow = EventsOn("popup:show", () => {
      ignoreBlurUntil.current = Date.now() + 600;
      refresh();
      setQuery("");
      setSel(0);
      setPhSnippet(null);
      setPhValues({});
      setPhNames([]);
      // If on snippets tab, reload snippets too
      if (tab === "snippets") {
        loadSnippets();
      }
      setTimeout(() => searchRef.current?.focus(), 50);
    });
    const offSettingsShow = EventsOn("settings:show", () => {
      ignoreBlurUntil.current = Date.now() + 600;
      setTab("settings");
      setPhSnippet(null);
      setPhValues({});
      setPhNames([]);
      loadSettings();
    });
    return () => {
      offClip();
      offShow();
      offSettingsShow();
    };
  }, [refresh, loadSnippets, loadSettings, tab]);

  // Load snippets or settings when switching tabs
  useEffect(() => {
    if (tab === "snippets") {
      loadSnippets();
    } else if (tab === "settings") {
      loadSettings();
    }
  }, [tab, loadSnippets, loadSettings]);

  // Feature 2: auto-hide when the window loses focus (Win+V behavior)
  useEffect(() => {
    const onBlur = () => {
      if (Date.now() < ignoreBlurUntil.current) return; // grace period after show
      if (tab === "settings") return; // ignore blur auto-hide in settings to prevent flickering on dropdown click

      // A transient blur during the show transition is immediately followed by
      // focus coming back. Debounce and re-check real focus before hiding so we
      // only close on a genuine click-away, not a focus bounce.
      window.setTimeout(() => {
        if (Date.now() < ignoreBlurUntil.current) return;
        if (!document.hasFocus()) {
          // If the preview put us in fullscreen, exit it before hiding so the
          // next Alt+V opens the normal small popup.
          if (fullscreenRef.current) {
            WindowUnfullscreen();
            fullscreenRef.current = false;
            setPreviewItem(null);
          }
          Hide();
        }
      }, 120);
    };
    window.addEventListener("blur", onBlur);
    return () => window.removeEventListener("blur", onBlur);
  }, [tab]);

  // Lazily fetch a small base64 thumbnail for each image item once.
  useEffect(() => {
    items.forEach((it) => {
      if (it.kind === 1 && thumbs[it.id] === undefined) {
        Thumbnail(it.id).then((url) =>
          setThumbs((prev) => ({ ...prev, [it.id]: url }))
        );
      }
    });
  }, [items, thumbs]);

  // Autofocus first placeholder input when form opens
  useEffect(() => {
    if (phSnippet && phNames.length > 0) {
      setTimeout(() => firstPhRef.current?.focus(), 30);
    }
  }, [phSnippet, phNames]);

  // ── Recent tab logic ──
  const filtered = items.filter((it) =>
    it.preview.toLowerCase().includes(query.toLowerCase())
  );
  const clampedSel = Math.min(sel, Math.max(0, filtered.length - 1));

  useEffect(() => {
    const el = itemRefs.current[clampedSel];
    el?.scrollIntoView({ block: "nearest" });
  }, [clampedSel, filtered.length]);

  const doPaste = async (item: ClipItem) => {
    await PasteItem(item.id);
    await Hide();
    setQuery("");
    setSel(0);
  };

  const doTransform = async (item: ClipItem, op: string) => {
    await PasteTransformed(item.id, op);
    await Hide();
    setQuery("");
    setSel(0);
  };

  const doPretty = async (item: ClipItem) => {
    await PasteFormatted(item.id);
    await Hide();
    setQuery("");
    setSel(0);
  };

  // ── Pin + drag-to-reorder logic ──
  const doTogglePin = async (item: ClipItem) => {
    await TogglePin(item.id);
    await refresh();
  };

  const onDragStart = (item: ClipItem) => {
    if (item.pinned) setDragId(item.id);
  };

  const onDragOverItem = (e: React.DragEvent, item: ClipItem) => {
    // Only pinned rows are valid drop targets while a pinned drag is active.
    if (dragId !== null && item.pinned) e.preventDefault();
  };

  const onDropItem = async (e: React.DragEvent, target: ClipItem) => {
    e.preventDefault();
    if (dragId === null || !target.pinned || target.id === dragId) {
      setDragId(null);
      return;
    }
    const pinnedIds = items.filter((i) => i.pinned).map((i) => i.id);
    const from = pinnedIds.indexOf(dragId);
    const to = pinnedIds.indexOf(target.id);
    if (from === -1 || to === -1) {
      setDragId(null);
      return;
    }
    pinnedIds.splice(from, 1);
    pinnedIds.splice(to, 0, dragId);
    setDragId(null);
    await ReorderPins(pinnedIds);
    await refresh();
  };

  // ── Snippets tab logic ──
  const openSnippet = async (sn: snippet.Snippet) => {
    const names = (await SnippetPlaceholders(sn.id) as unknown as string[]) ?? [];
    if (names.length === 0) {
      await PasteSnippet(sn.id, {});
      await Hide();
    } else {
      const defaults: Record<string, string> = {};
      names.forEach((n) => { defaults[n] = ""; });
      setPhSnippet(sn);
      setPhNames(names);
      setPhValues(defaults);
    }
  };

  const doPasteSnippet = async () => {
    if (!phSnippet) return;
    await PasteSnippet(phSnippet.id, phValues);
    await Hide();
    setPhSnippet(null);
    setPhValues({});
    setPhNames([]);
  };

  const cancelPhForm = () => {
    setPhSnippet(null);
    setPhValues({});
    setPhNames([]);
  };

  const handleApplyHotkey = async () => {
    if (!settings) return;
    const label = getHotkeyLabel(hotkeyMod, hotkeyKey);
    await runSave(
      { ...settings, hotkey_mod: hotkeyMod, hotkey_key: hotkeyKey, hotkey_label: label },
      `Hotkey set to ${label}`
    );
  };

  const handleToggleAutoStart = async (val: boolean) => {
    if (!settings) return;
    setAutoStart(val);
    await runSave({ ...settings, auto_start: val }, `Auto-start ${val ? "enabled" : "disabled"}`);
  };

  const handleCapacityChange = async (val: number) => {
    if (!settings) return;
    setMaxItems(val);
    await runSave({ ...settings, max_items: val }, "Capacity saved (applies on restart)");
  };

  // ── Preview popup: go fullscreen while viewing, restore on close. Wails'
  // WindowUnfullscreen restores the exact previous size/position, so this is
  // immune to the off-screen/DPI issues of manual resize+center. ──
  const openPreview = async (item: ClipItem) => {
    // Ensure the full-size image is available before showing the popup.
    if (item.kind === 1 && thumbs[item.id] === undefined) {
      const url = await Thumbnail(item.id);
      setThumbs((prev) => ({ ...prev, [item.id]: url }));
    }
    WindowFullscreen();
    fullscreenRef.current = true;
    setPreviewItem(item);
  };

  const closePreview = () => {
    if (fullscreenRef.current) {
      WindowUnfullscreen();
      fullscreenRef.current = false;
    }
    setPreviewItem(null);
  };

  // ── Delete / clear-all ──
  const doDelete = async (item: ClipItem) => {
    if (previewItem && previewItem.id === item.id) closePreview();
    await DeleteItem(item.id);
    await refresh();
  };

  const doClearAll = async () => {
    closePreview();
    await ClearAll();
    setSel(0);
    await refresh();
  };



  // ── Keyboard handler ──
  const onKeyDown = async (e: React.KeyboardEvent) => {
    // While the preview popup is open, Escape just closes it.
    if (previewItem) {
      if (e.key === "Escape") { e.preventDefault(); closePreview(); }
      return;
    }
    if (tab === "recent") {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSel((s) => Math.min(s + 1, filtered.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setSel((s) => Math.max(s - 1, 0));
      } else if (e.key === "Enter" && filtered[clampedSel]) {
        await doPaste(filtered[clampedSel]);
      } else if (e.key === "Escape") {
        await Hide();
      }
    } else if (tab === "snippets") {
      if (e.key === "Escape") {
        if (phSnippet) {
          cancelPhForm();
        } else {
          await Hide();
        }
      } else if (!phSnippet) {
        if (e.key === "ArrowDown") {
          e.preventDefault();
          setSnippetSel((s) => Math.min(s + 1, snippetList.length - 1));
        } else if (e.key === "ArrowUp") {
          e.preventDefault();
          setSnippetSel((s) => Math.max(s - 1, 0));
        } else if (e.key === "Enter" && snippetList[snippetSel]) {
          await openSnippet(snippetList[snippetSel]);
        }
      }
    } else if (tab === "settings") {
      if (e.key === "Escape") {
        await Hide();
      }
    }
  };

  useEffect(() => {
    const el = snippetRefs.current[snippetSel];
    el?.scrollIntoView({ block: "nearest" });
  }, [snippetSel, snippetList.length]);

  return (
    <div className="panel" onKeyDown={onKeyDown}>
      {/* Settings (Gear) toggle button */}
      <button
        className={`settings-toggle-btn${tab === "settings" ? " active" : ""}`}
        onClick={() => {
          if (tab === "settings") {
            setTab("recent");
            setTimeout(() => searchRef.current?.focus(), 30);
          } else {
            setTab("settings");
          }
        }}
        title="Settings"
        aria-label="Settings"
      >
        <Icon name="settings" size={16} />
      </button>

      {/* Close button in top-right corner */}
      <button className="close-btn" onClick={() => Hide()} aria-label="Close">
        <Icon name="close" size={16} />
      </button>

      {/* Tab switcher */}
      <div className="tabs">
        <button
          className={`tab${tab === "recent" ? " tab--active" : ""}`}
          onClick={() => { setTab("recent"); setTimeout(() => searchRef.current?.focus(), 30); }}
        >
          Recent
        </button>
        {/* Snippets feature temporarily disabled
        <button
          className={`tab${tab === "snippets" ? " tab--active" : ""}`}
          onClick={() => setTab("snippets")}
        >
          Snippets
        </button>
        */}
      </div>

      {tab === "recent" && (
        <>
          <div className="search-wrap">
            <input
              ref={searchRef}
              className="search-input"
              autoFocus
              placeholder="Search clipboard..."
              value={query}
              onChange={(e) => {
                setQuery(e.target.value);
                setSel(0);
              }}
            />
            {items.length > 0 && (
              <button
                className="clear-all-btn"
                title="Xoá toàn bộ lịch sử"
                onClick={(e) => { e.stopPropagation(); doClearAll(); }}
              >
                Clear all
              </button>
            )}
          </div>

          <ul className="clip-list">
            {filtered.length === 0 && (
              <li className="empty-state">No clipboard history yet</li>
            )}
            {filtered.map((it, i) => {
              const isSelected = i === clampedSel;
              const isText = it.kind === 0;
              const hasFmt = isText && (it.format === "json" || it.format === "sql");
              const isImage = it.kind === 1;
              // Show a Preview button for images and for text that was truncated.
              const canPreview = isImage || (isText && it.text.length > it.preview.length);
              return (
                <li
                  key={it.id}
                  ref={(el) => { itemRefs.current[i] = el; }}
                  className={`clip-item${isSelected ? " selected" : ""}${it.pinned ? " pinned" : ""}${dragId === it.id ? " dragging" : ""}`}
                  draggable={it.pinned}
                  onDragStart={() => onDragStart(it)}
                  onDragOver={(e) => onDragOverItem(e, it)}
                  onDrop={(e) => onDropItem(e, it)}
                  onDragEnd={() => setDragId(null)}
                  onClick={() => doPaste(it)}
                  onMouseEnter={() => setSel(i)}
                >
                  <div className="item-row">
                    <div className="item-main">
                      {isImage ? (
                        <span className="image-label">
                          {thumbs[it.id] ? (
                            <img className="thumb" src={thumbs[it.id]} alt="clipboard image" />
                          ) : (
                            <span className="image-icon"><Icon name="image" size={20} /></span>
                          )}
                        </span>
                      ) : (
                        <span className="item-text">{it.preview}</span>
                      )}
                    </div>

                    {/* Function buttons — revealed only on hover/selection */}
                    <div className="item-actions" onClick={(e) => e.stopPropagation()}>
                      <button
                        className={`pin-btn${it.pinned ? " pinned" : ""}`}
                        title={it.pinned ? "Bỏ ghim" : "Ghim lên đầu"}
                        aria-label={it.pinned ? "Unpin" : "Pin"}
                        onClick={(e) => { e.stopPropagation(); doTogglePin(it); }}
                      >
                        <Icon name="pin" size={14} />
                      </button>
                      <button
                        className="del-btn"
                        title="Xoá mục này"
                        aria-label="Delete"
                        onClick={(e) => { e.stopPropagation(); doDelete(it); }}
                      >
                        <Icon name="trash" size={14} />
                      </button>
                    </div>

                    {/* Labels (format badge, Preview) — always pinned hard right */}
                    <div className="item-labels">
                      {hasFmt && (
                        <span className={`fmt-badge fmt-badge--${it.format}`}>
                          {it.format.toUpperCase()}
                        </span>
                      )}
                      {canPreview && (
                        <button
                          className="preview-btn"
                          title="Xem nội dung đầy đủ"
                          aria-label="Preview"
                          onClick={(e) => { e.stopPropagation(); openPreview(it); }}
                        >
                          <Icon name="eye" size={15} />
                        </button>
                      )}
                    </div>
                  </div>
                  {isSelected && isText && (
                    <div
                      className="action-row"
                      onClick={(e) => e.stopPropagation()}
                    >
                      {TRANSFORM_OPS.map(({ label, op }) => (
                        <button
                          key={op}
                          className="action-btn"
                          title={op}
                          onClick={(e) => { e.stopPropagation(); doTransform(it, op); }}
                        >
                          {label}
                        </button>
                      ))}
                      {hasFmt && (
                        <button
                          className="action-btn action-btn--pretty"
                          title="Pretty print"
                          onClick={(e) => { e.stopPropagation(); doPretty(it); }}
                        >
                          Pretty
                        </button>
                      )}
                    </div>
                  )}
                </li>
              );
            })}
          </ul>
        </>
      )}

      {tab === "snippets" && (
        <>
          {phSnippet ? (
            <div className="ph-form">
              <div className="ph-form-title">{phSnippet.name}</div>
              {phNames.map((name, i) => (
                <div key={name} className="ph-field">
                  <label className="ph-label">{name}</label>
                  <input
                    ref={i === 0 ? firstPhRef : undefined}
                    className="ph-input"
                    value={phValues[name] ?? ""}
                    onChange={(e) => setPhValues((prev) => ({ ...prev, [name]: e.target.value }))}
                    onKeyDown={async (e) => {
                      if (e.key === "Enter") { e.preventDefault(); await doPasteSnippet(); }
                      if (e.key === "Escape") { e.preventDefault(); cancelPhForm(); }
                    }}
                    placeholder={`{{${name}}}`}
                  />
                </div>
              ))}
              <div className="ph-actions">
                <button className="ph-btn ph-btn--paste" onClick={doPasteSnippet}>Paste</button>
                <button className="ph-btn ph-btn--cancel" onClick={cancelPhForm}>Cancel</button>
              </div>
            </div>
          ) : (
            <ul className="clip-list">
              {snippetList.length === 0 && (
                <li className="empty-state">No snippets configured</li>
              )}
              {snippetList.map((sn, i) => (
                <li
                  key={sn.id}
                  ref={(el) => { snippetRefs.current[i] = el; }}
                  className={`clip-item${i === snippetSel ? " selected" : ""}`}
                  onClick={() => openSnippet(sn)}
                  onMouseEnter={() => setSnippetSel(i)}
                >
                  <span className="item-text">{sn.name}</span>
                </li>
              ))}
            </ul>
          )}
        </>
      )}

      {tab === "settings" && (
        <div className="settings-panel">
          <div className={`settings-status settings-status--${saveStatus.kind}`}>
            {saveStatus.kind === "saving" && <span className="status-spinner" />}
            {saveStatus.kind === "success" && <span className="status-icon"><Icon name="check" size={14} /></span>}
            {saveStatus.kind === "error" && <span className="status-icon"><Icon name="alert" size={14} /></span>}
            <span className="status-msg">{saveStatus.msg}</span>
          </div>

          <div className="settings-scroll">
            {/* Section 1: Hotkey Picker */}
            <div className="settings-section">
              <div className="settings-section-title"><Icon name="hotkey" size={14} /> Hotkey</div>
              <div className="settings-row">
                <select
                  className="settings-select"
                  value={hotkeyMod}
                  onChange={(e) => setHotkeyMod(Number(e.target.value))}
                >
                  {MODIFIERS.map((m) => (
                    <option key={m.value} value={m.value}>
                      {m.label}
                    </option>
                  ))}
                </select>
                <span className="settings-plus">+</span>
                <select
                  className="settings-select"
                  value={hotkeyKey}
                  onChange={(e) => setHotkeyKey(Number(e.target.value))}
                >
                  {KEYS.map((k) => (
                    <option key={k.value} value={k.value}>
                      {k.label}
                    </option>
                  ))}
                </select>
                <button className="settings-btn settings-btn--primary" onClick={handleApplyHotkey}>
                  Apply
                </button>
              </div>
            </div>

            {/* Section 2: Capacity */}
            <div className="settings-section">
              <div className="settings-section-title"><Icon name="capacity" size={14} /> History Capacity</div>
              <div className="capacity-control">
                <input
                  type="number"
                  className="settings-input settings-input--number"
                  min="10"
                  max="500"
                  value={maxItems}
                  onChange={(e) => {
                    const val = Number(e.target.value);
                    setMaxItems(val);
                  }}
                  onBlur={() => {
                    let val = maxItems;
                    if (isNaN(val) || val < 10) {
                      val = 10;
                    } else if (val > 500) {
                      val = 500;
                    }
                    setMaxItems(val);
                    handleCapacityChange(val);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault();
                      (e.target as HTMLInputElement).blur();
                    }
                  }}
                />
              </div>
              <div className="settings-help">Range: 10 – 500. Takes effect on restart.</div>
            </div>

            {/* Section 3: Auto-Start */}
            <div className="settings-section">
              <div className="settings-section-title"><Icon name="startup" size={14} /> Startup</div>
              <label className="toggle-label">
                <input
                  type="checkbox"
                  className="toggle-checkbox"
                  checked={autoStart}
                  onChange={(e) => handleToggleAutoStart(e.target.checked)}
                />
                <span className="toggle-text">Start with Windows</span>
              </label>
            </div>

            {/* Section 4: About */}
            <div className="settings-section settings-section--about">
              <div className="about-title">DevClip v1.0</div>
              <div className="about-desc">Alt+V clipboard manager for developers.</div>
            </div>
          </div>
        </div>
      )}

      {previewItem && (
        <div className="preview-overlay" onClick={closePreview}>
          <div className="preview-card" onClick={(e) => e.stopPropagation()}>
            <button
              className="preview-close"
              onClick={closePreview}
              aria-label="Close preview"
            >
              <Icon name="close" size={16} />
            </button>
            {previewItem.kind === 1 ? (
              thumbs[previewItem.id] ? (
                <img className="preview-image" src={thumbs[previewItem.id]} alt="preview" />
              ) : (
                <div className="preview-loading">Loading…</div>
              )
            ) : (
              <pre className="preview-text">{previewItem.text}</pre>
            )}
          </div>
        </div>
      )}

      <div className="hint-bar">
        <div className="hint-keys">
          {tab === "recent" && (
            <>
              <span>&#8593;&#8595; select</span>
              <span className="hint-sep">&middot;</span>
              <span>Enter paste</span>
              <span className="hint-sep">&middot;</span>
              <span>Esc close</span>
            </>
          )}
          {tab === "snippets" && !phSnippet && (
            <>
              <span>&#8593;&#8595; select</span>
              <span className="hint-sep">&middot;</span>
              <span>Enter open</span>
              <span className="hint-sep">&middot;</span>
              <span>Esc close</span>
            </>
          )}
          {tab === "snippets" && phSnippet && (
            <>
              <span>Enter paste</span>
              <span className="hint-sep">&middot;</span>
              <span>Esc back</span>
            </>
          )}
          {tab === "settings" && (
            <span>Esc close</span>
          )}
        </div>
        <button
          className="author-link"
          title="Author · github.com/hieudeptrai196"
          onClick={() => BrowserOpenURL("https://github.com/hieudeptrai196")}
        >
          <Icon name="github" size={13} />
          Author
        </button>
      </div>
    </div>
  );
}

export default App;
