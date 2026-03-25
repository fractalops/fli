package aws

import (
	"errors"
	"strings"

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

	// Fallback: string matching for EC2 and wrapped errors
	msg := err.Error()
	return strings.Contains(msg, "NotFound") ||
		strings.Contains(msg, "NoSuchEntity") ||
		strings.Contains(msg, "ResourceNotFoundException") ||
		strings.Contains(msg, "not found")
}
