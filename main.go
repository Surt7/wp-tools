package main

import (
	"SetSecrets/internal/dotnet"
	"SetSecrets/internal/enroll"
	"SetSecrets/internal/envfile"
	"SetSecrets/internal/generator"
	"SetSecrets/internal/prompt"
	"SetSecrets/internal/session"
	"SetSecrets/internal/tpm"
	"fmt"
	"os"
	"strings"

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
				// Mode normal : fill only if  CHANGE_ME or empty
				// Mode force (-f / --force) : replace absolutely all keys
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
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {

			entries, err := envfile.LoadFile(".env")
			if err != nil {
				fmt.Printf("Error loading .env file: %v\n", err)
				os.Exit(1)
			}
			var targetKeys []string

			if len(args) == 0 {
				var availableKeys []string
				for _, entry := range entries {
					if entry.IsKey {
						availableKeys = append(availableKeys, entry.Key)
					}
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
				targetKeys = args
			}
			for _, key := range targetKeys {
				val, err := generator.GenerateRandomPassword(24)
				if err != nil {
					fmt.Printf("Error generating password for %s: %v\n", key, err)
					os.Exit(1)
				}
				entries = envfile.SetEntry(entries, key, val)
				fmt.Printf("✔ Generated secret for: %s\n", key)
			}

			if err := envfile.SaveFile(".env", entries); err != nil {
				fmt.Printf("Error saving .env file: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("\nSuccessfully updated .env file!")
		},
	}
	rootCmd.AddCommand(setCmd)

	var dryRun bool
	var projectPath string

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

			if err := dotnet.SyncSecrets(envMap, dryRun, projectPath); err != nil {
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

	var enrollMachineId string
	var enrollToken string
	var enrollSerial string
	var enrollApiUrl string
	var enrollOutFile string
	enrollCmd := &cobra.Command{
		Use:   "enroll",
		Short: "Generate TPM key and enroll device to get X.509 certificate",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			// 1. Auto-detect local enroll.json file (e.g. from USB key)
			if cfg, err := enroll.LoadConfigFile("enroll.json"); err == nil {
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

			if err := machine.SaveKeyToFile("device.Key"); err != nil {
				fmt.Printf("Error saving TPM key: %v\n", err)
				os.Exit(1)
			}

			pubKeyPem, err := machine.PublicKeyPem()
			if err != nil {
				fmt.Printf("Error getting public key PEM: %v\n", err)
				os.Exit(1)
			}
			req := enroll.Request{
				MachineId:            enrollMachineId,
				PlainEnrollmentToken: enrollToken,
				PublicKeyPem:         pubKeyPem,
				DeviceSerialNumber:   enrollSerial,
			}

			resp, err := enroll.Submit(enrollApiUrl, req)
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

			// Render elegant TUI Card with Lipgloss
			prompt.PrintEnrollSuccess(resp.DeviceId, resp.CertSerialNumber, enrollOutFile, enrollSerial)
		},
	}

	var startDeviceId string
	var startMode string
	var startApiUrl string
	var startEnv string
	var startKeyFile string
	var startCertFile string
	var startCaFile string

	resolveConfig := func(envName, customUrl, customCa, customDevice string) (string, string, string) {
		apiUrl := "https://localhost:7010"
		caFile := ""

		switch strings.ToLower(envName) {
		case "staging":
			apiUrl = "https://localhost:7010"
			caFile = "root_ca.crt"
		case "prod", "production":
			apiUrl = "https://api.worldpinball.com"
			caFile = ""
		default: // dev
			apiUrl = "https://localhost:7010"
			caFile = ""
		}

		if customUrl != "" {
			apiUrl = customUrl
		}
		if customCa != "" {
			caFile = customCa
		}

		deviceId := customDevice
		if deviceId == "" {
			if cfg, err := enroll.LoadConfigFile("enroll.json"); err == nil && cfg.MachineId != "" {
				deviceId = cfg.MachineId
				fmt.Printf("ℹ Auto-detected Device ID from enroll.json: %s\n", deviceId)
			}
		}

		return apiUrl, caFile, deviceId
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start an authenticated game session",
		Run: func(cmd *cobra.Command, args []string) {
			apiUrl, caFile, deviceId := resolveConfig(startEnv, startApiUrl, startCaFile, startDeviceId)
			if deviceId == "" {
				fmt.Println("Error: --device-id is required (or provide enroll.json)")
				os.Exit(1)
			}

			machine := tpm.NewSimulatedMachine()
			defer func() {
				_ = machine.Close()
			}()
			if err := machine.LoadKeyFromFile(startKeyFile); err != nil {
				fmt.Printf("Error loading private key from %s: %v\n", startKeyFile, err)
				os.Exit(1)
			}
			fmt.Printf("⏳ [%s] Signing canonical request and contacting server (%s)...\n", strings.ToUpper(startEnv), apiUrl)
			resp, err := session.StartSession(apiUrl, deviceId, machine, startMode, startCertFile, startKeyFile, caFile)
			if err != nil {
				fmt.Printf("Failed to start session: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✔ Session started successfully!\n")
			fmt.Printf("  Session ID  : %s\n", resp.SessionID)
			fmt.Printf("  Server Time : %d\n", resp.ServerTime)
		},
	}

	var submitDeviceId string
	var submitSessionId string
	var submitScore int64
	var submitSlot int
	var submitApiUrl string
	var submitEnv string
	var submitKeyFile string
	var submitCertFile string
	var submitCaFile string

	submitCmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit player score and end the game session",
		Run: func(cmd *cobra.Command, args []string) {
			apiUrl, caFile, deviceId := resolveConfig(submitEnv, submitApiUrl, submitCaFile, submitDeviceId)
			if deviceId == "" || submitSessionId == "" {
				fmt.Println("Error: --session-id is required (and --device-id or enroll.json)")
				os.Exit(1)
			}

			machine := tpm.NewSimulatedMachine()
			defer func() {
				_ = machine.Close()
			}()
			if err := machine.LoadKeyFromFile(submitKeyFile); err != nil {
				fmt.Printf("Error loading private key from %s: %v\n", submitKeyFile, err)
				os.Exit(1)
			}
			fmt.Printf("⏳ [%s] Signing score submission and contacting server (%s)...\n", strings.ToUpper(submitEnv), apiUrl)
			resp, err := session.SubmitScore(apiUrl, deviceId, machine, submitSessionId, submitScore, submitSlot, submitCertFile, submitKeyFile, caFile)
			if err != nil {
				fmt.Printf("Failed to submit score: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✔ Score submitted successfully!\n")
			fmt.Printf("  Score Entry ID : %s\n", resp.ScoreEntryID)
		},
	}

	// enroll a device
	enrollCmd.Flags().StringVar(&enrollMachineId, "machine-id", "", "Machine GUID provisioned by Admin")
	enrollCmd.Flags().StringVar(&enrollToken, "token", "", "One-time enrollment token")
	enrollCmd.Flags().StringVar(&enrollSerial, "serial", "", "Physical device serial number")
	enrollCmd.Flags().StringVar(&enrollApiUrl, "url", "http://localhost:5000", "Base URL of the ASP.NET Core API")
	enrollCmd.Flags().StringVarP(&enrollOutFile, "out", "o", "device.crt", "Output path for the X.509 certificate")

	//start game session
	startCmd.Flags().StringVar(&startEnv, "env", "dev", "Target environment profile: dev, staging, prod")
	startCmd.Flags().StringVar(&startApiUrl, "url", "", "Override API base URL")
	startCmd.Flags().StringVar(&startDeviceId, "device-id", "", "Device GUID registered on the server (falls back to enroll.json)")
	startCmd.Flags().StringVar(&startMode, "mode", "mode-1", "Game mode code")
	startCmd.Flags().StringVar(&startKeyFile, "key", "device.key", "Path to the private key PEM file")
	startCmd.Flags().StringVar(&startCertFile, "cert", "device.crt", "Path to the X.509 certificate file")
	startCmd.Flags().StringVar(&startCaFile, "ca", "", "Path to the root CA certificate file (override)")

	// submit score
	submitCmd.Flags().StringVar(&submitEnv, "env", "dev", "Target environment profile: dev, staging, prod")
	submitCmd.Flags().StringVar(&submitApiUrl, "url", "", "Override API base URL")
	submitCmd.Flags().StringVar(&submitDeviceId, "device-id", "", "Device GUID registered on the server (falls back to enroll.json)")
	submitCmd.Flags().StringVar(&submitSessionId, "session-id", "", "Game Session GUID")
	submitCmd.Flags().Int64Var(&submitScore, "score", 1250000, "Final score to submit")
	submitCmd.Flags().IntVar(&submitSlot, "slot", 1, "Player slot number (1-4)")
	submitCmd.Flags().StringVar(&submitKeyFile, "key", "device.key", "Path to the private key PEM file")
	submitCmd.Flags().StringVar(&submitCertFile, "cert", "device.crt", "Path to the X.509 certificate file")
	submitCmd.Flags().StringVar(&submitCaFile, "ca", "", "Path to the root CA certificate file (override)")

	// miscelinous
	dotnetCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Simulate without executing dotnet commands")
	dotnetCmd.Flags().StringVarP(&projectPath, "project", "p", "", "Path to the .csproj or project directory")

	//add command
	rootCmd.AddCommand(enrollCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(submitCmd)
	rootCmd.AddCommand(dotnetCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
