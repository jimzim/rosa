package network

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

//go:embed templates/*
var templates embed.FS

// VPCConfig holds configuration for VPC creation
type VPCConfig struct {
	Name                  string
	Region                string
	CIDR                  string
	AvailabilityZoneCount int
	AvailabilityZones     []string
	Tags                  map[string]string
}

// VPCInfo contains information about a created VPC
type VPCInfo struct {
	VPCId           string
	PublicSubnetIds []string
	PrivateSubnetIds []string
	StackName       string
	Region          string
}

// Service provides network operations
type Service interface {
	CreateVPC(ctx context.Context, config VPCConfig) (*VPCInfo, error)
	GetVPCInfo(ctx context.Context, stackName string) (*VPCInfo, error)
	DeleteVPC(ctx context.Context, stackName string) error
	ListVPCs(ctx context.Context) ([]VPCInfo, error)
}

type service struct {
	cfnClient *cloudformation.Client
	ec2Client *ec2.Client
	logger    *slog.Logger
}

// NewService creates a new network service
func NewService(ctx context.Context, region string, logger *slog.Logger) (Service, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &service{
		cfnClient: cloudformation.NewFromConfig(cfg),
		ec2Client: ec2.NewFromConfig(cfg),
		logger:    logger,
	}, nil
}

// CreateVPC creates a new VPC using CloudFormation
func (s *service) CreateVPC(ctx context.Context, cfg VPCConfig) (*VPCInfo, error) {
	s.logger.Info("Creating VPC", "name", cfg.Name, "region", cfg.Region)

	// Load CloudFormation template
	templateBody, err := templates.ReadFile("templates/rosa-quickstart-vpc.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	// Set default values
	if cfg.CIDR == "" {
		cfg.CIDR = "10.0.0.0/16"
	}
	if cfg.AvailabilityZoneCount == 0 {
		cfg.AvailabilityZoneCount = 2
	}

	// Prepare parameters
	params := []types.Parameter{
		{
			ParameterKey:   aws.String("Name"),
			ParameterValue: aws.String(cfg.Name),
		},
		{
			ParameterKey:   aws.String("Region"),
			ParameterValue: aws.String(cfg.Region),
		},
		{
			ParameterKey:   aws.String("VpcCidr"),
			ParameterValue: aws.String(cfg.CIDR),
		},
		{
			ParameterKey:   aws.String("AvailabilityZoneCount"),
			ParameterValue: aws.String(fmt.Sprintf("%d", cfg.AvailabilityZoneCount)),
		},
	}

	// Add specific AZs if provided
	for i, az := range cfg.AvailabilityZones {
		if i >= 4 {
			break // Template supports max 4 AZs
		}
		params = append(params, types.Parameter{
			ParameterKey:   aws.String(fmt.Sprintf("AZ%d", i+1)),
			ParameterValue: aws.String(az),
		})
	}

	// Prepare tags
	var tags []types.Tag
	for k, v := range cfg.Tags {
		tags = append(tags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	// Add default tags
	tags = append(tags, 
		types.Tag{
			Key:   aws.String("service"),
			Value: aws.String("ROSA"),
		},
		types.Tag{
			Key:   aws.String("rosa_hcp_policies"),
			Value: aws.String("true"),
		},
		types.Tag{
			Key:   aws.String("created_by"),
			Value: aws.String("rosa-hcp-cli"),
		},
	)

	stackName := fmt.Sprintf("rosa-vpc-%s", cfg.Name)

	// Create the stack
	s.logger.Info("Creating CloudFormation stack", "stack", stackName)
	_, err = s.cfnClient.CreateStack(ctx, &cloudformation.CreateStackInput{
		StackName:    aws.String(stackName),
		TemplateBody: aws.String(string(templateBody)),
		Parameters:   params,
		Tags:         tags,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityIam,
			types.CapabilityCapabilityNamedIam,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create stack: %w", err)
	}

	// Wait for stack creation
	s.logger.Info("Waiting for stack creation to complete...")
	waiter := cloudformation.NewStackCreateCompleteWaiter(s.cfnClient)
	err = waiter.Wait(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}, 10*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for stack creation: %w", err)
	}

	// Get stack outputs
	return s.GetVPCInfo(ctx, stackName)
}

// GetVPCInfo retrieves information about a VPC stack
func (s *service) GetVPCInfo(ctx context.Context, stackName string) (*VPCInfo, error) {
	// Describe the stack
	stackOutput, err := s.cfnClient.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe stack: %w", err)
	}

	if len(stackOutput.Stacks) == 0 {
		return nil, fmt.Errorf("stack not found: %s", stackName)
	}

	// Get resources
	resources, err := s.cfnClient.DescribeStackResources(ctx, &cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe stack resources: %w", err)
	}

	info := &VPCInfo{
		StackName: stackName,
		Region:    s.cfnClient.Options().Region,
	}

	// Extract VPC and subnet IDs
	for _, resource := range resources.StackResources {
		resourceType := aws.ToString(resource.ResourceType)
		logicalId := aws.ToString(resource.LogicalResourceId)
		physicalId := aws.ToString(resource.PhysicalResourceId)

		switch resourceType {
		case "AWS::EC2::VPC":
			info.VPCId = physicalId
		case "AWS::EC2::Subnet":
			if logicalId[:6] == "Public" {
				info.PublicSubnetIds = append(info.PublicSubnetIds, physicalId)
			} else if logicalId[:7] == "Private" {
				info.PrivateSubnetIds = append(info.PrivateSubnetIds, physicalId)
			}
		}
	}

	s.logger.Info("VPC info retrieved", 
		"vpc_id", info.VPCId,
		"public_subnets", info.PublicSubnetIds,
		"private_subnets", info.PrivateSubnetIds,
	)

	return info, nil
}

// DeleteVPC deletes a VPC stack
func (s *service) DeleteVPC(ctx context.Context, stackName string) error {
	s.logger.Info("Deleting VPC stack", "stack", stackName)

	_, err := s.cfnClient.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete stack: %w", err)
	}

	// Wait for deletion
	s.logger.Info("Waiting for stack deletion to complete...")
	waiter := cloudformation.NewStackDeleteCompleteWaiter(s.cfnClient)
	err = waiter.Wait(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}, 10*time.Minute)
	if err != nil {
		return fmt.Errorf("failed waiting for stack deletion: %w", err)
	}

	s.logger.Info("VPC stack deleted successfully")
	return nil
}

