package externalauthprovider

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/externalauthprovider"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// ListOptions contains options for listing external auth providers
type ListOptions struct {
	ClusterName  string
	OutputFormat string
}

// NewListCommand creates the external-auth-provider list command
func NewListCommand(logger *slog.Logger) *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:     "external-auth-providers",
		Aliases: []string{"external-auth-provider"},
		Short:   "List external authentication providers for a cluster",
		Long:    "List all external authentication providers configured for a cluster.",
		Example: `  # List all external auth providers for a cluster
  rosa list external-auth-providers --cluster my-cluster

  # Output in JSON format
  rosa list external-auth-providers --cluster my-cluster --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListExternalAuth(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format (text, json, yaml)")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runListExternalAuth(ctx context.Context, logger *slog.Logger, opts *ListOptions) error {
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

	// Check if HCP cluster
	if cluster.Hypershift() == nil || !cluster.Hypershift().Enabled() {
		return fmt.Errorf("external authentication is only supported for HCP clusters")
	}
	
	// Check if external auth is enabled
	if cluster.ExternalAuthConfig() == nil || !cluster.ExternalAuthConfig().Enabled() {
		writer := output.NewWriter(output.FormatText)
		writer.Warning("External authentication is not enabled for this cluster")
		writer.Info("Create cluster with --external-auth-providers-enabled to enable this feature")
		return nil
	}

	// List providers
	providers, err := authSvc.List(ctx, cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to list external auth providers: %w", err)
	}

	// Display results based on output format
	var writer *output.Writer

	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		writer = output.NewWriter(output.FormatJSON)
		return writer.Print(providers)
	case "yaml":
		writer = output.NewWriter(output.FormatYAML)
		return writer.Print(providers)
	default:
		writer = output.NewWriter(output.FormatText)

		if len(providers) == 0 {
			writer.Info("No external authentication providers configured for cluster '%s'", opts.ClusterName)
			writer.Info("\nTo create an external auth provider:")
			fmt.Printf("  rosa create external-auth-provider --cluster %s\n", opts.ClusterName)
			return nil
		}

		writer.Title(fmt.Sprintf("External Auth Providers for cluster '%s'", cluster.Name()))

		// Check if external auth is enabled on the cluster
		if cluster.ExternalAuthConfig() != nil && cluster.ExternalAuthConfig().Enabled() {
			writer.Success("External authentication is ENABLED for this cluster")
		} else {
			writer.Warning("External authentication is configured but NOT YET ENABLED")
		}
		fmt.Println()

		// Display as table
		fmt.Printf("%-20s %-50s %-30s\n", "NAME", "ISSUER URL", "CLIENT ID")
		fmt.Println(strings.Repeat("-", 100))

		for _, provider := range providers {
			fmt.Printf("%-20s %-50s %-30s\n",
				truncate(provider.Name, 20),
				truncate(provider.IssuerURL, 50),
				truncate(provider.ClientID, 30))

			// Show additional details if present  
			hasClaimMappings := provider.Claims.Username != "" || provider.Claims.Email != "" || 
			                   provider.Claims.Groups != "" || provider.Claims.Name != "" || 
			                   provider.Claims.PreferredUsername != ""
			if hasClaimMappings {
				claimMappings := []string{}
				if provider.Claims.Username != "" {
					claimMappings = append(claimMappings, fmt.Sprintf("username=%s", provider.Claims.Username))
				}
				if provider.Claims.Groups != "" {
					claimMappings = append(claimMappings, fmt.Sprintf("groups=%s", provider.Claims.Groups))
				}
				if len(claimMappings) > 0 {
					fmt.Printf("  Claim Mappings: %s\n", strings.Join(claimMappings, ", "))
				}
			}
		}
		fmt.Println()

		// Show helpful commands
		writer.Info("To view details of a specific provider:")
		fmt.Printf("  rosa describe external-auth-provider <name> --cluster %s\n", opts.ClusterName)

		writer.Info("\nTo delete a provider:")
		fmt.Printf("  rosa delete external-auth-provider <name> --cluster %s\n", opts.ClusterName)

		if len(providers) > 0 {
			writer.Info("\n⚠️  Remember to configure break-glass credentials for emergency access:")
			fmt.Printf("  rosa create break-glass-credential --cluster %s\n", opts.ClusterName)
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
