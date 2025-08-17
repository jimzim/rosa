package region

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
	"github.com/openshift/rosa-hcp/pkg/region"
)

// ListOptions contains options for listing regions
type ListOptions struct {
	HCPOnly      bool
	OutputFormat string
}

// NewListCommand creates the list regions command
func NewListCommand(logger *slog.Logger) *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:     "regions",
		Aliases: []string{"region"},
		Short:   "List available AWS regions",
		Long: `List AWS regions that are available for ROSA HCP cluster deployment.

This command displays all AWS regions where you can create ROSA clusters.
You can filter to show only regions that support Hosted Control Planes (HCP).`,
		Example: `  # List all available regions
  rosa list regions

  # List only regions that support HCP
  rosa list regions --hcp

  # Output in JSON format
  rosa list regions --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListRegions(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.HCPOnly, "hcp", false, "Show only regions that support Hosted Control Planes")
	flags.StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format (text, json, yaml)")

	return cmd
}

func runListRegions(ctx context.Context, logger *slog.Logger, opts *ListOptions) error {
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

	// Create region service
	regionSvc, err := region.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create region service: %w", err)
	}

	// List regions
	logger.InfoContext(ctx, "listing regions", slog.Bool("hcp_only", opts.HCPOnly))
	regions, err := regionSvc.ListRegions(ctx, opts.HCPOnly)
	if err != nil {
		return fmt.Errorf("failed to list regions: %w", err)
	}

	if len(regions) == 0 {
		logger.InfoContext(ctx, "no regions found")
		if opts.HCPOnly {
			fmt.Println("No HCP-enabled regions available")
		} else {
			fmt.Println("No regions available")
		}
		return nil
	}

	// Sort regions by ID for consistent output
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].ID < regions[j].ID
	})

	// Display results based on output format
	var writer *output.Writer

	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		writer = output.NewWriter(output.FormatJSON)
		return writer.Print(regions)
	case "yaml":
		writer = output.NewWriter(output.FormatYAML)
		return writer.Print(regions)
	default:
		writer = output.NewWriter(output.FormatText)
		// Text format - display as table
		writer.Title("Available AWS Regions")
		writer.KeyValue(map[string]string{"Total": fmt.Sprintf("%d regions", len(regions))})
		if opts.HCPOnly {
			writer.Info("Showing only HCP-enabled regions")
		}
		fmt.Println()

		// Display header
		fmt.Printf("%-15s %-35s %-8s %-10s %-8s\n", 
			"REGION ID", "DISPLAY NAME", "HCP", "MULTI-AZ", "GOV")
		fmt.Println(strings.Repeat("-", 90))

		// Display regions
		for _, r := range regions {
			hcp := "No"
			if r.SupportsHCP {
				hcp = "Yes"
			}
			multiAZ := "No"
			if r.SupportsMultiAZ {
				multiAZ = "Yes"
			}
			gov := ""
			if r.GovCloud {
				gov = "Yes"
			}
			
			fmt.Printf("%-15s %-35s %-8s %-10s %-8s\n",
				r.ID, r.DisplayName, hcp, multiAZ, gov)
		}
		fmt.Println()

		// Show summary
		hcpCount := 0
		multiAZCount := 0
		for _, r := range regions {
			if r.SupportsHCP {
				hcpCount++
			}
			if r.SupportsMultiAZ {
				multiAZCount++
			}
		}
		
		fmt.Printf("Summary:\n")
		fmt.Printf("  • %d regions support HCP\n", hcpCount)
		fmt.Printf("  • %d regions support Multi-AZ deployments\n", multiAZCount)
	}

	return nil
}
