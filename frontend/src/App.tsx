import { useEffect, useRef, useState, useCallback } from "react";
import { History, PasteItem, Hide, Thumbnail, FormatItem, PasteTransformed, PasteFormatted, Snippets, SnippetPlaceholders, PasteSnippet } from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import type { ClipItem } from "./types";
import { snippet } from "../wailsjs/go/models";
import "./App.css";

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
  const [tab, setTab] = useState<"recent" | "snippets">("recent");

  // ── Recent tab state ──
  const [items, setItems] = useState<ClipItem[]>([]);
  const [query, setQuery] = useState("");
  const [sel, setSel] = useState(0);
  const [thumbs, setThumbs] = useState<Record<number, string>>({});
  const searchRef = useRef<HTMLInputElement>(null);
  const itemRefs = useRef<(HTMLLIElement | null)[]>([]);

  // ── Snippets tab state ──
  const [snippetList, setSnippetList] = useState<snippet.Snippet[]>([]);
  const [snippetSel, setSnippetSel] = useState(0);
  const snippetRefs = useRef<(HTMLLIElement | null)[]>([]);
  // Placeholder form state
  const [phSnippet, setPhSnippet] = useState<snippet.Snippet | null>(null);
  const [phValues, setPhValues] = useState<Record<string, string>>({});
  const [phNames, setPhNames] = useState<string[]>([]);
  const firstPhRef = useRef<HTMLInputElement>(null);

  // Suppress the blur-to-hide for a short grace period right after the popup is
  // shown, otherwise the window fires a transient `blur` during the show
  // transition and the popup closes itself immediately on Alt+V.
  const ignoreBlurUntil = useRef(0);

  const refresh = useCallback(async () => {
    const list = (await History()) as unknown as ClipItem[];
    setItems(list ?? []);
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
    return () => {
      offClip();
      offShow();
    };
  }, [refresh, loadSnippets, tab]);

  // Load snippets when switching to snippets tab
  useEffect(() => {
    if (tab === "snippets") {
      loadSnippets();
    }
  }, [tab, loadSnippets]);

  // Feature 2: auto-hide when the window loses focus (Win+V behavior)
  useEffect(() => {
    const onBlur = () => {
      if (Date.now() < ignoreBlurUntil.current) return; // grace period after show
      // A transient blur during the show transition is immediately followed by
      // focus coming back. Debounce and re-check real focus before hiding so we
      // only close on a genuine click-away, not a focus bounce.
      window.setTimeout(() => {
        if (Date.now() < ignoreBlurUntil.current) return;
        if (!document.hasFocus()) Hide();
      }, 120);
    };
    window.addEventListener("blur", onBlur);
    return () => window.removeEventListener("blur", onBlur);
  }, []);

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

  // ── Keyboard handler ──
  const onKeyDown = async (e: React.KeyboardEvent) => {
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
    } else {
      // Snippets tab
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
    }
  };

  useEffect(() => {
    const el = snippetRefs.current[snippetSel];
    el?.scrollIntoView({ block: "nearest" });
  }, [snippetSel, snippetList.length]);

  return (
    <div className="panel" onKeyDown={onKeyDown}>
      {/* Close button in top-right corner */}
      <button className="close-btn" onClick={() => Hide()} aria-label="Close">&#xd7;</button>

      {/* Tab switcher */}
      <div className="tabs">
        <button
          className={`tab${tab === "recent" ? " tab--active" : ""}`}
          onClick={() => { setTab("recent"); setTimeout(() => searchRef.current?.focus(), 30); }}
        >
          Recent
        </button>
        <button
          className={`tab${tab === "snippets" ? " tab--active" : ""}`}
          onClick={() => setTab("snippets")}
        >
          Snippets
        </button>
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
          </div>

          <ul className="clip-list">
            {filtered.length === 0 && (
              <li className="empty-state">No clipboard history yet</li>
            )}
            {filtered.map((it, i) => {
              const isSelected = i === clampedSel;
              const isText = it.kind === 0;
              const hasFmt = isText && (it.format === "json" || it.format === "sql");
              return (
                <li
                  key={it.id}
                  ref={(el) => { itemRefs.current[i] = el; }}
                  className={`clip-item${isSelected ? " selected" : ""}`}
                  onClick={() => doPaste(it)}
                  onMouseEnter={() => setSel(i)}
                >
                  <div className="item-row">
                    {it.kind === 1 ? (
                      <span className="image-label">
                        {thumbs[it.id] ? (
                          <img className="thumb" src={thumbs[it.id]} alt="clipboard image" />
                        ) : (
                          <span className="image-icon">&#9638;</span>
                        )}
                        <span className="image-text">Image</span>
                      </span>
                    ) : (
                      <span className="item-text">{it.preview}</span>
                    )}
                    {hasFmt && (
                      <span className={`fmt-badge fmt-badge--${it.format}`}>
                        {it.format.toUpperCase()}
                      </span>
                    )}
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

      <div className="hint-bar">
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
      </div>
    </div>
  );
}

export default App;
