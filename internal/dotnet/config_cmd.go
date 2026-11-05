package dotnet

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"SetSecrets/internal/prompt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

// NewConfigCmd creates the 'config' command to inspect and modify target paths.
func NewConfigCmd() *cobra.Command {
	var envFilter string
	var targetEnv string

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

	// Subcommand: add (and set)
	addCmd := &cobra.Command{
		Use:     "add [alias] [path]",
		Aliases: []string{"set"},
		Short:   "Add or update a project path in wp-config.json",
		Args:    cobra.MaximumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			cfgPath := FindConfigFile("")
			cfg, err := LoadConfig(cfgPath)
			if err != nil {
				fmt.Printf("Error loading config: %v\n", err)
				os.Exit(1)
			}

			var alias, path, env string

			if len(args) == 0 {
				// Interactive Huh TUI mode
				var availableEnvs []string
				for e := range cfg.Environments {
					availableEnvs = append(availableEnvs, e)
				}
				sort.Strings(availableEnvs)
				if len(availableEnvs) == 0 {
					availableEnvs = []string{"dev", "prod"}
				}

				defEnv := targetEnv
				if defEnv == "" {
					defEnv = cfg.DefaultEnvironment
				}

				pAlias, pPath, pEnv, err := prompt.PromptAddTarget(availableEnvs, defEnv)
				if err != nil {
					fmt.Printf("Operation cancelled: %v\n", err)
					os.Exit(0)
				}
				alias = pAlias
				path = pPath
				env = pEnv
			} else if len(args) == 2 {
				alias = strings.ToLower(strings.TrimSpace(args[0]))
				path = strings.TrimSpace(args[1])
				env = strings.ToLower(strings.TrimSpace(targetEnv))
				if env == "" {
					env = cfg.DefaultEnvironment
					if env == "" {
						env = "dev"
					}
				}
			} else {
				fmt.Println("Error: please specify both <alias> and <path>, or run without arguments for interactive form.")
				os.Exit(1)
			}

			if cfg.Environments == nil {
				cfg.Environments = make(map[string]map[string]string)
			}
			if cfg.Environments[env] == nil {
				cfg.Environments[env] = make(map[string]string)
			}

			cfg.Environments[env][alias] = path
			if err := SaveConfig(cfgPath, cfg); err != nil {
				fmt.Printf("Error saving config: %v\n", err)
				os.Exit(1)
			}

			// Feedback on disk validation
			if _, err := os.Stat(path); err == nil {
				hasSecrets, _ := HasUserSecretsId(path)
				if hasSecrets {
					fmt.Printf("✔ Validated project: exists on disk (UserSecrets configured)\n")
				} else {
					fmt.Printf("ℹ Validated project: exists on disk (UserSecrets not yet initialized)\n")
				}
			} else {
				fmt.Printf("⚠️ Warning: project file not found on disk at '%s'\n", path)
			}

			fmt.Printf("✔ Saved '%s' (%s) -> %s\nSaved in %s\n", alias, env, path, cfgPath)
		},
	}
	addCmd.Flags().StringVarP(&targetEnv, "env", "e", "dev", "Target environment (dev, prod)")

	// Subcommand: remove
	removeCmd := &cobra.Command{
		Use:     "remove [alias]",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a project target from wp-config.json",
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfgPath := FindConfigFile("")
			cfg, err := LoadConfig(cfgPath)
			if err != nil {
				fmt.Printf("Error loading config: %v\n", err)
				os.Exit(1)
			}

			env := strings.ToLower(strings.TrimSpace(targetEnv))
			if env == "" {
				env = cfg.DefaultEnvironment
				if env == "" {
					env = "dev"
				}
			}

			targets, ok := cfg.Environments[env]
			if !ok || len(targets) == 0 {
				fmt.Printf("No targets found in environment '%s'\n", env)
				os.Exit(0)
			}

			var alias string
			if len(args) == 0 {
				var targetNames []string
				for t := range targets {
					targetNames = append(targetNames, t)
				}
				sort.Strings(targetNames)

				selected, err := prompt.SelectTargetToRemove(targetNames)
				if err != nil {
					fmt.Printf("Operation cancelled: %v\n", err)
					os.Exit(0)
				}
				alias = selected
			} else {
				alias = strings.ToLower(strings.TrimSpace(args[0]))
			}

			if _, exists := targets[alias]; !exists {
				fmt.Printf("Target '%s' not found in environment '%s'\n", alias, env)
				os.Exit(1)
			}

			delete(cfg.Environments[env], alias)
			if err := SaveConfig(cfgPath, cfg); err != nil {
				fmt.Printf("Error saving config: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("✔ Removed target '%s' (%s) from %s\n", alias, env, cfgPath)
		},
	}
	removeCmd.Flags().StringVarP(&targetEnv, "env", "e", "dev", "Target environment (dev, prod)")

	// Subcommand: path
	pathCmd := &cobra.Command{
		Use:   "path",
		Short: "Print the path of the active wp-config.json file",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(FindConfigFile(""))
		},
	}

	cmd.AddCommand(addCmd)
	cmd.AddCommand(removeCmd)
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
	sb.WriteString("\n\n" + dim.Render("Commands: wp-tools config add [alias] [path] | wp-tools config remove [alias]") + "\n")
	return sb.String()
}
