import { useEffect, useRef, useState, useCallback } from "react";
import { History, PasteItem, Hide } from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import type { ClipItem } from "./types";
import "./App.css";

function App() {
  const [items, setItems] = useState<ClipItem[]>([]);
  const [query, setQuery] = useState("");
  const [sel, setSel] = useState(0);
  const searchRef = useRef<HTMLInputElement>(null);
  const itemRefs = useRef<(HTMLLIElement | null)[]>([]);
  // Suppress the blur-to-hide for a short grace period right after the popup is
  // shown, otherwise the window fires a transient `blur` during the show
  // transition and the popup closes itself immediately on Alt+V.
  const ignoreBlurUntil = useRef(0);

  const refresh = useCallback(async () => {
    const list = (await History()) as unknown as ClipItem[];
    setItems(list ?? []);
  }, []);

  useEffect(() => {
    refresh();
    const offClip = EventsOn("clip:changed", refresh);
    const offShow = EventsOn("popup:show", () => {
      ignoreBlurUntil.current = Date.now() + 600;
      refresh();
      setQuery("");
      setSel(0);
      setTimeout(() => searchRef.current?.focus(), 50);
    });
    return () => {
      offClip();
      offShow();
    };
  }, [refresh]);

  // Feature 2: auto-hide when the window loses focus (Win+V behavior)
  useEffect(() => {
    const onBlur = () => {
      if (Date.now() < ignoreBlurUntil.current) return; // grace period after show
      Hide();
    };
    window.addEventListener("blur", onBlur);
    return () => window.removeEventListener("blur", onBlur);
  }, []);

  const filtered = items.filter((it) =>
    it.preview.toLowerCase().includes(query.toLowerCase())
  );

  // Clamp selection when filtered list shrinks
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

  const onKeyDown = async (e: React.KeyboardEvent) => {
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
  };

  return (
    <div className="panel" onKeyDown={onKeyDown}>
      {/* Feature 1: close button in top-right corner */}
      <button className="close-btn" onClick={() => Hide()} aria-label="Close">&#xd7;</button>
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
      </div>

      <ul className="clip-list">
        {filtered.length === 0 && (
          <li className="empty-state">No clipboard history yet</li>
        )}
        {filtered.map((it, i) => (
          <li
            key={it.id}
            ref={(el) => { itemRefs.current[i] = el; }}
            className={`clip-item${i === clampedSel ? " selected" : ""}`}
            onClick={() => doPaste(it)}
            onMouseEnter={() => setSel(i)}
          >
            {it.kind === 1 ? (
              <span className="image-label">
                <span className="image-icon">&#9638;</span> Image
              </span>
            ) : (
              <span className="item-text">{it.preview}</span>
            )}
          </li>
        ))}
      </ul>

      <div className="hint-bar">
        <span>&#8593;&#8595; select</span>
        <span className="hint-sep">&middot;</span>
        <span>Enter paste</span>
        <span className="hint-sep">&middot;</span>
        <span>Esc close</span>
      </div>
    </div>
  );
}

export default App;
