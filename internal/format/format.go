package format

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"
	"unicode"
)

// Kind classifies clipboard text content.
type Kind int

const (
	Plain Kind = iota
	JSON
	SQL
)

// sqlStartWords are the keywords that, when they appear as the first word of
// trimmed text (case-insensitive), identify the text as SQL.
var sqlStartWords = []string{
	"select", "insert", "update", "delete", "with", "create", "alter", "drop",
}

// Detect classifies text as JSON, SQL, or Plain.
func Detect(text string) Kind {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) == 0 {
		return Plain
	}

	// JSON: valid JSON that starts with { or [
	if (trimmed[0] == '{' || trimmed[0] == '[') && json.Valid([]byte(trimmed)) {
		return JSON
	}

	// SQL: first word matches a known SQL keyword
	lower := strings.ToLower(trimmed)
	for _, kw := range sqlStartWords {
		// match "keyword " (with trailing space) or exact keyword (whole string)
		if strings.HasPrefix(lower, kw+" ") || lower == kw {
			return SQL
		}
	}

	return Plain
}

// PrettyJSON returns indented JSON (2 spaces). Returns error if invalid JSON.
func PrettyJSON(text string) (string, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(strings.TrimSpace(text)), "", "  "); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// sqlKeywords are uppercased wherever they appear as whole words.
var sqlKeywords = []string{
	"SELECT", "FROM", "WHERE", "AND", "OR", "JOIN", "INNER", "LEFT", "RIGHT",
	"OUTER", "ON", "GROUP", "BY", "ORDER", "HAVING", "LIMIT", "INSERT", "INTO",
	"VALUES", "UPDATE", "SET", "DELETE", "CREATE", "TABLE", "AS", "DISTINCT",
	"NULL", "NOT", "IN", "LIKE", "BETWEEN", "UNION", "ALL",
}

// clauseNewlines defines multi-word and single-word clauses that get a newline
// prepended. Order matters: longer phrases must come before their sub-words.
var clauseNewlines = []string{
	"GROUP BY",
	"ORDER BY",
	"LEFT JOIN",
	"RIGHT JOIN",
	"INNER JOIN",
	"FROM",
	"WHERE",
	"HAVING",
	"LIMIT",
	"JOIN",
	"UNION",
}

// FormatSQL uppercases SQL keywords and puts major clauses on new lines.
func FormatSQL(text string) string {
	// Step 1: collapse runs of whitespace to single spaces.
	s := collapseSpaces(strings.TrimSpace(text))

	// Step 2: uppercase keywords (whole-word, case-insensitive).
	for _, kw := range sqlKeywords {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(kw) + `\b`)
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			return strings.ToUpper(m)
		})
	}

	// Step 3: insert newline before clause keywords (already uppercased).
	// We replace " CLAUSE" with "\nCLAUSE" to avoid double-newlines.
	for _, clause := range clauseNewlines {
		// Replace occurrences that are preceded by a space (i.e. not at start).
		s = strings.ReplaceAll(s, " "+clause, "\n"+clause)
	}

	return strings.TrimSpace(s)
}

// collapseSpaces replaces runs of whitespace with a single space.
func collapseSpaces(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return b.String()
}

// Transform applies a string transformation to text.
// op: "upper", "lower", "camel", "snake", "kebab", "base64encode", "base64decode".
// Unknown op returns input unchanged.
func Transform(text string, op string) string {
	switch op {
	case "upper":
		return strings.ToUpper(text)
	case "lower":
		return strings.ToLower(text)
	case "camel":
		return toCamel(text)
	case "snake":
		return toDelimited(text, '_')
	case "kebab":
		return toDelimited(text, '-')
	case "base64encode":
		return base64.StdEncoding.EncodeToString([]byte(text))
	case "base64decode":
		decoded, err := base64.StdEncoding.DecodeString(text)
		if err != nil {
			return text
		}
		return string(decoded)
	default:
		return text
	}
}

// splitWords splits text into lowercase words by splitting on spaces, underscores,
// hyphens, and camelCase boundaries.
func splitWords(text string) []string {
	// First expand camelCase boundaries: insert space before each uppercase
	// letter that follows a lowercase letter or digit.
	var expanded strings.Builder
	runes := []rune(text)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				expanded.WriteRune(' ')
			}
		}
		expanded.WriteRune(r)
	}

	// Split on spaces, underscores, hyphens.
	raw := regexp.MustCompile(`[\s_\-]+`).Split(expanded.String(), -1)

	var words []string
	for _, w := range raw {
		w = strings.TrimSpace(w)
		if w != "" {
			words = append(words, strings.ToLower(w))
		}
	}
	return words
}

// toCamel converts text to camelCase.
func toCamel(text string) string {
	words := splitWords(text)
	if len(words) == 0 {
		return text
	}
	var b strings.Builder
	for i, w := range words {
		if i == 0 {
			b.WriteString(w)
		} else {
			if len(w) > 0 {
				b.WriteString(strings.ToUpper(w[:1]) + w[1:])
			}
		}
	}
	return b.String()
}

// toDelimited converts text to snake_case (sep='_') or kebab-case (sep='-').
func toDelimited(text string, sep rune) string {
	words := splitWords(text)
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = w
	}
	return strings.Join(parts, string(sep))
}
