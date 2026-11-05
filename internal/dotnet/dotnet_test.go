package dotnet

import (
	"os"
	"testing"
)

type MockRunner struct {
	Calls [][]string
	Err   error
	Out   []byte
}

func (m *MockRunner) Run(name string, args ...string) ([]byte, error) {
	call := append([]string{name}, args...)
	m.Calls = append(m.Calls, call)
	return m.Out, m.Err
}

func TestSetSecret_WithProject(t *testing.T) {
	mock := &MockRunner{}
	mgr := NewManager(mock)

	err := mgr.SetSecret("PostGres:Password", "mon_secret", "src/WebApi/WebApi.csproj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.Calls))
	}

	expectedArgs := []string{
		"dotnet", "user-secrets", "set", "PostGres:Password", "mon_secret", "--project", "src/WebApi/WebApi.csproj",
	}

	for i, arg := range expectedArgs {
		if mock.Calls[0][i] != arg {
			t.Errorf("at index %d: expected %s, got %s", i, arg, mock.Calls[0][i])
		}
	}
}

func TestSyncSecrets(t *testing.T) {
	mock := &MockRunner{}
	mgr := NewManager(mock)

	envValues := map[string]string{
		"POSTGRES_PASSWORD": "secret_postgres",
	}

	err := mgr.SyncSecrets(envValues, true, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 0 {
		t.Errorf("expected 0 calls in dry-run, got %d", len(mock.Calls))
	}

}
func TestSyncSecrets_Normal(t *testing.T) {
	mock := &MockRunner{}
	mgr := NewManager(mock)

	envValues := map[string]string{
		"POSTGRES_PASSWORD": "secret_postgres",
	}

	err := mgr.SyncSecrets(envValues, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(mock.Calls))
	}
	expectedCall := []string{
		"dotnet", "user-secrets", "set", "PostGres:Password", "secret_postgres",
	}

	for i, arg := range expectedCall {
		if mock.Calls[0][i] != arg {
			t.Errorf("at index %d: expected %s, got %s", i, arg, mock.Calls[0][i])
		}
	}
}

func TestInitUserSecrets(t *testing.T) {
	mock := &MockRunner{}
	mgr := NewManager(mock)

	err := mgr.InitUserSecrets("src/WebApi/WebApi.csproj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.Calls))
	}

	expectedCall := []string{
		"dotnet", "user-secrets", "init", "--project", "src/WebApi/WebApi.csproj",
	}

	for i, arg := range expectedCall {
		if mock.Calls[0][i] != arg {
			t.Errorf("at index %d: expected %s, got %s", i, arg, mock.Calls[0][i])
		}
	}
}

func TestHasUserSecretsId(t *testing.T) {
	// Create a temporary directory with a dummy csproj
	tmpDir := t.TempDir()
	
	// 1. Without UserSecretsId
	noSecretFile := tmpDir + "/NoSecret.csproj"
	if err := os.WriteFile(noSecretFile, []byte("<Project Sdk=\"Microsoft.NET.Sdk\"><PropertyGroup><TargetFramework>net10.0</TargetFramework></PropertyGroup></Project>"), 0644); err != nil {
		t.Fatalf("failed creating temp file: %v", err)
	}

	has, err := HasUserSecretsId(noSecretFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if has {
		t.Errorf("expected false, got true")
	}

	// 2. With UserSecretsId
	withSecretFile := tmpDir + "/WithSecret.csproj"
	if err := os.WriteFile(withSecretFile, []byte("<Project Sdk=\"Microsoft.NET.Sdk\"><PropertyGroup><UserSecretsId>guid-123</UserSecretsId></PropertyGroup></Project>"), 0644); err != nil {
		t.Fatalf("failed creating temp file: %v", err)
	}

	has, err = HasUserSecretsId(withSecretFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has {
		t.Errorf("expected true, got false")
	}
}
