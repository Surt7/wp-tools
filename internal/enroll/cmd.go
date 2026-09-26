package enroll

import (
	"fmt"
	"os"

	"SetSecrets/internal/prompt"
	"SetSecrets/internal/tpm"

	"github.com/spf13/cobra"
)

// NewEnrollCmd creates the 'enroll' command to provision and enroll a physical machine device.
func NewEnrollCmd() *cobra.Command {
	var (
		enrollMachineId string
		enrollToken     string
		enrollSerial    string
		enrollApiUrl    string
		enrollOutFile   string
	)

	cmd := &cobra.Command{
		Use:   "enroll",
		Short: "Generate TPM key and enroll device to get X.509 certificate",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			// 1. Auto-detect local enroll.json file (e.g. from USB key)
			if cfg, err := LoadConfigFile("enroll.json"); err == nil {
				if enrollMachineId == "" {
					enrollMachineId = cfg.MachineId
				}
				if enrollToken == "" {
					enrollToken = cfg.Token
				}
				if enrollSerial == "" {
					enrollSerial = cfg.DeviceSerialNumber
				}
				if cfg.ApiUrl != "" && enrollApiUrl == "http://localhost:5000" {
					enrollApiUrl = cfg.ApiUrl
				}
				fmt.Println("📁 Loaded configuration from enroll.json")
			}

			// 2. If flags are still missing, launch QR scan / interactive form
			if enrollMachineId == "" || enrollToken == "" || enrollSerial == "" {
				if err := prompt.EnrollForm(&enrollMachineId, &enrollToken, &enrollSerial); err != nil {
					fmt.Printf("Prompt error: %v\n", err)
					os.Exit(1)
				}
			}

			machine := tpm.NewSimulatedMachine()
			defer func() {
				_ = machine.Close()
			}()
			if err := machine.GenerateKey(); err != nil {
				fmt.Printf("Error generating TPM key: %v\n", err)
				os.Exit(1)
			}

			if err := machine.SaveKeyToFile("device.key"); err != nil {
				fmt.Printf("Error saving TPM key: %v\n", err)
				os.Exit(1)
			}

			pubKeyPem, err := machine.PublicKeyPem()
			if err != nil {
				fmt.Printf("Error getting public key PEM: %v\n", err)
				os.Exit(1)
			}
			req := Request{
				MachineId:            enrollMachineId,
				PlainEnrollmentToken: enrollToken,
				PublicKeyPem:         pubKeyPem,
				DeviceSerialNumber:   enrollSerial,
			}

			resp, err := Submit(enrollApiUrl, req)
			if err != nil {
				fmt.Printf("Enrollment failed: %v\n", err)
				os.Exit(1)
			}

			if err := os.WriteFile(enrollOutFile, []byte(resp.CertPem), 0644); err != nil {
				fmt.Printf("Error saving certificate: %v\n", err)
				os.Exit(1)
			}

			if err := machine.SaveKeyToFile("device.key"); err != nil {
				fmt.Printf("Error saving private key: %v\n", err)
				os.Exit(1)
			}

			if cfg, err := LoadConfigFile("enroll.json"); err == nil {
				cfg.DeviceId = resp.DeviceId
				_ = SaveConfigFile("enroll.json", cfg)
			}

			// Render elegant TUI Card with Lipgloss
			prompt.PrintEnrollSuccess(resp.DeviceId, resp.CertSerialNumber, enrollOutFile, enrollSerial)
		},
	}

	cmd.Flags().StringVar(&enrollMachineId, "machine-id", "", "Machine GUID provisioned by Admin")
	cmd.Flags().StringVar(&enrollToken, "token", "", "One-time enrollment token")
	cmd.Flags().StringVar(&enrollSerial, "serial", "", "Physical device serial number")
	cmd.Flags().StringVar(&enrollApiUrl, "url", "http://localhost:5000", "Base URL of the ASP.NET Core API")
	cmd.Flags().StringVarP(&enrollOutFile, "out", "o", "device.crt", "Output path for the X.509 certificate")

	return cmd
}
