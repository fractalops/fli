package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"fli/internal/flowlog"
)

// FlowLogsAPI defines the EC2 methods needed for flow log operations.
type FlowLogsAPI interface {
	DescribeFlowLogs(context.Context, *ec2.DescribeFlowLogsInput, ...func(*ec2.Options)) (*ec2.DescribeFlowLogsOutput, error)
	CreateFlowLogs(context.Context, *ec2.CreateFlowLogsInput, ...func(*ec2.Options)) (*ec2.CreateFlowLogsOutput, error)
	DeleteFlowLogs(context.Context, *ec2.DeleteFlowLogsInput, ...func(*ec2.Options)) (*ec2.DeleteFlowLogsOutput, error)
	CreateTags(context.Context, *ec2.CreateTagsInput, ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
	DescribeVpcs(context.Context, *ec2.DescribeVpcsInput, ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DescribeSubnets(context.Context, *ec2.DescribeSubnetsInput, ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	DescribeNetworkInterfaces(context.Context, *ec2.DescribeNetworkInterfacesInput, ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error)
}

// FlowLogInfo holds parsed information about a discovered flow log.
type FlowLogInfo struct {
	FlowLogID    string
	ResourceID   string
	ResourceType string
	DestType     string
	LogGroupName string
	LogFormat    string
	Version      int
	Status       string
	TrafficType  string
	AggInterval  int32
	ManagedByFli bool
	ProfileName  string
}

// VPCInfo holds information about a VPC.
type VPCInfo struct {
	ID   string
	Name string
}

// SubnetInfo holds information about a subnet.
type SubnetInfo struct {
	ID   string
	Name string
	CIDR string
	AZ   string
}

// ENIInfo holds information about a network interface.
type ENIInfo struct {
	ID          string
	Name        string
	Description string
	PrivateIP   string
	InstanceID  string
}

// DiscoverFlowLogs retrieves all flow logs in the region using pagination.
func DiscoverFlowLogs(ctx context.Context, client FlowLogsAPI) ([]FlowLogInfo, error) {
	var allFlowLogs []FlowLogInfo
	paginator := ec2.NewDescribeFlowLogsPaginator(client, &ec2.DescribeFlowLogsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe flow logs: %w", err)
		}

		for _, fl := range page.FlowLogs {
			info := FlowLogInfo{
				FlowLogID:    aws.ToString(fl.FlowLogId),
				ResourceID:   aws.ToString(fl.ResourceId),
				LogGroupName: aws.ToString(fl.LogGroupName),
				LogFormat:    aws.ToString(fl.LogFormat),
				Status:       aws.ToString(fl.FlowLogStatus),
				TrafficType:  string(fl.TrafficType),
				AggInterval:  aws.ToInt32(fl.MaxAggregationInterval),
			}

			info.DestType = string(fl.LogDestinationType)
			info.Version = flowlog.DetectVersion(info.LogFormat)

			// Check tags for managed-by
			for _, tag := range fl.Tags {
				if aws.ToString(tag.Key) == "managed-by" && aws.ToString(tag.Value) == "fli" {
					info.ManagedByFli = true
				}
				if aws.ToString(tag.Key) == "fli-profile" {
					info.ProfileName = aws.ToString(tag.Value)
				}
			}

			allFlowLogs = append(allFlowLogs, info)
		}
	}

	return allFlowLogs, nil
}

// HasExistingFlowLog checks if a CloudWatch Logs flow log already exists on the given resource.
func HasExistingFlowLog(flowLogs []FlowLogInfo, resourceID string) *FlowLogInfo {
	for _, fl := range flowLogs {
		if fl.ResourceID == resourceID && fl.DestType == "cloud-watch-logs" {
			return &fl
		}
	}
	return nil
}

// ListVPCs retrieves all VPCs in the region using pagination.
func ListVPCs(ctx context.Context, client FlowLogsAPI) ([]VPCInfo, error) {
	var vpcs []VPCInfo
	paginator := ec2.NewDescribeVpcsPaginator(client, &ec2.DescribeVpcsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe VPCs: %w", err)
		}

		for _, vpc := range page.Vpcs {
			info := VPCInfo{
				ID:   aws.ToString(vpc.VpcId),
				Name: getNameTag(vpc.Tags),
			}
			vpcs = append(vpcs, info)
		}
	}

	return vpcs, nil
}

// ListSubnets retrieves all subnets for a VPC using pagination.
func ListSubnets(ctx context.Context, client FlowLogsAPI, vpcID string) ([]SubnetInfo, error) {
	var subnets []SubnetInfo
	input := &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
	}
	paginator := ec2.NewDescribeSubnetsPaginator(client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe subnets: %w", err)
		}

		for _, subnet := range page.Subnets {
			info := SubnetInfo{
				ID:   aws.ToString(subnet.SubnetId),
				Name: getNameTag(subnet.Tags),
				CIDR: aws.ToString(subnet.CidrBlock),
				AZ:   aws.ToString(subnet.AvailabilityZone),
			}
			subnets = append(subnets, info)
		}
	}

	return subnets, nil
}

