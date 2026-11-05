package dotnet

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}
type RealCommandRunner struct{}

func (r RealCommandRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

type Manager struct {
	Runner CommandRunner
}

func NewManager(runner CommandRunner) *Manager {
	if runner == nil {
		runner = RealCommandRunner{}
	}
	return &Manager{
		Runner: runner,
	}
}

func DefaultMapping() map[string]string {
	return map[string]string{
		"POSTGRES_PASSWORD":     "PostGres:Password",
		"REDIS_PASSWORD":        "Redis:Password",
		"RABBITMQ_DEFAULT_PASS": "RabbitMQ:Password",
		"KEYCLOAK_DB_PASSWORD":  "Keycloak:ClientSecret",
		"KAFKA_PASSWORD":        "Kafka:Password",
	}
}

func (m *Manager) SetSecret(key string, value string, projectPath string) error {
	args := []string{"user-secrets", "set", key, value}
	if projectPath != "" {
		args = append(args, "--project", projectPath)
	}

	output, err := m.Runner.Run("dotnet", args...)
	if err != nil {
		return fmt.Errorf("dotnet user-secrets failed: %s (%w)", string(output), err)
	}
	return nil
}

// InitUserSecrets runs 'dotnet user-secrets init' on the specified project.
func (m *Manager) InitUserSecrets(projectPath string) error {
	args := []string{"user-secrets", "init"}
	if projectPath != "" {
		args = append(args, "--project", projectPath)
	}

	output, err := m.Runner.Run("dotnet", args...)
	if err != nil {
		return fmt.Errorf("dotnet user-secrets init failed: %s (%w)", string(output), err)
	}
	return nil
}

// HasUserSecretsId checks whether the specified .csproj project file contains a UserSecretsId element.
func HasUserSecretsId(projectPath string) (bool, error) {
	targetFile := projectPath
	if targetFile == "" {
		targetFile = "."
	}
	fi, err := os.Stat(targetFile)
	if err != nil {
		return false, err
	}
	if fi.IsDir() {
		files, err := filepath.Glob(filepath.Join(targetFile, "*.csproj"))
		if err != nil || len(files) == 0 {
			return false, fmt.Errorf("no .csproj found in directory %s", targetFile)
		}
		targetFile = files[0]
	}
	data, err := os.ReadFile(targetFile)
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), "<UserSecretsId>"), nil
}

var defaultManager = NewManager(nil)

func InitUserSecrets(projectPath string) error {
	return defaultManager.InitUserSecrets(projectPath)
}

func (m *Manager) SyncSecrets(envValues map[string]string, dryRun bool, projectPath string, allowedEnvKeys []string) error {
	mapping := DefaultMapping()

	var filterMap map[string]bool
	if len(allowedEnvKeys) > 0 {
		filterMap = make(map[string]bool)
		for _, k := range allowedEnvKeys {
			filterMap[strings.ToUpper(strings.TrimSpace(k))] = true
		}
	}

	for envKey, dotnetKey := range mapping {
		if filterMap != nil && !filterMap[strings.ToUpper(envKey)] {
			continue
		}

		// If the variable exists in our .env and has a value
		if val, exists := envValues[envKey]; exists && val != "" {

			if dryRun {
				fmt.Printf("[DRY-RUN] Would set: %s = %s\n", dotnetKey, val)
				continue
			}

			if err := m.SetSecret(dotnetKey, val, projectPath); err != nil {
				return fmt.Errorf("failed setting %s (%s): %w", dotnetKey, envKey, err)
			}
			fmt.Printf("✔ Synced to .NET User Secrets: %s\n", dotnetKey)
		}
	}

	return nil
}
func SyncSecrets(envValues map[string]string, dryRun bool, projectPath string, allowedEnvKeys []string) error {
	return defaultManager.SyncSecrets(envValues, dryRun, projectPath, allowedEnvKeys)
}
