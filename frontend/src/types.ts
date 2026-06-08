export interface ClipItem {
  id: number;
  kind: number; // 0 = text, 1 = image
  text: string;
  preview: string;
  createdAt: string;
  pinned: boolean;
  format: string; // "plain" | "json" | "sql"
}

export interface Snippet {
  id: number;
  name: string;
  content: string;
}
