package envfile

import (
	"bytes"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	input := `# Commentary
POSTGRES_PASSWORD=CHANGE_ME
KEYCLOAK_ADMIN=admin
`
	reader := strings.NewReader(input)
	entries, err := Parse(reader)

	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(entries) != 3 {
		t.Fatal("Parse returned wrong number of entries")
	}

	if !entries[1].NeedsValue() {
		t.Errorf("Expected entry to need a value")
	}
}

func TestWrite(t *testing.T) {
	originalEntries := []Entry{
		{RawLine: "# Test Commentary", IsKey: false},
		{Key: "POSTGRES_PASSWORD", Value: "secret123", IsKey: true},
	}

	var buf bytes.Buffer

	err := Write(&buf, originalEntries)
	if err != nil {
		t.Fatalf("Parse failed : %v", err)
	}

	parsedEntries, err := Parse(&buf)
	if err != nil {
		t.Fatalf("Parse failed : %v", err)
	}
	if len(parsedEntries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(parsedEntries))
	}

	if parsedEntries[1].Value != "secret123" {
		t.Errorf("expected 'secret123', got '%s'", parsedEntries[1].Value)
	}

}

func TestSetEntry(t *testing.T) {
	entries := []Entry{
		{Key: "POSTGRES_PASSWORD", Value: "old_pass", IsKey: true},
	}

	entries = SetEntry(entries, "POSTGRES_PASSWORD", "new_pass")
	if entries[0].Value != "new_pass" {
		t.Errorf("expected 'new_pass', got '%s'", entries[0].Value)
	}

	entries = SetEntry(entries, "MINIO_PORT", "9000")
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[1].Key != "MINIO_PORT" || entries[1].Value != "9000" {
		t.Errorf("expected MINIO_PORT=9000, got %s=%s", entries[1].Key, entries[1].Value)
	}
}
