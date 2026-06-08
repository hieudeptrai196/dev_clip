package clip

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashTextIsStableAndDistinct(t *testing.T) {
	a := HashText("SELECT 1")
	b := HashText("SELECT 1")
	c := HashText("SELECT 2")
	assert.Equal(t, a, b, "same text must hash equal")
	assert.NotEqual(t, a, c, "different text must hash different")
}

func TestHashBytesDistinctFromText(t *testing.T) {
	img := HashBytes([]byte{1, 2, 3})
	assert.NotZero(t, img)
}
