import { useEffect, useState, useCallback } from "react";
import { History, PasteItem, Hide } from "../wailsjs/go/main/App";
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
      await Hide();
    } else if (e.key === "Escape") {
      await Hide();
    }
  };

  return (
    <div className="container" onKeyDown={onKeyDown} tabIndex={0}>
      <input
        autoFocus
        placeholder="Search clipboard..."
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
            {it.kind === 1 ? `[image] ${it.preview}` : it.preview}
          </li>
        ))}
      </ul>
    </div>
  );
}

export default App;
