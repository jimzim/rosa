package network

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	
	"github.com/openshift/rosa-hcp/pkg/network"
	"github.com/openshift/rosa-hcp/pkg/output"
)

type listOptions struct {
	region string
	output string
}

func NewListCommand(logger *slog.Logger) *cobra.Command {
	opts := &listOptions{}
	
	cmd := &cobra.Command{
		Use:     "networks",
		Aliases: []string{"network"},
		Short:   "List VPC networks created for ROSA HCP",
		Long:    `List all VPC networks created by rosa create network command.`,
		Example: `  # List networks in a specific region
  rosa list networks --region us-west-2

  # List networks with JSON output
  rosa list networks --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), logger, opts)
		},
	}
	
	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region to list networks from")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "table", "Output format (table, json, yaml)")
	
	return cmd
}

func runList(ctx context.Context, logger *slog.Logger, opts *listOptions) error {
	if opts.region == "" {
		return fmt.Errorf("AWS region is required")
	}
	
	// Create the network service
	netService, err := network.NewService(ctx, opts.region, logger)
	if err != nil {
		return fmt.Errorf("failed to create network service: %w", err)
	}
	
	// List VPCs
	writer := output.NewWriter(output.FormatText)
	writer.Info("Fetching VPC networks...")
	
	vpcs, err := netService.ListVPCs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list VPCs: %w", err)
	}
	
	if len(vpcs) == 0 {
		writer.Info("No VPC networks found in region %s", opts.region)
		return nil
	}
	
	// Display results based on output format
	switch opts.output {
	case "json":
		// TODO: Implement JSON output
		return fmt.Errorf("JSON output not yet implemented")
		
	case "yaml":
		// TODO: Implement YAML output
		return fmt.Errorf("YAML output not yet implemented")
		
	default: // table
		writer.Title(fmt.Sprintf("VPC Networks in %s", opts.region))
		fmt.Println()
		
		// Prepare table data
		headers := []string{"STACK NAME", "VPC ID", "REGION", "SUBNETS"}
		var rows [][]string
		
		for _, vpc := range vpcs {
			subnetInfo := fmt.Sprintf("%d public, %d private", 
				len(vpc.PublicSubnetIds), 
				len(vpc.PrivateSubnetIds))
			rows = append(rows, []string{
				truncate(vpc.StackName, 30),
				vpc.VPCId,
				vpc.Region,
				subnetInfo,
			})
		}
		
		writer.Table(headers, rows)
	}
	
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
