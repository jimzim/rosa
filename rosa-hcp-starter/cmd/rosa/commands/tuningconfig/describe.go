package tuningconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
	"github.com/openshift/rosa-hcp/pkg/tuningconfig"
)

// DescribeOptions contains options for describing a TuningConfig
type DescribeOptions struct {
	ClusterName  string
	Name         string
	OutputFormat string
}

// NewDescribeCommand creates the tuning-config describe command
func NewDescribeCommand(logger *slog.Logger) *cobra.Command {
	opts := &DescribeOptions{}

	cmd := &cobra.Command{
		Use:     "tuning-config",
		Aliases: []string{"tuningconfig"},
		Short:   "Show details of a TuningConfig",
		Long:    "Display detailed information about a specific TuningConfig for an HCP cluster.",
		Example: `  # Describe a TuningConfig
  rosa describe tuning-config --cluster my-cluster --name my-config

  # Output the spec in JSON format
  rosa describe tuning-config --cluster my-cluster --name my-config --output json

  # Output the spec in YAML format
  rosa describe tuning-config --cluster my-cluster --name my-config --output yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDescribeTuningConfig(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVar(&opts.Name, "name", "", "Name of the TuningConfig (required)")
	flags.StringVarP(&opts.OutputFormat, "output", "o", "", "Output format (json|yaml)")

	cmd.MarkFlagRequired("cluster")
	cmd.MarkFlagRequired("name")

	return cmd
}

func runDescribeTuningConfig(ctx context.Context, logger *slog.Logger, opts *DescribeOptions) error {
	writer := output.NewWriter(output.FormatText)

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

	// Create TuningConfig service
	tcSvc, err := tuningconfig.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create tuningconfig service: %w", err)
	}

	// Get cluster to verify it's HCP
	clusterResp, err := apiClient.GetCluster(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	cluster := clusterResp.Body()

	// Verify HCP cluster
	if cluster.Hypershift() == nil || !cluster.Hypershift().Enabled() {
		return fmt.Errorf("TuningConfig is only supported for Hosted Control Plane clusters")
	}

	// Get the TuningConfig
	config, err := tcSvc.Get(ctx, opts.ClusterName, opts.Name)
	if err != nil {
		return fmt.Errorf("failed to get TuningConfig: %w", err)
	}

	// Handle different output formats
	switch opts.OutputFormat {
	case "json":
		data, err := json.MarshalIndent(config.Spec, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal spec to JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
		
	case "yaml":
		data, err := yaml.Marshal(config.Spec)
		if err != nil {
			return fmt.Errorf("failed to marshal spec to YAML: %w", err)
		}
		fmt.Println(string(data))
		return nil
		
	default:
		// Display TuningConfig details in text format
		writer.Title("TuningConfig Details")
		
		details := map[string]string{
			"Name":         config.Name,
			"Cluster":      opts.ClusterName,
			"Created":      config.CreatedAt,
		}
		
		writer.KeyValue(details)

		// Show TuneD profile information
		if config.Spec != nil {
			writer.Info("\n📋 TuneD Profile Information:")
			
			// Try to extract profile data
			if profile, ok := config.Spec["profile"].([]interface{}); ok && len(profile) > 0 {
				if p, ok := profile[0].(map[string]interface{}); ok {
					if name, ok := p["name"].(string); ok {
						fmt.Printf("  Profile Name: %s\n", name)
					}
					if data, ok := p["data"].(string); ok {
						fmt.Printf("  Profile Data Length: %d bytes\n", len(data))
					}
				}
			}
			
			// Try to extract recommend settings
			if recommend, ok := config.Spec["recommend"].([]interface{}); ok {
				fmt.Printf("  Recommend Rules: %d\n", len(recommend))
			}
		}

		// Show associated NodePools
		writer.Info("\n📋 Associated NodePools:")
		
		if len(config.NodePools) > 0 {
			for _, np := range config.NodePools {
				fmt.Printf("  - %s\n", np)
			}
		} else {
			fmt.Println("  No NodePools are using this TuningConfig")
		}

		// Show usage instructions
		writer.Info("\n📋 Usage:")
		fmt.Println("  To apply this TuningConfig to a NodePool:")
		fmt.Printf("  rosa create nodepool --cluster %s --tuning-configs %s ...\n", opts.ClusterName, config.Name)
		
		fmt.Println("\n  To view the full spec in JSON:")
		fmt.Printf("  rosa describe tuning-config --cluster %s --name %s --output json\n", opts.ClusterName, config.Name)
		
		if len(config.NodePools) == 0 {
			writer.Info("\n⚠ Note: This TuningConfig is not applied to any NodePools yet.")
			writer.Info("Create or edit a NodePool with --tuning-configs to apply it.")
		}
	}

	return nil
}
