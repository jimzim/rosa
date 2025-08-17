package addon

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/addon"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// UninstallOptions contains options for uninstalling an add-on
type UninstallOptions struct {
	ClusterName string
	AddonID     string
	Yes         bool
}

// NewUninstallCommand creates the addon uninstall command
func NewUninstallCommand(logger *slog.Logger) *cobra.Command {
	opts := &UninstallOptions{}

	cmd := &cobra.Command{
		Use:   "addon ADDON_ID",
		Short: "Uninstall an add-on from a cluster",
		Long: `Uninstall a Red Hat managed add-on from a cluster.

This will remove the add-on operators and resources from the cluster.
Some add-ons may take time to fully clean up their resources.`,
		Example: `  # Uninstall an add-on
  rosa uninstall addon cluster-logging-operator --cluster my-cluster

  # Uninstall without confirmation prompt
  rosa uninstall addon my-addon --cluster my-cluster --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.AddonID = args[0]
			return runUninstallAddon(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runUninstallAddon(ctx context.Context, logger *slog.Logger, opts *UninstallOptions) error {
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

	// Get cluster
	clusterResp, err := apiClient.GetCluster(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	cluster := clusterResp.Body()

	// Create Add-on service
	addonSvc, err := addon.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create add-on service: %w", err)
	}

	// Get the installed add-on
	installation, err := addonSvc.Get(ctx, cluster.ID(), opts.AddonID)
	if err != nil {
		return fmt.Errorf("add-on '%s' is not installed on cluster '%s'", opts.AddonID, opts.ClusterName)
	}

	// Check if already being deleted
	if installation.State == addon.StateDeleting {
		writer.Warning("Add-on '%s' is already being uninstalled", installation.AddonName)
		return nil
	}

	// Display uninstall summary
	writer.Title("Add-on Uninstall Summary")
	writer.KeyValue(map[string]string{
		"Cluster":     cluster.Name(),
		"Add-on ID":   installation.AddonID,
		"Add-on Name": installation.AddonName,
		"Version":     installation.Version,
		"State":       string(installation.State),
	})

	if len(installation.Parameters) > 0 {
		writer.KeyValue(map[string]string{
			"Parameters": addon.FormatParameters(installation.Parameters),
		})
	}

	// Get add-on details for additional info
	addonDetails, err := addonSvc.GetAddOn(ctx, opts.AddonID)
	if err == nil && addonDetails.RequiresSTS {
		isSTS, _, accountID := addon.GetClusterInfo(cluster)
		if isSTS {
			writer.Warning("\nThis add-on uses STS authentication.")
			writer.Info("After uninstalling, you may want to clean up the associated IAM roles:")
			fmt.Printf("  rosa delete operator-roles --cluster %s --addon %s\n", opts.ClusterName, opts.AddonID)
			writer.Info("Account ID: %s", accountID)
		}
	}

	// Confirm uninstallation
	if !opts.Yes {
		writer.Warning("\nUninstalling an add-on will remove all its resources from the cluster.")
		writer.Warning("This action cannot be undone.")
		
		var confirm bool
		err = huh.NewConfirm().
			Title(fmt.Sprintf("Uninstall add-on '%s' from cluster '%s'?", installation.AddonName, opts.ClusterName)).
			Value(&confirm).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirm {
			writer.Info("Uninstallation cancelled")
			return nil
		}
	}

	// Uninstall the add-on
	writer.Info("Uninstalling add-on...")
	err = addonSvc.Uninstall(ctx, cluster.ID(), opts.AddonID)
	if err != nil {
		return fmt.Errorf("failed to uninstall add-on: %w", err)
	}

	writer.Success("Add-on '%s' uninstallation initiated", installation.AddonName)
	
	writer.Info("\nThe add-on uninstallation may take several minutes to complete.")
	writer.Info("Resources created by the add-on may take additional time to be fully removed.")
	writer.Info("\nTo check the uninstallation status:")
	fmt.Printf("  rosa list addons --cluster %s\n", opts.ClusterName)

	return nil
}
