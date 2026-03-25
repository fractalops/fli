package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// IAMAPI defines the IAM methods needed for flow log role management.
type IAMAPI interface {
	CreateRole(context.Context, *iam.CreateRoleInput, ...func(*iam.Options)) (*iam.CreateRoleOutput, error)
	PutRolePolicy(context.Context, *iam.PutRolePolicyInput, ...func(*iam.Options)) (*iam.PutRolePolicyOutput, error)
	DeleteRole(context.Context, *iam.DeleteRoleInput, ...func(*iam.Options)) (*iam.DeleteRoleOutput, error)
	DeleteRolePolicy(context.Context, *iam.DeleteRolePolicyInput, ...func(*iam.Options)) (*iam.DeleteRolePolicyOutput, error)
	GetRole(context.Context, *iam.GetRoleInput, ...func(*iam.Options)) (*iam.GetRoleOutput, error)
	SimulatePrincipalPolicy(context.Context, *iam.SimulatePrincipalPolicyInput, ...func(*iam.Options)) (*iam.SimulatePrincipalPolicyOutput, error)
}

const (
	// FlowLogPolicyName is the name of the IAM policy attached to flow log roles.
	FlowLogPolicyName = "fli-flow-logs-publish"
)

// trustPolicy builds the IAM trust policy for VPC flow logs.
func trustPolicy(accountID string) (string, error) {
	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect":    "Allow",
				"Principal": map[string]string{"Service": "vpc-flow-logs.amazonaws.com"},
				"Action":    "sts:AssumeRole",
				"Condition": map[string]interface{}{
					"StringEquals": map[string]string{
						"aws:SourceAccount": accountID,
					},
				},
			},
		},
	}
	data, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal trust policy: %w", err)
	}
	return string(data), nil
}

// permissionPolicy builds the IAM permission policy for publishing flow logs to CloudWatch.
func permissionPolicy() (string, error) {
	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Action": []string{
					"logs:CreateLogGroup",
					"logs:CreateLogStream",
					"logs:PutLogEvents",
					"logs:DescribeLogGroups",
					"logs:DescribeLogStreams",
				},
				"Resource": "*",
			},
		},
	}
	data, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal permission policy: %w", err)
	}
	return string(data), nil
}

// CreateFlowLogRole creates the IAM role needed for flow log publishing.
// Returns the role ARN.
func CreateFlowLogRole(ctx context.Context, client IAMAPI, roleName, accountID string) (string, error) {
	trustDoc, err := trustPolicy(accountID)
	if err != nil {
		return "", err
	}

	createResp, err := client.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(trustDoc),
		Description:              aws.String("IAM role for VPC flow log publishing, managed by fli"),
		Tags: []iamTypes.Tag{
			{Key: aws.String("managed-by"), Value: aws.String("fli")},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create IAM role: %w", err)
	}

	permDoc, err := permissionPolicy()
	if err != nil {
		return "", err
	}

	_, err = client.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyName:     aws.String(FlowLogPolicyName),
		PolicyDocument: aws.String(permDoc),
	})
	if err != nil {
		return "", fmt.Errorf("failed to attach policy to role: %w", err)
	}

	return aws.ToString(createResp.Role.Arn), nil
}

// DeleteFlowLogRole deletes the IAM role and its inline policy.
func DeleteFlowLogRole(ctx context.Context, client IAMAPI, roleName, policyName string) error {
	// Delete the inline policy first
	_, err := client.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
		RoleName:   aws.String(roleName),
		PolicyName: aws.String(policyName),
	})
	if err != nil && !IsNotFound(err) {
		return fmt.Errorf("failed to delete role policy: %w", err)
	}

	// Delete the role
	_, err = client.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil && !IsNotFound(err) {
		return fmt.Errorf("failed to delete IAM role: %w", err)
	}

	return nil
}

// RoleExists checks if an IAM role exists.
func RoleExists(ctx context.Context, client IAMAPI, roleName string) (bool, error) {
	_, err := client.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get IAM role: %w", err)
	}
	return true, nil
}

// FlowLogRoleName generates the IAM role name for a given resource ID.
func FlowLogRoleName(resourceID string) string {
	return fmt.Sprintf("fli-flow-logs-%s", resourceID)
}
