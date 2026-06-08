package snippet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Load tests ----

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snippets.json")

	data := `{
		"snippets": [
			{"name": "greet",  "content": "Hello {{name}}"},
			{"name": "bye",    "content": "Goodbye {{name}}"},
			{"name": "plain",  "content": "no placeholders"}
		]
	}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0644))

	snippets, err := Load(path)
	require.NoError(t, err)
	require.Len(t, snippets, 3)

	assert.Equal(t, uint64(1), snippets[0].ID)
	assert.Equal(t, "greet", snippets[0].Name)
	assert.Equal(t, "Hello {{name}}", snippets[0].Content)

	assert.Equal(t, uint64(2), snippets[1].ID)
	assert.Equal(t, "bye", snippets[1].Name)

	assert.Equal(t, uint64(3), snippets[2].ID)
	assert.Equal(t, "plain", snippets[2].Name)
}

func TestLoad_NonExistentPath(t *testing.T) {
	snippets, err := Load("/no/such/file/snippets.json")
	assert.NoError(t, err)
	assert.Empty(t, snippets)
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte(`{not valid json`), 0644))

	_, err := Load(path)
	assert.Error(t, err)
}

func TestLoad_EmptySnippetsArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"snippets": []}`), 0644))

	snippets, err := Load(path)
	require.NoError(t, err)
	assert.Empty(t, snippets)
}

// ---- Placeholders tests ----

func TestPlaceholders_DeduplicatedInOrder(t *testing.T) {
	result := Placeholders("Hello {{name}}, id={{ id }}, again {{name}}")
	assert.Equal(t, []string{"name", "id"}, result)
}

func TestPlaceholders_None(t *testing.T) {
	result := Placeholders("no placeholders here")
	assert.Empty(t, result)
}

func TestPlaceholders_SpacesAreStripped(t *testing.T) {
	result := Placeholders("{{ first }}  {{  second  }}")
	assert.Equal(t, []string{"first", "second"}, result)
}

func TestPlaceholders_SinglePlaceholder(t *testing.T) {
	result := Placeholders("{{token}}")
	assert.Equal(t, []string{"token"}, result)
}

func TestPlaceholders_AllowedCharacters(t *testing.T) {
	result := Placeholders("{{my-var_1}}")
	assert.Equal(t, []string{"my-var_1"}, result)
}

// ---- Render tests ----

func TestRender_SubstitutesValues(t *testing.T) {
	out := Render("Hello {{name}}, you are {{age}} years old.", map[string]string{
		"name": "Alice",
		"age":  "30",
	})
	assert.Equal(t, "Hello Alice, you are 30 years old.", out)
}

func TestRender_SpacesAroundPlaceholder(t *testing.T) {
	out := Render("Hi {{ name }}!", map[string]string{"name": "Bob"})
	assert.Equal(t, "Hi Bob!", out)
}

func TestRender_MissingValueBecomesEmpty(t *testing.T) {
	out := Render("{{greeting}} world", map[string]string{})
	assert.Equal(t, " world", out)
}

func TestRender_ReplacesAllOccurrences(t *testing.T) {
	out := Render("{{x}} and {{x}} and {{x}}", map[string]string{"x": "Y"})
	assert.Equal(t, "Y and Y and Y", out)
}

func TestRender_NoPlaceholders(t *testing.T) {
	out := Render("plain text", map[string]string{"x": "y"})
	assert.Equal(t, "plain text", out)
}

func TestRender_NilMap(t *testing.T) {
	out := Render("{{key}}", nil)
	assert.Equal(t, "", out)
}
