package dotnet

import (
	"fmt"
	"os"
	"strings"

	"SetSecrets/internal/envfile"
	"SetSecrets/internal/prompt"

	"github.com/spf13/cobra"
)

// NewDotnetCmd creates the 'dotnet' command to sync .env secrets to .NET user-secrets.
func NewDotnetCmd() *cobra.Command {
	var (
		dryRun      bool
		projectPath string
		target      string
		envName     string
		only        string
		interactive bool
		autoInit    bool
	)

	cmd := &cobra.Command{
		Use:   "dotnet",
		Short: "Sync .env secrets to .NET User Secrets",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			entries, err := EnsureEnvFile(".env", ".env.example", autoInit, dryRun)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			envMap := envfile.ToMap(entries)

			// Load targets configuration
			cfg, err := LoadConfig("")
			if err != nil {
				fmt.Printf("Error loading config: %v\n", err)
				os.Exit(1)
			}

			// Interactive Huh TUI mode triggered explicitly (-i) or when no target or path is provided
			isInteractive := interactive || (target == "" && projectPath == "")
			if isInteractive {
				activeEnv := strings.ToLower(strings.TrimSpace(envName))
				if activeEnv == "" {
					activeEnv = cfg.DefaultEnvironment
					if activeEnv == "" {
						activeEnv = "dev"
					}
				}

				targetsMap, ok := cfg.Environments[activeEnv]
				if !ok {
					fmt.Printf("Error: unknown environment '%s'\n", activeEnv)
					os.Exit(1)
				}

				// 1. Choose target project interactively with Huh
				selectedTarget, err := prompt.SelectTarget(getMapKeys(targetsMap))
				if err != nil {
					fmt.Printf("Target selection cancelled or error: %v\n", err)
					os.Exit(0)
				}
				target = selectedTarget

				// 2. Choose secrets interactively with Huh multi-select
				var availableKeys []string
				mapping := DefaultMapping()
				preselected := make(map[string]bool)

				targetLower := strings.ToLower(target)
				for envKey := range mapping {
					if _, hasVal := envMap[envKey]; hasVal {
						availableKeys = append(availableKeys, envKey)

						// Smart preselection: for 'hub', preselect redis & rabbit; for others, preselect all
						if targetLower == "hub" {
							if strings.Contains(envKey, "REDIS") || strings.Contains(envKey, "RABBIT") {
								preselected[envKey] = true
							}
						} else {
							preselected[envKey] = true
						}
					}
				}

				chosenKeys, err := prompt.SelectSecretsToSync(availableKeys, preselected)
				if err != nil {
					fmt.Printf("Secrets selection cancelled or error: %v\n", err)
					os.Exit(0)
				}

				if len(chosenKeys) == 0 {
					fmt.Println("No secrets selected. Aborting.")
					os.Exit(0)
				}

				resolvedPath, err := ResolveProjectPath(target, projectPath, envName, "")
				if err != nil {
					fmt.Printf("Error resolving target: %v\n", err)
					os.Exit(1)
				}

				if resolvedPath != "" {
					if _, err := os.Stat(resolvedPath); err != nil {
						fmt.Printf("Error: project file not found on disk at '%s'\nPlease check the path in %s\n", resolvedPath, ConfigFileName)
						os.Exit(1)
					}
					if err := EnsureUserSecretsInitialized(resolvedPath, target, autoInit, dryRun); err != nil {
						fmt.Printf("Error: %v\n", err)
						os.Exit(1)
					}
				}

				if err := SyncSecrets(envMap, dryRun, resolvedPath, chosenKeys); err != nil {
					fmt.Printf("Error syncing secrets: %v\n", err)
					os.Exit(1)
				}
				if dryRun {
					fmt.Println("[DRY-RUN] Simulation complete: secrets would be synced to .NET User Secrets")
				} else {
					fmt.Println("✔ Successfully synced secrets to .NET User Secrets")
				}
				return
			}

			// CLI Non-Interactive Mode
			resolvedPath, err := ResolveProjectPath(target, projectPath, envName, "")
			if err != nil {
				fmt.Printf("Error resolving target: %v\n", err)
				os.Exit(1)
			}

			if resolvedPath != "" {
				if _, err := os.Stat(resolvedPath); err != nil {
					fmt.Printf("Error: project file not found on disk at '%s'\nPlease check the path in %s\n", resolvedPath, ConfigFileName)
					os.Exit(1)
				}
				if err := EnsureUserSecretsInitialized(resolvedPath, target, autoInit, dryRun); err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
			}

			var allowedKeys []string
			if only != "" {
				tokens := strings.Split(only, ",")
				mapping := DefaultMapping()
				for envKey := range mapping {
					envUpper := strings.ToUpper(envKey)
					for _, tok := range tokens {
						trimmed := strings.ToUpper(strings.TrimSpace(tok))
						if trimmed != "" && strings.Contains(envUpper, trimmed) {
							allowedKeys = append(allowedKeys, envKey)
							break
						}
					}
				}
			}

			if err := SyncSecrets(envMap, dryRun, resolvedPath, allowedKeys); err != nil {
				fmt.Printf("Error syncing secrets: %v\n", err)
				os.Exit(1)
			}
			if dryRun {
				fmt.Println("[DRY-RUN] Simulation complete: secrets would be synced to .NET User Secrets")
			} else {
				fmt.Println("✔ Successfully synced secrets to .NET User Secrets")
			}
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Simulate without executing dotnet commands")
	cmd.Flags().StringVarP(&projectPath, "project", "p", "", "Path to the .csproj or project directory")
	cmd.Flags().StringVarP(&target, "target", "t", "", "Target project alias (hub, leaderboard, logistics)")
	cmd.Flags().StringVarP(&envName, "env", "e", "dev", "Environment scenario (dev, prod)")
	cmd.Flags().StringVarP(&only, "only", "o", "", "Filter secrets by comma-separated keywords (e.g. redis,rabbit)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive terminal form (Huh UI)")
	cmd.Flags().BoolVar(&autoInit, "auto-init", false, "Auto-initialize dotnet user-secrets if missing without prompting")

	return cmd
}
