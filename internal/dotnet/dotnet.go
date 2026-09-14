package dotnet

import (
	"fmt"
	"os/exec"
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

var defaultManager = NewManager(nil)

func (m *Manager) SyncSecrets(envValues map[string]string, dryRun bool, projectPath string) error {
	mapping := DefaultMapping()

	for envKey, dotnetKey := range mapping {
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
func SyncSecrets(envValues map[string]string, dryRun bool, projectPath string) error {
	return defaultManager.SyncSecrets(envValues, dryRun, projectPath)
}
