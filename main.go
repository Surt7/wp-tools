package main

import (
	"os"

	"SetSecrets/internal/dotnet"
	"SetSecrets/internal/enroll"
	"SetSecrets/internal/envfile"
	"SetSecrets/internal/session"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "wp-tools",
		Short: "WorldPinball operator and developer CLI toolkit",
	}

	// Environment & Secrets
	rootCmd.AddCommand(envfile.NewInitCmd())
	rootCmd.AddCommand(envfile.NewSetCmd())
	rootCmd.AddCommand(dotnet.NewDotnetCmd())

	// Machine Provisioning & Enrollment
	rootCmd.AddCommand(enroll.NewEnrollCmd())

	// Game Sessions & Lobby
	rootCmd.AddCommand(session.NewStartCmd())
	rootCmd.AddCommand(session.NewSubmitCmd())
	rootCmd.AddCommand(session.NewLobbyCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
