package envfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"SetSecrets/internal/config"
	"SetSecrets/internal/generator"
	"SetSecrets/internal/prompt"

	"github.com/spf13/cobra"
)

// NewInitCmd creates the 'init' command to generate .env from .env.example.
func NewInitCmd() *cobra.Command {
	var (
		force       bool
		target      string
		envName     string
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize secrets in .env file (supports interactive target selection and remote projects)",
		Run: func(cmd *cobra.Command, args []string) {
			isInteractive := interactive || target == ""

			if isInteractive {
				cfg, err := config.LoadConfig("")
				if err == nil {
					activeEnv := strings.ToLower(strings.TrimSpace(envName))
					if activeEnv == "" {
						activeEnv = cfg.DefaultEnvironment
						if activeEnv == "" {
							activeEnv = "dev"
						}
					}

					targetsMap, ok := cfg.Environments[activeEnv]
					if ok && len(targetsMap) > 0 {
						targetOptions := append([]string{"local (current directory)"}, config.GetMapKeys(targetsMap)...)
						selected, err := prompt.SelectTarget(targetOptions)
						if err != nil {
							fmt.Printf("Target selection cancelled: %v\n", err)
							os.Exit(0)
						}
						if strings.HasPrefix(selected, "local") {
							target = "local"
						} else {
							target = selected
						}
					}
				}
			}

			targetDir, err := config.ResolveEnvDir(target, envName, "")
			if err != nil {
				fmt.Printf("Error resolving target '%s': %v\n", target, err)
				os.Exit(1)
			}

			examplePath := filepath.Join(targetDir, ".env.example")
			destPath := filepath.Join(targetDir, ".env")

			if _, err := os.Stat(examplePath); os.IsNotExist(err) {
				fmt.Printf("Error: '%s' not found.\nPlease make sure .env.example exists in %s\n", examplePath, targetDir)
				os.Exit(1)
			}

			if err := InitFromExample(examplePath, destPath, force); err != nil {
				fmt.Printf("Error generating .env file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("\n✔ Successfully generated '%s' from '%s'\n", destPath, examplePath)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force regeneration of all variables, even if already set")
	cmd.Flags().StringVarP(&target, "target", "t", "", "Target project name from wp-config.json (e.g. media-vault, hub, leaderboard)")
	cmd.Flags().StringVarP(&envName, "env", "e", "dev", "Target environment (dev, prod)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode with Huh TUI")
	return cmd
}

// NewSetCmd creates the 'set' command to generate/update secrets for specific keys in .env.
func NewSetCmd() *cobra.Command {
	var (
		target      string
		envName     string
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "set [KEY] [VALUE]",
		Short: "Set or generate a secret for a specific key in .env (supports interactive target selection and remote projects)",
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			isInteractive := interactive || (target == "" && len(args) == 0)

			if isInteractive && target == "" {
				cfg, err := config.LoadConfig("")
				if err == nil {
					activeEnv := strings.ToLower(strings.TrimSpace(envName))
					if activeEnv == "" {
						activeEnv = cfg.DefaultEnvironment
						if activeEnv == "" {
							activeEnv = "dev"
						}
					}

					targetsMap, ok := cfg.Environments[activeEnv]
					if ok && len(targetsMap) > 0 {
						targetOptions := append([]string{"local (current directory)"}, config.GetMapKeys(targetsMap)...)
						selected, err := prompt.SelectTarget(targetOptions)
						if err != nil {
							fmt.Printf("Target selection cancelled: %v\n", err)
							os.Exit(0)
						}
						if strings.HasPrefix(selected, "local") {
							target = "local"
						} else {
							target = selected
						}
					}
				}
			}

			targetDir, err := config.ResolveEnvDir(target, envName, "")
			if err != nil {
				fmt.Printf("Error resolving target '%s': %v\n", target, err)
				os.Exit(1)
			}

			destPath := filepath.Join(targetDir, ".env")
			examplePath := filepath.Join(targetDir, ".env.example")

			// If .env doesn't exist yet, offer to initialize it from .env.example
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				if _, errEx := os.Stat(examplePath); errEx == nil {
					fmt.Printf("'%s' not found. Initializing from '%s'...\n", destPath, examplePath)
					if errInit := InitFromExample(examplePath, destPath, false); errInit != nil {
						fmt.Printf("Error creating .env from example: %v\n", errInit)
						os.Exit(1)
					}
				} else {
					fmt.Printf("Error: '%s' not found and no .env.example present.\n", destPath)
					os.Exit(1)
				}
			}

			entries, err := LoadFile(destPath)
			if err != nil {
				fmt.Printf("Error loading '%s': %v\n", destPath, err)
				os.Exit(1)
			}

			var targetKeys []string
			var explicitValue string

			if len(args) == 0 {
				var availableKeys []string
				for _, entry := range entries {
					if entry.IsKey {
						availableKeys = append(availableKeys, entry.Key)
					}
				}

				if len(availableKeys) == 0 {
					fmt.Printf("No keys found in '%s'.\n", destPath)
					os.Exit(0)
				}

				selected, err := prompt.SelectKeys(availableKeys)
				if err != nil {
					fmt.Printf("Error selecting keys: %v\n", err)
					os.Exit(1)
				}
				if len(selected) == 0 {
					fmt.Println("No keys selected.")
					os.Exit(0)
				}
				targetKeys = selected
			} else {
				targetKeys = []string{args[0]}
				if len(args) >= 2 {
					explicitValue = args[1]
				}
			}

			for _, key := range targetKeys {
				val := explicitValue
				if val == "" {
					generated, err := generator.GenerateRandomPassword(24)
					if err != nil {
						fmt.Printf("Error generating password for %s: %v\n", key, err)
						os.Exit(1)
					}
					val = generated
					fmt.Printf("✔ Generated random secret for: %s\n", key)
				} else {
					fmt.Printf("✔ Set explicit secret for: %s\n", key)
				}
				entries = SetEntry(entries, key, val)
			}

			if err := SaveFile(destPath, entries); err != nil {
				fmt.Printf("Error saving '%s': %v\n", destPath, err)
				os.Exit(1)
			}

			fmt.Printf("\n✔ Successfully updated '%s'\n", destPath)
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "", "Target project name from wp-config.json (e.g. media-vault, hub, leaderboard)")
	cmd.Flags().StringVarP(&envName, "env", "e", "dev", "Target environment (dev, prod)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode with Huh TUI")
	return cmd
}
