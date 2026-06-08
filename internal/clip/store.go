package clip

import (
	"sync"
	"sync/atomic"
	"time"
)

// ClipStore is a fixed-size in-memory FIFO ring buffer with O(1) dedup.
type ClipStore struct {
	mu     sync.RWMutex
	buf    []*ClipItem
	head   int // next write index
	count  int
	byHash map[uint64]*ClipItem
	nextID uint64
}

func NewStore(capacity int) *ClipStore {
	if capacity < 1 {
		capacity = 1
	}
	return &ClipStore{
		buf:    make([]*ClipItem, capacity),
		byHash: make(map[uint64]*ClipItem, capacity),
	}
}

// Push inserts item. Duplicate (same Hash) refreshes the existing item to top
// instead of adding a new one. Evicted items are cleared to release memory.
func (s *ClipStore) Push(item *ClipItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.byHash[item.Hash]; ok {
		existing.CreatedAt = time.Now()
		existing.ID = atomic.AddUint64(&s.nextID, 1) // new ID => sorts to top
		return
	}

	if old := s.buf[s.head]; old != nil {
		delete(s.byHash, old.Hash)
		old.Text = ""
		old.Image = nil
		s.buf[s.head] = nil
	}

	item.ID = atomic.AddUint64(&s.nextID, 1)
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	s.buf[s.head] = item
	s.byHash[item.Hash] = item
	s.head = (s.head + 1) % len(s.buf)
	if s.count < len(s.buf) {
		s.count++
	}
}

// List returns a snapshot of items, newest (highest ID) first.
func (s *ClipStore) List() []*ClipItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ClipItem, 0, s.count)
	for _, it := range s.buf {
		if it != nil {
			out = append(out, it)
		}
	}
	// newest first by ID (descending)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].ID > out[i].ID {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// Get returns the item with the given ID, or nil.
func (s *ClipStore) Get(id uint64) *ClipItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, it := range s.buf {
		if it != nil && it.ID == id {
			return it
		}
	}
	return nil
}

// Clear removes all items and releases memory.
func (s *ClipStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.buf {
		if it := s.buf[i]; it != nil {
			it.Text = ""
			it.Image = nil
		}
		s.buf[i] = nil
	}
	s.byHash = make(map[uint64]*ClipItem, len(s.buf))
	s.head = 0
	s.count = 0
}
