package kubeletconfig

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/kubeletconfig"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// DescribeOptions contains options for describing a KubeletConfig
type DescribeOptions struct {
	ClusterName string
	Name        string
}

// NewDescribeCommand creates the kubeletconfig describe command
func NewDescribeCommand(logger *slog.Logger) *cobra.Command {
	opts := &DescribeOptions{}

	cmd := &cobra.Command{
		Use:     "kubeletconfig",
		Aliases: []string{"kubelet-config"},
		Short:   "Show details of a KubeletConfig",
		Long:    "Display detailed information about a specific KubeletConfig for an HCP cluster.",
		Example: `  # Describe a KubeletConfig
  rosa describe kubeletconfig --cluster my-cluster --name my-config`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDescribeKubeletConfig(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVar(&opts.Name, "name", "", "Name of the KubeletConfig (required)")

	cmd.MarkFlagRequired("cluster")
	cmd.MarkFlagRequired("name")

	return cmd
}

func runDescribeKubeletConfig(ctx context.Context, logger *slog.Logger, opts *DescribeOptions) error {
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

	// Create KubeletConfig service
	kcSvc, err := kubeletconfig.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create kubeletconfig service: %w", err)
	}

	// Get cluster to verify it's HCP
	clusterResp, err := apiClient.GetCluster(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	cluster := clusterResp.Body()

	// Verify HCP cluster
	if cluster.Hypershift() == nil || !cluster.Hypershift().Enabled() {
		return fmt.Errorf("KubeletConfig is only supported for Hosted Control Plane clusters")
	}

	// Get the KubeletConfig
	config, err := kcSvc.Get(ctx, opts.ClusterName, opts.Name)
	if err != nil {
		return fmt.Errorf("failed to get KubeletConfig: %w", err)
	}

	// Display KubeletConfig details
	writer.Title("KubeletConfig Details")
	
	details := map[string]string{
		"Name":         config.Name,
		"Cluster":      opts.ClusterName,
		"Created":      config.CreatedAt,
	}
	
	// Add pod pids limit if set
	if config.PodPidsLimit > 0 {
		details["Pod PIDs Limit"] = fmt.Sprintf("%d", config.PodPidsLimit)
	}
	
	writer.KeyValue(details)

	// Show associated NodePools
	writer.Info("\n📋 Associated NodePools:")
	
	if len(config.NodePools) > 0 {
		for _, np := range config.NodePools {
			fmt.Printf("  - %s (replicas: %d)\n", np, getNodePoolReplicas(ctx, apiClient, opts.ClusterName, np))
		}
	} else {
		fmt.Println("  No NodePools are using this KubeletConfig")
	}

	// Show usage instructions
	writer.Info("\n📋 Usage:")
	fmt.Println("  To apply this KubeletConfig to a NodePool:")
	fmt.Printf("  rosa create nodepool --cluster %s --kubelet-configs %s ...\n", opts.ClusterName, config.Name)
	
	if len(config.NodePools) == 0 {
		writer.Info("\n⚠ Note: This KubeletConfig is not applied to any NodePools yet.")
		writer.Info("Create or edit a NodePool with --kubelet-configs to apply it.")
	}

	return nil
}

// Helper function to get NodePool replica count
func getNodePoolReplicas(ctx context.Context, apiClient api.Client, clusterName, nodePoolName string) int {
	// This would normally query the actual NodePool
	// For now, return a placeholder
	return 2
}
