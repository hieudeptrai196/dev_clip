package clip

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func textItem(s string) *ClipItem {
	return &ClipItem{Kind: KindText, Text: s, Preview: s, Hash: HashText(s)}
}

func TestPushAndListNewestFirst(t *testing.T) {
	s := NewStore(3)
	s.Push(textItem("a"))
	s.Push(textItem("b"))
	items := s.List()
	require.Len(t, items, 2)
	assert.Equal(t, "b", items[0].Text, "newest first")
	assert.Equal(t, "a", items[1].Text)
}

func TestPushAssignsMonotonicIDs(t *testing.T) {
	s := NewStore(3)
	s.Push(textItem("a"))
	s.Push(textItem("b"))
	items := s.List()
	assert.Equal(t, uint64(2), items[0].ID)
	assert.Equal(t, uint64(1), items[1].ID)
}

func TestRingBufferEvictsOldestFIFO(t *testing.T) {
	s := NewStore(3)
	for _, v := range []string{"a", "b", "c", "d"} {
		s.Push(textItem(v))
	}
	items := s.List()
	require.Len(t, items, 3)
	assert.Equal(t, []string{"d", "c", "b"}, []string{items[0].Text, items[1].Text, items[2].Text})
}

func TestDedupMovesExistingToTop(t *testing.T) {
	s := NewStore(3)
	s.Push(textItem("a"))
	s.Push(textItem("b"))
	s.Push(textItem("a")) // duplicate of first
	items := s.List()
	require.Len(t, items, 2, "no new item added")
	assert.Equal(t, "a", items[0].Text, "duplicate moved to top")
}

func TestEvictedItemDataIsCleared(t *testing.T) {
	s := NewStore(1)
	first := textItem("a")
	s.Push(first)
	s.Push(textItem("b"))
	assert.Empty(t, first.Text, "evicted item's Text must be cleared for GC")
	assert.Nil(t, first.Image)
}

func TestPinMovesItemToTopAndSurvivesEviction(t *testing.T) {
	s := NewStore(2)
	s.Push(textItem("a"))
	s.Push(textItem("b"))
	aID := s.List()[1].ID // "a" is the oldest

	s.TogglePin(aID)

	// Pinned "a" must lead the list even though it was the oldest.
	items := s.List()
	require.Len(t, items, 2)
	assert.Equal(t, "a", items[0].Text)
	assert.True(t, items[0].Pinned)

	// Overflow the cap: "a" must NOT be evicted because it is pinned.
	s.Push(textItem("c"))
	s.Push(textItem("d"))
	items = s.List()
	texts := []string{}
	for _, it := range items {
		texts = append(texts, it.Text)
	}
	assert.Contains(t, texts, "a", "pinned item must survive eviction")
	assert.Equal(t, "a", items[0].Text, "pinned item stays on top")
}

func TestUnpinReturnsItemToHistory(t *testing.T) {
	s := NewStore(5)
	s.Push(textItem("a"))
	id := s.List()[0].ID
	s.TogglePin(id)
	assert.True(t, s.List()[0].Pinned)

	s.TogglePin(s.List()[0].ID)
	got := s.List()[0]
	assert.False(t, got.Pinned, "item is unpinned")
	assert.Equal(t, "a", got.Text)
}

func TestReorderPinsAppliesGivenOrder(t *testing.T) {
	s := NewStore(5)
	for _, v := range []string{"a", "b", "c"} {
		s.Push(textItem(v))
	}
	for _, it := range s.List() {
		s.TogglePin(it.ID)
	}
	// All three pinned now; capture their IDs by text.
	idOf := map[string]uint64{}
	for _, it := range s.List() {
		idOf[it.Text] = it.ID
	}

	s.ReorderPins([]uint64{idOf["b"], idOf["a"], idOf["c"]})

	items := s.List()
	require.Len(t, items, 3)
	assert.Equal(t, []string{"b", "a", "c"},
		[]string{items[0].Text, items[1].Text, items[2].Text})
}

func TestDuplicateOfPinnedItemKeepsItPinned(t *testing.T) {
	s := NewStore(5)
	s.Push(textItem("a"))
	s.TogglePin(s.List()[0].ID)

	s.Push(textItem("a")) // copy the same text again

	items := s.List()
	require.Len(t, items, 1, "no duplicate added")
	assert.True(t, items[0].Pinned, "stays pinned")
}

func TestClearKeepsPinnedItems(t *testing.T) {
	s := NewStore(5)
	s.Push(textItem("keep"))
	s.TogglePin(s.List()[0].ID)
	s.Push(textItem("drop"))

	s.Clear()

	items := s.List()
	require.Len(t, items, 1)
	assert.Equal(t, "keep", items[0].Text)
	assert.True(t, items[0].Pinned)
}

func TestDeleteItemRemovesUnpinned(t *testing.T) {
	s := NewStore(5)
	s.Push(textItem("a"))
	s.Push(textItem("b"))
	bID := s.List()[0].ID

	s.DeleteItem(bID)

	items := s.List()
	require.Len(t, items, 1)
	assert.Equal(t, "a", items[0].Text)
	// Same text can be re-added after delete (dedup map was cleaned).
	s.Push(textItem("b"))
	assert.Len(t, s.List(), 2)
}

func TestDeleteItemRemovesPinned(t *testing.T) {
	s := NewStore(5)
	s.Push(textItem("a"))
	s.TogglePin(s.List()[0].ID)
	require.True(t, s.List()[0].Pinned)

	s.DeleteItem(s.List()[0].ID)
	assert.Len(t, s.List(), 0)
}

func TestClearAllRemovesEverythingIncludingPinned(t *testing.T) {
	s := NewStore(5)
	s.Push(textItem("a"))
	s.TogglePin(s.List()[0].ID)
	s.Push(textItem("b"))

	s.ClearAll()

	assert.Len(t, s.List(), 0)
}

func TestConcurrentPushIsSafe(t *testing.T) {
	s := NewStore(100)
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 50; j++ {
				s.Push(textItem(fmt.Sprintf("g%d-%d", n, j)))
			}
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	assert.Len(t, s.List(), 100)
}
