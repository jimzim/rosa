package aws

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Client provides AWS operations
type Client interface {
	GetCallerIdentity(ctx context.Context) (*CallerIdentity, error)
	AssumeRole(ctx context.Context, roleARN string, sessionName string) (*Credentials, error)
	ValidateRoles(ctx context.Context, roles RoleSet) error
	DescribeSubnets(ctx context.Context, subnetIDs []string) ([]ec2Types.Subnet, error)
	VPCHasInternetGateway(ctx context.Context, vpcID string) (bool, error)
}

// Config holds AWS client configuration
type Config struct {
	Region  string
	Profile string
	RoleARN string
}

// CallerIdentity represents AWS caller identity
type CallerIdentity struct {
	AccountID string
	UserID    string
	ARN       string
}

// Credentials represents temporary AWS credentials
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// RoleSet represents the HCP-specific roles
type RoleSet struct {
	InstallerRoleARN string
	SupportRoleARN   string
	WorkerRoleARN    string
}

type awsClient struct {
	cfg       aws.Config
	stsClient *sts.Client
	iamClient *iam.Client
	ec2Client *ec2.Client
}

// NewClient creates a new AWS client with string parameters
func NewClient(ctx context.Context, region string, profile string, logger *slog.Logger) (Client, error) {
	cfg := Config{
		Region:  region,
		Profile: profile,
	}
	return NewClientWithConfig(ctx, cfg)
}

// NewClientWithConfig creates a new AWS client from a Config struct
func NewClientWithConfig(ctx context.Context, cfg Config) (Client, error) {
	// Build config options
	var opts []func(*config.LoadOptions) error

	if cfg.Region != "" {
		opts = append(opts, config.WithRegion(cfg.Region))
	}

	if cfg.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(cfg.Profile))
	}

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// If role ARN is provided, assume it
	if cfg.RoleARN != "" {
		stsClient := sts.NewFromConfig(awsCfg)

		result, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         aws.String(cfg.RoleARN),
			RoleSessionName: aws.String("rosa-hcp-cli"),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to assume role %s: %w", cfg.RoleARN, err)
		}

		// Update config with assumed role credentials
		awsCfg.Credentials = credentials.NewStaticCredentialsProvider(
			*result.Credentials.AccessKeyId,
			*result.Credentials.SecretAccessKey,
			*result.Credentials.SessionToken,
		)
	}

	return &awsClient{
		cfg:       awsCfg,
		stsClient: sts.NewFromConfig(awsCfg),
		iamClient: iam.NewFromConfig(awsCfg),
		ec2Client: ec2.NewFromConfig(awsCfg),
	}, nil
}

// GetCallerIdentity returns the AWS caller identity
func (c *awsClient) GetCallerIdentity(ctx context.Context) (*CallerIdentity, error) {
	result, err := c.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get caller identity: %w", err)
	}

	return &CallerIdentity{
		AccountID: *result.Account,
		UserID:    *result.UserId,
		ARN:       *result.Arn,
	}, nil
}

// AssumeRole assumes an AWS IAM role
func (c *awsClient) AssumeRole(ctx context.Context, roleARN string, sessionName string) (*Credentials, error) {
	result, err := c.stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleARN),
		RoleSessionName: aws.String(sessionName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to assume role %s: %w", roleARN, err)
	}

	return &Credentials{
		AccessKeyID:     *result.Credentials.AccessKeyId,
		SecretAccessKey: *result.Credentials.SecretAccessKey,
		SessionToken:    *result.Credentials.SessionToken,
	}, nil
}

// ValidateRoles validates that the HCP roles exist and are accessible
func (c *awsClient) ValidateRoles(ctx context.Context, roles RoleSet) error {
	// Validate installer role
	if err := c.validateRole(ctx, roles.InstallerRoleARN, "installer"); err != nil {
		return err
	}

	// Validate support role
	if err := c.validateRole(ctx, roles.SupportRoleARN, "support"); err != nil {
		return err
	}

	// Validate worker role
	if err := c.validateRole(ctx, roles.WorkerRoleARN, "worker"); err != nil {
		return err
	}

	return nil
}

func (c *awsClient) validateRole(ctx context.Context, roleARN string, roleType string) error {
	if roleARN == "" {
		return fmt.Errorf("%s role ARN is required", roleType)
	}

	// Parse role name from ARN
	// Format: arn:aws:iam::123456789012:role/role-name
	var roleName string
	if _, err := fmt.Sscanf(roleARN, "arn:aws:iam::%*[^:]]:role/%s", &roleName); err != nil {
		return fmt.Errorf("invalid %s role ARN format: %s", roleType, roleARN)
	}

	// Check if role exists
	_, err := c.iamClient.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to validate %s role %s: %w", roleType, roleName, err)
	}

	return nil
}

// DescribeSubnets describes EC2 subnets by their IDs
func (c *awsClient) DescribeSubnets(ctx context.Context, subnetIDs []string) ([]ec2Types.Subnet, error) {
	if len(subnetIDs) == 0 {
		return nil, fmt.Errorf("no subnet IDs provided")
	}

	input := &ec2.DescribeSubnetsInput{
		SubnetIds: subnetIDs,
	}

	result, err := c.ec2Client.DescribeSubnets(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe subnets: %w", err)
	}

	return result.Subnets, nil
}

// VPCHasInternetGateway checks if a VPC has an internet gateway attached
func (c *awsClient) VPCHasInternetGateway(ctx context.Context, vpcID string) (bool, error) {
	input := &ec2.DescribeInternetGatewaysInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("attachment.vpc-id"),
				Values: []string{vpcID},
			},
		},
	}

	result, err := c.ec2Client.DescribeInternetGateways(ctx, input)
	if err != nil {
		return false, fmt.Errorf("failed to describe internet gateways: %w", err)
	}

	return len(result.InternetGateways) > 0, nil
}
