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
