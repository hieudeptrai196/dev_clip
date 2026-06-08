package format

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Kind classifies clipboard text content.
type Kind int

const (
	Plain Kind = iota
	JSON
	SQL
	JWT
	Timestamp
)

// sqlStartWords are the keywords that, when they appear as the first word of
// trimmed text (case-insensitive), identify the text as SQL.
var sqlStartWords = []string{
	"select", "insert", "update", "delete", "with", "create", "alter", "drop",
}

// Detect classifies text as JSON, SQL, JWT, Timestamp, or Plain.
func Detect(text string) Kind {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) == 0 {
		return Plain
	}

	if IsJWT(trimmed) {
		return JWT
	}

	if _, _, ok := DetectTimestamp(trimmed); ok {
		return Timestamp
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

// DetectTimestamp detects if a string is a Unix timestamp (seconds or milliseconds).
func DetectTimestamp(text string) (t time.Time, isMs bool, ok bool) {
	s := strings.TrimSpace(text)
	if len(s) != 10 && len(s) != 13 {
		return time.Time{}, false, false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return time.Time{}, false, false
		}
	}
	var val int64
	for i := 0; i < len(s); i++ {
		val = val*10 + int64(s[i]-'0')
	}
	if len(s) == 10 {
		t = time.Unix(val, 0)
		if t.Year() > 1975 && t.Year() < 2100 {
			return t, false, true
		}
	} else {
		t = time.UnixMilli(val)
		if t.Year() > 1975 && t.Year() < 2100 {
			return t, true, true
		}
	}
	return time.Time{}, false, false
}

var dateLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02",
	"02/01/2006 15:04:05",
	"02/01/2006",
}

// ConvertToTimestamp parses a human-readable date to a Unix timestamp string.
func ConvertToTimestamp(text string) (string, error) {
	s := strings.TrimSpace(text)
	for _, layout := range dateLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return fmt.Sprintf("%d", t.Unix()), nil
		}
	}
	return "", fmt.Errorf("invalid date format")
}

// IsJWT checks if a string is a base64url-encoded JWT.
func IsJWT(text string) bool {
	s := strings.TrimSpace(text)
	if !strings.HasPrefix(s, "eyJ") {
		return false
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts[:2] {
		if _, err := decodeBase64URL(p); err != nil {
			return false
		}
	}
	return true
}

func decodeBase64URL(s string) ([]byte, error) {
	if len(s)%4 != 0 {
		return base64.RawURLEncoding.DecodeString(s)
	}
	return base64.URLEncoding.DecodeString(s)
}

// DecodeJWT decodes the header and payload of a JWT.
func DecodeJWT(token string) (string, string, error) {
	s := strings.TrimSpace(token)
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid JWT format")
	}
	headerBytes, err := decodeBase64URL(parts[0])
	if err != nil {
		return "", "", fmt.Errorf("failed to decode header: %w", err)
	}
	payloadBytes, err := decodeBase64URL(parts[1])
	if err != nil {
		return "", "", fmt.Errorf("failed to decode payload: %w", err)
	}

	var headerBuf, payloadBuf bytes.Buffer
	if err := json.Indent(&headerBuf, headerBytes, "", "  "); err != nil {
		headerBuf.Write(headerBytes)
	}
	if err := json.Indent(&payloadBuf, payloadBytes, "", "  "); err != nil {
		payloadBuf.Write(payloadBytes)
	}

	return headerBuf.String(), payloadBuf.String(), nil
}

// MinifyJSON compresses JSON text.
func MinifyJSON(text string) (string, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(strings.TrimSpace(text))); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// MinifySQL compresses SQL text.
func MinifySQL(text string) string {
	s := strings.ReplaceAll(text, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	return collapseSpaces(s)
}
