package envfile

import (
	"fmt"
	"os"

	"SetSecrets/internal/generator"
	"SetSecrets/internal/prompt"

	"github.com/spf13/cobra"
)

// NewInitCmd creates the 'init' command to generate .env from .env.example.
func NewInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize secrets in .env file",
		Run: func(cmd *cobra.Command, args []string) {
			entries, err := LoadFile(".env.example")
			if err != nil {
				fmt.Printf("Error loading .env.example: %v\n", err)
				os.Exit(1)
			}

			for i := range entries {
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

			if err := SaveFile(".env", entries); err != nil {
				fmt.Printf("Error saving .env file: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("\n Successfully generated .env file!")
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force regeneration of all variables, even if already set")
	return cmd
}

// NewSetCmd creates the 'set' command to generate/update secrets for specific keys in .env.
func NewSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <KEY> [VALUE]",
		Short: "Set or generate a secret for a specific key in .env",
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			entries, err := LoadFile(".env")
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
				entries = SetEntry(entries, key, val)
				fmt.Printf("✔ Generated secret for: %s\n", key)
			}

			if err := SaveFile(".env", entries); err != nil {
				fmt.Printf("Error saving .env file: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("\nSuccessfully updated .env file!")
		},
	}
}
