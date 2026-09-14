package main

import (
	"SetSecrets/internal/dotnet"
	"SetSecrets/internal/envfile"
	"SetSecrets/internal/generator"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "setsecrets",
		Short: "Configure app user-secrets",
		Args:  cobra.NoArgs,
	}

	var force bool
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize secrets on .env file",
		Run: func(cmd *cobra.Command, args []string) {
			entries, err := envfile.LoadFile(".env.example")
			if err != nil {
				fmt.Printf("Error loading .env.example: %v\n", err)
				os.Exit(1)
			}

			for i := range entries {
				// Mode normal : remplit seulement si CHANGE_ME ou vide
				// Mode force (-f / --force) : remplace absolument toutes les clés
				if entries[i].IsKey && (entries[i].NeedsValue() || force) {
					pwd, err := generator.GenerateRandomPassword(24)
					if err != nil {
						fmt.Printf("Error generating password for %s: %v\n", entries[i].Key, err)
						os.Exit(1)
					}

					entries[i].Value = pwd
					fmt.Printf("✔ Generated secret for: %s\n", entries[i].Key)
				}
			}

			if err := envfile.SaveFile(".env", entries); err != nil {
				fmt.Printf("Error saving .env file: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("\n Successfully generated .env file!")
		},
	}
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "Force regeneration of all variables, even if already set")
	rootCmd.AddCommand(initCmd)

	setCmd := &cobra.Command{
		Use:   "set <KEY> [VALUE]",
		Short: "Set or generate a secret for a specific key in .env",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]
			var value string

			if len(args) == 2 {
				value = args[1]
			} else {
				var err error
				value, err = generator.GenerateRandomPassword(24)
				if err != nil {
					fmt.Printf("Error generating password: %v\n", err)
					os.Exit(1)
				}
			}

			entries, _ := envfile.LoadFile(".env")
			entries = envfile.SetEntry(entries, key, value)

			if err := envfile.SaveFile(".env", entries); err != nil {
				fmt.Printf("Error saving .env file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("✔ Successfully set %s in .env\n", key)
		},
	}
	rootCmd.AddCommand(setCmd)

	var dryRun bool
	dotnetCmd := &cobra.Command{
		Use:   "dotnet",
		Short: "Sync .env secrets to .NET User Secrets",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, arg []string) {
			entries, err := envfile.LoadFile(".env")
			if err != nil {
				fmt.Printf("Error loading .env file: %v\n", err)
				os.Exit(1)
			}
			envMap := envfile.ToMap(entries)

			if err := dotnet.SyncSecrets(envMap, dryRun); err != nil {
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
	dotnetCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Simulate without executing dotnet commands")
	rootCmd.AddCommand(dotnetCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
