package menu

import (
	"fmt"
	"os"
	"strings"

	"SetSecrets/internal/audit"
	"SetSecrets/internal/config"
	"SetSecrets/internal/dotnet"
	"SetSecrets/internal/enroll"
	"SetSecrets/internal/envfile"
	"SetSecrets/internal/prompt"
	"SetSecrets/internal/session"
	"SetSecrets/internal/software"
	"SetSecrets/internal/workstation"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	bannerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 2).
			MarginBottom(1)

	breadcrumbStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#04B575"))

	targetStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C"))
)

type SessionContext struct {
	Target string
	Env    string
}

func PrintContextBanner(breadcrumb string, ctx SessionContext) {
	targetInfo := ctx.Target
	if targetInfo == "" {
		targetInfo = "local (current directory)"
	}
	envInfo := ctx.Env
	if envInfo == "" {
		envInfo = "dev"
	}

	header := fmt.Sprintf("%s  %s",
		breadcrumbStyle.Render("📍 Navigation :"),
		breadcrumb,
	)
	targetLine := fmt.Sprintf("%s  %s (%s)",
		targetStyle.Render("🎯 Projet Cible:"),
		targetInfo,
		envInfo,
	)

	content := fmt.Sprintf("%s\n%s", header, targetLine)
	fmt.Println(bannerBoxStyle.Render(content))
}

func Run(defaultTarget string, defaultEnv string) {
	ctx := SessionContext{
		Target: defaultTarget,
		Env:    defaultEnv,
	}

	cfg, _ := config.LoadConfig("")
	if ctx.Env == "" {
		ctx.Env = cfg.DefaultEnvironment
		if ctx.Env == "" {
			ctx.Env = "dev"
		}
	}

	for {
		PrintContextBanner("Menu Principal", ctx)

		var choice string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Que souhaites-tu faire ?").
					Options(
						huh.NewOption("🔐 Secrets & Environnement (.env, User-Secrets, Audit)", "secrets"),
						huh.NewOption("🛠️ Machines & Certificats (Enrollment, Workstation, Signature)", "machines"),
						huh.NewOption("🎮 Game Sessions & Lobby (Créer, Soumettre, Lobby)", "gaming"),
						huh.NewOption(fmt.Sprintf("🎯 Changer de Projet Cible (Actuel: %s)", displayTarget(ctx.Target)), "switch_target"),
						huh.NewOption("🚪 Quitter", "exit"),
					).
					Value(&choice),
			),
		)

		if err := form.Run(); err != nil || choice == "exit" {
			fmt.Println("Au revoir !")
			return
		}

		switch choice {
		case "secrets":
			runSecretsMenu(&ctx)
		case "machines":
			runMachinesMenu(&ctx)
		case "gaming":
			runGamingMenu(&ctx)
		case "switch_target":
			switchTargetInteractive(&ctx, cfg)
		}
	}
}

func displayTarget(t string) string {
	if t == "" {
		return "local (dossier courant)"
	}
	return t
}

func switchTargetInteractive(ctx *SessionContext, cfg config.Config) {
	targetsMap := cfg.Environments[ctx.Env]
	options := append([]string{"local (current directory)"}, config.GetMapKeys(targetsMap)...)

	selected, err := prompt.SelectTarget(options)
	if err == nil {
		if strings.HasPrefix(selected, "local") {
			ctx.Target = "local"
		} else {
			ctx.Target = selected
		}
		fmt.Printf("✔ Projet cible défini sur: %s\n\n", ctx.Target)
	}
}

