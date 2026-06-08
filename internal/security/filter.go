package security

import (
	"path/filepath"
	"strings"
)

// SecurityFilter decides whether a clipboard capture from a given source
// application should be dropped (e.g. password managers).
type SecurityFilter struct {
	blocked map[string]struct{} // lowercased exe base names
}

func NewFilter(exeNames []string) *SecurityFilter {
	m := make(map[string]struct{}, len(exeNames))
	for _, n := range exeNames {
		m[strings.ToLower(n)] = struct{}{}
	}
	return &SecurityFilter{blocked: m}
}

// IsBlocked reports whether the given full exe path belongs to a blocked app.
func (f *SecurityFilter) IsBlocked(exePath string) bool {
	if exePath == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(exePath))
	_, ok := f.blocked[base]
	return ok
}

// Note: the built-in blocklist is defined in internal/settings.DefaultSettings()
// to keep a single source of truth. This package only enforces the list.
