package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
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
