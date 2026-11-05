package dotnet

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

// NewConfigCmd creates the 'config' command to inspect and modify target paths.
func NewConfigCmd() *cobra.Command {
	var envFilter string
	var setEnv string

	cmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"targets", "paths"},
		Short:   "View and manage project paths and target aliases",
		Run: func(cmd *cobra.Command, args []string) {
			cfgPath := FindConfigFile("")
			cfg, err := LoadConfig(cfgPath)
			if err != nil {
				fmt.Printf("Error loading config: %v\n", err)
				os.Exit(1)
			}

			fmt.Println(RenderConfigTable(cfg, cfgPath, envFilter))
		},
	}

	cmd.Flags().StringVarP(&envFilter, "env", "e", "", "Filter to a specific environment (e.g. dev, prod)")

	// Subcommand: set
	setCmd := &cobra.Command{
		Use:   "set <alias> <path>",
		Short: "Set or update a project path in wp-config.json",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			alias := strings.ToLower(strings.TrimSpace(args[0]))
			path := strings.TrimSpace(args[1])
			targetEnv := strings.ToLower(strings.TrimSpace(setEnv))
			if targetEnv == "" {
				targetEnv = "dev"
			}

			cfgPath := FindConfigFile("")
			cfg, err := LoadConfig(cfgPath)
			if err != nil {
				fmt.Printf("Error loading config: %v\n", err)
				os.Exit(1)
			}

			if cfg.Environments == nil {
				cfg.Environments = make(map[string]map[string]string)
			}
			if cfg.Environments[targetEnv] == nil {
				cfg.Environments[targetEnv] = make(map[string]string)
			}

			cfg.Environments[targetEnv][alias] = path
			if err := SaveConfig(cfgPath, cfg); err != nil {
				fmt.Printf("Error saving config: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("✔ Updated '%s' (%s) -> %s\nSaved in %s\n", alias, targetEnv, path, cfgPath)
		},
	}
	setCmd.Flags().StringVarP(&setEnv, "env", "e", "dev", "Target environment (dev, prod)")

	// Subcommand: path
	pathCmd := &cobra.Command{
		Use:   "path",
		Short: "Print the path of the active wp-config.json file",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(FindConfigFile(""))
		},
	}

	cmd.AddCommand(setCmd)
	cmd.AddCommand(pathCmd)

	return cmd
}

// RenderConfigTable renders a lipgloss table displaying all configured target paths.
func RenderConfigTable(cfg Config, configPath string, envFilter string) string {
	var sb strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	sb.WriteString(titleStyle.Render("\n=== Configured Project Paths ===") + "\n")
	sb.WriteString(fmt.Sprintf("Config file: %s %s\n\n", configPath, dim.Render(fmt.Sprintf("(Default Env: %s)", cfg.DefaultEnvironment))))

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Headers("ALIAS", "ENVIRONMENT", "PROJECT PATH", "ON DISK")

	var envNames []string
	for env := range cfg.Environments {
		if envFilter != "" && !strings.EqualFold(env, envFilter) {
			continue
		}
		envNames = append(envNames, env)
	}
	sort.Strings(envNames)

	for _, env := range envNames {
		targets := cfg.Environments[env]
		var aliases []string
		for a := range targets {
			aliases = append(aliases, a)
		}
		sort.Strings(aliases)

		for _, alias := range aliases {
			path := targets[alias]
			var existsStatus string
			if _, err := os.Stat(path); err == nil {
				existsStatus = green.Render("✔ Found")
			} else {
				existsStatus = red.Render("✖ Not Found")
			}
			t.Row(alias, env, path, existsStatus)
		}
	}

	sb.WriteString(t.Render())
	sb.WriteString("\n\n" + dim.Render("To update a path: wp-tools config set <alias> <path> [-e <env>]") + "\n")
	return sb.String()
}
