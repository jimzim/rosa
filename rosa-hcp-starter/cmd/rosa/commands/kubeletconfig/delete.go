package kubeletconfig

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/kubeletconfig"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// DeleteOptions contains options for deleting a KubeletConfig
type DeleteOptions struct {
	ClusterName string
	ConfigID    string
	Yes         bool
}

// NewDeleteCommand creates the kubeletconfig delete command
func NewDeleteCommand(logger *slog.Logger) *cobra.Command {
	opts := &DeleteOptions{}

	cmd := &cobra.Command{
		Use:   "kubeletconfig CONFIG_ID",
		Short: "Delete a KubeletConfig from a cluster",
		Long: `Delete a KubeletConfig from a cluster.

The KubeletConfig cannot be deleted if it is currently in use by any node pools.`,
		Example: `  # Delete a kubeletconfig by ID
  rosa delete kubeletconfig high-density --cluster my-cluster

  # Delete without confirmation prompt
  rosa delete kubeletconfig high-density --cluster my-cluster --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ConfigID = args[0]
			return runDeleteKubeletConfig(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runDeleteKubeletConfig(ctx context.Context, logger *slog.Logger, opts *DeleteOptions) error {
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

	// Get the KubeletConfig to show details
	kc, err := kcSvc.Get(ctx, cluster.ID(), opts.ConfigID)
	if err != nil {
		return fmt.Errorf("failed to get kubeletconfig: %w", err)
	}

	writer.Title("Delete KubeletConfig")
	writer.KeyValue(map[string]string{
		"Cluster":        cluster.Name(),
		"Config ID":      kc.ID,
		"Config Name":    kc.Name,
		"Pod PIDs Limit": fmt.Sprintf("%d", kc.PodPidsLimit),
	})

	// Confirm deletion
	if !opts.Yes {
		var confirm bool
		err = huh.NewConfirm().
			Title("Are you sure you want to delete this KubeletConfig?").
			Description("This action cannot be undone.").
			Value(&confirm).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirm {
			writer.Info("Deletion cancelled")
			return nil
		}
	}

	// Delete the KubeletConfig
	writer.Info("Deleting KubeletConfig...")
	err = kcSvc.Delete(ctx, cluster.ID(), opts.ConfigID)
	if err != nil {
		return fmt.Errorf("failed to delete kubeletconfig: %w", err)
	}

	writer.Success("KubeletConfig '%s' deleted successfully", opts.ConfigID)
	return nil
}
