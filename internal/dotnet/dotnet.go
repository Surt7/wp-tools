package dotnet

import (
	"fmt"
	"os/exec"
)

func DefaultMapping() map[string]string {
	return map[string]string{
		"POSTGRES_PASSWORD":     "PostGres:Password",
		"REDIS_PASSWORD":        "Redis:Password",
		"RABBITMQ_DEFAULT_PASS": "RabbitMQ:Password",
		"KEYCLOAK_DB_PASSWORD":  "Keycloak:ClientSecret",
	}
}

func SetSecret(key string, value string) error {
	cmd := exec.Command("dotnet", "user-secrets", "set", key, value)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dotnet user-secrets failed: %s (%w)", string(output), err)
	}
	return nil
}
func SyncSecrets(envValues map[string]string, dryRun bool) error {
	mapping := DefaultMapping()

	for envKey, dotnetKey := range mapping {
		// If the variable exists in our .env and has a value
		if val, exists := envValues[envKey]; exists && val != "" {

			if dryRun {
				fmt.Printf("[DRY-RUN] Would set: %s = %s\n", dotnetKey, val)
				continue
			}

			if err := SetSecret(dotnetKey, val); err != nil {
				return fmt.Errorf("failed setting %s (%s): %w", dotnetKey, envKey, err)
			}
			fmt.Printf("✔ Synced to .NET User Secrets: %s\n", dotnetKey)
		}
	}

	return nil
}
