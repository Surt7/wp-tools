package software

import (
	"fmt"
	"os"
	"time"

	"SetSecrets/internal/prompt"
	"SetSecrets/internal/target"
	"SetSecrets/internal/tpm"

	"github.com/spf13/cobra"
)

// NewSoftwareCmd creates the parent 'software' command.
func NewSoftwareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "software",
		Short: "Manage software code signing keys and certificates",
	}
	cmd.AddCommand(NewCertCmd())
	return cmd
}

// NewCertCmd generates a code signing key and requests an X.509 certificate.
func NewCertCmd() *cobra.Command {
	var (
		token         string
		publisherName string
		apiUrl        string
	)

	cmd := &cobra.Command{
		Use:   "cert",
		Short: "Generate code signing key and request certificate from Step-CA",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Auto-detect local software-cert.json file
			if cfg, err := LoadConfigFile("software-cert.json"); err == nil {
				if token == "" {
					token = cfg.Token
				}
				if publisherName == "" && cfg.PublisherName != "" {
					publisherName = cfg.PublisherName
				}
				if cfg.ApiUrl != "" {
					apiUrl = cfg.ApiUrl
				}
				fmt.Println("📁 Loaded configuration from software-cert.json")
			}

			if apiUrl == "" {
				apiUrl = target.Current().ApiUrl
			}
			if publisherName == "" {
				publisherName = "WorldPinball Software"
			}
			if token == "" {
				return fmt.Errorf("admin JWT token is required (use --token or software-cert.json)")
			}

			// 2. Generate ECDSA P-256 key pair
			machine := tpm.NewSimulatedMachine()
			defer machine.Close()

			if err := machine.GenerateKey(); err != nil {
				return fmt.Errorf("failed to generate key: %w", err)
			}

			if err := machine.SaveKeyToFile("software-signing.key"); err != nil {
				return fmt.Errorf("failed to save private key: %w", err)
			}

			pubKeyPem, err := machine.PublicKeyPem()
			if err != nil {
				return fmt.Errorf("failed to export public key: %w", err)
			}

			fmt.Printf("Requesting code signing certificate for '%s' with %s...\n", publisherName, apiUrl)

			// 3. Request certificate from backend CA
			resp, err := Issue(apiUrl, IssueRequest{
				PublisherName: publisherName,
				PublicKeyPem:  pubKeyPem,
			}, token)
			if err != nil {
				return fmt.Errorf("certificate issuance failed: %w", err)
			}

			// 4. Save certificate locally
			if err := os.WriteFile("software-signing.crt", []byte(resp.CertPem), 0600); err != nil {
				return fmt.Errorf("failed to save certificate: %w", err)
			}

			prompt.PrintSoftwareCertSuccess(
				publisherName,
				resp.SerialNumber,
				resp.ThumbprintSha256,
				resp.NotAfterUtc.Format(time.RFC3339),
				"software-signing.crt",
			)

			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Admin JWT Bearer token")
	cmd.Flags().StringVar(&publisherName, "publisher", "", "Publisher common name")
	cmd.Flags().StringVar(&apiUrl, "api-url", "", "Target backend API URL")

	return cmd
}
