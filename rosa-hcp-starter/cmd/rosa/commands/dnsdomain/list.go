package dnsdomain

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/dnsdomain"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// ListOptions contains options for listing DNS domains
type ListOptions struct {
	HostedCPOnly bool
	OutputFormat string
}

// NewListCommand creates the dns-domains list command
func NewListCommand(logger *slog.Logger) *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:     "dns-domains",
		Aliases: []string{"dns-domain", "dnsdomain", "dnsdomains"},
		Short:   "List DNS domain reservations",
		Long:    "List all DNS domain reservations for your organization.",
		Example: `  # List all DNS domains
  rosa list dns-domains

  # List only HCP DNS domains
  rosa list dns-domains --hosted-cp

  # Output in JSON format
  rosa list dns-domains --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListDNSDomains(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.HostedCPOnly, "hosted-cp", false, "List only DNS domains for HCP clusters")
	flags.StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format (text, json, yaml)")

	return cmd
}

func runListDNSDomains(ctx context.Context, logger *slog.Logger, opts *ListOptions) error {
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

	// List domains
	domains, err := dnsSvc.List(ctx, opts.HostedCPOnly)
	if err != nil {
		return fmt.Errorf("failed to list DNS domains: %w", err)
	}

	// Display results based on output format
	var writer *output.Writer

	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		writer = output.NewWriter(output.FormatJSON)
		return writer.Print(domains)
	case "yaml":
		writer = output.NewWriter(output.FormatYAML)
		return writer.Print(domains)
	default:
		writer = output.NewWriter(output.FormatText)
		
		if len(domains) == 0 {
			writer.Info("No DNS domains found for your organization")
			writer.Info("\nTo create a DNS domain:")
			fmt.Println("  rosa create dns-domain --hosted-cp")
			return nil
		}

		writer.Title("DNS Domains")
		
		if opts.HostedCPOnly {
			writer.Info("Showing only HCP DNS domains\n")
		}
		
		// Display as table
		fmt.Printf("%-25s %-15s %-25s %-15s %-10s\n", 
			"DOMAIN ID", "ARCHITECTURE", "RESERVED TIME", "CLUSTER ID", "USER DEFINED")
		fmt.Println(strings.Repeat("-", 100))
		
		for _, domain := range domains {
			userDefined := "No"
			if domain.UserDefined {
				userDefined = "Yes"
			}
			
			clusterID := domain.ClusterID
			if clusterID == "" {
				clusterID = "-"
			}
			
			fmt.Printf("%-25s %-15s %-25s %-15s %-10s\n",
				truncate(domain.ID, 25),
				domain.Architecture,
				domain.ReservedAt,
				truncate(clusterID, 15),
				userDefined)
		}
		fmt.Println()

		// Summary
		writer.Info("Summary:")
		hcpCount := 0
		classicCount := 0
		for _, d := range domains {
			if d.Architecture == "HCP" {
				hcpCount++
			} else if d.Architecture == "Classic" {
				classicCount++
			}
		}
		
		writer.KeyValue(map[string]string{
			"Total Domains":   fmt.Sprintf("%d", len(domains)),
			"HCP Domains":     fmt.Sprintf("%d", hcpCount),
			"Classic Domains": fmt.Sprintf("%d", classicCount),
		})

		// Available commands
		writer.Info("\n📋 Available Commands:")
		writer.Info("  # Create a new DNS domain:")
		fmt.Println("  rosa create dns-domain --hosted-cp")
		
		if len(domains) > 0 {
			writer.Info("\n  # Delete a DNS domain:")
			fmt.Println("  rosa delete dns-domain <domain-id>")
		}
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
