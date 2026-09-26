package dotnet

import (
	"fmt"
	"os"

	"SetSecrets/internal/envfile"

	"github.com/spf13/cobra"
)

// NewDotnetCmd creates the 'dotnet' command to sync .env secrets to .NET user-secrets.
func NewDotnetCmd() *cobra.Command {
	var (
		dryRun      bool
		projectPath string
	)

	cmd := &cobra.Command{
		Use:   "dotnet",
		Short: "Sync .env secrets to .NET User Secrets",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			entries, err := envfile.LoadFile(".env")
			if err != nil {
				fmt.Printf("Error loading .env file: %v\n", err)
				os.Exit(1)
			}
			envMap := envfile.ToMap(entries)

			if err := SyncSecrets(envMap, dryRun, projectPath); err != nil {
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

	return cmd
}