// ListVPCs lists all VPC stacks created by rosa
func (s *service) ListVPCs(ctx context.Context) ([]VPCInfo, error) {
	// List stacks with rosa-vpc prefix
	output, err := s.cfnClient.ListStacks(ctx, &cloudformation.ListStacksInput{
		StackStatusFilter: []types.StackStatus{
			types.StackStatusCreateComplete,
			types.StackStatusUpdateComplete,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list stacks: %w", err)
	}

	var vpcs []VPCInfo
	for _, stack := range output.StackSummaries {
		stackName := aws.ToString(stack.StackName)
		// Filter for rosa VPC stacks
		if len(stackName) > 8 && stackName[:8] == "rosa-vpc" {
			info, err := s.GetVPCInfo(ctx, stackName)
			if err != nil {
				s.logger.Warn("Failed to get VPC info", "stack", stackName, "error", err)
				continue
			}
			vpcs = append(vpcs, *info)
		}
	}

	return vpcs, nil
}

// ValidateVPC checks if a VPC and subnets exist and are suitable for ROSA HCP
func ValidateVPC(ctx context.Context, region string, vpcID string, subnetIDs []string) error {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)

	// Check VPC exists
	vpcResult, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	})
	if err != nil {
		return fmt.Errorf("failed to describe VPC: %w", err)
	}
	if len(vpcResult.Vpcs) == 0 {
		return fmt.Errorf("VPC not found: %s", vpcID)
	}

	// Check subnets exist and belong to the VPC
	subnetResult, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: subnetIDs,
	})
	if err != nil {
		return fmt.Errorf("failed to describe subnets: %w", err)
	}

	for _, subnet := range subnetResult.Subnets {
		if aws.ToString(subnet.VpcId) != vpcID {
			return fmt.Errorf("subnet %s does not belong to VPC %s", 
				aws.ToString(subnet.SubnetId), vpcID)
		}
	}

	return nil
}

// GetSubnetsFromVPC retrieves all subnets for a given VPC
func GetSubnetsFromVPC(ctx context.Context, region string, vpcID string) (publicSubnets []string, privateSubnets []string, err error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)

	// Get all subnets for the VPC
	result, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to describe subnets: %w", err)
	}

	// Categorize subnets by checking if they have public IP assignment
	for _, subnet := range result.Subnets {
		subnetID := aws.ToString(subnet.SubnetId)
		if aws.ToBool(subnet.MapPublicIpOnLaunch) {
			publicSubnets = append(publicSubnets, subnetID)
		} else {
			privateSubnets = append(privateSubnets, subnetID)
		}
	}

	return publicSubnets, privateSubnets, nil
}

// GetAvailabilityZones retrieves available AZs for a region
func GetAvailabilityZones(ctx context.Context, region string) ([]string, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)

	result, err := ec2Client.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"available"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe availability zones: %w", err)
	}

	var zones []string
	for _, zone := range result.AvailabilityZones {
		zones = append(zones, aws.ToString(zone.ZoneName))
	}

	return zones, nil
}

// VPCOutput formats VPC information for display
type VPCOutput struct {
	Name             string   `json:"name"`
	VPCID            string   `json:"vpc_id"`
	Region           string   `json:"region"`
	PublicSubnetIDs  []string `json:"public_subnet_ids"`
	PrivateSubnetIDs []string `json:"private_subnet_ids"`
	StackName        string   `json:"stack_name"`
}

// ToOutput converts VPCInfo to VPCOutput
func (v *VPCInfo) ToOutput(name string) VPCOutput {
	return VPCOutput{
		Name:             name,
		VPCID:            v.VPCId,
		Region:           v.Region,
		PublicSubnetIDs:  v.PublicSubnetIds,
		PrivateSubnetIDs: v.PrivateSubnetIds,
		StackName:        v.StackName,
	}
}

// ToJSON returns JSON representation of VPCOutput
func (v VPCOutput) ToJSON() (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
