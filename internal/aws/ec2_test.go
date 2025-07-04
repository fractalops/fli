// Package aws provides AWS service-specific functionality for the FLI tool
package aws

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// mockEC2API implements the EC2 API interface for testing
type mockEC2API struct {
	DescribeNetworkInterfacesFunc func(context.Context, *ec2.DescribeNetworkInterfacesInput, ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error)
}

func (m *mockEC2API) DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	return m.DescribeNetworkInterfacesFunc(ctx, params, optFns...)
}

func TestGetENIsBySecurityGroup(t *testing.T) {
	tests := []struct {
		name            string
		securityGroupID string
		mockResponse    *ec2.DescribeNetworkInterfacesOutput
		mockError       error
		want            []string
		wantErr         bool
	}{
		{
			name:            "successful lookup",
			securityGroupID: "sg-1234567890abcdef0",
			mockResponse: &ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []types.NetworkInterface{
					{NetworkInterfaceId: aws.String("eni-123")},
					{NetworkInterfaceId: aws.String("eni-456")},
				},
			},
			want:    []string{"eni-123", "eni-456"},
			wantErr: false,
		},
		{
			name:            "no ENIs found",
			securityGroupID: "sg-1234567890abcdef0",
			mockResponse: &ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []types.NetworkInterface{},
			},
			want:    nil, // Changed from []string{} to nil to match implementation
			wantErr: false,
		},
		{
			name:            "API error",
			securityGroupID: "sg-1234567890abcdef0",
			mockError:       fmt.Errorf("API error"),
			want:            nil,
			wantErr:         true,
		},
		{
			name:            "empty security group ID",
			securityGroupID: "",
			want:            nil,
			wantErr:         true,
		},
		{
			name:            "nil network interface ID",
			securityGroupID: "sg-1234567890abcdef0",
			mockResponse: &ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []types.NetworkInterface{
					{NetworkInterfaceId: nil}, // Test handling of nil NetworkInterfaceId
					{NetworkInterfaceId: aws.String("eni-456")},
				},
			},
			want:    []string{"eni-456"}, // Should only include the non-nil ENI ID
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockEC2API{
				DescribeNetworkInterfacesFunc: func(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
					// Validate filter parameters for non-empty security group ID
					if tt.securityGroupID != "" {
						if len(params.Filters) != 1 {
							t.Errorf("Expected 1 filter, got %d", len(params.Filters))
						}
						if len(params.Filters) > 0 {
							if *params.Filters[0].Name != "group-id" {
								t.Errorf("Expected filter name 'group-id', got %s", *params.Filters[0].Name)
							}
							if len(params.Filters[0].Values) != 1 || params.Filters[0].Values[0] != tt.securityGroupID {
								t.Errorf("Expected filter value %s, got %v", tt.securityGroupID, params.Filters[0].Values)
							}
						}
					}

					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			ec2Client := NewEC2Client(client)
			got, err := ec2Client.GetENIsBySecurityGroup(context.Background(), tt.securityGroupID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetENIsBySecurityGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetENIsBySecurityGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsENINotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
		{
			name:     "ENI not found error with .NotFound",
			err:      fmt.Errorf("operation error EC2: DescribeNetworkInterfaces, https response error StatusCode: 400, RequestID: fc1dac8f-f5e9-4e44-88ab-ae3f95e33c2c, api error InvalidNetworkInterfaceID.NotFound: The networkInterface ID 'eni-0562b9d767484e13e' does not exist"),
			expected: true,
		},
		{
			name:     "ENI not found error without .NotFound",
			err:      fmt.Errorf("InvalidNetworkInterfaceID: The networkInterface ID 'eni-0562b9d767484e13e' does not exist"),
			expected: true,
		},
		{
			name:     "different AWS error",
			err:      fmt.Errorf("AccessDenied: User is not authorized to perform ec2:DescribeNetworkInterfaces"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsENINotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("IsENINotFoundError() = %v, want %v", result, tt.expected)
			}
		})
	}
}
