package externalauthprovider

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/externalauthprovider"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// CreateOptions contains options for creating an external auth provider
type CreateOptions struct {
	ClusterName         string
	Name                string
	IssuerURL           string
	ClientID            string
	ClientSecret        string
	IssuerAudiences     []string
	ConsoleClientID     string
	ConsoleClientSecret string
	ClaimMappings       map[string]string
	Interactive         bool
}

// NewCreateCommand creates the external-auth-provider create command
func NewCreateCommand(logger *slog.Logger) *cobra.Command {
	opts := &CreateOptions{
		ClaimMappings: make(map[string]string),
	}

	cmd := &cobra.Command{
		Use:   "external-auth-provider",
		Short: "Create an external authentication provider for a cluster",
		Long: `Configure a cluster to use an external OIDC authentication provider 
instead of the internal OpenShift OAuth server.

This enables single sign-on (SSO) with external identity providers like
Azure AD, Okta, Keycloak, or any OIDC-compliant provider.

Note: This feature is only supported for HCP (Hosted Control Plane) clusters.`,
		Example: `  # Create an external auth provider interactively
  rosa create external-auth-provider --cluster my-cluster --interactive

  # Create with specific configuration
  rosa create external-auth-provider --cluster my-cluster \
    --name my-sso \
    --issuer-url https://auth.example.com/realms/myrealm \
    --client-id openshift \
    --client-secret <secret> \
    --console-client-id console`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateExternalAuth(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVar(&opts.Name, "name", "", "Name for the external auth provider")
	flags.StringVar(&opts.IssuerURL, "issuer-url", "", "OIDC issuer URL (e.g., https://auth.example.com/realms/myrealm)")
	flags.StringVar(&opts.ClientID, "client-id", "", "OIDC client ID for the cluster")
	flags.StringVar(&opts.ClientSecret, "client-secret", "", "OIDC client secret (optional)")
	flags.StringSliceVar(&opts.IssuerAudiences, "issuer-audiences", nil, "Valid audiences for the issuer (optional)")
	flags.StringVar(&opts.ConsoleClientID, "console-client-id", "", "OIDC client ID for the web console")
	flags.StringVar(&opts.ConsoleClientSecret, "console-client-secret", "", "OIDC client secret for the console (optional)")
	flags.StringToStringVar(&opts.ClaimMappings, "claim-mappings", nil, "Claim mappings (e.g., username=preferred_username)")
	flags.BoolVarP(&opts.Interactive, "interactive", "i", false, "Interactive mode")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runCreateExternalAuth(ctx context.Context, logger *slog.Logger, opts *CreateOptions) error {
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

	// Get cluster to verify it exists and is HCP
	clusterResp, err := apiClient.GetCluster(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	cluster := clusterResp.Body()

	// Check if cluster is HCP
	if !cluster.Hypershift().Enabled() {
		return fmt.Errorf("external authentication providers are only supported for HCP clusters")
	}

	// Check if external auth is already configured
	if cluster.ExternalAuthConfig() != nil && cluster.ExternalAuthConfig().Enabled() {
		writer.Warning("External authentication is already configured for this cluster")
		writer.Info("You can create additional providers or modify existing ones")
	}

	// Interactive mode
	if opts.Interactive || (opts.Name == "" && opts.IssuerURL == "" && opts.ClientID == "") {
		if !opts.Interactive {
			opts.Interactive = true
			writer.Info("Enabling interactive mode")
		}

		// Provider name
		if opts.Name == "" {
			err = huh.NewInput().
				Title("Provider name").
				Description("Unique name for this external auth provider").
				Value(&opts.Name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name cannot be empty")
					}
					if !isValidProviderName(s) {
						return fmt.Errorf("name must contain only lowercase letters, numbers, and hyphens")
					}
					return nil
				}).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get provider name: %w", err)
			}
		}

		// Issuer URL
		if opts.IssuerURL == "" {
			err = huh.NewInput().
				Title("OIDC Issuer URL").
				Description("The OIDC provider's issuer URL (must use HTTPS)").
				Placeholder("https://auth.example.com/realms/myrealm").
				Value(&opts.IssuerURL).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("issuer URL cannot be empty")
					}
					if !strings.HasPrefix(s, "https://") {
						return fmt.Errorf("issuer URL must use HTTPS")
					}
					if strings.HasSuffix(s, "/") {
						return fmt.Errorf("issuer URL should not end with /")
					}
					return nil
				}).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get issuer URL: %w", err)
			}
		}

		// Client ID
		if opts.ClientID == "" {
			err = huh.NewInput().
				Title("Client ID").
				Description("OIDC client ID for cluster authentication").
				Value(&opts.ClientID).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("client ID cannot be empty")
					}
					return nil
				}).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get client ID: %w", err)
			}
		}

		// Client Secret (optional)
		if opts.ClientSecret == "" {
			var useSecret bool
			err = huh.NewConfirm().
				Title("Configure client secret?").
				Description("Some OIDC providers require a client secret").
				Value(&useSecret).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get secret option: %w", err)
			}

			if useSecret {
				err = huh.NewInput().
					Title("Client Secret").
					Description("OIDC client secret (will be stored securely)").
					Value(&opts.ClientSecret).
					Run()
				if err != nil {
					return fmt.Errorf("failed to get client secret: %w", err)
				}
			}
		}

		// Console client configuration
		var configureConsole bool
		err = huh.NewConfirm().
			Title("Configure web console authentication?").
			Description("Set up separate OIDC client for the OpenShift web console").
			Value(&configureConsole).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get console option: %w", err)
		}

		if configureConsole {
			err = huh.NewInput().
				Title("Console Client ID").
				Description("OIDC client ID for web console authentication").
				Value(&opts.ConsoleClientID).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get console client ID: %w", err)
			}

			if opts.ConsoleClientID != "" {
				err = huh.NewInput().
					Title("Console Client Secret (optional)").
					Description("OIDC client secret for console (leave empty if not required)").
					Value(&opts.ConsoleClientSecret).
					Run()
				if err != nil {
					return fmt.Errorf("failed to get console client secret: %w", err)
				}
			}
		}

		// Claim mappings
		var customizeMappings bool
		err = huh.NewConfirm().
			Title("Customize claim mappings?").
			Description("Map OIDC claims to OpenShift user attributes").
			Value(&customizeMappings).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get mappings option: %w", err)
		}

		if customizeMappings {
			writer.Info("Enter claim names from your OIDC provider (leave empty to use defaults)")

			var usernameClaim string
			err = huh.NewInput().
				Title("Username claim").
				Placeholder("preferred_username").
				Value(&usernameClaim).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get username claim: %w", err)
			}
			if usernameClaim != "" {
				opts.ClaimMappings["username"] = usernameClaim
			}

			var emailClaim string
			err = huh.NewInput().
				Title("Email claim").
				Placeholder("email").
				Value(&emailClaim).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get email claim: %w", err)
			}
			if emailClaim != "" {
				opts.ClaimMappings["email"] = emailClaim
			}

			var groupsClaim string
			err = huh.NewInput().
				Title("Groups claim").
				Placeholder("groups").
				Value(&groupsClaim).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get groups claim: %w", err)
			}
			if groupsClaim != "" {
				opts.ClaimMappings["groups"] = groupsClaim
			}
		}
	}

	// Validate required fields
	if opts.Name == "" {
		return fmt.Errorf("provider name is required")
	}
	if opts.IssuerURL == "" {
		return fmt.Errorf("issuer URL is required")
	}
	if opts.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}

	// Create External Auth Provider service
	authSvc, err := externalauthprovider.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create external auth service: %w", err)
	}

	// Check if supported
	supported, err := authSvc.IsSupported(ctx, cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to check external auth support: %w", err)
	}
	if !supported {
		return fmt.Errorf("external authentication is not supported for this cluster")
	}

	// Prepare configuration
	authConfig := externalauthprovider.Config{
		Name:                opts.Name,
		IssuerURL:           opts.IssuerURL,
		ClientID:            opts.ClientID,
		ClientSecret:        opts.ClientSecret,
		IssuerAudiences:     opts.IssuerAudiences,
		ConsoleClientID:     opts.ConsoleClientID,
		ConsoleClientSecret: opts.ConsoleClientSecret,
	}

	// Add claim mappings if provided
	if len(opts.ClaimMappings) > 0 {
		authConfig.ClaimMappings = &externalauthprovider.ClaimMappings{}

		if username, ok := opts.ClaimMappings["username"]; ok {
			authConfig.ClaimMappings.Username = &externalauthprovider.ClaimMapping{
				Claim: username,
			}
		}
		if email, ok := opts.ClaimMappings["email"]; ok {
			authConfig.ClaimMappings.Email = &externalauthprovider.ClaimMapping{
				Claim: email,
			}
		}
		if groups, ok := opts.ClaimMappings["groups"]; ok {
			authConfig.ClaimMappings.Groups = &externalauthprovider.ClaimMapping{
				Claim: groups,
			}
		}
		if name, ok := opts.ClaimMappings["name"]; ok {
			authConfig.ClaimMappings.Name = &externalauthprovider.ClaimMapping{
				Claim: name,
			}
		}
		if preferredUsername, ok := opts.ClaimMappings["preferred_username"]; ok {
			authConfig.ClaimMappings.PreferredUsername = &externalauthprovider.ClaimMapping{
				Claim: preferredUsername,
			}
		}
	}

	// Display configuration summary
	writer.Title("Creating External Authentication Provider")
	writer.KeyValue(map[string]string{
		"Cluster":    cluster.Name(),
		"Name":       authConfig.Name,
		"Issuer URL": authConfig.IssuerURL,
		"Client ID":  authConfig.ClientID,
	})

	if authConfig.ConsoleClientID != "" {
		writer.KeyValue(map[string]string{
			"Console Client ID": authConfig.ConsoleClientID,
		})
	}

	if authConfig.ClaimMappings != nil {
		writer.KeyValue(map[string]string{
			"Claim Mappings": externalauthprovider.FormatClaimMappings(authConfig.ClaimMappings),
		})
	}

	// Create the external auth provider
	writer.Info("Creating external authentication provider...")
	provider, err := authSvc.Create(ctx, cluster.ID(), authConfig)
	if err != nil {
		return fmt.Errorf("failed to create external auth provider: %w", err)
	}

	writer.Success("External authentication provider '%s' created successfully", provider.Name)

	writer.Info("\n⚠️  Important Next Steps:")
	writer.Info("1. Configure your OIDC provider with the correct redirect URIs:")
	fmt.Printf("   - https://oauth-%s.%s/oauth2callback/%s\n",
		cluster.Name(), cluster.DNS().BaseDomain(), provider.Name)
	if provider.ConsoleClientID != "" {
		fmt.Printf("   - https://console-%s.%s/auth/callback\n",
			cluster.Name(), cluster.DNS().BaseDomain())
	}

	writer.Info("\n2. The change may take a few minutes to propagate")

	writer.Info("\n3. Once active, users will authenticate via your external provider")

	writer.Info("\n4. To enable break-glass credentials for emergency access:")
	fmt.Printf("   rosa create break-glass-credential --cluster %s\n", opts.ClusterName)

	writer.Info("\nTo view the provider details:")
	fmt.Printf("  rosa describe external-auth-provider %s --cluster %s\n", provider.Name, opts.ClusterName)

	return nil
}

func isValidProviderName(name string) bool {
	if name == "" {
		return false
	}
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return false
		}
	}
	return true
}
