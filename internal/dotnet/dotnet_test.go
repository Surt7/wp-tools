package dotnet

import "testing"

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

	err := mgr.SyncSecrets(envValues, true, "")
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

	err := mgr.SyncSecrets(envValues, false, "")
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
