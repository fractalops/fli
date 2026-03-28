package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	fliaws "fli/internal/aws"
	"fli/internal/flowlog"
)

const (
	checkMark          = "✓"
	emDash             = "—"
	defaultProfileName = "default"

	// Resource type constants used in wizard selections.
	resourceTypeVPC              = "VPC"
	resourceTypeSubnet           = "Subnet"
	resourceTypeNetworkInterface = "NetworkInterface"
)

// InitConfig captures all the user's choices from the init wizard.
type InitConfig struct {
	// Common
	ProfileName string
	UseExisting bool
	Region      string
	Confirmed   bool

	// Existing flow log
	LogGroupName string
	Version      int

	// New flow log creation
	ResourceType  string // "VPC", "Subnet", "NetworkInterface"
	ResourceID    string
	ResourceLabel string // human-readable label (e.g. "vpc-abc (prod)")
	TrafficType   string // "ALL", "ACCEPT", "REJECT"
	FieldSet      string // "default", "security", "troubleshooting", "full", "custom"
	CustomFields  []string
	AggInterval   string // "60" or "600"
	RetentionDays int
}

func runInitWizard(ctx context.Context, ec2Client fliaws.FlowLogsAPI, flowLogs []fliaws.FlowLogInfo, profileFlag string, noTUI bool, region string) (*InitConfig, error) {
	cfg := &InitConfig{}

	// Filter to CloudWatch Logs flow logs
	var cwlFlowLogs []fliaws.FlowLogInfo
	for _, fl := range flowLogs {
		if fl.DestType == "cloud-watch-logs" {
			cwlFlowLogs = append(cwlFlowLogs, fl)
		}
	}

	// Display discovery results
	displayDiscoveryTable(flowLogs, region)

	// Case A: No flow logs at all
	if len(flowLogs) == 0 {
		fmt.Fprintln(os.Stderr, "\n● No existing flow logs found in "+region)
		fmt.Fprintln(os.Stderr, "\n→ Proceeding to create a new flow log...")
		return runCreateWizard(ctx, ec2Client, flowLogs, cfg, profileFlag, noTUI, region)
	}

	// Case B: Flow logs exist but none are CloudWatch
	if len(cwlFlowLogs) == 0 {
		fmt.Fprintln(os.Stderr, "\n● No CloudWatch Logs flow logs found (required for fli)")
		fmt.Fprintln(os.Stderr, "\n→ Proceeding to create a new flow log...")
		return runCreateWizard(ctx, ec2Client, flowLogs, cfg, profileFlag, noTUI, region)
	}

	// Case C: Usable flow logs exist — ask what to do
	var choice string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What would you like to do?").
				Options(
					huh.NewOption("Use an existing flow log", "existing"),
					huh.NewOption("Create a new flow log", "create"),
				).
				Value(&choice),
		),
	).WithAccessible(noTUI).Run()
	if err != nil {
		return nil, err
	}

	if choice == "existing" {
		return runExistingWizard(cwlFlowLogs, cfg, profileFlag, noTUI)
	}
	return runCreateWizard(ctx, ec2Client, flowLogs, cfg, profileFlag, noTUI, region)
}

func runExistingWizard(cwlFlowLogs []fliaws.FlowLogInfo, cfg *InitConfig, profileFlag string, noTUI bool) (*InitConfig, error) {
	cfg.UseExisting = true
	cfg.ProfileName = defaultProfileName
	if profileFlag != "" {
		cfg.ProfileName = profileFlag
	}

	// Build options from discovered flow logs
	options := make([]huh.Option[string], 0, len(cwlFlowLogs))
	flowLogMap := make(map[string]fliaws.FlowLogInfo)
	for _, fl := range cwlFlowLogs {
		label := fmt.Sprintf("%s → %s (v%d)", fl.ResourceID, fl.LogGroupName, fl.Version)
		options = append(options, huh.NewOption(label, fl.FlowLogID))
		flowLogMap[fl.FlowLogID] = fl
	}

	var selectedID string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Profile name").
				Value(&cfg.ProfileName).
				Validate(validateProfileName),
			huh.NewSelect[string]().
				Title("Select a flow log").
				Options(options...).
				Value(&selectedID),
		),
	).WithAccessible(noTUI).Run()
	if err != nil {
		return nil, err
	}

	selected := flowLogMap[selectedID]
	cfg.LogGroupName = selected.LogGroupName
	cfg.Version = selected.Version
	cfg.ResourceID = selected.ResourceID

	return cfg, nil
}

