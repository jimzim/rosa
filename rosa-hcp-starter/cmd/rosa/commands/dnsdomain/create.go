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

// CreateOptions contains options for creating a DNS domain
type CreateOptions struct {
	HostedCP bool
}

// NewCreateCommand creates the dns-domain create command
func NewCreateCommand(logger *slog.Logger) *cobra.Command {
	opts := &CreateOptions{}

	cmd := &cobra.Command{
		Use:     "dns-domain",
		Aliases: []string{"dnsdomain"},
		Short:   "Create a DNS domain reservation",
		Long: `Create a DNS domain reservation for ROSA clusters.

DNS domains are used to reserve custom domain names for your clusters.
For HCP clusters, use the --hosted-cp flag.`,
		Example: `  # Create a DNS domain for HCP cluster
  rosa create dns-domain --hosted-cp

  # Create a DNS domain for Classic cluster
  rosa create dns-domain`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateDNSDomain(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.HostedCP, "hosted-cp", true, "Create DNS domain for Hosted Control Plane (HCP) clusters")

	return cmd
}

func runCreateDNSDomain(ctx context.Context, logger *slog.Logger, opts *CreateOptions) error {
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

	// Display configuration
	writer.Title("Creating DNS Domain")
	
	archType := "Classic"
	if opts.HostedCP {
		archType = "HCP (Hosted Control Plane)"
	}
	
	writer.KeyValue(map[string]string{
		"Architecture": archType,
	})

	// Create the DNS domain
	writer.Info("Creating DNS domain reservation...")
	domain, err := dnsSvc.Create(ctx, opts.HostedCP)
	if err != nil {
		return fmt.Errorf("failed to create DNS domain: %w", err)
	}

	writer.Success("DNS domain '%s' created successfully", domain.ID)
	
	writer.KeyValue(map[string]string{
		"Domain ID":     domain.ID,
		"Architecture":  domain.Architecture,
		"Reserved Time": domain.ReservedAt,
	})

	// Show next steps
	writer.Info("\n📋 Next Steps:")
	writer.Info("1. Use this DNS domain ID when creating your cluster")
	writer.Info("2. View all DNS domains:")
	fmt.Println("   rosa list dns-domains")
	
	if opts.HostedCP {
		writer.Info("\n3. When creating an HCP cluster with this domain:")
		fmt.Println("   rosa create cluster --dns-domain-id " + domain.ID)
	}

	return nil
}
