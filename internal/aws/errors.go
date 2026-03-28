package aws

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/smithy-go"

	cwlTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// IsNotFound returns true if the error indicates a resource was not found.
// It uses SDK type assertions where available and falls back to string matching
// for services (like EC2) that don't export typed error types.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	// IAM: NoSuchEntityException
	var iamNotFound *iamTypes.NoSuchEntityException
	if errors.As(err, &iamNotFound) {
		return true
	}

	// CloudWatch Logs: ResourceNotFoundException
	var cwlNotFound *cwlTypes.ResourceNotFoundException
	if errors.As(err, &cwlNotFound) {
		return true
	}

	// Fallback: check smithy API error codes for services (like EC2)
	// that don't export typed error types.
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return strings.Contains(code, "NotFound") ||
			code == "NoSuchEntity"
	}

	return false
}

// IsAccessDenied returns true if the error is an AWS permissions error.
func IsAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == "AccessDeniedException" ||
			code == "UnauthorizedAccess" ||
			code == "AccessDenied"
	}
	return false
}

// permissionHints maps AWS API action patterns to the IAM permissions needed.
var permissionHints = map[string]string{
	"CreateRole":              "iam:CreateRole",
	"PutRolePolicy":           "iam:PutRolePolicy",
	"GetRole":                 "iam:GetRole",
	"DeleteRole":              "iam:DeleteRole",
	"DeleteRolePolicy":        "iam:DeleteRolePolicy",
	"CreateFlowLogs":          "ec2:CreateFlowLogs",
	"DescribeFlowLogs":        "ec2:DescribeFlowLogs",
	"DeleteFlowLogs":          "ec2:DeleteFlowLogs",
	"DescribeVpcs":            "ec2:DescribeVpcs",
	"CreateLogGroup":          "logs:CreateLogGroup",
	"DeleteLogGroup":          "logs:DeleteLogGroup",
	"PutRetentionPolicy":      "logs:PutRetentionPolicy",
	"DescribeLogGroups":       "logs:DescribeLogGroups",
	"StartQuery":              "logs:StartQuery",
	"GetQueryResults":         "logs:GetQueryResults",
	"SimulatePrincipalPolicy": "iam:SimulatePrincipalPolicy",
}

// WrapError wraps an AWS error with a user-friendly hint if it's an access denied error.
// The context parameter describes what was being attempted (e.g. "create IAM role").
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	if !IsAccessDenied(err) {
		return err
	}

	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return err
	}

	// Try to find the specific permission needed
	code := apiErr.ErrorCode()
	msg := apiErr.ErrorMessage()

	// Look for the API action in the error message or operation name
	var hint string
	for action, permission := range permissionHints {
		if strings.Contains(msg, action) || strings.Contains(err.Error(), action) {
			hint = permission
			break
		}
	}

	if hint != "" {
		return fmt.Errorf("%s: permission denied\n\n  Your IAM principal needs: %s\n  Error: %s: %s\n\n  Run \"fli init --check-permissions\" to see all required permissions: %w", context, hint, code, msg, err)
	}

	return fmt.Errorf("%s: permission denied\n\n  Error: %s: %s\n\n  Run \"fli init --check-permissions\" to see all required permissions: %w", context, code, msg, err)
}
