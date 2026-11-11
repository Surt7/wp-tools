package dotnet

import (
	"SetSecrets/internal/config"
)

const ConfigFileName = config.ConfigFileName

// Config represents the multi-environment targets configuration.
type Config = config.Config

// DefaultConfig returns the default starting configuration with dev and prod scenarios.
func DefaultConfig() Config {
	return config.DefaultConfig()
}

// FindConfigFile locates the configuration file either at customPath, in current dir, or next to the executable.
func FindConfigFile(customPath string) string {
	return config.FindConfigFile(customPath)
}

// LoadConfig loads the config from file, or creates it with defaults if missing.
func LoadConfig(filePath string) (Config, error) {
	return config.LoadConfig(filePath)
}

// SaveConfig persists the configuration to a JSON file.
func SaveConfig(filePath string, cfg Config) error {
	return config.SaveConfig(filePath, cfg)
}

// ResolveProjectPath returns the target project path for the given environment and target alias.
func ResolveProjectPath(targetName string, directPath string, envName string, configPath string) (string, error) {
	return config.ResolveProjectPath(targetName, directPath, envName, configPath)
}

func getMapKeys[V any](m map[string]V) []string {
	return config.GetMapKeys(m)
}
