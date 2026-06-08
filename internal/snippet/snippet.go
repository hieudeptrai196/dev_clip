package snippet

import (
	"encoding/json"
	"errors"
	"os"
	"regexp"
)

// Snippet represents a pinned reusable snippet loaded from a JSON config.
type Snippet struct {
	ID      uint64 `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

// configFile is the JSON structure we decode from the config file.
type configFile struct {
	Snippets []struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	} `json:"snippets"`
}

// placeholderRe matches {{name}} with optional surrounding spaces.
var placeholderRe = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_-]+)\s*\}\}`)

// Load reads a config file of the form:
//
//	{ "snippets": [ { "name": "...", "content": "..." }, ... ] }
//
// Returns snippets with sequential IDs (starting at 1) in file order.
// If the file does not exist, returns an empty slice and no error.
// If the file exists but is invalid JSON, returns an error.
func Load(path string) ([]Snippet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Snippet{}, nil
		}
		return nil, err
	}

	var cfg configFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	snippets := make([]Snippet, 0, len(cfg.Snippets))
	for i, s := range cfg.Snippets {
		snippets = append(snippets, Snippet{
			ID:      uint64(i + 1),
			Name:    s.Name,
			Content: s.Content,
		})
	}
	return snippets, nil
}

// Placeholders returns the unique placeholder names found in content, in order
// of first appearance. A placeholder is {{name}} where name may contain
// letters, digits, underscore, or hyphen. Spaces inside braces are trimmed.
// Returns nil if none are found.
func Placeholders(content string) []string {
	matches := placeholderRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	result := make([]string, 0, len(matches))
	for _, m := range matches {
		name := m[1]
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			result = append(result, name)
		}
	}
	return result
}

// Render substitutes each {{name}} (allowing surrounding spaces) with
// values[name]. Placeholders with no provided value are replaced with "".
func Render(content string, values map[string]string) string {
	return placeholderRe.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the name from the match
		sub := placeholderRe.FindStringSubmatch(match)
		if sub == nil {
			return match
		}
		name := sub[1]
		if values == nil {
			return ""
		}
		return values[name]
	})
}
