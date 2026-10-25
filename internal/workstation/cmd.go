package workstation

import (
	"SetSecrets/internal/target"
	"SetSecrets/internal/tpm"
	"fmt"
	"os"
	"time"

	"SetSecrets/internal/prompt"

	"github.com/spf13/cobra"
)

// NewWorkstationCmd creates the parent 'workstation' command.
func NewWorkstationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workstation",
		Short: "Manage operator workstation identity and certificates",
	}
	cmd.AddCommand(NewEnrollCmd())

	return cmd
}

// NewEnrollCmd provisions a TPM key and enrolls the workstation.
func NewEnrollCmd() *cobra.Command {
	var (
		token           string
		workstationName string
		apiUrl          string
	)

	cmd := &cobra.Command{
		Use:   "enroll",
		Short: "Generate workstation key and request mTLS certificate",
		RunE: func(cmd *cobra.Command, args []string) error {

			// 1. Auto-detect local workstation-enroll.json file
			if cfg, err := LoadConfigFile("workstation-enroll.json"); err == nil {
				if token == "" {
					token = cfg.Token
				}
				if workstationName == "" && cfg.WorkstationName != "" {
					workstationName = cfg.WorkstationName
				}
				if cfg.ApiUrl != "" {
					apiUrl = cfg.ApiUrl
				}
				fmt.Println("📁 Loaded configuration from workstation-enroll.json")
			}

			if apiUrl == "" {
				apiUrl = target.Current().ApiUrl
			}
			if workstationName == "" {
				workstationName = "admin-workstation-01"
			}
			if token == "" {
				return fmt.Errorf("bootstrap token is required (use --token)")
			}

			machine := tpm.NewSimulatedMachine()
			defer machine.Close()

			if err := machine.GenerateKey(); err != nil {
				return fmt.Errorf("failed to generate key: %w", err)
			}

			if err := machine.SaveKeyToFile("workstation.key"); err != nil {
				return fmt.Errorf("failed to save private key: %w", err)
			}

			pubKeyPem, err := machine.PublicKeyPem()
			if err != nil {
				return fmt.Errorf("failed to export public key: %w", err)
			}

			fmt.Printf("Enrolling workstation '%s' with %s...\n", workstationName, apiUrl)

			resp, err := Enroll(
				apiUrl, EnrollRequest{
					PlainBootstrapToken: token,
					WorkstationName:     workstationName,
					PublicKeyPem:        pubKeyPem,
				})

			if err != nil {
				return fmt.Errorf("enrollement failed: %w", err)
			}

			if err := os.WriteFile("workstation.crt", []byte(resp.CertPem), 0600); err != nil {
				return fmt.Errorf("failed to save certificate: %w", err)
			}

			prompt.PrintWorkstationEnrollSuccess(
				workstationName,
				resp.SerialNumber,
				resp.ThumbprintSha256,
				resp.NotAfterUtc.Format(time.RFC3339),
				"workstation.crt",
			)

			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Bootstrap OTP token")
	cmd.Flags().StringVar(&workstationName, "name", "", "Workstation identifier name")
	cmd.Flags().StringVar(&apiUrl, "api-url", "", "Target backend API URL")
	return cmd
}
