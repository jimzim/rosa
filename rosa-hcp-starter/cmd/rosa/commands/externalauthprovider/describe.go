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

// DescribeOptions contains options for describing an external auth provider
type DescribeOptions struct {
	ClusterName  string
	ProviderName string
	OutputFormat string
}

// NewDescribeCommand creates the external-auth-provider describe command
func NewDescribeCommand(logger *slog.Logger) *cobra.Command {
	opts := &DescribeOptions{}

	cmd := &cobra.Command{
		Use:   "external-auth-provider PROVIDER_NAME",
		Short: "Describe an external authentication provider",
		Long:  "Show detailed information about a specific external authentication provider.",
		Example: `  # Describe an external auth provider
  rosa describe external-auth-provider my-sso --cluster my-cluster

  # Output in JSON format
  rosa describe external-auth-provider my-sso --cluster my-cluster --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ProviderName = args[0]
			return runDescribeExternalAuth(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format (text, json, yaml)")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runDescribeExternalAuth(ctx context.Context, logger *slog.Logger, opts *DescribeOptions) error {
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

	// Get the provider
	provider, err := authSvc.Get(ctx, cluster.ID(), opts.ProviderName)
	if err != nil {
		return fmt.Errorf("failed to get external auth provider '%s': %w", opts.ProviderName, err)
	}

	// Display results based on output format
	var writer *output.Writer

	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		writer = output.NewWriter(output.FormatJSON)
		return writer.Print(provider)
	case "yaml":
		writer = output.NewWriter(output.FormatYAML)
		return writer.Print(provider)
	default:
		writer = output.NewWriter(output.FormatText)

		writer.Title(fmt.Sprintf("External Auth Provider '%s'", provider.Name))

		// Basic information
		writer.KeyValue(map[string]string{
			"Cluster":     cluster.Name(),
			"Provider ID": provider.ID,
			"Name":        provider.Name,
		})

		// OIDC Configuration
		writer.Info("\nOIDC Configuration:")
		writer.KeyValue(map[string]string{
			"Issuer URL": provider.IssuerURL,
			"Client ID":  provider.ClientID,
		})

		if len(provider.IssuerAudiences) > 0 {
			writer.KeyValue(map[string]string{
				"Audiences": strings.Join(provider.IssuerAudiences, ", "),
			})
		}

		// Console configuration
		if provider.ConsoleClientID != "" {
			writer.Info("\nConsole Configuration:")
			writer.KeyValue(map[string]string{
				"Console Client ID": provider.ConsoleClientID,
			})
		}

		// Claim mappings
		writer.Info("\nClaim Mappings:")
		if provider.ClaimMappings != nil {
			if provider.ClaimMappings.Username != nil {
				writer.KeyValue(map[string]string{
					"Username": fmt.Sprintf("claim=%s", provider.ClaimMappings.Username.Claim),
				})
				if provider.ClaimMappings.Username.Prefix != "" {
					writer.KeyValue(map[string]string{
						"  Prefix": provider.ClaimMappings.Username.Prefix,
					})
				}
			}
			if provider.ClaimMappings.Email != nil {
				writer.KeyValue(map[string]string{
					"Email": fmt.Sprintf("claim=%s", provider.ClaimMappings.Email.Claim),
				})
			}
			if provider.ClaimMappings.Name != nil {
				writer.KeyValue(map[string]string{
					"Name": fmt.Sprintf("claim=%s", provider.ClaimMappings.Name.Claim),
				})
			}
			if provider.ClaimMappings.Groups != nil {
				writer.KeyValue(map[string]string{
					"Groups": fmt.Sprintf("claim=%s", provider.ClaimMappings.Groups.Claim),
				})
			}
			if provider.ClaimMappings.PreferredUsername != nil {
				writer.KeyValue(map[string]string{
					"Preferred Username": fmt.Sprintf("claim=%s", provider.ClaimMappings.PreferredUsername.Claim),
				})
			}
		} else {
			writer.Info("  Using default claim mappings")
		}

		// Status and redirect URIs
		writer.Info("\nIntegration Details:")

		if cluster.ExternalAuthConfig() != nil && cluster.ExternalAuthConfig().Enabled() {
			writer.Success("Status: ACTIVE")
		} else {
			writer.Warning("Status: CONFIGURED (may take a few minutes to activate)")
		}

		writer.Info("\nRequired Redirect URIs in your OIDC provider:")
		fmt.Printf("  • OAuth Callback:   https://oauth-%s.%s/oauth2callback/%s\n",
			cluster.Name(), cluster.DNS().BaseDomain(), provider.Name)

		if provider.ConsoleClientID != "" {
			fmt.Printf("  • Console Callback: https://console-%s.%s/auth/callback\n",
				cluster.Name(), cluster.DNS().BaseDomain())
		}

		// Well-known endpoints
		writer.Info("\nOIDC Discovery Endpoints:")
		fmt.Printf("  • Configuration: %s/.well-known/openid-configuration\n", provider.IssuerURL)
		fmt.Printf("  • JWKS:          %s/.well-known/jwks.json\n", provider.IssuerURL)

		// Actions
		writer.Info("\nAvailable Actions:")
		fmt.Printf("  • Edit:   rosa edit external-auth-provider %s --cluster %s\n",
			provider.Name, opts.ClusterName)
		fmt.Printf("  • Delete: rosa delete external-auth-provider %s --cluster %s\n",
			provider.Name, opts.ClusterName)

		// Recommendations
		writer.Info("\n💡 Recommendations:")
		writer.Info("  1. Ensure your OIDC provider has the correct redirect URIs configured")
		writer.Info("  2. Test authentication with a non-privileged user first")
		writer.Info("  3. Configure break-glass credentials for emergency access:")
		fmt.Printf("     rosa create break-glass-credential --cluster %s\n", opts.ClusterName)
	}

	return nil
}
