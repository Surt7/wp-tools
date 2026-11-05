package audit

import (
	"fmt"
	"os"

	"SetSecrets/internal/dotnet"
	"SetSecrets/internal/prompt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

// NewAuditCmd creates the 'audit' command to verify git repositories against leaks.
func NewAuditCmd() *cobra.Command {
	var autoFix bool

	cmd := &cobra.Command{
		Use:     "audit",
		Aliases: []string{"check-env", "check"},
		Short:   "Audit repositories to ensure .env and secrets are properly ignored",
		Run: func(cmd *cobra.Command, args []string) {
			cfgPath := dotnet.FindConfigFile("")
			cfg, err := dotnet.LoadConfig(cfgPath)
			if err != nil {
				fmt.Printf("Error loading config: %v\n", err)
				os.Exit(1)
			}

			repoRoots := DiscoverRepositories(cfg)
			if len(repoRoots) == 0 {
				fmt.Println("No git repositories discovered to audit.")
				return
			}

			var results []RepoAuditResult
			hasDanger := false
			hasWarning := false

			for _, root := range repoRoots {
				res := AuditRepo(root)

				// Auto-fix or prompt to fix if missing from .gitignore
				if !res.EnvIgnored && !res.EnvTracked {
					shouldFix := autoFix
					if !shouldFix {
						promptMsg := fmt.Sprintf(".env is NOT ignored in '%s'. Add it to .gitignore?", res.Name)
						confirmed, err := prompt.Confirm(promptMsg, "Prevents committing local secrets into Git", true)
						if err == nil && confirmed {
							shouldFix = true
						}
					}
					if shouldFix {
						if err := AddToGitIgnore(root, ".env\n.env.*"); err == nil {
							fmt.Printf("✔ Added .env to .gitignore in '%s'\n", res.Name)
							res.EnvIgnored = true
						}
					}
				}

				if !res.ConfigIgnored && !res.ConfigTracked && res.Name != "wp-tools" {
					shouldFixConfig := autoFix
					if !shouldFixConfig {
						promptMsg := fmt.Sprintf("wp-config.json is NOT ignored in '%s'. Add it to .gitignore?", res.Name)
						confirmed, err := prompt.Confirm(promptMsg, "Prevents committing local machine project paths into Git", true)
						if err == nil && confirmed {
							shouldFixConfig = true
						}
					}
					if shouldFixConfig {
						if err := AddToGitIgnore(root, "wp-config.json"); err == nil {
							fmt.Printf("✔ Added wp-config.json to .gitignore in '%s'\n", res.Name)
							res.ConfigIgnored = true
						}
					}
				}

				if res.EnvTracked {
					res.Status = "DANGER"
				} else if !res.EnvIgnored || (!res.ConfigIgnored && res.Name != "wp-tools") {
					res.Status = "WARNING"
				} else {
					res.Status = "SAFE"
				}

				if res.Status == "DANGER" {
					hasDanger = true
				} else if res.Status == "WARNING" {
					hasWarning = true
				}

				results = append(results, res)
			}

			headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
			fmt.Println(headerStyle.Render("\n=== Git & Secrets Security Audit ==="))
			fmt.Println(RenderAuditTable(results))

			green := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
			red := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
			yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

			if hasDanger {
				fmt.Println("\n" + red.Render("🚨 CRITICAL: Secret files are currently TRACKED in Git!"))
				fmt.Println("To stop tracking without deleting your local file, run:")
				fmt.Println("  git rm --cached <file>")
			} else if hasWarning {
				fmt.Println("\n" + yellow.Render("ℹ Some repositories have warnings. Run `wp-tools audit --fix` to update .gitignore."))
			} else {
				fmt.Println("\n" + green.Render("✔ Zero secret files tracked in Git. All repositories properly protected!"))
			}
		},
	}

	cmd.Flags().BoolVar(&autoFix, "fix", false, "Automatically add missing ignore rules to .gitignore")

	return cmd
}

// RenderAuditTable produces a styled lipgloss summary table for the security audit.
func RenderAuditTable(results []RepoAuditResult) string {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Headers("REPOSITORY", ".ENV IGNORED", ".ENV TRACKED", "STATUS")

	green := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

	for _, r := range results {
		var ignoredText string
		if r.EnvIgnored {
			ignoredText = green.Render("✔ Ignored")
		} else {
			ignoredText = red.Render("✖ Not Ignored")
		}

		var trackedText string
		if r.EnvTracked {
			trackedText = red.Render("🚨 TRACKED (LEAK)")
		} else {
			trackedText = green.Render("✔ Untracked")
		}

		var statusText string
		switch r.Status {
		case "SAFE":
			statusText = green.Render("✔ SAFE")
		case "WARNING":
			statusText = yellow.Render("⚠ WARNING")
		case "DANGER":
			statusText = red.Render("🚨 DANGER")
		default:
			statusText = r.Status
		}

		t.Row(r.Name, ignoredText, trackedText, statusText)
	}

	return t.Render()
}
