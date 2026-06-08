package clip

import (
	"sync"
	"time"
)

// ClipStore is an in-memory history with two tiers:
//
//   - items:  unpinned entries, capped at `cap` with FIFO eviction (oldest out).
//   - pinned: pinned entries, NOT subject to eviction and kept in a
//     user-defined display order (reorderable via drag-and-drop).
//
// Dedup across both tiers is O(1) via byHash.
type ClipStore struct {
	mu     sync.RWMutex
	cap    int
	items  []*ClipItem // unpinned, oldest first
	pinned []*ClipItem // pinned, display order (top first)
	byHash map[uint64]*ClipItem
	nextID uint64
}

func NewStore(capacity int) *ClipStore {
	if capacity < 1 {
		capacity = 1
	}
	return &ClipStore{
		cap:    capacity,
		byHash: make(map[uint64]*ClipItem, capacity),
	}
}

// Push inserts item. A duplicate (same Hash) refreshes the existing entry
// instead of adding a new one: an unpinned duplicate is bumped to the top, a
// pinned duplicate keeps its pinned slot. Evicted items are cleared to release
// memory.
func (s *ClipStore) Push(item *ClipItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.byHash[item.Hash]; ok {
		existing.CreatedAt = time.Now()
		if !existing.Pinned {
			existing.ID = s.bumpID()
			s.moveUnpinnedToTop(existing)
		}
		return
	}

	s.nextID++
	item.ID = s.nextID
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	s.items = append(s.items, item)
	s.byHash[item.Hash] = item
	s.evictOverflow()
}

// List returns a snapshot: pinned entries first (in display order), then
// unpinned entries newest (highest ID) first.
func (s *ClipStore) List() []*ClipItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ClipItem, 0, len(s.pinned)+len(s.items))
	out = append(out, s.pinned...)
	for i := len(s.items) - 1; i >= 0; i-- {
		out = append(out, s.items[i])
	}
	return out
}

// Get returns the item with the given ID, or nil.
func (s *ClipStore) Get(id uint64) *ClipItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, it := range s.pinned {
		if it.ID == id {
			return it
		}
	}
	for _, it := range s.items {
		if it.ID == id {
			return it
		}
	}
	return nil
}

// TogglePin flips the pinned state of the item with the given ID. Pinning moves
// the item out of the capped ring to the top of the pinned group; unpinning
// returns it to the top of the unpinned history (with a fresh ID/timestamp).
// No-op if the ID is not found.
func (s *ClipStore) TogglePin(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Currently pinned -> unpin and move back into the unpinned history.
	for i, it := range s.pinned {
		if it.ID == id {
			it.Pinned = false
			s.pinned = append(s.pinned[:i], s.pinned[i+1:]...)
			it.ID = s.bumpID()
			it.CreatedAt = time.Now()
			s.items = append(s.items, it)
			s.evictOverflow()
			return
		}
	}

	// Currently unpinned -> pin and move to the top of the pinned group.
	for i, it := range s.items {
		if it.ID == id {
			it.Pinned = true
			s.items = append(s.items[:i], s.items[i+1:]...)
			s.pinned = append([]*ClipItem{it}, s.pinned...)
			return
		}
	}
}

// ReorderPins rearranges the pinned group to match the given ID order. IDs that
// are not currently pinned are ignored; any pinned item missing from ids keeps
// its relative order at the end (defensive against a stale UI).
func (s *ClipStore) ReorderPins(ids []uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	byID := make(map[uint64]*ClipItem, len(s.pinned))
	for _, it := range s.pinned {
		byID[it.ID] = it
	}

	next := make([]*ClipItem, 0, len(s.pinned))
	for _, id := range ids {
		if it, ok := byID[id]; ok {
			next = append(next, it)
			delete(byID, id)
		}
	}
	// Preserve any leftover pinned items not named in ids.
	for _, it := range s.pinned {
		if _, ok := byID[it.ID]; ok {
			next = append(next, it)
		}
	}
	s.pinned = next
}

// Clear removes all unpinned history and releases its memory. Pinned items are
// intentionally kept (pinning marks them as protected).
func (s *ClipStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range s.items {
		it.Text = ""
		it.Image = nil
	}
	s.items = nil
	s.byHash = make(map[uint64]*ClipItem, s.cap)
	for _, it := range s.pinned {
		s.byHash[it.Hash] = it
	}
}

// DeleteItem removes a single entry (pinned or unpinned) by ID and clears its
// payload for GC. No-op if the ID is not found.
func (s *ClipStore) DeleteItem(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, it := range s.pinned {
		if it.ID == id {
			delete(s.byHash, it.Hash)
			it.Text = ""
			it.Image = nil
			s.pinned = append(s.pinned[:i], s.pinned[i+1:]...)
			return
		}
	}
	for i, it := range s.items {
		if it.ID == id {
			delete(s.byHash, it.Hash)
			it.Text = ""
			it.Image = nil
			s.items = append(s.items[:i], s.items[i+1:]...)
			return
		}
	}
}

// ClearAll removes everything, including pinned items, and releases memory.
// This is the "clear all history" action (unlike Clear, which keeps pins).
func (s *ClipStore) ClearAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range s.items {
		it.Text = ""
		it.Image = nil
	}
	for _, it := range s.pinned {
		it.Text = ""
		it.Image = nil
	}
	s.items = nil
	s.pinned = nil
	s.byHash = make(map[uint64]*ClipItem, s.cap)
}

// bumpID returns the next monotonic ID. Caller must hold the lock.
func (s *ClipStore) bumpID() uint64 {
	s.nextID++
	return s.nextID
}

// moveUnpinnedToTop moves an existing unpinned item to the newest position.
// Caller must hold the lock.
func (s *ClipStore) moveUnpinnedToTop(it *ClipItem) {
	for i, x := range s.items {
		if x == it {
			s.items = append(s.items[:i], s.items[i+1:]...)
			s.items = append(s.items, it)
			return
		}
	}
}

// evictOverflow drops oldest unpinned items until len(items) <= cap, clearing
// their payload for GC. Caller must hold the lock.
func (s *ClipStore) evictOverflow() {
	for len(s.items) > s.cap {
		old := s.items[0]
		s.items = s.items[1:]
		delete(s.byHash, old.Hash)
		old.Text = ""
		old.Image = nil
	}
}
