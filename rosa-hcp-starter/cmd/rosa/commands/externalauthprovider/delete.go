package externalauthprovider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	extAuthSvc "github.com/openshift/rosa-hcp/pkg/externalauthprovider"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// DeleteOptions contains options for deleting an external auth provider
type DeleteOptions struct {
	ClusterName  string
	ProviderName string
	Force        bool
}

// NewDeleteCommand creates the external-auth-provider delete command
func NewDeleteCommand(logger *slog.Logger) *cobra.Command {
	opts := &DeleteOptions{}

	cmd := &cobra.Command{
		Use:     "external-auth-provider",
		Aliases: []string{"externalauthprovider"},
		Short:   "Delete an external authentication provider",
		Long:    "Delete an external authentication provider from a cluster.",
		Example: `  # Delete an external auth provider
  rosa delete external-auth-provider --cluster my-cluster --name my-provider

  # Force delete without confirmation
  rosa delete external-auth-provider --cluster my-cluster --name my-provider --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteExternalAuthProvider(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVar(&opts.ProviderName, "name", "", "Name of the provider to delete (required)")
	flags.BoolVarP(&opts.Force, "yes", "y", false, "Skip confirmation prompt")

	cmd.MarkFlagRequired("cluster")
	cmd.MarkFlagRequired("name")

	return cmd
}

func runDeleteExternalAuthProvider(ctx context.Context, logger *slog.Logger, opts *DeleteOptions) error {
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

	// Create external auth service
	extAuthService, err := extAuthSvc.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create external auth service: %w", err)
	}

	// Get cluster to verify it exists
	clusterResp, err := apiClient.GetCluster(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	cluster := clusterResp.Body()

	// Check if external auth is enabled
	if cluster.ExternalAuthConfig() == nil || !cluster.ExternalAuthConfig().Enabled() {
		writer.Warning("Cluster '%s' does not have external authentication enabled", opts.ClusterName)
		return nil
	}

	// List providers to verify the one we're deleting exists
	providers, err := extAuthService.List(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to list external auth providers: %w", err)
	}

	found := false
	for _, provider := range providers {
		if provider.Name == opts.ProviderName {
			found = true
			break
		}
	}

	if !found {
		writer.Warning("External auth provider '%s' not found on cluster '%s'", opts.ProviderName, opts.ClusterName)
		return nil
	}

	// Confirm deletion
	if !opts.Force {
		writer.Warning("This will permanently delete external auth provider '%s' from cluster '%s'", 
			opts.ProviderName, opts.ClusterName)
		writer.Warning("Users authenticating through this provider will lose access.")
	}

	// Delete the provider
	writer.Info("Deleting external auth provider '%s'...", opts.ProviderName)
	err = extAuthService.Delete(ctx, opts.ClusterName, opts.ProviderName)
	if err != nil {
		return fmt.Errorf("failed to delete external auth provider: %w", err)
	}

	writer.Success("Successfully deleted external auth provider '%s' from cluster '%s'", 
		opts.ProviderName, opts.ClusterName)

	// Show remaining providers
	remainingProviders, err := extAuthService.List(ctx, opts.ClusterName)
	if err == nil && len(remainingProviders) > 0 {
		writer.Info("\nRemaining external auth providers:")
		for _, provider := range remainingProviders {
			fmt.Printf("  - %s (%s)\n", provider.Name, provider.IssuerURL)
		}
	}

	return nil
}