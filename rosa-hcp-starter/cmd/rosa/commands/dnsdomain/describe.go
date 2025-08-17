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

// DescribeOptions contains options for describing a DNS domain
type DescribeOptions struct {
	DomainID string
}

// NewDescribeCommand creates the dns-domain describe command
func NewDescribeCommand(logger *slog.Logger) *cobra.Command {
	opts := &DescribeOptions{}

	cmd := &cobra.Command{
		Use:     "dns-domain DOMAIN_ID",
		Aliases: []string{"dnsdomain"},
		Short:   "Show details of a DNS domain",
		Long:    "Display detailed information about a specific DNS domain reservation.",
		Example: `  # Describe a DNS domain
  rosa describe dns-domain abc123`,
		Args: func(_ *cobra.Command, argv []string) error {
			if len(argv) != 1 {
				return fmt.Errorf("expected exactly one argument: the DNS domain ID")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.DomainID = args[0]
			return runDescribeDNSDomain(cmd.Context(), logger, opts)
		},
	}

	return cmd
}

func runDescribeDNSDomain(ctx context.Context, logger *slog.Logger, opts *DescribeOptions) error {
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

	// Get the DNS domain
	domain, err := dnsSvc.Get(ctx, opts.DomainID)
	if err != nil {
		return fmt.Errorf("failed to get DNS domain: %w", err)
	}

	// Display DNS domain details
	writer.Title("DNS Domain Details")
	
	details := map[string]string{
		"ID":          domain.ID,
		"Domain Name": fmt.Sprintf("%s.%s", domain.ID, domain.BaseDomain),
		"Base Domain": domain.BaseDomain,
		"Status":      domain.Status,
	}
	
	if domain.ClusterID != "" {
		details["Cluster ID"] = domain.ClusterID
		details["In Use"] = "Yes"
	} else {
		details["In Use"] = "No"
	}
	
	if domain.ReservedAt != "" {
		details["Reserved At"] = domain.ReservedAt
	}
	
	if domain.UserDefined {
		details["User Defined"] = "Yes"
	} else {
		details["User Defined"] = "No"
	}
	
	writer.KeyValue(details)

	// Show status-specific information
	switch domain.Status {
	case "ready":
		writer.Success("✓ DNS domain is ready for use")
		if domain.ClusterID == "" {
			writer.Info("\n📋 This domain is available. To use it:")
			fmt.Printf("  rosa create cluster --base-domain %s ...\n", fmt.Sprintf("%s.%s", domain.ID, domain.BaseDomain))
		} else {
			writer.Info("\n📋 This domain is currently in use by cluster: %s", domain.ClusterID)
		}
		
	case "pending":
		writer.Info("⏳ DNS domain is being provisioned. Please wait a few moments.")
		
	case "validating":
		writer.Info("🔍 DNS domain is being validated.")
		
	case "invalid":
		writer.Error("✗ DNS domain validation failed")
		writer.Info("Please check the domain configuration and try again.")
		
	default:
		writer.Info("Status: %s", domain.Status)
	}

	// Show DNS configuration requirements
	if domain.Status == "ready" && domain.ClusterID == "" {
		writer.Info("\n📋 DNS Configuration Requirements:")
		fmt.Println("  Ensure the following DNS records are configured:")
		fmt.Printf("  - NS records for %s.%s pointing to AWS Route53\n", domain.ID, domain.BaseDomain)
		fmt.Println("  - Proper delegation from parent domain")
	}

	// Show commands for actions
	writer.Info("\n📋 Available Actions:")
	if domain.ClusterID == "" {
		fmt.Printf("  # Delete this DNS domain:\n")
		fmt.Printf("  rosa delete dns-domain %s\n", opts.DomainID)
	} else {
		fmt.Println("  This domain is in use and cannot be deleted until the cluster is removed.")
	}

	return nil
}
