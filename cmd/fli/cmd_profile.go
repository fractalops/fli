package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"

	"fli/internal/config"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage flow log profiles",
	Long:  "Manage named flow log configurations. Profiles store log group, region, and version settings.",
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE:  runProfileList,
}

var profileUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the active profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileUse,
}

var profileShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show profile details",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runProfileShow,
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Remove a profile from config (does not delete AWS resources)",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileDelete,
}

func initProfileCommands() {
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileUseCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileDeleteCmd)
}

func runProfileList(_ *cobra.Command, _ []string) error {
	cfgPath, err := config.ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Profiles) == 0 {
		fmt.Fprintln(os.Stderr, "No profiles configured. Run \"fli init\" to create one.")
		return nil
	}

	headerStyle := lipgloss.NewStyle().Bold(true)
	activeMarker := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	rows := make([][]string, 0, len(cfg.Profiles))
	for name, p := range cfg.Profiles {
		marker := " "
		if name == cfg.ActiveProfile {
			marker = activeMarker.Render("▸")
		}
		rows = append(rows, []string{marker, name, p.LogGroup, fmt.Sprintf("%d", p.Version), p.Region})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		Headers("", "Name", "Log Group", "Version", "Region").
		Rows(rows...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return lipgloss.NewStyle()
		})

	fmt.Println(t)
	return nil
}

func runProfileUse(_ *cobra.Command, args []string) error {
	name := args[0]

	cfgPath, err := config.ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if _, ok := cfg.GetProfile(name); !ok {
		return fmt.Errorf("profile %q not found. Run \"fli profile list\" to see available profiles", name)
	}

	cfg.ActiveProfile = name
	if err := cfg.Save(cfgPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Active profile set to %q\n", name)
	return nil
}

func runProfileShow(_ *cobra.Command, args []string) error {
	cfgPath, err := config.ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	name := cfg.ActiveProfile
	if len(args) > 0 {
		name = args[0]
	}

	if name == "" {
		if len(cfg.Profiles) == 0 {
			return fmt.Errorf("no profiles configured. Run \"fli init\" to create one")
		}
		return fmt.Errorf("no active profile set. Run \"fli profile use <name>\" or specify one: fli profile show <name>")
	}

	profile, ok := cfg.GetProfile(name)
	if !ok {
		return fmt.Errorf("profile %q not found. Run \"fli profile list\" to see available profiles", name)
	}

	activeStr := "—"
	if name == cfg.ActiveProfile {
		activeStr = "✓"
	}

	// Build content
	content := fmt.Sprintf(
		"  Region               %s\n"+
			"  Log Group            %s\n"+
			"  Version              %d\n"+
			"  Active               %s",
		profile.Region, profile.LogGroup, profile.Version, activeStr,
	)

	// Check for managed resources in state
	statePath, err := config.StatePath()
	if err == nil {
		state, err := config.LoadState(statePath)
		if err == nil {
			if ps, ok := state.GetProfileState(name); ok && len(ps.Resources) > 0 {
				content += "\n\n  Managed Resources\n  ────────────────"
				for _, r := range ps.Resources {
					switch r.Type {
					case config.ResourceTypeLogGroup:
						content += fmt.Sprintf("\n  CloudWatch Log Group %s", r.Name)
					case config.ResourceTypeIAMRole:
						content += fmt.Sprintf("\n  IAM Role             %s", r.Name)
					case config.ResourceTypeRolePolicy:
						content += fmt.Sprintf("\n  IAM Role Policy      %s", r.PolicyName)
					case config.ResourceTypeFlowLog:
						content += fmt.Sprintf("\n  VPC Flow Log         %s (%s)", r.ID, r.ResourceID)
					}
				}
				content += fmt.Sprintf("\n\n  Created              %s", ps.CreatedAt.Format("2006-01-02T15:04:05Z"))
			} else {
				content += "\n\n  Managed Resources    none (using pre-existing flow log)"
			}
		}
	}

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Render(content)

	title := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Profile: %s", name))
	fmt.Printf("%s\n%s\n", title, panel)
	return nil
}

func runProfileDelete(_ *cobra.Command, args []string) error {
	name := args[0]

	cfgPath, err := config.ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if _, ok := cfg.GetProfile(name); !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	cfg.DeleteProfile(name)
	if err := cfg.Save(cfgPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Removed profile %q from config.\n", name)

	// Warn if state has resources
	statePath, err := config.StatePath()
	if err == nil {
		state, err := config.LoadState(statePath)
		if err == nil && state.HasResources(name) {
			fmt.Fprintf(os.Stderr, "\nWarning: This profile has managed AWS resources.\nRun \"fli cleanup --profile %s\" to delete them, or they will remain in AWS.\n", name)
		}
	}

	return nil
}
