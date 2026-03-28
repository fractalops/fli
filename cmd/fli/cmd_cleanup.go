package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	fliaws "fli/internal/aws"
	"fli/internal/config"
)

var (
	cleanupProfile  string
	cleanupKeepLogs bool
	cleanupForce    bool
	cleanupAll      bool
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Delete AWS resources created by fli init",
	Long: `Tear down VPC flow log infrastructure created by fli init.

Deletes the flow log, IAM role, and CloudWatch Logs log group in reverse
dependency order. Only deletes resources tagged with managed-by=fli.`,
	RunE: runCleanup,
}

func initCleanupCommand() {
	cleanupCmd.Flags().StringVar(&cleanupProfile, "profile", "", "Profile to clean up (default: \"default\")")
	cleanupCmd.Flags().BoolVar(&cleanupKeepLogs, "keep-logs", false, "Preserve the log group (keeps historical data)")
	cleanupCmd.Flags().BoolVar(&cleanupForce, "force", false, "Skip confirmation prompt")
	cleanupCmd.Flags().BoolVar(&cleanupAll, "all", false, "Clean up all profiles")
}

func runCleanup(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	statePath, err := config.StatePath()
	if err != nil {
		return fmt.Errorf("failed to resolve state path: %w", err)
	}

	state, err := config.LoadState(statePath)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	cfgPath, err := config.ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cleanupAll {
		// Iterate all profiles with state
		for profileName := range state.Profiles {
			if err := cleanupSingleProfile(ctx, cfg, state, cfgPath, statePath, profileName); err != nil {
				fmt.Fprintf(os.Stderr, "Error cleaning up profile %q: %v\n", profileName, err)
			}
		}
		// Also clean up config-only profiles
		for profileName := range cfg.Profiles {
			if _, ok := state.GetProfileState(profileName); !ok {
				cfg.DeleteProfile(profileName)
			}
		}
		if err := cfg.Save(cfgPath); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		return nil
	}

	profileName := cleanupProfile
	if profileName == "" {
		profileName = cfg.ResolveProfile("")
	}
	if profileName == "" {
		profileName = "default"
	}

	return cleanupSingleProfile(ctx, cfg, state, cfgPath, statePath, profileName)
}

