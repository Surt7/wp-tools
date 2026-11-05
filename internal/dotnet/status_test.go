package dotnet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetUserSecretsId(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Project with UserSecretsId
	withGuidPath := filepath.Join(tmpDir, "WithGuid.csproj")
	contentWithGuid := `<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
    <UserSecretsId>f293a81d-2454-4f6b-8e24-65a7f151f667</UserSecretsId>
  </PropertyGroup>
</Project>`
	if err := os.WriteFile(withGuidPath, []byte(contentWithGuid), 0644); err != nil {
		t.Fatalf("failed creating file: %v", err)
	}

	guid, err := GetUserSecretsId(withGuidPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if guid != "f293a81d-2454-4f6b-8e24-65a7f151f667" {
		t.Errorf("expected guid f293a81d-2454-4f6b-8e24-65a7f151f667, got %s", guid)
	}

	// 2. Project without UserSecretsId
	noGuidPath := filepath.Join(tmpDir, "NoGuid.csproj")
	contentNoGuid := `<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
  </PropertyGroup>
</Project>`
	if err := os.WriteFile(noGuidPath, []byte(contentNoGuid), 0644); err != nil {
		t.Fatalf("failed creating file: %v", err)
	}

	guid, err = GetUserSecretsId(noGuidPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if guid != "" {
		t.Errorf("expected empty guid, got %s", guid)
	}
}

func TestListSecrets(t *testing.T) {
	mockOutput := "Redis:Password = my_redis_secret\nRabbitMq:Password = my_rabbit_secret\n"
	mock := &MockRunner{
		Out: []byte(mockOutput),
	}
	mgr := NewManager(mock)

	secrets, err := mgr.ListSecrets("src/WebApi.csproj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(secrets) != 2 {
		t.Fatalf("expected 2 secrets, got %d", len(secrets))
	}

	if secrets["Redis:Password"] != "my_redis_secret" {
		t.Errorf("expected my_redis_secret, got %s", secrets["Redis:Password"])
	}
	if secrets["RabbitMq:Password"] != "my_rabbit_secret" {
		t.Errorf("expected my_rabbit_secret, got %s", secrets["RabbitMq:Password"])
	}
}
