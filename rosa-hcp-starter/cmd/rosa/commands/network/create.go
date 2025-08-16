package network

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	
	"github.com/openshift/rosa-hcp/pkg/network"
	"github.com/openshift/rosa-hcp/pkg/output"
)

type createOptions struct {
	name   string
	region string
	cidr   string
	azCount int
	azs    []string
	dryRun bool
	interactive bool
}

func NewCreateCommand(logger *slog.Logger) *cobra.Command {
	opts := &createOptions{}
	
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Create a VPC network for ROSA HCP clusters",
		Long: `Create a VPC network with public and private subnets suitable for ROSA HCP clusters.
		
This command creates a CloudFormation stack with:
  - VPC with DNS support enabled
  - Public and private subnets across availability zones
  - Internet Gateway and NAT Gateway
  - Route tables configured for public and private access
  - Security groups for ROSA
  - VPC endpoints for AWS services`,
		Example: `  # Create a VPC with default settings
  rosa create network --name myvpc --region us-west-2

  # Create a VPC with custom CIDR
  rosa create network --name myvpc --region us-west-2 --cidr 10.1.0.0/16

  # Create a VPC with specific availability zones
  rosa create network --name myvpc --region us-west-2 --availability-zones us-west-2a,us-west-2b

  # Interactive mode
  rosa create network --interactive`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd.Context(), logger, opts)
		},
	}
	
	cmd.Flags().StringVar(&opts.name, "name", "", "Name for the VPC resources")
	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region for the VPC")
	cmd.Flags().StringVar(&opts.cidr, "cidr", "10.0.0.0/16", "CIDR block for the VPC")
	cmd.Flags().IntVar(&opts.azCount, "availability-zone-count", 2, "Number of availability zones (1-4)")
	cmd.Flags().StringSliceVar(&opts.azs, "availability-zones", nil, "Specific availability zones to use")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Show what would be created without creating it")
	cmd.Flags().BoolVarP(&opts.interactive, "interactive", "i", false, "Interactive mode")
	
	return cmd
}

func runCreate(ctx context.Context, logger *slog.Logger, opts *createOptions) error {
	// Interactive mode
	if opts.interactive {
		if err := promptForOptions(ctx, opts); err != nil {
			return err
		}
	}
	
	// Validate options
	if opts.name == "" {
		return fmt.Errorf("VPC name is required")
	}
	if opts.region == "" {
		return fmt.Errorf("AWS region is required")
	}
	if opts.azCount < 1 || opts.azCount > 4 {
		return fmt.Errorf("availability zone count must be between 1 and 4")
	}
	if len(opts.azs) > 0 && len(opts.azs) != opts.azCount {
		return fmt.Errorf("number of availability zones provided (%d) must match availability zone count (%d)", 
			len(opts.azs), opts.azCount)
	}
	
	// Show what will be created
	writer := output.NewWriter(output.FormatText)
	writer.Title("VPC Configuration")
	writer.KeyValue(map[string]string{
		"Name": opts.name,
		"Region": opts.region,
		"CIDR": opts.cidr,
		"Availability Zones": fmt.Sprintf("%d", opts.azCount),
	})
	if len(opts.azs) > 0 {
		writer.KeyValue(map[string]string{
			"Specific AZs": strings.Join(opts.azs, ", "),
		})
	}
	
	if opts.dryRun {
		writer.Info("Dry run mode - no resources will be created")
		return nil
	}
	
	// Create the network service
	netService, err := network.NewService(ctx, opts.region, logger)
	if err != nil {
		return fmt.Errorf("failed to create network service: %w", err)
	}
	
	// Prepare VPC configuration
	vpcConfig := network.VPCConfig{
		Name:                  opts.name,
		Region:                opts.region,
		CIDR:                  opts.cidr,
		AvailabilityZoneCount: opts.azCount,
		AvailabilityZones:     opts.azs,
		Tags: map[string]string{
			"created_by": "rosa-hcp-cli",
			"purpose":    "rosa-hcp-cluster",
		},
	}
	
	// Create the VPC
	writer.Info("Creating VPC network...")
	vpcInfo, err := netService.CreateVPC(ctx, vpcConfig)
	if err != nil {
		return fmt.Errorf("failed to create VPC: %w", err)
	}
	
	// Display results
	writer.Success("VPC created successfully!")
	fmt.Println()
	writer.Title("VPC Details")
	writer.KeyValue(map[string]string{
		"VPC ID": vpcInfo.VPCId,
		"Stack Name": vpcInfo.StackName,
		"Region": vpcInfo.Region,
	})
	
	fmt.Println()
	writer.Title("Subnets")
	writer.KeyValue(map[string]string{
		"Public Subnets": strings.Join(vpcInfo.PublicSubnetIds, ", "),
		"Private Subnets": strings.Join(vpcInfo.PrivateSubnetIds, ", "),
	})
	
	fmt.Println()
	writer.Info("You can now use these subnet IDs when creating a ROSA HCP cluster:")
	fmt.Printf("  rosa cluster create --name mycluster --region %s --subnet-ids %s\n",
		opts.region, strings.Join(vpcInfo.PrivateSubnetIds, ","))
	
	return nil
}

