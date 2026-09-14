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
