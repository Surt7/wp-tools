package main

import (
	"os"

	"SetSecrets/internal/dotnet"
	"SetSecrets/internal/enroll"
	"SetSecrets/internal/envfile"
	"SetSecrets/internal/session"
	"SetSecrets/internal/target"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "wp-tools",
		Short: "WorldPinball operator and developer CLI toolkit",
	}

	// Global target selection flags
	rootCmd.PersistentFlags().BoolVar(&target.TargetSpring, "spring", false, "Target Spring Boot backend (HTTP: 7085, gRPC: 9090)")
	rootCmd.PersistentFlags().BoolVar(&target.TargetSpring, "springb", false, "Alias for --spring")
	rootCmd.PersistentFlags().BoolVar(&target.TargetAsp, "asp", false, "Target ASP.NET Core backend (HTTP/gRPC: 7010, default)")

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
