package dotnet

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

type ProjectStatus struct {
	Target        string
	Path          string
	Exists        bool
	UserSecretsId string
	Secrets       map[string]string
	Error         error
}

// GetUserSecretsId extracts the UserSecretsId GUID from the project .csproj file.
func GetUserSecretsId(projectPath string) (string, error) {
	targetFile := projectPath
	if targetFile == "" {
		targetFile = "."
	}
	fi, err := os.Stat(targetFile)
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		files, err := filepath.Glob(filepath.Join(targetFile, "*.csproj"))
		if err != nil || len(files) == 0 {
			return "", fmt.Errorf("no .csproj found in directory %s", targetFile)
		}
		targetFile = files[0]
	}
	data, err := os.ReadFile(targetFile)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`(?i)<UserSecretsId>\s*([a-f0-9\-]+)\s*</UserSecretsId>`)
	match := re.FindStringSubmatch(string(data))
	if len(match) > 1 {
		return match[1], nil
	}
	return "", nil
}

// ListSecrets executes 'dotnet user-secrets list' and returns the key-value map.
func (m *Manager) ListSecrets(projectPath string) (map[string]string, error) {
	args := []string{"user-secrets", "list"}
	if projectPath != "" {
		args = append(args, "--project", projectPath)
	}

	output, err := m.Runner.Run("dotnet", args...)
	if err != nil {
		return nil, fmt.Errorf("dotnet user-secrets list failed: %s (%w)", string(output), err)
	}

	secrets := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " = ", 2)
		if len(parts) == 2 {
			secrets[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		} else {
			parts = strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				secrets[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
	return secrets, nil
}

func ListSecrets(projectPath string) (map[string]string, error) {
	return defaultManager.ListSecrets(projectPath)
}

// InspectProject checks a specific project file on disk and its user-secrets state.
func InspectProject(targetName string, projectPath string) ProjectStatus {
	st := ProjectStatus{
		Target: targetName,
		Path:   projectPath,
	}

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		st.Exists = false
		return st
	}
	st.Exists = true

	guid, err := GetUserSecretsId(projectPath)
	if err != nil {
		st.Error = err
		return st
	}
	st.UserSecretsId = guid

	if guid != "" {
		sec, err := ListSecrets(projectPath)
		if err != nil {
			st.Error = err
		} else {
			st.Secrets = sec
		}
	}

	return st
}

// RenderStatusTable produces a styled lipgloss summary table for all targets.
func RenderStatusTable(statuses []ProjectStatus) string {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Headers("TARGET", "USERSECRETS ID", "STATUS", "SECRETS CONFIGURED")

	green := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	for _, s := range statuses {
		var statusText string
		var idText string
		var secretsSummary string

		if !s.Exists {
			statusText = red.Render("✖ Missing .csproj")
			idText = dim.Render("—")
			secretsSummary = dim.Render("file not found on disk")
		} else if s.UserSecretsId == "" {
			statusText = yellow.Render("⚠ Uninitialized")
			idText = dim.Render("not configured")
			secretsSummary = dim.Render("run: dotnet user-secrets init")
		} else {
			statusText = green.Render("✔ Active")
			idText = s.UserSecretsId
			if len(s.Secrets) == 0 {
				secretsSummary = dim.Render("0 secrets (empty)")
			} else {
				var keys []string
				for k := range s.Secrets {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				if len(keys) > 3 {
					secretsSummary = fmt.Sprintf("%s, +%d more", strings.Join(keys[:3], ", "), len(keys)-3)
				} else {
					secretsSummary = strings.Join(keys, ", ")
				}
			}
		}

		t.Row(s.Target, idText, statusText, secretsSummary)
	}

	return t.Render()
}

// RenderProjectDetail produces a styled detailed view for a specific project.
func RenderProjectDetail(s ProjectStatus, reveal bool) string {
	var sb strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	sb.WriteString(titleStyle.Render(fmt.Sprintf("\n=== Target: %s ===", s.Target)) + "\n")
	sb.WriteString(fmt.Sprintf("Project: %s\n", s.Path))

	if !s.Exists {
		sb.WriteString("Status: ✖ .csproj file does not exist on disk\n")
		return sb.String()
	}

	if s.UserSecretsId == "" {
		sb.WriteString("UserSecretsId: ⚠ Not initialized\n")
		sb.WriteString("Tip: Run `wp-tools dotnet -t " + s.Target + "` to initialize and sync secrets.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("UserSecretsId: %s\n\n", s.UserSecretsId))

	if len(s.Secrets) == 0 {
		sb.WriteString("No secrets configured yet in UserSecrets.\n")
		return sb.String()
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Headers("KEY", "VALUE")

	var keys []string
	for k := range s.Secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		val := s.Secrets[k]
		displayVal := val
		if !reveal && len(val) > 4 {
			displayVal = val[:2] + strings.Repeat("*", 6) + val[len(val)-2:]
		}
		t.Row(k, displayVal)
	}

	sb.WriteString(t.Render())
	return sb.String()
}

// NewStatusCmd creates the 'status' command to inspect UserSecrets state.
func NewStatusCmd() *cobra.Command {
	var (
		target  string
		envName string
		reveal  bool
	)

	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"list", "ls"},
		Short:   "Supervise and inspect .NET User Secrets across projects",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := LoadConfig("")
			if err != nil {
				fmt.Printf("Error loading config: %v\n", err)
				os.Exit(1)
			}

			activeEnv := strings.ToLower(strings.TrimSpace(envName))
			if activeEnv == "" {
				activeEnv = cfg.DefaultEnvironment
				if activeEnv == "" {
					activeEnv = "dev"
				}
			}

			targetsMap, ok := cfg.Environments[activeEnv]
			if !ok {
				fmt.Printf("Error: unknown environment '%s'\n", activeEnv)
				os.Exit(1)
			}

			if target != "" {
				targetLower := strings.ToLower(strings.TrimSpace(target))
				path, exists := targetsMap[targetLower]
				if !exists {
					fmt.Printf("Error: target '%s' not found in environment '%s'\n", target, activeEnv)
					os.Exit(1)
				}
				status := InspectProject(targetLower, path)
				fmt.Println(RenderProjectDetail(status, reveal))
				return
			}

			var statuses []ProjectStatus
			for _, tName := range getMapKeys(targetsMap) {
				statuses = append(statuses, InspectProject(tName, targetsMap[tName]))
			}

			headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
			fmt.Println(headerStyle.Render(fmt.Sprintf("\n=== .NET User Secrets Supervision (Environment: %s) ===", activeEnv)))
			fmt.Println(RenderStatusTable(statuses))
			fmt.Println("\nTip: Run with -t <target> (e.g. -t hub) to view secrets, or --reveal to unmask values.")
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "", "Filter to a specific target project")
	cmd.Flags().StringVarP(&envName, "env", "e", "dev", "Environment scenario (dev, prod)")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal secret values instead of masking them")

	return cmd
}
