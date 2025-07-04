// Package aws provides AWS service-specific functionality for the FLI tool.
// It includes helpers for interacting with EC2, CloudWatch Logs, and other AWS services.
package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// EC2API defines the interface for the EC2 client, allowing for mock implementations.
type EC2API interface {
	DescribeNetworkInterfaces(context.Context, *ec2.DescribeNetworkInterfacesInput, ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error)
}

// ENITag represents ENI information returned by AWS.
type ENITag struct {
	ENI        string
	Label      string
	SGNames    []string
	PrivateIPs []string
}

// EC2Client is a client for EC2 operations.
type EC2Client struct {
	client EC2API
}

// NewEC2Client creates a new EC2Client.
func NewEC2Client(client EC2API) *EC2Client {
	return &EC2Client{client: client}
}

// DescribeNetworkInterfaces is a wrapper for the AWS SDK's DescribeNetworkInterfaces.
func (c *EC2Client) DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	result, err := c.client.DescribeNetworkInterfaces(ctx, params, optFns...)
	if err != nil {
		return nil, fmt.Errorf("failed to describe network interfaces: %w", err)
	}
	return result, nil
}

// GetENIsBySecurityGroup returns a list of ENI IDs associated with the given security group.
// Parameters:
// - ctx: Context for the API call.
// - client: EC2 client implementation.
// - sgID: Security Group ID to query.
// Returns:
// - A slice of ENI IDs associated with the security group.
// - Any error that occurred during the API call.
func (c *EC2Client) GetENIsBySecurityGroup(ctx context.Context, sgID string) ([]string, error) {
	if sgID == "" {
		return nil, fmt.Errorf("security group ID cannot be empty")
	}

	// Create filter for security group.
	filters := []types.Filter{
		{
			Name:   stringPtr("group-id"),
			Values: []string{sgID},
		},
	}

	// Call EC2 API to get network interfaces.
	resp, err := c.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe network interfaces: %w", err)
	}

	// Extract ENI IDs.
	var eniIDs []string
	for _, eni := range resp.NetworkInterfaces {
		if eni.NetworkInterfaceId != nil {
			eniIDs = append(eniIDs, *eni.NetworkInterfaceId)
		}
	}

	return eniIDs, nil
}

// GetENITag fetches the security group names and returns an ENITag for the given ENI ID.
func (c *EC2Client) GetENITag(ctx context.Context, eniID string) (ENITag, error) {
	if eniID == "" {
		return ENITag{}, fmt.Errorf("ENI ID cannot be empty")
	}
	resp, err := c.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []string{eniID},
	})
	if err != nil {
		return ENITag{}, fmt.Errorf("failed to describe network interface: %w", err)
	}
	if len(resp.NetworkInterfaces) == 0 {
		return ENITag{}, fmt.Errorf("ENI not found: %s", eniID)
	}
	ni := resp.NetworkInterfaces[0]
	var sgNames []string
	label := "unknown"
	for i, sg := range ni.Groups {
		if sg.GroupName != nil {
			sgNames = append(sgNames, *sg.GroupName)
			if i == 0 {
				label = *sg.GroupName // Use first SG name as label.
			}
		}
	}
	var privateIPs []string
	for _, ip := range ni.PrivateIpAddresses {
		if ip.PrivateIpAddress != nil {
			privateIPs = append(privateIPs, *ip.PrivateIpAddress)
		}
	}
	return ENITag{
		ENI:        eniID,
		Label:      label,
		SGNames:    sgNames,
		PrivateIPs: privateIPs,
	}, nil
}

// Helper function to get a pointer to a string.
func stringPtr(s string) *string {
	return &s
}

// IsENINotFoundError checks if the error indicates that an ENI was not found.
// Check for AWS SDK v2 error types.
// Also check for the operation error structure.
func IsENINotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check for AWS SDK v2 error types
	if ok := strings.Contains(err.Error(), "InvalidNetworkInterfaceID.NotFound"); ok {
		return true
	}

	// Also check for the operation error structure
	if ok := strings.Contains(err.Error(), "InvalidNetworkInterfaceID"); ok {
		return true
	}

	return false
}
