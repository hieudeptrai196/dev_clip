package format

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Detect ───────────────────────────────────────────────────────────────────

func TestDetect_JSON_Object(t *testing.T) {
	assert.Equal(t, JSON, Detect(`{"key": "value"}`))
}

func TestDetect_JSON_Array(t *testing.T) {
	assert.Equal(t, JSON, Detect(`[1, 2, 3]`))
}

func TestDetect_JSON_WithLeadingWhitespace(t *testing.T) {
	assert.Equal(t, JSON, Detect("  \n{\"a\":1}\n"))
}

func TestDetect_SQL_Select(t *testing.T) {
	assert.Equal(t, SQL, Detect("SELECT * FROM users"))
}

func TestDetect_SQL_Insert(t *testing.T) {
	assert.Equal(t, SQL, Detect("INSERT INTO foo VALUES (1)"))
}

func TestDetect_SQL_Update(t *testing.T) {
	assert.Equal(t, SQL, Detect("UPDATE foo SET bar=1"))
}

func TestDetect_SQL_Delete(t *testing.T) {
	assert.Equal(t, SQL, Detect("DELETE FROM foo"))
}

func TestDetect_SQL_With(t *testing.T) {
	assert.Equal(t, SQL, Detect("WITH cte AS (SELECT 1)"))
}

func TestDetect_SQL_Create(t *testing.T) {
	assert.Equal(t, SQL, Detect("CREATE TABLE foo (id INT)"))
}

func TestDetect_SQL_Alter(t *testing.T) {
	assert.Equal(t, SQL, Detect("ALTER TABLE foo ADD COLUMN bar INT"))
}

func TestDetect_SQL_Drop(t *testing.T) {
	assert.Equal(t, SQL, Detect("DROP TABLE foo"))
}

func TestDetect_SQL_CaseInsensitive(t *testing.T) {
	assert.Equal(t, SQL, Detect("select * from users"))
}

func TestDetect_Plain(t *testing.T) {
	assert.Equal(t, Plain, Detect("hello world"))
}

func TestDetect_PlainNumber(t *testing.T) {
	assert.Equal(t, Plain, Detect("42"))
}

func TestDetect_PlainEmpty(t *testing.T) {
	assert.Equal(t, Plain, Detect(""))
}

// ─── PrettyJSON ───────────────────────────────────────────────────────────────

func TestPrettyJSON_Object(t *testing.T) {
	input := `{"b":2,"a":1}`
	out, err := PrettyJSON(input)
	require.NoError(t, err)
	assert.Contains(t, out, "\n")
	// 2-space indent
	assert.Contains(t, out, "  ")
}

func TestPrettyJSON_Array(t *testing.T) {
	input := `[1,2,3]`
	out, err := PrettyJSON(input)
	require.NoError(t, err)
	assert.Contains(t, out, "\n")
}

func TestPrettyJSON_AlreadyPretty(t *testing.T) {
	input := "{\n  \"a\": 1\n}"
	out, err := PrettyJSON(input)
	require.NoError(t, err)
	assert.Equal(t, input, out)
}

func TestPrettyJSON_InvalidJSON(t *testing.T) {
	_, err := PrettyJSON("not json")
	assert.Error(t, err)
}

func TestPrettyJSON_InvalidJSON_Partial(t *testing.T) {
	_, err := PrettyJSON(`{"a": 1`)
	assert.Error(t, err)
}

// ─── FormatSQL ────────────────────────────────────────────────────────────────

func TestFormatSQL_UppercasesKeywords(t *testing.T) {
	out := FormatSQL("select * from users where id = 1")
	assert.True(t, strings.Contains(out, "SELECT"), "expected SELECT: %s", out)
	assert.True(t, strings.Contains(out, "FROM"), "expected FROM: %s", out)
	assert.True(t, strings.Contains(out, "WHERE"), "expected WHERE: %s", out)
}

func TestFormatSQL_NewlineBeforeFrom(t *testing.T) {
	out := FormatSQL("select * from users")
	assert.Contains(t, out, "\nFROM")
}

