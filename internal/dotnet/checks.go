package dotnet

import (
	"fmt"
	"os"
	"path/filepath"

	"SetSecrets/internal/envfile"
	"SetSecrets/internal/prompt"
)

// EnsureEnvFile ensures that the .env file exists.
// If it is missing but .env.example exists, it prompts the user to create it or creates it automatically if autoInit is true.
func EnsureEnvFile(envPath string, examplePath string, autoInit bool, dryRun bool) ([]envfile.Entry, error) {
	if envPath == "" {
		envPath = ".env"
	}
	if examplePath == "" {
		examplePath = ".env.example"
	}

	if _, err := os.Stat(envPath); err == nil {
		// .env already exists
		return envfile.LoadFile(envPath)
	}

	// .env does not exist. Check if .env.example exists.
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("neither '%s' nor '%s' exists", envPath, examplePath)
	}

	if dryRun {
		fmt.Printf("[DRY-RUN] '%s' not found. Would prompt to generate from '%s'.\n", envPath, examplePath)
		return envfile.LoadFile(examplePath)
	}

	shouldInit := autoInit
	if !shouldInit {
		title := fmt.Sprintf("'%s' file was not found. Create it from '%s'?", envPath, examplePath)
		desc := "Generates secure random secrets and creates a local environment file."
		confirmed, promptErr := prompt.Confirm(title, desc, true)
		if promptErr != nil {
			return nil, fmt.Errorf("aborted by user: %w", promptErr)
		}
		shouldInit = confirmed
	}

	if !shouldInit {
		return nil, fmt.Errorf("'%s' is required to sync secrets", envPath)
	}

	fmt.Printf("⏳ Generating '%s' from '%s'...\n", envPath, examplePath)
	if err := envfile.InitFromExample(examplePath, envPath, false); err != nil {
		return nil, fmt.Errorf("failed creating '%s': %w", envPath, err)
	}
	fmt.Printf("✔ Successfully created '%s'\n", envPath)

	return envfile.LoadFile(envPath)
}

// EnsureUserSecretsInitialized ensures that the target .NET project has a <UserSecretsId> configured.
// If missing, it prompts the user to run 'dotnet user-secrets init' or runs it automatically if autoInit is true.
func EnsureUserSecretsInitialized(projectPath string, targetName string, autoInit bool, dryRun bool) error {
	if projectPath == "" {
		return nil
	}

	hasSecrets, err := HasUserSecretsId(projectPath)
	if err != nil {
		// Could not inspect .csproj (maybe directory or special file) - do not block
		return nil
	}

	if hasSecrets {
		return nil
	}

	displayName := targetName
	if displayName == "" {
		displayName = filepath.Base(projectPath)
	}

	if dryRun {
		fmt.Printf("[DRY-RUN] UserSecretsId is not configured in '%s'. Would initialize.\n", displayName)
		return nil
	}

	shouldInit := autoInit
	if !shouldInit {
		title := fmt.Sprintf("UserSecretsId is not configured in '%s'. Initialize it now?", displayName)
		desc := "Runs 'dotnet user-secrets init' to add <UserSecretsId> to the project."
		confirmed, promptErr := prompt.Confirm(title, desc, true)
		if promptErr != nil {
			return fmt.Errorf("user-secrets initialization cancelled: %w", promptErr)
		}
		shouldInit = confirmed
	}

	if !shouldInit {
		return fmt.Errorf("UserSecrets is not initialized for '%s'. Aborting sync", displayName)
	}

	fmt.Printf("⏳ Initializing UserSecrets for %s...\n", displayName)
	if err := InitUserSecrets(projectPath); err != nil {
		return fmt.Errorf("failed initializing UserSecrets: %w", err)
	}
	fmt.Printf("✔ UserSecrets initialized for %s\n", displayName)
	return nil
}
