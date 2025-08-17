package kubeletconfig

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/kubeletconfig"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// ListOptions contains options for listing KubeletConfigs
type ListOptions struct {
	ClusterName  string
	OutputFormat string
}

// NewListCommand creates the kubeletconfig list command
func NewListCommand(logger *slog.Logger) *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:     "kubeletconfigs",
		Aliases: []string{"kubeletconfig", "kubelet-configs", "kubelet-config"},
		Short:   "List KubeletConfigs for a cluster",
		Long: `List all KubeletConfigs configured for a cluster.

Shows the name, ID, and pod PIDs limit for each KubeletConfig.`,
		Example: `  # List all kubeletconfigs for a cluster
  rosa list kubeletconfigs --cluster my-cluster

  # Output in JSON format
  rosa list kubeletconfigs --cluster my-cluster --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListKubeletConfigs(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format (text, json, yaml)")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runListKubeletConfigs(ctx context.Context, logger *slog.Logger, opts *ListOptions) error {
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

	// Create KubeletConfig service
	kcSvc, err := kubeletconfig.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create kubeletconfig service: %w", err)
	}

	// List KubeletConfigs
	configs, err := kcSvc.List(ctx, cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to list kubeletconfigs: %w", err)
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
			writer.Info("No KubeletConfigs found for cluster '%s'", opts.ClusterName)
			return nil
		}

		writer.Title(fmt.Sprintf("KubeletConfigs for cluster '%s'", cluster.Name()))
		
		// Display as table
		fmt.Printf("%-20s %-30s %-15s\n", "ID", "NAME", "POD PIDS LIMIT")
		fmt.Println(strings.Repeat("-", 70))
		
		for _, kc := range configs {
			name := kc.Name
			if name == "" {
				name = "-"
			}
			fmt.Printf("%-20s %-30s %-15d\n", kc.ID, name, kc.PodPidsLimit)
		}
		fmt.Println()

		// Show which node pools are using these configs if HCP
		if cluster.Hypershift().Enabled() {
			fmt.Println("\nNote: To see which node pools use these configs, run:")
			fmt.Printf("  rosa list nodepools --cluster %s\n", opts.ClusterName)
		}
	}

	return nil
}
