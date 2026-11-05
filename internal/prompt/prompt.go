package prompt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// SelectKeys displays an interactive multi-select menu to choose keys.
func SelectKeys(keys []string) ([]string, error) {
	var selected []string

	// Convert string slice into huh options
	var options []huh.Option[string]
	for _, k := range keys {
		options = append(options, huh.NewOption(k, k))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select keys to regenerate").
				Options(options...).
				Value(&selected),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, err
	}

	return selected, nil
}

// SelectTarget displays an interactive select menu to choose a project target.
func SelectTarget(targets []string) (string, error) {
	if len(targets) == 0 {
		return "", fmt.Errorf("no targets available")
	}

	var selected string
	var options []huh.Option[string]
	for _, t := range targets {
		options = append(options, huh.NewOption(t, t))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select target project").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return "", err
	}

	return selected, nil
}

// SelectSecretsToSync displays an interactive multi-select menu to choose secrets to sync.
func SelectSecretsToSync(availableKeys []string, defaultSelected map[string]bool) ([]string, error) {
	if len(availableKeys) == 0 {
		return nil, fmt.Errorf("no secrets available to sync")
	}

	var selected []string
	var options []huh.Option[string]
	for _, k := range availableKeys {
		opt := huh.NewOption(k, k)
		if defaultSelected != nil && defaultSelected[k] {
			opt = opt.Selected(true)
		}
		options = append(options, opt)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select secrets to sync to project").
				Description("Press Space to toggle, Enter to confirm").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return selected, nil
}

// Confirm displays an interactive Yes/No confirmation prompt with Huh.
func Confirm(title string, description string, defaultYes bool) (bool, error) {
	confirmed := defaultYes
	confirmField := huh.NewConfirm().
		Title(title).
		Affirmative("Yes").
		Negative("No").
		Value(&confirmed)

	if description != "" {
		confirmField.Description(description)
	}

	form := huh.NewForm(
		huh.NewGroup(confirmField),
	)

	if err := form.Run(); err != nil {
		return false, err
	}

	return confirmed, nil
}

// PromptAddTarget prompts for target alias, project path, and environment.
func PromptAddTarget(availableEnvs []string, defaultEnv string) (alias string, path string, env string, err error) {
	env = defaultEnv
	if env == "" && len(availableEnvs) > 0 {
		env = availableEnvs[0]
	}

	var envOptions []huh.Option[string]
	for _, e := range availableEnvs {
		envOptions = append(envOptions, huh.NewOption(e, e))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Target Alias").
				Description("Short name to refer to this project (e.g. billing, hub)").
				Placeholder("e.g. billing").
				Value(&alias).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("alias cannot be empty")
					}
					return nil
				}),
			huh.NewInput().
				Title("Project Path").
				Description("Path to .csproj file or project directory").
				Placeholder(`e.g. D:\Workspace\Project\Project.csproj`).
				Value(&path).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("path cannot be empty")
					}
					return nil
				}),
			huh.NewSelect[string]().
				Title("Target Environment").
				Options(envOptions...).
				Value(&env),
		),
	)

	if err := form.Run(); err != nil {
		return "", "", "", err
	}

	return strings.ToLower(strings.TrimSpace(alias)), strings.TrimSpace(path), strings.ToLower(strings.TrimSpace(env)), nil
}

// SelectTargetToRemove displays an interactive select menu to pick which target to remove.
func SelectTargetToRemove(targets []string) (string, error) {
	if len(targets) == 0 {
		return "", fmt.Errorf("no targets available to remove")
	}

	var selected string
	var options []huh.Option[string]
	for _, t := range targets {
		options = append(options, huh.NewOption(t, t))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select target to remove from config").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return "", err
	}

	return selected, nil
}

type PlayerCandidate struct {
	Slot     int32  `json:"slot"`
	Initials string `json:"initials"`
	Score    int64  `json:"score,omitempty"`
}