func runSecretsMenu(ctx *SessionContext) {
	for {
		PrintContextBanner("Menu Principal > 🔐 Secrets & Environnement", *ctx)

		var action string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Gestion des Secrets").
					Options(
						huh.NewOption("1. Initialiser / Régénérer les secrets (.env)", "init"),
						huh.NewOption("2. Modifier / Générer une clé spécifique (set)", "set"),
						huh.NewOption("3. Synchroniser vers .NET User-Secrets (dotnet)", "dotnet"),
						huh.NewOption("4. Auditer la sécurité des secrets (audit)", "audit"),
						huh.NewOption("5. Voir le statut des secrets .NET (dotnet status)", "status"),
						huh.NewOption("← Retour au menu principal", "back"),
					).
					Value(&action),
			),
		)

		if err := form.Run(); err != nil || action == "back" {
			return
		}

		switch action {
		case "init":
			cmd := envfile.NewInitCmd()
			cmd.SetArgs([]string{"-t", ctx.Target, "-e", ctx.Env, "-i"})
			_ = cmd.Execute()
		case "set":
			cmd := envfile.NewSetCmd()
			cmd.SetArgs([]string{"-t", ctx.Target, "-e", ctx.Env, "-i"})
			_ = cmd.Execute()
		case "dotnet":
			cmd := dotnet.NewDotnetCmd()
			cmd.SetArgs([]string{"-t", ctx.Target, "-e", ctx.Env, "-i"})
			_ = cmd.Execute()
		case "audit":
			cmd := audit.NewAuditCmd()
			cmd.SetArgs([]string{})
			_ = cmd.Execute()
		case "status":
			cmd := dotnet.NewStatusCmd()
			cmd.SetArgs([]string{"-t", ctx.Target, "-e", ctx.Env})
			_ = cmd.Execute()
		}

		waitEnter()
	}
}

func runMachinesMenu(ctx *SessionContext) {
	for {
		PrintContextBanner("Menu Principal > 🛠️ Machines & Certificats", *ctx)

		var action string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Machines & Certificats").
					Options(
						huh.NewOption("1. Enrôler une machine / automate (enroll)", "enroll"),
						huh.NewOption("2. Enrôler un poste de travail (workstation)", "workstation"),
						huh.NewOption("3. Signer un binaire logiciel (software)", "software"),
						huh.NewOption("← Retour au menu principal", "back"),
					).
					Value(&action),
			),
		)

		if err := form.Run(); err != nil || action == "back" {
			return
		}

		switch action {
		case "enroll":
			cmd := enroll.NewEnrollCmd()
			cmd.SetArgs([]string{})
			_ = cmd.Execute()
		case "workstation":
			cmd := workstation.NewWorkstationCmd()
			cmd.SetArgs([]string{})
			_ = cmd.Execute()
		case "software":
			cmd := software.NewSoftwareCmd()
			cmd.SetArgs([]string{})
			_ = cmd.Execute()
		}

		waitEnter()
	}
}

func runGamingMenu(ctx *SessionContext) {
	for {
		PrintContextBanner("Menu Principal > 🎮 Game Sessions & Lobby", *ctx)

		var action string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Sessions & Lobby").
					Options(
						huh.NewOption("1. Créer une session de jeu (session create)", "create"),
						huh.NewOption("2. Soumettre un score de partie (session submit)", "submit"),
						huh.NewOption("3. Consulter le lobby multijoueur (session lobby)", "lobby"),
						huh.NewOption("← Retour au menu principal", "back"),
					).
					Value(&action),
			),
		)

		if err := form.Run(); err != nil || action == "back" {
			return
		}

		switch action {
		case "create":
			cmd := session.NewCreateCmd()
			cmd.SetArgs([]string{})
			_ = cmd.Execute()
		case "submit":
			cmd := session.NewSubmitCmd()
			cmd.SetArgs([]string{})
			_ = cmd.Execute()
		case "lobby":
			cmd := session.NewLobbyCmd()
			cmd.SetArgs([]string{})
			_ = cmd.Execute()
		}

		waitEnter()
	}
}

func waitEnter() {
	fmt.Print("\nAppuyez sur [Entrée] pour continuer...")
	var b [1]byte
	_, _ = os.Stdin.Read(b[:])
	fmt.Println()
}