func runCreateWizard(ctx context.Context, ec2Client fliaws.FlowLogsAPI, flowLogs []fliaws.FlowLogInfo, cfg *InitConfig, profileFlag string, noTUI bool, region string) (*InitConfig, error) {
	cfg.UseExisting = false
	cfg.Region = region
	cfg.ProfileName = defaultProfileName
	if profileFlag != "" {
		cfg.ProfileName = profileFlag
	}
	cfg.AggInterval = "600"
	cfg.RetentionDays = 30

	// Pre-fetch VPCs with spinner (needed by multiple groups)
	vpcOptions, err := fetchVPCOptions(ctx, ec2Client)
	if err != nil {
		return nil, err
	}

	// Cache for lazily-loaded subnet/ENI options (fetched when group becomes visible)
	var filterVpcID string
	optionCache := &resourceOptionCache{ctx: ctx, ec2Client: ec2Client}

	// If only one VPC, auto-select it for filtering
	singleVPC := len(vpcOptions) == 1
	if singleVPC {
		filterVpcID = vpcOptions[0].Value
		cfg.ResourceID = filterVpcID
	}

	// Build the single form with all groups — back-navigation works across all of them,
	// including the final confirmation step.
	form := huh.NewForm(
		buildProfileAndScopeGroup(cfg),
		buildVPCSelectGroup(cfg, vpcOptions, singleVPC),
		buildSubnetFilterGroup(cfg, vpcOptions, &filterVpcID, singleVPC),
		buildSubnetSelectGroup(cfg, &filterVpcID, optionCache),
		buildENIFilterGroup(cfg, vpcOptions, &filterVpcID, singleVPC),
		buildENISelectGroup(cfg, &filterVpcID, optionCache),
		buildTrafficAndFieldsGroup(cfg),
		buildCustomFieldsGroup(cfg),
		buildRetentionGroup(cfg),
		buildConfirmGroup(cfg, cfg.Region),
	).WithAccessible(noTUI)

	if err := form.Run(); err != nil {
		return nil, err
	}

	// Check for duplicate flow log after form completes
	if existing := fliaws.HasExistingFlowLog(flowLogs, cfg.ResourceID); existing != nil {
		return nil, fmt.Errorf("a flow log already exists on %s:\n  %s → %s (v%d, %s traffic)\n\nUse \"fli init\" and select \"Use an existing flow log\" to configure a profile for it",
			cfg.ResourceID, existing.FlowLogID, existing.LogGroupName, existing.Version, existing.TrafficType)
	}

	// Set version from field set
	cfg.Version = flowlog.PresetVersion(cfg.FieldSet)

	return cfg, nil
}

func buildProfileAndScopeGroup(cfg *InitConfig) *huh.Group {
	return huh.NewGroup(
		huh.NewNote().
			Title("New Flow Log").
			Description("Configure a new VPC flow log and save it as a named profile."),
		huh.NewInput().
			Title("Profile name").
			Value(&cfg.ProfileName).
			Validate(validateProfileName),
		huh.NewSelect[string]().
			Title("What would you like to monitor?").
			Options(
				huh.NewOption("Entire VPC (covers all ENIs automatically)", resourceTypeVPC),
				huh.NewOption("Specific subnet", resourceTypeSubnet),
				huh.NewOption("Specific network interface", resourceTypeNetworkInterface),
			).
			Value(&cfg.ResourceType),
	)
}

func buildVPCSelectGroup(cfg *InitConfig, vpcOptions []huh.Option[string], singleVPC bool) *huh.Group {
	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select a VPC").
			Options(vpcOptions...).
			Value(&cfg.ResourceID),
	).WithHideFunc(func() bool {
		return cfg.ResourceType != resourceTypeVPC || singleVPC
	})
}

