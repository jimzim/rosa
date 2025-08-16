package network

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	
	"github.com/openshift/rosa-hcp/pkg/network"
	"github.com/openshift/rosa-hcp/pkg/output"
)

type deleteOptions struct {
	stackName string
	region    string
	force     bool
}

func NewDeleteCommand(logger *slog.Logger) *cobra.Command {
	opts := &deleteOptions{}
	
	cmd := &cobra.Command{
		Use:   "network STACK_NAME",
		Short: "Delete a VPC network",
		Long:  `Delete a VPC network created by rosa create network command.`,
		Example: `  # Delete a VPC network
  rosa delete network rosa-vpc-myvpc --region us-west-2

  # Delete without confirmation
  rosa delete network rosa-vpc-myvpc --region us-west-2 --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.stackName = args[0]
			}
			return runDelete(cmd.Context(), logger, opts)
		},
	}
	
	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region where the VPC is located")
	cmd.Flags().BoolVarP(&opts.force, "yes", "y", false, "Skip confirmation prompt")
	
	return cmd
}

func runDelete(ctx context.Context, logger *slog.Logger, opts *deleteOptions) error {
	if opts.stackName == "" {
		return fmt.Errorf("stack name is required")
	}
	if opts.region == "" {
		return fmt.Errorf("AWS region is required")
	}
	
	writer := output.NewWriter(output.FormatText)
	
	// Create the network service
	netService, err := network.NewService(ctx, opts.region, logger)
	if err != nil {
		return fmt.Errorf("failed to create network service: %w", err)
	}
	
	// Get VPC info first to show what will be deleted
	writer.Info("Fetching VPC information...")
	vpcInfo, err := netService.GetVPCInfo(ctx, opts.stackName)
	if err != nil {
		return fmt.Errorf("failed to get VPC info: %w", err)
	}
	
	// Show what will be deleted
	writer.Title("VPC to be deleted")
	writer.KeyValue(map[string]string{
		"Stack Name": vpcInfo.StackName,
		"VPC ID": vpcInfo.VPCId,
		"Region": vpcInfo.Region,
		"Public Subnets": fmt.Sprintf("%d", len(vpcInfo.PublicSubnetIds)),
		"Private Subnets": fmt.Sprintf("%d", len(vpcInfo.PrivateSubnetIds)),
	})
	fmt.Println()
	
	// Confirm deletion
	if !opts.force {
		writer.Warning("This action cannot be undone!")
		
		var confirm bool
		err := huh.NewConfirm().
			Title("Delete VPC?").
			Description(fmt.Sprintf("Are you sure you want to delete VPC %s?", opts.stackName)).
			Value(&confirm).
			Run()
		if err != nil {
			return err
		}
		
		if !confirm {
			writer.Info("Deletion cancelled")
			return nil
		}
	}
	
	// Delete the VPC
	writer.Info("Deleting VPC network...")
	err = netService.DeleteVPC(ctx, opts.stackName)
	if err != nil {
		return fmt.Errorf("failed to delete VPC: %w", err)
	}
	
	writer.Success("VPC %s deleted successfully", opts.stackName)
	return nil
}