func cleanupSingleProfile(ctx context.Context, cfg *config.Config, state *config.State, cfgPath, statePath, profileName string) error {
	ps, hasState := state.GetProfileState(profileName)
	_, hasConfig := cfg.GetProfile(profileName)

	if !hasState && !hasConfig {
		return fmt.Errorf("no resources managed by fli for profile %q. Nothing to clean up", profileName)
	}

	// Config-only profile (used existing flow log)
	if !hasState {
		cfg.DeleteProfile(profileName)
		if err := cfg.Save(cfgPath); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Removed profile %q from config (no managed resources).\n", profileName)
		return nil
	}

	// Show what will be deleted
	if !cleanupForce {
		showCleanupPanel(ps, profileName)

		var confirmed bool
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Delete these resources?").
					Affirmative("Yes, delete").
					Negative("Cancel").
					Value(&confirmed),
			),
		).Run()
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	// Resolve region for AWS client
	region := ps.Region
	if region == "" {
		if p, ok := cfg.GetProfile(profileName); ok {
			region = p.Region
		}
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(awsCfg)
	iamClient := iam.NewFromConfig(awsCfg)
	cwlClient := cloudwatchlogs.NewFromConfig(awsCfg)

	// Delete in reverse dependency order
	// 1. Flow log
	if r, ok := state.FindResource(profileName, config.ResourceTypeFlowLog); ok {
		if err := deleteFlowLogResource(ctx, ec2Client, r); err != nil {
			return err
		}
		_ = state.RemoveResource(statePath, profileName, r)
	}

	// 2. IAM role policy
	if r, ok := state.FindResource(profileName, config.ResourceTypeRolePolicy); ok {
		_ = state.RemoveResource(statePath, profileName, r)
	}

	// 3. IAM role (deletes policy too)
	if r, ok := state.FindResource(profileName, config.ResourceTypeIAMRole); ok {
		if err := deleteIAMRoleResource(ctx, iamClient, r); err != nil {
			return err
		}
		_ = state.RemoveResource(statePath, profileName, r)
	}

	// 4. Log group (unless --keep-logs)
	if r, ok := state.FindResource(profileName, config.ResourceTypeLogGroup); ok {
		if err := deleteLogGroupResource(ctx, cwlClient, r); err != nil {
			return err
		}
		if !cleanupKeepLogs {
			_ = state.RemoveResource(statePath, profileName, r)
		}
	}

	// Remove from config
	cfg.DeleteProfile(profileName)
	if err := cfg.Save(cfgPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nProfile %q cleaned up.\n", profileName)
	return nil
}

func deleteFlowLogResource(ctx context.Context, ec2Client *ec2.Client, r config.Resource) error {
	if err := fliaws.DeleteFlowLog(ctx, ec2Client, r.ID); err != nil {
		if fliaws.IsNotFound(err) {
			fmt.Fprintf(os.Stderr, "⚠ Warning: VPC flow log %s not found (may have been manually deleted)\n", r.ID)
			return nil
		}
		return fliaws.WrapError(err, "delete flow log")
	}
	fmt.Fprintf(os.Stderr, "✓ Deleted flow log %s\n", r.ID)
	return nil
}

func deleteIAMRoleResource(ctx context.Context, iamClient *iam.Client, r config.Resource) error {
	if err := fliaws.DeleteFlowLogRole(ctx, iamClient, r.Name, fliaws.FlowLogPolicyName); err != nil {
		if fliaws.IsNotFound(err) {
			fmt.Fprintf(os.Stderr, "⚠ Warning: IAM role %s not found (may have been manually deleted)\n", r.Name)
			return nil
		}
		return fliaws.WrapError(err, "delete IAM role")
	}
	fmt.Fprintf(os.Stderr, "✓ Deleted IAM role %s\n", r.Name)
	return nil
}

func deleteLogGroupResource(ctx context.Context, cwlClient *cloudwatchlogs.Client, r config.Resource) error {
	if cleanupKeepLogs {
		fmt.Fprintf(os.Stderr, "● Keeping log group %s (--keep-logs)\n", r.Name)
		return nil
	}
	if err := fliaws.DeleteLogGroup(ctx, cwlClient, r.Name); err != nil {
		if fliaws.IsNotFound(err) {
			fmt.Fprintf(os.Stderr, "⚠ Warning: Log group %s not found (may have been manually deleted)\n", r.Name)
			return nil
		}
		return fliaws.WrapError(err, "delete log group")
	}
	fmt.Fprintf(os.Stderr, "✓ Deleted log group %s\n", r.Name)
	return nil
}

func showCleanupPanel(ps *config.ProfileState, profileName string) {
	var lines []string
	for _, r := range ps.Resources {
		switch r.Type {
		case config.ResourceTypeFlowLog:
			lines = append(lines, fmt.Sprintf("  ✗ VPC Flow Log      %s  (%s)", r.ID, r.ResourceID))
		case config.ResourceTypeRolePolicy:
			lines = append(lines, fmt.Sprintf("  ✗ IAM Role Policy   %s", r.PolicyName))
		case config.ResourceTypeIAMRole:
			lines = append(lines, fmt.Sprintf("  ✗ IAM Role          %s", r.Name))
		case config.ResourceTypeLogGroup:
			lines = append(lines, fmt.Sprintf("  ✗ Log Group         %s", r.Name))
			lines = append(lines, "                      ⚠ This deletes all stored logs")
		}
	}

	content := strings.Join(lines, "\n")

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Render(content)

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Cleanup: %s", profileName))
	fmt.Fprintf(os.Stderr, "\n%s\n%s\n\n", title, panel)
}
