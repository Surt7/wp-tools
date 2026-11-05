package dotnet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureEnvFile_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	examplePath := filepath.Join(tmpDir, ".env.example")

	content := "REDIS_PASSWORD=my_existing_secret\n"
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed creating test env: %v", err)
	}

	entries, err := EnsureEnvFile(envPath, examplePath, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 || entries[0].Key != "REDIS_PASSWORD" || entries[0].Value != "my_existing_secret" {
		t.Errorf("unexpected entries: %+v", entries)
	}
}

func TestEnsureEnvFile_AutoInitFromExample(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	examplePath := filepath.Join(tmpDir, ".env.example")

	exampleContent := "POSTGRES_PASSWORD=CHANGE_ME\nREDIS_HOST=localhost\n"
	if err := os.WriteFile(examplePath, []byte(exampleContent), 0644); err != nil {
		t.Fatalf("failed creating test example: %v", err)
	}

	// .env does NOT exist; autoInit=true should create it
	entries, err := EnsureEnvFile(envPath, examplePath, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Verify .env was written to disk
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Errorf("expected .env to be created on disk")
	}

	// Verify POSTGRES_PASSWORD was generated and is not CHANGE_ME
	for _, entry := range entries {
		if entry.Key == "POSTGRES_PASSWORD" {
			if entry.Value == "CHANGE_ME" || entry.Value == "" {
				t.Errorf("expected generated password, got %s", entry.Value)
			}
		}
		if entry.Key == "REDIS_HOST" {
			if entry.Value != "localhost" {
				t.Errorf("expected host to be preserved, got %s", entry.Value)
			}
		}
	}
}

func TestEnsureEnvFile_NeitherExists(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	examplePath := filepath.Join(tmpDir, ".env.example")

	_, err := EnsureEnvFile(envPath, examplePath, true, false)
	if err == nil {
		t.Fatalf("expected error when neither exists, got nil")
	}
}
