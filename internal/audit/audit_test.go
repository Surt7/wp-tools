package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGitRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create fake .git folder
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed creating test .git: %v", err)
	}

	// Create nested project directory
	subDir := filepath.Join(tmpDir, "src", "Project.Api")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed creating test subDir: %v", err)
	}

	foundRoot, err := FindGitRoot(subDir)
	if err != nil {
		t.Fatalf("unexpected error finding git root: %v", err)
	}

	if foundRoot != tmpDir {
		t.Errorf("expected %s, got %s", tmpDir, foundRoot)
	}
}