func SelectPlayersMulti(candidates []PlayerCandidate) ([]PlayerCandidate, error) {
	var selectedLabels []string
	var options []huh.Option[string]

	for _, c := range candidates {
		label := fmt.Sprintf("Slot %d: %s", c.Slot, c.Initials)
		options = append(options, huh.NewOption(label, label).Selected(true))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select players to join the lobby").
				Description("Space to toggle, Enter to confirm").
				Options(options...).
				Value(&selectedLabels),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	var result []PlayerCandidate
	for _, label := range selectedLabels {
		for _, c := range candidates {
			if fmt.Sprintf("Slot %d: %s", c.Slot, c.Initials) == label {
				result = append(result, c)
				break
			}
		}
	}
	return result, nil
}

// EnrollForm prompts the user interactively with QR code / barcode scan or manual inputs.
func EnrollForm(machineId, token, serial *string) error {
	var rawScan string

	// Step 1: Offer quick scan input (e.g. from USB barcode scanner or QR code copy)
	quickScanForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Quick QR Code / Barcode Scan").
				Description("Scan or paste payload (Format: MachineId|Token|Serial) or leave empty for manual entry").
				Placeholder("b3c8f123...|wptok_...|WP-DEV-001").
				Value(&rawScan),
		),
	)

	if err := quickScanForm.Run(); err != nil {
		return err
	}

	// If the user scanned a combined payload, parse it
	rawScan = strings.TrimSpace(rawScan)
	if rawScan != "" {
		parts := strings.Split(rawScan, "|")
		if len(parts) == 1 {
			parts = strings.Split(rawScan, ":")
		}

		if len(parts) >= 2 {
			*machineId = strings.TrimSpace(parts[0])
			*token = strings.TrimSpace(parts[1])
			if len(parts) >= 3 {
				*serial = strings.TrimSpace(parts[2])
			}
			return nil
		}
	}

	// Step 2: Fallback to manual entry if no quick scan was provided
	manualForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Machine ID (GUID)").
				Placeholder("b3c8f123-4567-89ab-cdef-0123456789ab").
				Value(machineId).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("machine ID is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Enrollment Token").
				Placeholder("wptok_...").
				Value(token).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("enrollment token is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Device Serial Number").
				Placeholder("WP-DEV-001").
				Value(serial).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("serial number is required")
					}
					return nil
				}),
		),
	)

	return manualForm.Run()
}

// PrintEnrollSuccess renders a beautiful stylized terminal card with certificate details.
func PrintEnrollSuccess(deviceId, certSerial, certPath, serialNumber string) {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#04B575")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Width(18)

	valStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#04B575")).
		Padding(1, 3).
		Margin(1, 0)

	content := fmt.Sprintf(
		"%s\n\n%s%s\n%s%s\n%s%s\n%s%s",
		titleStyle.Render("✔ DEVICE ENROLLED SUCCESSFULLY"),
		labelStyle.Render("Device Serial:"), valStyle.Render(serialNumber),
		labelStyle.Render("Device ID:"), valStyle.Render(deviceId),
		labelStyle.Render("Cert Serial:"), valStyle.Render(certSerial),
		labelStyle.Render("Cert Location:"), valStyle.Render(certPath),
	)

	fmt.Println(cardStyle.Render(content))
}

// PrintWorkstationEnrollSuccess renders a stylized terminal card for operator workstations.
func PrintWorkstationEnrollSuccess(name, serial, thumbprint, expires, certPath string) {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00D2FF")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Width(18)

	valStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00D2FF")).
		Padding(1, 3).
		Margin(1, 0)

	content := fmt.Sprintf(
		"%s\n\n%s%s\n%s%s\n%s%s\n%s%s\n%s%s",
		titleStyle.Render("🔏 WORKSTATION ENROLLED SUCCESSFULLY > NEXT-STEP: PROTECT THE SOFTWARE "),
		labelStyle.Render("Workstation:"), valStyle.Render(name),
		labelStyle.Render("Cert Serial:"), valStyle.Render(serial),
		labelStyle.Render("Thumbprint:"), valStyle.Render(thumbprint),
		labelStyle.Render("Expires:"), valStyle.Render(expires),
		labelStyle.Render("Cert Location:"), valStyle.Render(certPath),
	)

	fmt.Println(cardStyle.Render(content))
}

// PrintSoftwareCertSuccess renders a stylized terminal card for software code signing certificates.
func PrintSoftwareCertSuccess(publisher, serial, thumbprint, expires, certPath string) {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFB703")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Width(18)

	valStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFB703")).
		Padding(1, 3).
		Margin(1, 0)

	content := fmt.Sprintf(
		"%s\n\n%s%s\n%s%s\n%s%s\n%s%s\n%s%s",
		titleStyle.Render("📜 SOFTWARE CODE SIGNING CERTIFICATE ISSUED"),
		labelStyle.Render("Publisher:"), valStyle.Render(publisher),
		labelStyle.Render("Cert Serial:"), valStyle.Render(serial),
		labelStyle.Render("Thumbprint:"), valStyle.Render(thumbprint),
		labelStyle.Render("Expires:"), valStyle.Render(expires),
		labelStyle.Render("Cert Location:"), valStyle.Render(certPath),
	)

	fmt.Println(cardStyle.Render(content))
}
