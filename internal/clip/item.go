package clip

import (
	"hash/fnv"
	"time"
)

type ClipKind int

const (
	KindText ClipKind = iota
	KindImage
)

// ClipItem is one entry in the in-memory history.
type ClipItem struct {
	ID        uint64    `json:"id"`
	Kind      ClipKind  `json:"kind"`
	Text      string    `json:"text"`    // empty if image
	Image     []byte    `json:"-"`       // PNG bytes; nil if text (not sent raw to UI)
	Preview   string    `json:"preview"` // short text or "image NxM" label for UI
	Hash      uint64    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	Pinned    bool      `json:"pinned"`
	Format    string    `json:"format"` // "plain", "json", "sql"
}

func HashText(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

func HashBytes(b []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(b)
	return h.Sum64()
}
