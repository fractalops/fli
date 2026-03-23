package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	fliaws "fli/internal/aws"
	"fli/internal/flowlog"
)

func showConfirmation(initCfg *InitConfig, region string, noTUI bool) (bool, error) {
	roleName := fliaws.FlowLogRoleName(initCfg.ResourceID)

	// Build the field set label
	fieldSetLabel := initCfg.FieldSet
	switch initCfg.FieldSet {
	case flowlog.PresetDefault:
		fieldSetLabel = "Default (v2)"
	case flowlog.PresetSecurity:
		fieldSetLabel = "Security (v5)"
	case flowlog.PresetTroubleshooting:
		fieldSetLabel = "Troubleshooting (v4)"
	case flowlog.PresetFull:
		fieldSetLabel = "Full (v5)"
	case flowlog.PresetCustom:
		fieldSetLabel = fmt.Sprintf("Custom (%d fields)", len(initCfg.CustomFields))
	}

	intervalLabel := "10 minutes"
	if initCfg.AggInterval == "60" {
		intervalLabel = "1 minute"
	}

	var retentionLabel string
	switch initCfg.RetentionDays {
	case 0:
		retentionLabel = "never expire"
	case 365:
		retentionLabel = "1 year"
	default:
		retentionLabel = fmt.Sprintf("%d days", initCfg.RetentionDays)
	}

	// Build content lines
	lines := []string{
		fmt.Sprintf("  Profile              %s", initCfg.ProfileName),
		fmt.Sprintf("  Region               %s", region),
		fmt.Sprintf("  Resource             %s", initCfg.ResourceID),
		fmt.Sprintf("  CloudWatch Log Group %s  (%s retention)", initCfg.LogGroupName, retentionLabel),
		fmt.Sprintf("  IAM Role             %s", roleName),
		fmt.Sprintf("  Flow Log             %s traffic | %s | %s interval", initCfg.TrafficType, fieldSetLabel, intervalLabel),
		"",
		fmt.Sprintf("  Tags                 managed-by=fli, fli-profile=%s", initCfg.ProfileName),
	}

	content := strings.Join(lines, "\n")

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Render(content)

	title := lipgloss.NewStyle().Bold(true).Render("Review")
	fmt.Printf("\n%s\n%s\n\n", title, panel)

	var confirmed bool
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Create these resources?").
				Affirmative("Yes, create").
				Negative("Cancel").
				Value(&confirmed),
		),
	).WithAccessible(noTUI).Run()
	if err != nil {
		return false, err
	}

	return confirmed, nil
}