func TestFormatSQL_NewlineBeforeWhere(t *testing.T) {
	out := FormatSQL("select * from users where id=1")
	assert.Contains(t, out, "\nWHERE")
}

func TestFormatSQL_NewlineBeforeJoin(t *testing.T) {
	out := FormatSQL("select * from a join b on a.id=b.id")
	assert.Contains(t, out, "\nJOIN")
}

func TestFormatSQL_NewlineBeforeOrderBy(t *testing.T) {
	out := FormatSQL("select * from users order by name")
	assert.Contains(t, out, "\nORDER BY")
}

func TestFormatSQL_NewlineBeforeGroupBy(t *testing.T) {
	out := FormatSQL("select count(*) from users group by status")
	assert.Contains(t, out, "\nGROUP BY")
}

func TestFormatSQL_NewlineBeforeHaving(t *testing.T) {
	out := FormatSQL("select count(*) from users group by status having count(*)>1")
	assert.Contains(t, out, "\nHAVING")
}

func TestFormatSQL_NewlineBeforeLimit(t *testing.T) {
	out := FormatSQL("select * from users limit 10")
	assert.Contains(t, out, "\nLIMIT")
}

func TestFormatSQL_KeywordNotInIdentifier(t *testing.T) {
	// "orders" should not become "ORDERs"
	out := FormatSQL("select * from orders")
	assert.Contains(t, out, "orders", "identifier 'orders' should not be broken: %s", out)
}

// ─── Transform ────────────────────────────────────────────────────────────────

func TestTransform_Upper(t *testing.T) {
	assert.Equal(t, "HELLO WORLD", Transform("hello world", "upper"))
}

func TestTransform_Lower(t *testing.T) {
	assert.Equal(t, "hello world", Transform("HELLO WORLD", "lower"))
}

func TestTransform_CamelFromSpaces(t *testing.T) {
	assert.Equal(t, "helloWorld", Transform("hello world", "camel"))
}

func TestTransform_CamelFromSnake(t *testing.T) {
	assert.Equal(t, "userId", Transform("user_id", "camel"))
}

func TestTransform_CamelFromKebab(t *testing.T) {
	assert.Equal(t, "myVariableName", Transform("my-variable-name", "camel"))
}

func TestTransform_CamelSingleWord(t *testing.T) {
	assert.Equal(t, "hello", Transform("hello", "camel"))
}

func TestTransform_SnakeFromSpaces(t *testing.T) {
	assert.Equal(t, "hello_world", Transform("Hello World", "snake"))
}

func TestTransform_SnakeFromCamel(t *testing.T) {
	assert.Equal(t, "user_id", Transform("userId", "snake"))
}

func TestTransform_SnakeFromKebab(t *testing.T) {
	assert.Equal(t, "my_variable_name", Transform("my-variable-name", "snake"))
}

func TestTransform_KebabFromSpaces(t *testing.T) {
	assert.Equal(t, "hello-world", Transform("Hello World", "kebab"))
}

func TestTransform_KebabFromCamel(t *testing.T) {
	assert.Equal(t, "user-id", Transform("userId", "kebab"))
}

func TestTransform_KebabFromSnake(t *testing.T) {
	assert.Equal(t, "my-variable-name", Transform("my_variable_name", "kebab"))
}

func TestTransform_Base64Encode(t *testing.T) {
	out := Transform("hello", "base64encode")
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("hello")), out)
}

func TestTransform_Base64Decode(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("hello"))
	assert.Equal(t, "hello", Transform(encoded, "base64decode"))
}

func TestTransform_Base64RoundTrip(t *testing.T) {
	original := "Hello, World! 123"
	encoded := Transform(original, "base64encode")
	decoded := Transform(encoded, "base64decode")
	assert.Equal(t, original, decoded)
}

func TestTransform_Base64DecodeInvalid(t *testing.T) {
	invalid := "not-valid-base64!!!"
	assert.Equal(t, invalid, Transform(invalid, "base64decode"))
}

func TestTransform_UnknownOp(t *testing.T) {
	assert.Equal(t, "hello", Transform("hello", "unknownop"))
}
