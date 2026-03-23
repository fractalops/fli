package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/spf13/cobra"

	fliaws "fli/internal/aws"
	"fli/internal/config"
)

var (
	initProfile          string
	initRegion           string
	initCheckPermissions bool
	initNoTUI            bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up VPC flow log infrastructure",
	Long: `Discover existing VPC flow logs or create new ones with all required AWS resources.

Creates a CloudWatch Logs log group, IAM role, and VPC flow log, then saves
the configuration as a named profile for use with fli query commands.`,
	RunE: runInit,
}

func initInitCommand() {
	initCmd.Flags().StringVar(&initProfile, "profile", "", "Profile name (default: \"default\")")
	initCmd.Flags().StringVar(&initRegion, "region", "", "AWS region override")
	initCmd.Flags().BoolVar(&initCheckPermissions, "check-permissions", false, "Check IAM permissions and exit")
	initCmd.Flags().BoolVar(&initNoTUI, "no-tui", false, "Use plain line-by-line prompts instead of TUI")
}

func runInit(cmd *cobra.Command, _ []string) error {
	// Set up signal-aware context so Ctrl+C cancels ongoing AWS calls
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()

	// Resolve region
	region, err := resolveRegion(ctx, initRegion)
	if err != nil {
		return err
	}

	// Load AWS config
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(awsCfg)
	stsClient := sts.NewFromConfig(awsCfg)
	iamClient := iam.NewFromConfig(awsCfg)
	cwlClient := cloudwatchlogs.NewFromConfig(awsCfg)

	// Get caller identity (needed for account ID)
	var identity fliaws.CallerIdentity
	var identityErr error
	err = spinner.New().
		Title("Verifying AWS credentials...").
		Action(func() {
			identity, identityErr = fliaws.GetCallerIdentity(ctx, stsClient)
		}).
		Run()
	if err != nil {
		return handleAbort(err)
	}
	if identityErr != nil {
		return fmt.Errorf("failed to verify AWS credentials: %w", identityErr)
	}
	fmt.Fprintf(os.Stderr, "✓ Authenticated as %s\n", identity.ARN)

	// Check permissions only mode
	if initCheckPermissions {
		return runPermissionCheck(ctx, ec2Client, iamClient, cwlClient, identity)
	}

	// Discovery
	var flowLogs []fliaws.FlowLogInfo
	var discoveryErr error
	err = spinner.New().
		Title("Discovering flow logs in " + region + "...").
		Action(func() {
			flowLogs, discoveryErr = fliaws.DiscoverFlowLogs(ctx, ec2Client)
		}).
		Run()
	if err != nil {
		return handleAbort(err)
	}
	if discoveryErr != nil {
		return fmt.Errorf("failed to discover flow logs: %w", discoveryErr)
	}

	// Run the wizard
	initCfg, err := runInitWizard(ctx, ec2Client, flowLogs, initProfile, initNoTUI, region)
	if err != nil {
		return handleAbort(err)
	}

	// If user selected an existing flow log, just save the profile
	if initCfg.UseExisting {
		return saveExistingProfile(initCfg, region)
	}

	// Show confirmation panel
	confirmed, err := showConfirmation(initCfg, region, initNoTUI)
	if err != nil {
		return handleAbort(err)
	}
	if !confirmed {
		fmt.Fprintln(os.Stderr, "Cancelled.")
		return nil
	}

	// Create resources
	err = createResources(ctx, ec2Client, iamClient, cwlClient, initCfg, region, identity.AccountID)
	if err != nil {
		return handleAbort(err)
	}
	return nil
}

// handleAbort checks if an error is a user abort (Ctrl+C) and returns a clean message.
// For all other errors, it returns them unchanged.
func handleAbort(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, huh.ErrUserAborted) || errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "\nAborted.")
		return nil
	}
	return err
}

func resolveRegion(ctx context.Context, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	// Try loading from profile config
	cfgPath, err := config.ConfigPath()
	if err == nil {
		cfg, err := config.LoadConfig(cfgPath)
		if err == nil {
			profileName := cfg.ResolveProfile(initProfile)
			if p, ok := cfg.GetProfile(profileName); ok && p.Region != "" {
				return p.Region, nil
			}
		}
	}

	// Fall back to AWS SDK region resolution
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("no AWS region configured.\n\nSet a region using one of:\n  fli init --region us-east-1\n  export AWS_REGION=us-east-1\n  aws configure set region us-east-1")
	}
	if awsCfg.Region == "" {
		return "", fmt.Errorf("no AWS region configured.\n\nSet a region using one of:\n  fli init --region us-east-1\n  export AWS_REGION=us-east-1\n  aws configure set region us-east-1")
	}
	return awsCfg.Region, nil
}

func saveExistingProfile(initCfg *InitConfig, region string) error {
	cfgPath, err := config.ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg.SetProfile(initCfg.ProfileName, config.ProfileConfig{
		Region:   region,
		LogGroup: initCfg.LogGroupName,
		Version:  initCfg.Version,
	})

	if cfg.ActiveProfile == "" {
		cfg.ActiveProfile = initCfg.ProfileName
	}

	if err := cfg.Save(cfgPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nProfile %q is ready. Run: fli raw --profile %s srcaddr dstaddr\n", initCfg.ProfileName, initCfg.ProfileName)
	return nil
}

func runPermissionCheck(_ context.Context, _ *ec2.Client, _ *iam.Client, _ *cloudwatchlogs.Client, _ fliaws.CallerIdentity) error {
	// TODO: Implement SimulatePrincipalPolicy-based permission check
	fmt.Fprintln(os.Stderr, "Permission check not yet implemented. AWS credentials are valid.")
	return nil
}
