package envfile

import (
	"bytes"
	"os"
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

func TestIsSecretKey(t *testing.T) {
	tests := []struct {
		key      string
		isSecret bool
	}{
		{"POSTGRES_PASSWORD", true},
		{"RABBITMQ_DEFAULT_PASS", true},
		{"KEYCLOAK_DB_PASSWORD", true},
		{"JWT_SECRET", true},
		{"API_TOKEN", true},
		{"DEVICE_API_KEY", true},
		{"KEYCLOAK_ADMIN", false},
		{"RABBITMQ_DEFAULT_USER", false},
		{"REDIS_HOST", false},
		{"MONGO_DATABASE", false},
		{"MONGO_INITDB_ROOT_USERNAME", false},
	}

	for _, tt := range tests {
		e := Entry{Key: tt.key, IsKey: true}
		if got := e.IsSecretKey(); got != tt.isSecret {
			t.Errorf("IsSecretKey(%q) = %v, want %v", tt.key, got, tt.isSecret)
		}
	}
}

func TestMergeAndInitFromExample(t *testing.T) {
	exampleContent := `POSTGRES_PASSWORD=CHANGE_ME
MINIO_ROOT_USER=mediavault_admin
MINIO_ROOT_PASSWORD=CHANGE_ME
`
	existingContent := `POSTGRES_PASSWORD=my_existing_secret_123
`
	tmpDir := t.TempDir()
	examplePath := tmpDir + "/.env.example"
	destPath := tmpDir + "/.env"

	if err := os.WriteFile(examplePath, []byte(exampleContent), 0644); err != nil {
		t.Fatalf("failed to write example: %v", err)
	}
	if err := os.WriteFile(destPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write dest: %v", err)
	}

	// Only MINIO_ROOT_PASSWORD is selected to be generated. POSTGRES_PASSWORD should be preserved!
	selected := []string{"MINIO_ROOT_PASSWORD"}
	err := MergeAndInitFromExample(examplePath, destPath, selected, false)
	if err != nil {
		t.Fatalf("MergeAndInitFromExample failed: %v", err)
	}

	entries, err := LoadFile(destPath)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	m := ToMap(entries)
	if m["POSTGRES_PASSWORD"] != "my_existing_secret_123" {
		t.Errorf("expected POSTGRES_PASSWORD to be preserved as 'my_existing_secret_123', got '%s'", m["POSTGRES_PASSWORD"])
	}
	if m["MINIO_ROOT_PASSWORD"] == "CHANGE_ME" || len(m["MINIO_ROOT_PASSWORD"]) < 8 {
		t.Errorf("expected MINIO_ROOT_PASSWORD to be generated, got '%s'", m["MINIO_ROOT_PASSWORD"])
	}
	if m["MINIO_ROOT_USER"] != "mediavault_admin" {
		t.Errorf("expected MINIO_ROOT_USER to be 'mediavault_admin', got '%s'", m["MINIO_ROOT_USER"])
	}
}
