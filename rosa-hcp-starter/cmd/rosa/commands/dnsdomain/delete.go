package dnsdomain

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/dnsdomain"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// DeleteOptions contains options for deleting a DNS domain
type DeleteOptions struct {
	DomainID string
	Force    bool
}

// NewDeleteCommand creates the dns-domain delete command
func NewDeleteCommand(logger *slog.Logger) *cobra.Command {
	opts := &DeleteOptions{}

	cmd := &cobra.Command{
		Use:     "dns-domain DOMAIN_ID",
		Aliases: []string{"dnsdomain"},
		Short:   "Delete a DNS domain",
		Long:    "Delete a DNS domain reservation.",
		Example: `  # Delete a DNS domain
  rosa delete dns-domain abc123

  # Force delete without confirmation
  rosa delete dns-domain abc123 --yes`,
		Args: func(_ *cobra.Command, argv []string) error {
			if len(argv) != 1 {
				return fmt.Errorf("expected exactly one argument: the DNS domain ID")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.DomainID = args[0]
			return runDeleteDNSDomain(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.Force, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runDeleteDNSDomain(ctx context.Context, logger *slog.Logger, opts *DeleteOptions) error {
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

	// Create DNS Domain service
	dnsSvc, err := dnsdomain.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create DNS domain service: %w", err)
	}

	// Confirm deletion
	if !opts.Force {
		writer.Warning("This will permanently delete DNS domain '%s'", opts.DomainID)
		writer.Warning("This action cannot be undone.")
	}

	// Delete the DNS domain
	writer.Info("Deleting DNS domain '%s'...", opts.DomainID)
	err = dnsSvc.Delete(ctx, opts.DomainID)
	if err != nil {
		return fmt.Errorf("failed to delete DNS domain: %w", err)
	}

	writer.Success("Successfully deleted DNS domain '%s'", opts.DomainID)
	return nil
}
