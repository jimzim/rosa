package externalauthprovider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/externalauthprovider"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// DeleteOptions contains options for deleting an external auth provider
type DeleteOptions struct {
	ClusterName  string
	ProviderName string
	Yes          bool
}

// NewDeleteCommand creates the external-auth-provider delete command
func NewDeleteCommand(logger *slog.Logger) *cobra.Command {
	opts := &DeleteOptions{}

	cmd := &cobra.Command{
		Use:   "external-auth-provider PROVIDER_NAME",
		Short: "Delete an external authentication provider",
		Long:  "Delete an external authentication provider from a cluster.",
		Example: `  # Delete an external auth provider
  rosa delete external-auth-provider my-sso --cluster my-cluster

  # Delete without confirmation prompt
  rosa delete external-auth-provider my-sso --cluster my-cluster --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ProviderName = args[0]
			return runDeleteExternalAuth(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runDeleteExternalAuth(ctx context.Context, logger *slog.Logger, opts *DeleteOptions) error {
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

	// Create External Auth Provider service
	authSvc, err := externalauthprovider.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create external auth service: %w", err)
	}

	// Get the provider to show details
	provider, err := authSvc.Get(ctx, cluster.ID(), opts.ProviderName)
	if err != nil {
		return fmt.Errorf("failed to get external auth provider '%s': %w", opts.ProviderName, err)
	}

	writer.Title("Delete External Authentication Provider")
	writer.KeyValue(map[string]string{
		"Cluster":    cluster.Name(),
		"Provider":   provider.Name,
		"Issuer URL": provider.IssuerURL,
		"Client ID":  provider.ClientID,
	})

	// Check if this is the last provider
	providers, err := authSvc.List(ctx, cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to list external auth providers: %w", err)
	}

	if len(providers) == 1 {
		writer.Warning("\n⚠️  WARNING: This is the only external auth provider configured.")
		writer.Warning("Deleting it will disable external authentication for the cluster.")
		writer.Warning("Users will need to use break-glass credentials or other IDPs to authenticate.")
	}

	// Confirm deletion
	if !opts.Yes {
		writer.Warning("\nThis action cannot be undone.")

		var confirm bool
		err = huh.NewConfirm().
			Title(fmt.Sprintf("Delete external auth provider '%s'?", provider.Name)).
			Description("All users authenticating through this provider will lose access").
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

	// Delete the provider
	writer.Info("Deleting external auth provider...")
	err = authSvc.Delete(ctx, cluster.ID(), opts.ProviderName)
	if err != nil {
		return fmt.Errorf("failed to delete external auth provider: %w", err)
	}

	writer.Success("External auth provider '%s' deleted successfully", opts.ProviderName)

	if len(providers) == 1 {
		writer.Info("\n✓ External authentication has been disabled for this cluster")
		writer.Info("Users can authenticate using:")
		writer.Info("  • Break-glass credentials (if configured)")
		writer.Info("  • Other configured identity providers")
		writer.Info("  • Cluster admin user (if created)")
	} else {
		writer.Info("\nRemaining external auth providers:")
		for _, p := range providers {
			if p.Name != opts.ProviderName {
				fmt.Printf("  • %s (%s)\n", p.Name, p.IssuerURL)
			}
		}
	}

	writer.Info("\nThe change may take a few minutes to propagate.")

	return nil
}
