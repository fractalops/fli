package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// STSAPI defines the STS methods needed for identity verification.
type STSAPI interface {
	GetCallerIdentity(context.Context, *sts.GetCallerIdentityInput, ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

// CallerIdentity holds the resolved AWS caller identity.
type CallerIdentity struct {
	AccountID string
	ARN       string
	UserID    string
}

// GetCallerIdentity returns the account ID and ARN of the current caller.
func GetCallerIdentity(ctx context.Context, client STSAPI) (CallerIdentity, error) {
	resp, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return CallerIdentity{}, fmt.Errorf("failed to get caller identity: %w", err)
	}
	return CallerIdentity{
		AccountID: aws.ToString(resp.Account),
		ARN:       aws.ToString(resp.Arn),
		UserID:    aws.ToString(resp.UserId),
	}, nil
}

// InitActions are the IAM actions required to run fli init.
var InitActions = []string{
	"iam:CreateRole",
	"iam:PutRolePolicy",
	"iam:GetRole",
	"ec2:CreateFlowLogs",
	"ec2:DescribeFlowLogs",
	"ec2:DescribeVpcs",
	"ec2:DescribeSubnets",
	"ec2:DescribeNetworkInterfaces",
	"logs:DescribeLogGroups",
	"logs:PutRetentionPolicy",
	"logs:TagResource",
}

// CleanupActions are the IAM actions required to run fli cleanup.
var CleanupActions = []string{
	"ec2:DeleteFlowLogs",
	"iam:DeleteRole",
	"iam:DeleteRolePolicy",
	"logs:DeleteLogGroup",
}

// QueryActions are the IAM actions required to run fli queries.
var QueryActions = []string{
	"logs:StartQuery",
	"logs:GetQueryResults",
	"logs:DescribeLogGroups",
}

// PermissionResult describes whether a specific IAM action is allowed or denied.
type PermissionResult struct {
	Action  string
	Allowed bool
}

// CheckPermissions uses SimulatePrincipalPolicy to verify whether the caller
// has the required IAM permissions.
func CheckPermissions(ctx context.Context, iamClient IAMAPI, callerARN string, actions []string) ([]PermissionResult, error) {
	input := &iam.SimulatePrincipalPolicyInput{
		PolicySourceArn: aws.String(callerARN),
		ActionNames:     actions,
	}

	resp, err := iamClient.SimulatePrincipalPolicy(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to simulate IAM policy: %w", err)
	}

	results := make([]PermissionResult, len(resp.EvaluationResults))
	for i, eval := range resp.EvaluationResults {
		results[i] = PermissionResult{
			Action:  aws.ToString(eval.EvalActionName),
			Allowed: string(eval.EvalDecision) == "allowed",
		}
	}
	return results, nil
}