func promptForOptions(ctx context.Context, opts *createOptions) error {
	// Get available regions if not set
	if opts.region == "" {
		regions := []string{
			"us-east-1", "us-east-2", "us-west-1", "us-west-2",
			"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1",
			"ap-northeast-1", "ap-northeast-2", "ap-southeast-1", "ap-southeast-2",
		}
		
		var region string
		err := huh.NewSelect[string]().
			Title("Select AWS Region").
			Options(huh.NewOptions(regions...)...).
			Value(&region).
			Run()
		if err != nil {
			return err
		}
		opts.region = region
	}
	
	// Prompt for VPC name
	if opts.name == "" {
		err := huh.NewInput().
			Title("VPC Name").
			Description("Name for the VPC resources").
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("name cannot be empty")
				}
				return nil
			}).
			Value(&opts.name).
			Run()
		if err != nil {
			return err
		}
	}
	
	// Prompt for CIDR if not set
	if opts.cidr == "" {
		var useDefault bool
		err := huh.NewConfirm().
			Title("Use default CIDR?").
			Description("Default CIDR is 10.0.0.0/16").
			Value(&useDefault).
			Run()
		if err != nil {
			return err
		}
		
		if useDefault {
			opts.cidr = "10.0.0.0/16"
		} else {
			err := huh.NewInput().
				Title("VPC CIDR").
				Description("CIDR block for the VPC (e.g., 10.0.0.0/16)").
				Value(&opts.cidr).
				Run()
			if err != nil {
				return err
			}
		}
	}
	
	// Prompt for AZ count
	if opts.azCount == 0 {
		azOptions := []huh.Option[int]{
			huh.NewOption("1 - Single AZ", 1),
			huh.NewOption("2 - Two AZs (recommended)", 2),
			huh.NewOption("3 - Three AZs", 3),
			huh.NewOption("4 - Four AZs", 4),
		}
		
		err := huh.NewSelect[int]().
			Title("Number of Availability Zones").
			Description("How many availability zones should the VPC span?").
			Options(azOptions...).
			Value(&opts.azCount).
			Run()
		if err != nil {
			return err
		}
	}
	
	// Ask if user wants to specify AZs
	if len(opts.azs) == 0 && opts.azCount > 0 {
		var specifyAZs bool
		err := huh.NewConfirm().
			Title("Specify availability zones?").
			Description("Would you like to specify which availability zones to use?").
			Value(&specifyAZs).
			Run()
		if err != nil {
			return err
		}
		
		if specifyAZs {
			// Get available AZs for the region
			azs, err := network.GetAvailabilityZones(ctx, opts.region)
			if err != nil {
				return fmt.Errorf("failed to get availability zones: %w", err)
			}
			
			selectedAZs := make([]string, 0, opts.azCount)
			for i := 0; i < opts.azCount; i++ {
				var az string
				err := huh.NewSelect[string]().
					Title(fmt.Sprintf("Select Availability Zone %d", i+1)).
					Options(huh.NewOptions(azs...)...).
					Value(&az).
					Run()
				if err != nil {
					return err
				}
				selectedAZs = append(selectedAZs, az)
			}
			opts.azs = selectedAZs
		}
	}
	
	return nil
}
