package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/charmbracelet/huh/spinner"

	fliaws "fli/internal/aws"
	"fli/internal/config"
	"fli/internal/flowlog"
)

const iamPropagationDelay = 10 * time.Second

func createResources(ctx context.Context, ec2Client fliaws.FlowLogsAPI, iamClient fliaws.IAMAPI, cwlClient fliaws.CloudWatchLogsManagementAPI, initCfg *InitConfig, region, accountID string) error {
	initCfg.Region = region

	statePath, err := config.StatePath()
	if err != nil {
		return fmt.Errorf("failed to resolve state path: %w", err)
	}

	state, err := config.LoadState(statePath)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	roleName := fliaws.FlowLogRoleName(initCfg.ResourceID)
	profileName := initCfg.ProfileName

	// Check for partial state from a previous failed run
	existingRole, _ := state.FindResource(profileName, config.ResourceTypeIAMRole)
	existingFlowLog, _ := state.FindResource(profileName, config.ResourceTypeFlowLog)

	var roleARN string

	// Step 1: Create IAM Role
	if existingRole.Name != "" { //nolint:nestif // readability preferred over refactoring
		exists, err := fliaws.RoleExists(ctx, iamClient, existingRole.Name)
		if err != nil {
			return fmt.Errorf("failed to check role existence: %w", err)
		}
		if exists {
			fmt.Fprintf(os.Stderr, "✓ IAM role %s already exists (from previous run)\n", existingRole.Name)
			roleARN = existingRole.ARN
		} else {
			_ = state.RemoveResource(statePath, profileName, existingRole)
			roleARN, err = createIAMRole(ctx, iamClient, state, statePath, initCfg, roleName, accountID)
			if err != nil {
				return err
			}
		}
	} else {
		roleARN, err = createIAMRole(ctx, iamClient, state, statePath, initCfg, roleName, accountID)
		if err != nil {
			return err
		}
	}

	// Step 2: Wait for IAM propagation
	err = spinner.New().
		Title("Waiting for IAM propagation...").
		Action(func() { time.Sleep(iamPropagationDelay) }).
		Run()
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "✓ IAM propagation complete")

	// Step 3: Create VPC Flow Log (AWS auto-creates the log group)
	var flowLogID string
	if existingFlowLog.ID != "" {
		fmt.Fprintf(os.Stderr, "✓ Flow log %s already exists (from previous run)\n", existingFlowLog.ID)
		flowLogID = existingFlowLog.ID
	} else {
		flowLogID, err = createFlowLog(ctx, ec2Client, state, statePath, initCfg, roleARN)
		if err != nil {
			return err
		}
	}
	_ = flowLogID

	// Step 4: Configure the auto-created log group (retention + tags)
	err = configureLogGroup(ctx, cwlClient, state, statePath, initCfg)
	if err != nil {
		// Non-fatal — flow log is working, just missing retention/tags
		if errors.Is(err, fliaws.ErrLogGroupNotYetCreated) {
			fmt.Fprintf(os.Stderr, "⚠ Log group not yet created by AWS (no traffic recorded yet).\n")
			fmt.Fprintf(os.Stderr, "  Retention will be set automatically on next run of \"fli init --profile %s\",\n", profileName)
			fmt.Fprintf(os.Stderr, "  or set it manually:\n")
			fmt.Fprintf(os.Stderr, "  aws logs put-retention-policy --log-group-name %s --retention-in-days %d\n", initCfg.LogGroupName, initCfg.RetentionDays)
		} else {
			fmt.Fprintf(os.Stderr, "⚠ Warning: failed to configure log group: %v\n", err)
			fmt.Fprintf(os.Stderr, "  The flow log is active but the log group has no retention policy.\n")
			fmt.Fprintf(os.Stderr, "  Re-run \"fli init --profile %s\" to retry.\n", profileName)
		}
	}

	// Save profile config
	cfgPath, err := config.ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg.SetProfile(profileName, config.ProfileConfig{
		Region:   region,
		LogGroup: initCfg.LogGroupName,
		Version:  initCfg.Version,
	})

	if cfg.ActiveProfile == "" {
		cfg.ActiveProfile = profileName
	}

	if err := cfg.Save(cfgPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nProfile %q is ready. Run: fli raw --profile %s srcaddr dstaddr\n", profileName, profileName)
	return nil
}