func buildSubnetFilterGroup(cfg *InitConfig, vpcOptions []huh.Option[string], filterVpcID *string, singleVPC bool) *huh.Group {
	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select a VPC to filter subnets").
			Options(vpcOptions...).
			Value(filterVpcID),
	).WithHideFunc(func() bool {
		return cfg.ResourceType != resourceTypeSubnet || singleVPC
	})
}

func buildSubnetSelectGroup(cfg *InitConfig, filterVpcID *string, optionCache *resourceOptionCache) *huh.Group {
	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select a subnet").
			OptionsFunc(func() []huh.Option[string] {
				if *filterVpcID == "" {
					return nil
				}
				return optionCache.getSubnetOptions(*filterVpcID)
			}, filterVpcID).
			Value(&cfg.ResourceID),
	).WithHideFunc(func() bool {
		return cfg.ResourceType != resourceTypeSubnet
	})
}

func buildENIFilterGroup(cfg *InitConfig, vpcOptions []huh.Option[string], filterVpcID *string, singleVPC bool) *huh.Group {
	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select a VPC to filter interfaces").
			Options(vpcOptions...).
			Value(filterVpcID),
	).WithHideFunc(func() bool {
		return cfg.ResourceType != resourceTypeNetworkInterface || singleVPC
	})
}

func buildENISelectGroup(cfg *InitConfig, filterVpcID *string, optionCache *resourceOptionCache) *huh.Group {
	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select a network interface").
			OptionsFunc(func() []huh.Option[string] {
				if *filterVpcID == "" {
					return nil
				}
				return optionCache.getENIOptions(*filterVpcID)
			}, filterVpcID).
			Value(&cfg.ResourceID),
	).WithHideFunc(func() bool {
		return cfg.ResourceType != resourceTypeNetworkInterface
	})
}

func buildTrafficAndFieldsGroup(cfg *InitConfig) *huh.Group {
	return huh.NewGroup(
		huh.NewNote().
			TitleFunc(func() string {
				return fmt.Sprintf("Configure flow log for %s", cfg.ResourceID)
			}, &cfg.ResourceID),
		huh.NewSelect[string]().
			Title("Which traffic to capture?").
			Options(
				huh.NewOption("All traffic", "ALL"),
				huh.NewOption("Accepted only", "ACCEPT"),
				huh.NewOption("Rejected only", "REJECT"),
			).
			Value(&cfg.TrafficType),
		huh.NewSelect[string]().
			Title("Which field set?").
			Description("Determines which VPC Flow Log fields are captured").
			Options(
				huh.NewOption("Default (v2) — 14 fields: basic 5-tuple + bytes/packets/action", flowlog.PresetDefault),
				huh.NewOption("Security (v5) — adds tcp-flags, pkt-src/dst, flow-direction, aws-service", flowlog.PresetSecurity),
				huh.NewOption("Troubleshooting (v4) — adds vpc-id, subnet-id, instance-id, region, az-id", flowlog.PresetTroubleshooting),
				huh.NewOption("Full (v5) — all 29 fields (highest ingestion cost)", flowlog.PresetFull),
				huh.NewOption("Custom — pick individual fields", flowlog.PresetCustom),
			).
			Value(&cfg.FieldSet),
		huh.NewSelect[string]().
			Title("Aggregation interval").
			Options(
				huh.NewOption("1 minute (more granular, ~6x more records)", "60"),
				huh.NewOption("10 minutes (lower cost)", "600"),
			).
			Value(&cfg.AggInterval),
	)
}

func buildCustomFieldsGroup(cfg *InitConfig) *huh.Group {
	return huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Select fields").
			Description("Fields are grouped by version. Higher versions include more metadata.").
			Options(buildFieldOptions()...).
			Value(&cfg.CustomFields),
	).WithHideFunc(func() bool {
		return cfg.FieldSet != flowlog.PresetCustom
	})
}

