package instance

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/instance"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// ListOptions contains options for listing instance types
type ListOptions struct {
	Region     string
	OutputFormat string
}

// NewListCommand creates the list instance-types command
func NewListCommand(logger *slog.Logger) *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:     "instance-types",
		Aliases: []string{"instancetypes", "instance-type"},
		Short:   "List available instance types",
		Long: `List instance types that are available for use with ROSA HCP clusters.

This command displays all AWS instance types that can be used when creating
clusters or node pools. You can optionally filter by region to see only
instances available in a specific region.`,
		Example: `  # List all instance types
  rosa list instance-types

  # List instance types available in a specific region
  rosa list instance-types --region us-west-2

  # Output in JSON format
  rosa list instance-types --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListInstanceTypes(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.Region, "region", "", "Filter instance types by AWS region")
	flags.StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format (text, json, yaml)")

	return cmd
}

func runListInstanceTypes(ctx context.Context, logger *slog.Logger, opts *ListOptions) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create API client
	apiClient, err := api.NewClient(ctx, api.Config{
		URL:   cfg.APIURL,
		Token: cfg.Token,
	})
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Create instance service
	instanceSvc, err := instance.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create instance service: %w", err)
	}

	// List instance types
	var instances []*instance.InstanceType

	if opts.Region != "" {
		logger.InfoContext(ctx, "listing instance types for region", slog.String("region", opts.Region))
		instances, err = instanceSvc.ListInstanceTypesForRegion(ctx, opts.Region)
	} else {
		logger.InfoContext(ctx, "listing all instance types")
		instances, err = instanceSvc.ListInstanceTypes(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to list instance types: %w", err)
	}

	if len(instances) == 0 {
		if opts.Region != "" {
			logger.InfoContext(ctx, "no instance types found for region", slog.String("region", opts.Region))
			fmt.Printf("No instance types available in region %s\n", opts.Region)
		} else {
			logger.InfoContext(ctx, "no instance types found")
			fmt.Println("No instance types available")
		}
		return nil
	}

	// Sort instances by category, then by CPU, then by memory
	sort.Slice(instances, func(i, j int) bool {
		if instances[i].Category != instances[j].Category {
			return instances[i].Category < instances[j].Category
		}
		if instances[i].CPU != instances[j].CPU {
			return instances[i].CPU < instances[j].CPU
		}
		return instances[i].Memory < instances[j].Memory
	})

	// Display results based on output format
	var writer *output.Writer

	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		writer = output.NewWriter(output.FormatJSON)
		return writer.Print(instances)
	case "yaml":
		writer = output.NewWriter(output.FormatYAML)
		return writer.Print(instances)
	default:
		writer = output.NewWriter(output.FormatText)
		// Text format - display as table
		writer.Title("Available Instance Types")
		kvMap := map[string]string{
			"Total": fmt.Sprintf("%d instance types", len(instances)),
		}
		if opts.Region != "" {
			kvMap["Region"] = opts.Region
		}
		writer.KeyValue(kvMap)
		fmt.Println()

		// Group by category for better readability
		categories := make(map[string][]*instance.InstanceType)
		for _, inst := range instances {
			category := inst.Category
			if category == "" {
				category = "general"
			}
			categories[category] = append(categories[category], inst)
		}

		// Display each category
		for category, catInstances := range categories {
			fmt.Printf("\n%s Instance Types:\n", strings.Title(category))
			fmt.Println(strings.Repeat("-", 80))
			fmt.Printf("%-20s %-15s %-10s %-15s\n", "INSTANCE TYPE", "VCPUS", "MEMORY", "SIZE")
			fmt.Println(strings.Repeat("-", 80))
			
			for _, inst := range catInstances {
				memoryStr := fmt.Sprintf("%.1f GiB", inst.Memory)
				fmt.Printf("%-20s %-15d %-10s %-15s\n", 
					inst.ID, inst.CPU, memoryStr, inst.Size)
			}
		}
		fmt.Println()
	}

	return nil
}
