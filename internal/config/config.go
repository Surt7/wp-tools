package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ConfigFileName = "wp-config.json"

// Config represents the multi-environment targets configuration.
type Config struct {
	DefaultEnvironment string                       `json:"defaultEnvironment"`
	Environments       map[string]map[string]string `json:"environments"`
}

// DefaultConfig returns the default starting configuration with dev and prod scenarios.
func DefaultConfig() Config {
	return Config{
		DefaultEnvironment: "dev",
		Environments: map[string]map[string]string{
			"dev": {
				"hub":         `D:\Workspace\WorldPinball.GgWp.Hub\WorldPinball.GgWp.Hub.Api\WorldPinball.GgWp.Hub.Api.csproj`,
				"leaderboard": `C:\Users\ippo_\source\repos\Leaderboard_WorldPinball\WorldPinball.Api\WorldPinball.Api.csproj`,
				"logistics":   `C:\Users\ippo_\source\repos\Leaderboard_WorldPinball\Logistics\WorldPinball.Logistics.Api\WorldPinball.Logistics.Api.csproj`,
				"media-vault": `D:\Workspace\MediaVault\MediaVault.Api\MediaVault.Api.csproj`,
			},
			"prod": {
				"hub":         `/var/www/hub/WorldPinball.GgWp.Hub.Api.csproj`,
				"leaderboard": `/var/www/leaderboard/WorldPinball.Api.csproj`,
				"logistics":   `/var/www/logistics/WorldPinball.Logistics.Api.csproj`,
			},
		},
	}
}

// FindConfigFile locates the configuration file either at customPath, in current dir, or next to the executable.
func FindConfigFile(customPath string) string {
	if customPath != "" {
		return customPath
	}
	if _, err := os.Stat(ConfigFileName); err == nil {
		return ConfigFileName
	}
	if exePath, err := os.Executable(); err == nil {
		exeConfig := filepath.Join(filepath.Dir(exePath), ConfigFileName)
		if _, err := os.Stat(exeConfig); err == nil {
			return exeConfig
		}
	}
	return ConfigFileName
}

// LoadConfig loads the config from file, or creates it with defaults if missing.
func LoadConfig(filePath string) (Config, error) {
	filePath = FindConfigFile(filePath)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			_ = SaveConfig(filePath, cfg)
			return cfg, nil
		}
		return Config{}, fmt.Errorf("reading %s: %w", filePath, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing %s: %w", filePath, err)
	}

	return cfg, nil
}

// SaveConfig persists the configuration to a JSON file.
func SaveConfig(filePath string, cfg Config) error {
	filePath = FindConfigFile(filePath)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// ResolveProjectPath returns the target project path for the given environment and target alias.
func ResolveProjectPath(targetName string, directPath string, envName string, configPath string) (string, error) {
	// 1. Direct path flag (-p) takes precedence if specified
	if directPath != "" {
		return directPath, nil
	}

	// 2. If no target specified or "local", fallback to default (current directory)
	if targetName == "" || strings.ToLower(targetName) == "local" {
		return "", nil
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		return "", err
	}

	// 3. Resolve active environment (defaults to cfg.DefaultEnvironment or "dev")
	activeEnv := strings.ToLower(strings.TrimSpace(envName))
	if activeEnv == "" {
		activeEnv = strings.ToLower(strings.TrimSpace(cfg.DefaultEnvironment))
		if activeEnv == "" {
			activeEnv = "dev"
		}
	}

	targets, envExists := cfg.Environments[activeEnv]
	if !envExists {
		return "", fmt.Errorf("unknown environment '%s'. Available environments: %v", activeEnv, GetMapKeys(cfg.Environments))
	}

	// 4. Lookup target alias case-insensitively
	targetNormalized := strings.ToLower(strings.TrimSpace(targetName))
	for k, v := range targets {
		if strings.ToLower(k) == targetNormalized {
			return v, nil
		}
	}

	return "", fmt.Errorf("unknown target '%s' in environment '%s'. Available targets: %v", targetName, activeEnv, GetMapKeys(targets))
}

// ResolveEnvDir finds the root directory of a target project where .env or .env.example lives.
func ResolveEnvDir(targetName string, envName string, configPath string) (string, error) {
	if targetName == "" || strings.ToLower(targetName) == "local" {
		return ".", nil
	}

	projectPath, err := ResolveProjectPath(targetName, "", envName, configPath)
	if err != nil {
		return "", err
	}
	if projectPath == "" {
		return ".", nil
	}

	// If projectPath is a file (e.g. .csproj), start searching from its directory
	current := projectPath
	fi, err := os.Stat(current)
	if err == nil && !fi.IsDir() {
		current = filepath.Dir(current)
	}

	// Walk up to 3 levels to locate .env.example, .env, or docker-compose.yml
	for i := 0; i < 3; i++ {
		if _, err := os.Stat(filepath.Join(current, ".env.example")); err == nil {
			return current, nil
		}
		if _, err := os.Stat(filepath.Join(current, ".env")); err == nil {
			return current, nil
		}
		if _, err := os.Stat(filepath.Join(current, "docker-compose.yml")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	// Fallback to the project's parent directory
	return filepath.Dir(projectPath), nil
}

// GetMapKeys returns all keys of a map as a slice of strings.
func GetMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