func buildRetentionGroup(cfg *InitConfig) *huh.Group {
	return huh.NewGroup(
		huh.NewSelect[int]().
			Title("Log group retention").
			Options(
				huh.NewOption("7 days", 7),
				huh.NewOption("14 days", 14),
				huh.NewOption("30 days (default)", 30),
				huh.NewOption("90 days", 90),
				huh.NewOption("1 year", 365),
				huh.NewOption("Never expire", 0),
			).
			Value(&cfg.RetentionDays),
	).WithHideFunc(func() bool {
		// Auto-set log group name from resource ID
		if cfg.LogGroupName == "" && cfg.ResourceID != "" {
			cfg.LogGroupName = fliaws.FlowLogLogGroupName(cfg.ResourceID)
		}
		return false
	})
}

func buildConfirmGroup(cfg *InitConfig, region string) *huh.Group {
	return huh.NewGroup(
		huh.NewNote().
			Title("Review").
			DescriptionFunc(func() string {
				return formatSummary(cfg, region)
			}, cfg),
		huh.NewConfirm().
			Title("Create these resources?").
			Affirmative("Yes, create").
			Negative("Go back").
			Value(&cfg.Confirmed),
	)
}

func formatSummary(cfg *InitConfig, region string) string {
	roleName := fliaws.FlowLogRoleName(cfg.ResourceID)

	fieldSetLabel := cfg.FieldSet
	switch cfg.FieldSet {
	case flowlog.PresetDefault:
		fieldSetLabel = "Default (v2)"
	case flowlog.PresetSecurity:
		fieldSetLabel = "Security (v5)"
	case flowlog.PresetTroubleshooting:
		fieldSetLabel = "Troubleshooting (v4)"
	case flowlog.PresetFull:
		fieldSetLabel = "Full (v5)"
	case flowlog.PresetCustom:
		fieldSetLabel = fmt.Sprintf("Custom (%d fields)", len(cfg.CustomFields))
	}

	intervalLabel := "10 minutes"
	if cfg.AggInterval == "60" {
		intervalLabel = "1 minute"
	}

	var retentionLabel string
	switch cfg.RetentionDays {
	case 0:
		retentionLabel = "never expire"
	case 365:
		retentionLabel = "1 year"
	default:
		retentionLabel = fmt.Sprintf("%d days", cfg.RetentionDays)
	}

	return fmt.Sprintf(
		"Profile              %s\n"+
			"Region               %s\n"+
			"Resource             %s\n"+
			"CloudWatch Log Group %s  (%s retention)\n"+
			"IAM Role             %s\n"+
			"Flow Log             %s traffic | %s | %s interval\n\n"+
			"Tags                 managed-by=fli, fli-profile=%s",
		cfg.ProfileName, region, cfg.ResourceID,
		cfg.LogGroupName, retentionLabel, roleName,
		cfg.TrafficType, fieldSetLabel, intervalLabel,
		cfg.ProfileName,
	)
}

// resourceOptionCache lazily loads and caches subnet/ENI options by VPC ID.
type resourceOptionCache struct {
	ctx       context.Context
	ec2Client fliaws.FlowLogsAPI

	mu      sync.Mutex
	subnets map[string][]huh.Option[string]
	enis    map[string][]huh.Option[string]
}

func (c *resourceOptionCache) getSubnetOptions(vpcID string) []huh.Option[string] {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.subnets == nil {
		c.subnets = make(map[string][]huh.Option[string])
	}
	if opts, ok := c.subnets[vpcID]; ok {
		return opts
	}

	subnets, err := fliaws.ListSubnets(c.ctx, c.ec2Client, vpcID)
	if err != nil || len(subnets) == 0 {
		return nil
	}

	opts := make([]huh.Option[string], len(subnets))
	for i, subnet := range subnets {
		var label string
		if subnet.Name != "" {
			label = fmt.Sprintf("%s (%s)  %s  %s", subnet.ID, subnet.Name, subnet.CIDR, subnet.AZ)
		} else {
			label = fmt.Sprintf("%s  %s  %s", subnet.ID, subnet.CIDR, subnet.AZ)
		}
		opts[i] = huh.NewOption(label, subnet.ID)
	}
	c.subnets[vpcID] = opts
	return opts
}

