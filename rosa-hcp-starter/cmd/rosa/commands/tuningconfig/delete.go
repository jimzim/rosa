package tuningconfig

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
	"github.com/openshift/rosa-hcp/pkg/tuningconfig"
)

// DeleteOptions contains options for deleting a TuningConfig
type DeleteOptions struct {
	ClusterName string
	ConfigID    string
	Yes         bool
}

// NewDeleteCommand creates the tuning-config delete command
func NewDeleteCommand(logger *slog.Logger) *cobra.Command {
	opts := &DeleteOptions{}

	cmd := &cobra.Command{
		Use:   "tuning-config CONFIG_ID",
		Short: "Delete a TuningConfig from a cluster",
		Long: `Delete a TuningConfig from a cluster.

The TuningConfig cannot be deleted if it is currently in use by any node pools.`,
		Example: `  # Delete a tuning config by ID or name
  rosa delete tuning-config database-tuning --cluster my-cluster

  # Delete without confirmation prompt
  rosa delete tuning-config database-tuning --cluster my-cluster --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ConfigID = args[0]
			return runDeleteTuningConfig(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runDeleteTuningConfig(ctx context.Context, logger *slog.Logger, opts *DeleteOptions) error {
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

	// Check if cluster is HCP
	if !cluster.Hypershift().Enabled() {
		return fmt.Errorf("TuningConfigs are only supported for HCP clusters")
	}

	// Create TuningConfig service
	tcSvc, err := tuningconfig.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create tuningconfig service: %w", err)
	}

	// Get the TuningConfig to show details
	tc, err := tcSvc.Get(ctx, cluster.ID(), opts.ConfigID)
	if err != nil {
		return fmt.Errorf("failed to get tuningconfig: %w", err)
	}

	writer.Title("Delete TuningConfig")
	writer.KeyValue(map[string]string{
		"Cluster":     cluster.Name(),
		"Config ID":   tc.ID,
		"Config Name": tc.Name,
	})

	// Show a summary of the spec
	if profiles, ok := tc.Spec["profile"].([]interface{}); ok && len(profiles) > 0 {
		var profileNames []string
		for _, p := range profiles {
			if profile, ok := p.(map[string]interface{}); ok {
				if name, ok := profile["name"].(string); ok {
					profileNames = append(profileNames, name)
				}
			}
		}
		if len(profileNames) > 0 {
			fmt.Printf("Profiles: %s\n", strings.Join(profileNames, ", "))
		}
	}

	// Confirm deletion
	if !opts.Yes {
		var confirm bool
		err = huh.NewConfirm().
			Title("Are you sure you want to delete this TuningConfig?").
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

	// Delete the TuningConfig
	writer.Info("Deleting TuningConfig...")
	err = tcSvc.Delete(ctx, cluster.ID(), opts.ConfigID)
	if err != nil {
		return fmt.Errorf("failed to delete tuningconfig: %w", err)
	}

	writer.Success("TuningConfig '%s' deleted successfully", opts.ConfigID)
	return nil
}
