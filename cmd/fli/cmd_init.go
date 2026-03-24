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
	"fli/internal/flowlog"
)

var (
	initProfile          string
	initRegion           string
	initCheckPermissions bool
	initNoTUI            bool
	initForce            bool

	// Non-interactive flags for CI.
	initVPC          string
	initSubnet       string
	initENI          string
	initTraffic      string
	initFields       string
	initInterval     int
	initRetention    int
	initLogGroupName string
	initUse          string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up VPC flow log infrastructure",
	Long: `Discover existing VPC flow logs or create new ones with all required AWS resources.

Creates an IAM role and VPC flow log, then saves the configuration as a named
profile for use with fli query commands.

Interactive mode (default):
  fli init

Non-interactive mode (for CI/automation):
  fli init --vpc vpc-0abc123 --traffic ALL --fields security --force
  fli init --subnet subnet-aaa111 --traffic REJECT --fields full --force
  fli init --use fl-0abc123 --force`,
	RunE: runInit,
}

func initInitCommand() {
	initCmd.Flags().StringVar(&initProfile, "profile", "", "Profile name (default: \"default\")")
	initCmd.Flags().StringVar(&initRegion, "region", "", "AWS region override")
	initCmd.Flags().BoolVar(&initCheckPermissions, "check-permissions", false, "Check IAM permissions and exit")
	initCmd.Flags().BoolVar(&initNoTUI, "no-tui", false, "Use plain line-by-line prompts instead of TUI")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Skip confirmation prompt")

	// Non-interactive resource selection (mutually exclusive)
	initCmd.Flags().StringVar(&initVPC, "vpc", "", "VPC ID to monitor (non-interactive)")
	initCmd.Flags().StringVar(&initSubnet, "subnet", "", "Subnet ID to monitor (non-interactive)")
	initCmd.Flags().StringVar(&initENI, "eni", "", "ENI ID to monitor (non-interactive)")
	initCmd.Flags().StringVar(&initUse, "use", "", "Use an existing flow log by ID (non-interactive)")

	// Non-interactive configuration
	initCmd.Flags().StringVar(&initTraffic, "traffic", "ALL", "Traffic type: ALL, ACCEPT, REJECT")
	initCmd.Flags().StringVar(&initFields, "fields", flowlog.PresetDefault, "Field set: default, security, troubleshooting, full")
	initCmd.Flags().IntVar(&initInterval, "interval", 600, "Aggregation interval in seconds: 60 or 600")
	initCmd.Flags().IntVar(&initRetention, "retention", 30, "Log group retention in days (0 for never expire)")
	initCmd.Flags().StringVar(&initLogGroupName, "log-group-name", "", "Custom log group name (default: /fli/flow-logs/<resource-id>)")

	initCmd.MarkFlagsMutuallyExclusive("vpc", "subnet", "eni", "use")
}

// isNonInteractive returns true if any resource flag was provided.
func isNonInteractive() bool {
	return initVPC != "" || initSubnet != "" || initENI != "" || initUse != ""
}

func runInit(cmd *cobra.Command, _ []string) error {
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()

	region, err := resolveRegion(ctx, initRegion)
	if err != nil {
		return err
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(awsCfg)
	stsClient := sts.NewFromConfig(awsCfg)
	iamClient := iam.NewFromConfig(awsCfg)
	cwlClient := cloudwatchlogs.NewFromConfig(awsCfg)

	// Get caller identity
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

	// Branch: non-interactive or interactive
	var initCfg *InitConfig
	if isNonInteractive() {
		initCfg, err = buildNonInteractiveConfig(flowLogs)
	} else {
		initCfg, err = runInitWizard(ctx, ec2Client, flowLogs, initProfile, initNoTUI, region)
	}
	if err != nil {
		return handleAbort(err)
	}

	// Use existing flow log — just save profile
	if initCfg.UseExisting {
		return saveExistingProfile(initCfg, region)
	}

	// Confirmation
	if !initForce {
		confirmed, err := showConfirmation(initCfg, region, initNoTUI)
		if err != nil {
			return handleAbort(err)
		}
		if !confirmed {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	// Create resources
	err = createResources(ctx, ec2Client, iamClient, cwlClient, initCfg, region, identity.AccountID)
	if err != nil {
		return handleAbort(err)
	}
	return nil
}

func buildNonInteractiveConfig(flowLogs []fliaws.FlowLogInfo) (*InitConfig, error) {
	cfg := &InitConfig{}

	profileName := initProfile
	if profileName == "" {
		profileName = defaultProfileName
	}
	cfg.ProfileName = profileName

	// Use existing flow log
	if initUse != "" {
		cfg.UseExisting = true
		// Find the flow log by ID
		for _, fl := range flowLogs {
			if fl.FlowLogID == initUse {
				cfg.LogGroupName = fl.LogGroupName
				cfg.Version = fl.Version
				cfg.ResourceID = fl.ResourceID
				return cfg, nil
			}
		}
		return nil, fmt.Errorf("flow log %q not found. Run \"fli init\" interactively to discover available flow logs", initUse)
	}

	// Determine resource type and ID
	switch {
	case initVPC != "":
		cfg.ResourceType = resourceTypeVPC
		cfg.ResourceID = initVPC
	case initSubnet != "":
		cfg.ResourceType = resourceTypeSubnet
		cfg.ResourceID = initSubnet
	case initENI != "":
		cfg.ResourceType = resourceTypeNetworkInterface
		cfg.ResourceID = initENI
	}

	// Validate traffic type
	switch initTraffic {
	case "ALL", "ACCEPT", "REJECT":
		cfg.TrafficType = initTraffic
	default:
		return nil, fmt.Errorf("invalid --traffic %q: must be ALL, ACCEPT, or REJECT", initTraffic)
	}

	// Validate field set
	switch initFields {
	case flowlog.PresetDefault, flowlog.PresetSecurity, flowlog.PresetTroubleshooting, flowlog.PresetFull:
		cfg.FieldSet = initFields
	default:
		return nil, fmt.Errorf("invalid --fields %q: must be default, security, troubleshooting, or full", initFields)
	}

	// Validate interval
	switch initInterval {
	case 60, 600:
		cfg.AggInterval = fmt.Sprintf("%d", initInterval)
	default:
		return nil, fmt.Errorf("invalid --interval %d: must be 60 or 600", initInterval)
	}

	cfg.RetentionDays = initRetention
	cfg.Version = flowlog.PresetVersion(cfg.FieldSet)

	// Log group name
	if initLogGroupName != "" {
		cfg.LogGroupName = initLogGroupName
	} else {
		cfg.LogGroupName = fliaws.FlowLogLogGroupName(cfg.ResourceID)
	}

	// Check for duplicate flow log
	if existing := fliaws.HasExistingFlowLog(flowLogs, cfg.ResourceID); existing != nil {
		return nil, fmt.Errorf("a flow log already exists on %s:\n  %s → %s (v%d, %s traffic)\n\nUse --use %s instead",
			cfg.ResourceID, existing.FlowLogID, existing.LogGroupName, existing.Version, existing.TrafficType, existing.FlowLogID)
	}

	return cfg, nil
}

// handleAbort checks if an error is a user abort (Ctrl+C) and returns a clean message.
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