func (c *resourceOptionCache) getENIOptions(vpcID string) []huh.Option[string] {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.enis == nil {
		c.enis = make(map[string][]huh.Option[string])
	}
	if opts, ok := c.enis[vpcID]; ok {
		return opts
	}

	enis, err := fliaws.ListENIs(c.ctx, c.ec2Client, vpcID)
	if err != nil || len(enis) == 0 {
		return nil
	}

	opts := make([]huh.Option[string], len(enis))
	for i, eni := range enis {
		instancePart := emDash
		if eni.InstanceID != "" {
			instancePart = eni.InstanceID
		}
		var label string
		if eni.Name != "" {
			label = fmt.Sprintf("%s (%s)  %s  %s", eni.ID, eni.Name, eni.PrivateIP, instancePart)
		} else {
			label = fmt.Sprintf("%s  %s  %s", eni.ID, eni.PrivateIP, instancePart)
		}
		opts[i] = huh.NewOption(label, eni.ID)
	}
	c.enis[vpcID] = opts
	return opts
}

// fetchVPCOptions loads VPCs with a spinner and returns huh options.
func fetchVPCOptions(ctx context.Context, ec2Client fliaws.FlowLogsAPI) ([]huh.Option[string], error) {
	var vpcs []fliaws.VPCInfo
	var fetchErr error

	err := spinner.New().
		Title("Loading VPCs...").
		Action(func() {
			vpcs, fetchErr = fliaws.ListVPCs(ctx, ec2Client)
		}).
		Run()
	if err != nil {
		return nil, err
	}
	if fetchErr != nil {
		return nil, fetchErr
	}
	if len(vpcs) == 0 {
		return nil, fmt.Errorf("no VPCs found in the region")
	}

	options := make([]huh.Option[string], len(vpcs))
	for i, vpc := range vpcs {
		label := vpc.ID
		if vpc.Name != "" {
			label = fmt.Sprintf("%s (%s)", vpc.ID, vpc.Name)
		}
		options[i] = huh.NewOption(label, vpc.ID)
	}
	return options, nil
}

func buildFieldOptions() []huh.Option[string] {
	type fieldEntry struct {
		name    string
		version int
	}

	entries := make([]fieldEntry, 0, len(flowlog.V2Fields)+len(flowlog.V3Fields)+len(flowlog.V4Fields)+len(flowlog.V5Fields))
	for _, f := range flowlog.V2Fields {
		entries = append(entries, fieldEntry{f, 2})
	}
	for _, f := range flowlog.V3Fields {
		entries = append(entries, fieldEntry{f, 3})
	}
	for _, f := range flowlog.V4Fields {
		entries = append(entries, fieldEntry{f, 4})
	}
	for _, f := range flowlog.V5Fields {
		entries = append(entries, fieldEntry{f, 5})
	}

	opts := make([]huh.Option[string], len(entries))
	for i, e := range entries {
		opts[i] = huh.NewOption(fmt.Sprintf("%s (v%d)", e.name, e.version), e.name)
	}
	return opts
}

func displayDiscoveryTable(flowLogs []fliaws.FlowLogInfo, region string) {
	if len(flowLogs) == 0 {
		return
	}

	headerStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Faint(true)

	rows := make([][]string, 0, len(flowLogs))
	for _, fl := range flowLogs {
		logGroup := fl.LogGroupName
		if logGroup == "" {
			logGroup = emDash
		}
		managed := emDash
		if fl.ManagedByFli {
			managed = checkMark
		}
		rows = append(rows, []string{
			fl.ResourceID,
			fl.DestType,
			logGroup,
			fmt.Sprintf("v%d", fl.Version),
			fl.Status,
			managed,
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		Headers("Resource", "Destination", "Log Group", "Format", "Status", "Managed").
		Rows(rows...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			if row >= 0 && row < len(flowLogs) && flowLogs[row].DestType != "cloud-watch-logs" {
				return dimStyle
			}
			return lipgloss.NewStyle()
		})

	fmt.Fprintf(os.Stderr, "\nDiscovered flow logs in %s:\n\n%s\n", region, t)
}

func validateProfileName(s string) error {
	if s == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if strings.Contains(s, " ") {
		return fmt.Errorf("profile name cannot contain spaces")
	}
	return nil
}