func createIAMRole(ctx context.Context, iamClient fliaws.IAMAPI, state *config.State, statePath string, initCfg *InitConfig, roleName, accountID string) (string, error) {
	var roleARN string
	var createErr error

	err := spinner.New().
		Title(fmt.Sprintf("Creating IAM role %s...", roleName)).
		Action(func() {
			roleARN, createErr = fliaws.CreateFlowLogRole(ctx, iamClient, roleName, accountID)
		}).
		Run()
	if err != nil {
		return "", err
	}
	if createErr != nil {
		return "", fliaws.WrapError(createErr, "create IAM role")
	}

	// Save role to state
	if err := state.AddResource(statePath, initCfg.ProfileName, initCfg.Region, config.Resource{
		Type: config.ResourceTypeIAMRole,
		Name: roleName,
		ARN:  roleARN,
	}); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	// Save role policy to state
	if err := state.AddResource(statePath, initCfg.ProfileName, initCfg.Region, config.Resource{
		Type:       config.ResourceTypeRolePolicy,
		RoleName:   roleName,
		PolicyName: fliaws.FlowLogPolicyName,
	}); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Created IAM role %s\n", roleName)
	return roleARN, nil
}

func createFlowLog(ctx context.Context, ec2Client fliaws.FlowLogsAPI, state *config.State, statePath string, initCfg *InitConfig, roleARN string) (string, error) {
	var flowLogID string
	var createErr error

	// Build format string
	var logFormat string
	if initCfg.FieldSet == flowlog.PresetCustom {
		logFormat = flowlog.FormatString(initCfg.CustomFields)
	} else if initCfg.FieldSet != flowlog.PresetDefault {
		logFormat = flowlog.FormatString(flowlog.PresetFields(initCfg.FieldSet))
	}

	// Map resource type to EC2 type
	resourceType := types.FlowLogsResourceTypeVpc
	switch initCfg.ResourceType {
	case resourceTypeSubnet:
		resourceType = types.FlowLogsResourceTypeSubnet
	case resourceTypeNetworkInterface:
		resourceType = types.FlowLogsResourceTypeNetworkInterface
	}

	// Map traffic type
	trafficType := types.TrafficTypeAll
	switch initCfg.TrafficType {
	case "ACCEPT":
		trafficType = types.TrafficTypeAccept
	case "REJECT":
		trafficType = types.TrafficTypeReject
	}

	aggInterval, _ := strconv.ParseInt(initCfg.AggInterval, 10, 32)
	if aggInterval < 0 || aggInterval > math.MaxInt32 {
		return "", fmt.Errorf("aggregation interval %d out of range", aggInterval)
	}

	err := spinner.New().
		Title(fmt.Sprintf("Creating flow log on %s...", initCfg.ResourceID)).
		Action(func() {
			flowLogID, createErr = fliaws.CreateFlowLog(ctx, ec2Client, fliaws.CreateFlowLogInput{
				ResourceID:   initCfg.ResourceID,
				ResourceType: resourceType,
				TrafficType:  trafficType,
				LogGroupName: initCfg.LogGroupName,
				IAMRoleARN:   roleARN,
				LogFormat:    logFormat,
				AggInterval:  int32(aggInterval), //nolint:gosec // bounds checked on line above
				ProfileName:  initCfg.ProfileName,
			})
		}).
		Run()
	if err != nil {
		return "", err
	}
	if createErr != nil {
		wrapped := fliaws.WrapError(createErr, "create flow log")
		fmt.Fprintf(os.Stderr, "\nResources already created have been saved. Run \"fli cleanup --profile %s\" to remove them, or re-run \"fli init --profile %s\" to retry.\n", initCfg.ProfileName, initCfg.ProfileName)
		return "", wrapped
	}

	// Save to state
	if err := state.AddResource(statePath, initCfg.ProfileName, initCfg.Region, config.Resource{
		Type:       config.ResourceTypeFlowLog,
		ID:         flowLogID,
		ResourceID: initCfg.ResourceID,
	}); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Created flow log on %s (%s)\n", initCfg.ResourceID, flowLogID)
	return flowLogID, nil
}

func configureLogGroup(ctx context.Context, cwlClient fliaws.CloudWatchLogsManagementAPI, state *config.State, statePath string, initCfg *InitConfig) error {
	var configErr error

	err := spinner.New().
		Title(fmt.Sprintf("Configuring log group %s...", initCfg.LogGroupName)).
		Action(func() {
			configErr = fliaws.ConfigureLogGroup(ctx, cwlClient, initCfg.LogGroupName, initCfg.RetentionDays, initCfg.ProfileName)
		}).
		Run()
	if err != nil {
		return err
	}
	if configErr != nil {
		return configErr
	}

	// Track the log group in state so cleanup can delete it
	logGroupARN, _ := fliaws.GetLogGroupARN(ctx, cwlClient, initCfg.LogGroupName)
	if err := state.AddResource(statePath, initCfg.ProfileName, initCfg.Region, config.Resource{
		Type: config.ResourceTypeLogGroup,
		Name: initCfg.LogGroupName,
		ARN:  logGroupARN,
	}); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Configured log group %s (retention: %s)\n", initCfg.LogGroupName, retentionLabel(initCfg.RetentionDays))
	return nil
}

func retentionLabel(days int) string {
	switch days {
	case 0:
		return "never expire"
	case 365:
		return "1 year"
	default:
		return fmt.Sprintf("%d days", days)
	}
}
