package tuningconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
	"github.com/openshift/rosa-hcp/pkg/tuningconfig"
)

// ListOptions contains options for listing TuningConfigs
type ListOptions struct {
	ClusterName  string
	OutputFormat string
	ShowSpec     bool
}

// NewListCommand creates the tuning-config list command
func NewListCommand(logger *slog.Logger) *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:     "tuning-configs",
		Aliases: []string{"tuningconfig", "tuningconfigs", "tuning-config"},
		Short:   "List TuningConfigs for a cluster",
		Long: `List all TuningConfigs configured for a cluster.

Shows the name and ID for each TuningConfig. Use --show-spec to also display
the TuneD profile specifications.`,
		Example: `  # List all tuning configs for a cluster
  rosa list tuning-configs --cluster my-cluster

  # Show specs as well
  rosa list tuning-configs --cluster my-cluster --show-spec

  # Output in JSON format
  rosa list tuning-configs --cluster my-cluster --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListTuningConfigs(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format (text, json, yaml)")
	flags.BoolVar(&opts.ShowSpec, "show-spec", false, "Show the TuneD spec for each config")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runListTuningConfigs(ctx context.Context, logger *slog.Logger, opts *ListOptions) error {
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

	// Get cluster to verify it exists
	clusterResp, err := apiClient.GetCluster(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	cluster := clusterResp.Body()

	// Check if cluster is HCP
	if !cluster.Hypershift().Enabled() {
		return fmt.Errorf("TuningConfigs are only supported for HCP clusters")
	}

	// Create TuningConfig service
	tcSvc, err := tuningconfig.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create tuningconfig service: %w", err)
	}

	// List TuningConfigs
	configs, err := tcSvc.List(ctx, cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to list tuningconfigs: %w", err)
	}

	// Display results based on output format
	var writer *output.Writer

	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		writer = output.NewWriter(output.FormatJSON)
		return writer.Print(configs)
	case "yaml":
		writer = output.NewWriter(output.FormatYAML)
		return writer.Print(configs)
	default:
		writer = output.NewWriter(output.FormatText)
		if len(configs) == 0 {
			writer.Info("No TuningConfigs found for cluster '%s'", opts.ClusterName)
			return nil
		}

		writer.Title(fmt.Sprintf("TuningConfigs for cluster '%s'", cluster.Name()))
		
		// Display as table
		fmt.Printf("%-20s %-30s\n", "ID", "NAME")
		fmt.Println(strings.Repeat("-", 50))
		
		for _, tc := range configs {
			name := tc.Name
			if name == "" {
				name = "-"
			}
			fmt.Printf("%-20s %-30s\n", tc.ID, name)
		}
		fmt.Println()

		// Show specs if requested
		if opts.ShowSpec {
			fmt.Println("\nTuningConfig Specifications:")
			fmt.Println(strings.Repeat("=", 60))
			
			for _, tc := range configs {
				fmt.Printf("\n%s (%s):\n", tc.Name, tc.ID)
				fmt.Println(strings.Repeat("-", 40))
				
				// Pretty print the spec
				specYAML, err := yaml.Marshal(tc.Spec)
				if err != nil {
					// Fall back to JSON if YAML fails
					specJSON, _ := json.MarshalIndent(tc.Spec, "  ", "  ")
					fmt.Println(string(specJSON))
				} else {
					fmt.Println(string(specYAML))
				}
			}
		}

		// Show which node pools are using these configs
		fmt.Println("\nNote: To see which node pools use these configs, run:")
		fmt.Printf("  rosa list nodepools --cluster %s\n", opts.ClusterName)
	}

	return nil
}
