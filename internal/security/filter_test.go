package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlocksKnownSensitiveAppByExeName(t *testing.T) {
	f := NewFilter([]string{"1Password.exe", "Bitwarden.exe"})
	assert.True(t, f.IsBlocked(`C:\Program Files\1Password\1Password.exe`))
	assert.True(t, f.IsBlocked(`D:\apps\Bitwarden.exe`))
}

func TestBlockingIsCaseInsensitive(t *testing.T) {
	f := NewFilter([]string{"1Password.exe"})
	assert.True(t, f.IsBlocked(`C:\x\1PASSWORD.EXE`))
}

func TestAllowsNormalApp(t *testing.T) {
	f := NewFilter([]string{"1Password.exe"})
	assert.False(t, f.IsBlocked(`C:\Windows\notepad.exe`))
}

func TestEmptyPathIsNotBlocked(t *testing.T) {
	f := NewFilter([]string{"1Password.exe"})
	assert.False(t, f.IsBlocked(""))
}
