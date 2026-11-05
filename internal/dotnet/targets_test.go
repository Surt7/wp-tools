package dotnet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveProjectPath(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.json")

	cfg := Config{
		DefaultEnvironment: "dev",
		Environments: map[string]map[string]string{
			"dev": {
				"hub":         `D:\Dev\Hub.csproj`,
				"leaderboard": `D:\Dev\Leaderboard.csproj`,
			},
			"prod": {
				"hub":         `/var/www/hub/Hub.csproj`,
				"leaderboard": `/var/www/leaderboard/Leaderboard.csproj`,
			},
		},
	}
	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}

	tests := []struct {
		name       string
		target     string
		directPath string
		envName    string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "direct path takes precedence over everything",
			target:     "hub",
			directPath: `C:\Custom\Project.csproj`,
			envName:    "prod",
			wantPath:   `C:\Custom\Project.csproj`,
			wantErr:    false,
		},
		{
			name:       "default dev scenario resolves dev path",
			target:     "hub",
			directPath: "",
			envName:    "",
			wantPath:   `D:\Dev\Hub.csproj`,
			wantErr:    false,
		},
		{
			name:       "explicit dev scenario resolves dev path",
			target:     "leaderboard",
			directPath: "",
			envName:    "dev",
			wantPath:   `D:\Dev\Leaderboard.csproj`,
			wantErr:    false,
		},
		{
			name:       "prod scenario resolves prod path",
			target:     "hub",
			directPath: "",
			envName:    "prod",
			wantPath:   `/var/www/hub/Hub.csproj`,
			wantErr:    false,
		},
		{
			name:       "case-insensitive target in prod",
			target:     "LeaderBoard",
			directPath: "",
			envName:    "PROD",
			wantPath:   `/var/www/leaderboard/Leaderboard.csproj`,
			wantErr:    false,
		},
		{
			name:       "unknown environment returns error",
			target:     "hub",
			directPath: "",
			envName:    "staging",
			wantPath:   "",
			wantErr:    true,
		},
		{
			name:       "unknown target returns error",
			target:     "billing",
			directPath: "",
			envName:    "dev",
			wantPath:   "",
			wantErr:    true,
		},
		{
			name:       "empty target returns empty path without error",
			target:     "",
			directPath: "",
			envName:    "dev",
			wantPath:   "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveProjectPath(tt.target, tt.directPath, tt.envName, configPath)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveProjectPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantPath {
				t.Errorf("ResolveProjectPath() got = %v, want %v", got, tt.wantPath)
			}
		})
	}

	// Test default file creation when missing
	missingPath := filepath.Join(tempDir, "missing.json")
	resolved, err := ResolveProjectPath("hub", "", "dev", missingPath)
	if err != nil {
		t.Fatalf("expected auto-created config to work, got err: %v", err)
	}
	if resolved == "" {
		t.Errorf("expected non-empty default resolved path")
	}
	if _, err := os.Stat(missingPath); err != nil {
		t.Errorf("expected %s to have been auto-created on disk", missingPath)
	}
}