// ListENIs retrieves all network interfaces for a VPC using pagination.
func ListENIs(ctx context.Context, client FlowLogsAPI, vpcID string) ([]ENIInfo, error) {
	var enis []ENIInfo
	input := &ec2.DescribeNetworkInterfacesInput{
		Filters: []types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
	}
	paginator := ec2.NewDescribeNetworkInterfacesPaginator(client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe network interfaces: %w", err)
		}

		for _, eni := range page.NetworkInterfaces {
			info := ENIInfo{
				ID:          aws.ToString(eni.NetworkInterfaceId),
				Name:        getNameTag(eni.TagSet),
				Description: aws.ToString(eni.Description),
				PrivateIP:   aws.ToString(eni.PrivateIpAddress),
			}
			if eni.Attachment != nil {
				info.InstanceID = aws.ToString(eni.Attachment.InstanceId)
			}
			// Use description as fallback for name
			if info.Name == "" {
				info.Name = info.Description
			}
			enis = append(enis, info)
		}
	}

	return enis, nil
}

// CreateFlowLogInput holds the parameters for creating a flow log.
type CreateFlowLogInput struct {
	ResourceID   string
	ResourceType types.FlowLogsResourceType
	TrafficType  types.TrafficType
	LogGroupName string
	IAMRoleARN   string
	LogFormat    string
	AggInterval  int32
	ProfileName  string
}

// CreateFlowLog creates a VPC flow log and returns the flow log ID.
func CreateFlowLog(ctx context.Context, client FlowLogsAPI, input CreateFlowLogInput) (string, error) {
	createInput := &ec2.CreateFlowLogsInput{
		ResourceIds:              []string{input.ResourceID},
		ResourceType:             input.ResourceType,
		TrafficType:              input.TrafficType,
		LogDestinationType:       types.LogDestinationTypeCloudWatchLogs,
		LogGroupName:             aws.String(input.LogGroupName),
		DeliverLogsPermissionArn: aws.String(input.IAMRoleARN),
		MaxAggregationInterval:   aws.Int32(input.AggInterval),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpcFlowLog,
				Tags: []types.Tag{
					{Key: aws.String("managed-by"), Value: aws.String("fli")},
					{Key: aws.String("fli-profile"), Value: aws.String(input.ProfileName)},
				},
			},
		},
	}

	if input.LogFormat != "" {
		createInput.LogFormat = aws.String(input.LogFormat)
	}

	resp, err := client.CreateFlowLogs(ctx, createInput)
	if err != nil {
		return "", fmt.Errorf("failed to create flow log: %w", err)
	}

	if len(resp.Unsuccessful) > 0 {
		msg := aws.ToString(resp.Unsuccessful[0].Error.Message)
		return "", fmt.Errorf("failed to create flow log: %s", msg)
	}

	if len(resp.FlowLogIds) == 0 {
		return "", fmt.Errorf("no flow log ID returned")
	}

	return resp.FlowLogIds[0], nil
}

// DeleteFlowLog deletes a VPC flow log by ID.
func DeleteFlowLog(ctx context.Context, client FlowLogsAPI, flowLogID string) error {
	resp, err := client.DeleteFlowLogs(ctx, &ec2.DeleteFlowLogsInput{
		FlowLogIds: []string{flowLogID},
	})
	if err != nil {
		return fmt.Errorf("failed to delete flow log: %w", err)
	}

	if len(resp.Unsuccessful) > 0 {
		msg := aws.ToString(resp.Unsuccessful[0].Error.Message)
		return fmt.Errorf("failed to delete flow log: %s", msg)
	}

	return nil
}

// TagResources adds tags to the given EC2 resources.
func TagResources(ctx context.Context, client FlowLogsAPI, resourceIDs []string, tags map[string]string) error {
	ec2Tags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		ec2Tags = append(ec2Tags, types.Tag{Key: aws.String(k), Value: aws.String(v)})
	}

	_, err := client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: resourceIDs,
		Tags:      ec2Tags,
	})
	if err != nil {
		return fmt.Errorf("failed to tag resources: %w", err)
	}

	return nil
}

// getNameTag extracts the "Name" tag value from a list of tags.
func getNameTag(tags []types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}
