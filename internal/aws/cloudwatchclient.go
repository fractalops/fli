package aws

import (
	"context"
	"fmt"
	"math"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

// CloudWatchLogsManagementAPI defines the CloudWatch Logs methods needed for log group management.
// This is separate from the existing runner.CloudWatchLogsClient which handles query operations.
type CloudWatchLogsManagementAPI interface {
	CreateLogGroup(context.Context, *cloudwatchlogs.CreateLogGroupInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.CreateLogGroupOutput, error)
	PutRetentionPolicy(context.Context, *cloudwatchlogs.PutRetentionPolicyInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutRetentionPolicyOutput, error)
	DeleteLogGroup(context.Context, *cloudwatchlogs.DeleteLogGroupInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DeleteLogGroupOutput, error)
	DescribeLogGroups(context.Context, *cloudwatchlogs.DescribeLogGroupsInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error)
}

// CreateLogGroupWithRetention creates a CloudWatch Logs log group with a retention policy.
// retentionDays of 0 means never expire.
// Returns the log group ARN.
func CreateLogGroupWithRetention(ctx context.Context, client CloudWatchLogsManagementAPI, name string, retentionDays int, profileName string) (string, error) {
	_, err := client.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: awssdk.String(name),
		Tags: map[string]string{
			"managed-by":  "fli",
			"fli-profile": profileName,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create log group: %w", err)
	}

	if retentionDays > 0 {
		if retentionDays > math.MaxInt32 {
			return "", fmt.Errorf("retentionDays %d exceeds max int32 value", retentionDays)
		}
		_, err = client.PutRetentionPolicy(ctx, &cloudwatchlogs.PutRetentionPolicyInput{
			LogGroupName:    awssdk.String(name),
			RetentionInDays: awssdk.Int32(int32(retentionDays)),
		})
		if err != nil {
			return "", fmt.Errorf("failed to set retention policy: %w", err)
		}
	}

	// Get the ARN by describing the log group
	arn, err := getLogGroupARN(ctx, client, name)
	if err != nil {
		return "", err
	}

	return arn, nil
}

// DeleteLogGroup deletes a CloudWatch Logs log group.
func DeleteLogGroup(ctx context.Context, client CloudWatchLogsManagementAPI, name string) error {
	_, err := client.DeleteLogGroup(ctx, &cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: awssdk.String(name),
	})
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete log group: %w", err)
	}
	return nil
}

// LogGroupExists checks if a CloudWatch Logs log group exists.
func LogGroupExists(ctx context.Context, client CloudWatchLogsManagementAPI, name string) (bool, error) {
	resp, err := client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: awssdk.String(name),
	})
	if err != nil {
		return false, fmt.Errorf("failed to describe log groups: %w", err)
	}

	for _, lg := range resp.LogGroups {
		if awssdk.ToString(lg.LogGroupName) == name {
			return true, nil
		}
	}
	return false, nil
}

// FlowLogLogGroupName generates the default log group name for a resource.
func FlowLogLogGroupName(resourceID string) string {
	return fmt.Sprintf("/fli/flow-logs/%s", resourceID)
}

// ErrLogGroupNotYetCreated is returned when the log group doesn't exist yet
// because AWS hasn't auto-created it (no traffic has been recorded).
var ErrLogGroupNotYetCreated = fmt.Errorf("log group not yet created by AWS")

// ConfigureLogGroup sets retention and tags on an existing log group (e.g. one auto-created by the flow log service).
// Returns ErrLogGroupNotYetCreated if the log group doesn't exist yet.
func ConfigureLogGroup(ctx context.Context, client CloudWatchLogsManagementAPI, name string, retentionDays int, _ string) error {
	// Check if the log group exists first
	exists, err := LogGroupExists(ctx, client, name)
	if err != nil {
		return err
	}
	if !exists {
		return ErrLogGroupNotYetCreated
	}

	if retentionDays > 0 {
		if retentionDays > math.MaxInt32 {
			return fmt.Errorf("retentionDays %d exceeds max int32 value", retentionDays)
		}
		_, err := client.PutRetentionPolicy(ctx, &cloudwatchlogs.PutRetentionPolicyInput{
			LogGroupName:    awssdk.String(name),
			RetentionInDays: awssdk.Int32(int32(retentionDays)),
		})
		if err != nil {
			return fmt.Errorf("failed to set retention policy: %w", err)
		}
	}

	return nil
}

// GetLogGroupARN returns the ARN of a log group by name.
func GetLogGroupARN(ctx context.Context, client CloudWatchLogsManagementAPI, name string) (string, error) {
	return getLogGroupARN(ctx, client, name)
}

func getLogGroupARN(ctx context.Context, client CloudWatchLogsManagementAPI, name string) (string, error) {
	resp, err := client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: awssdk.String(name),
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe log groups: %w", err)
	}

	for _, lg := range resp.LogGroups {
		if awssdk.ToString(lg.LogGroupName) == name {
			return awssdk.ToString(lg.Arn), nil
		}
	}
	return "", fmt.Errorf("log group %q not found after creation", name)
}
