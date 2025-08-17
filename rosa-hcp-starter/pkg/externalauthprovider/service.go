package externalauthprovider

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// Service provides External Auth Provider operations
type Service interface {
	Create(ctx context.Context, clusterID string, config Config) (*ExternalAuthProvider, error)
	List(ctx context.Context, clusterID string) ([]*ExternalAuthProvider, error)
	Get(ctx context.Context, clusterID, providerName string) (*ExternalAuthProvider, error)
	Delete(ctx context.Context, clusterID, providerName string) error
	IsSupported(ctx context.Context, clusterID string) (bool, error)
}

type service struct {
	logger *slog.Logger
	ocm    *sdk.Connection
}

// NewService creates a new External Auth Provider service
func NewService(ctx context.Context, logger *slog.Logger, ocm *sdk.Connection) (Service, error) {
	return &service{
		logger: logger,
		ocm:    ocm,
	}, nil
}

// Config holds external auth provider configuration
type Config struct {
	Name         string
	IssuerURL    string
	ClientID     string
	ClientSecret string // Optional
	// Issuer audiences (optional)
	IssuerAudiences []string
	// CA certificate (optional)
	IssuerCAFile string
	// Claim mappings
	ClaimMappings *ClaimMappings
	// Console client ID for web UI authentication
	ConsoleClientID string
	// Console client secret (optional)
	ConsoleClientSecret string
}

// ClaimMappings holds claim mapping configuration
type ClaimMappings struct {
	Username          *ClaimMapping
	Email             *ClaimMapping
	Name              *ClaimMapping
	Groups            *ClaimMapping
	PreferredUsername *ClaimMapping
}

// ClaimMapping represents a single claim mapping
type ClaimMapping struct {
	Claim  string
	Prefix string
}

// ExternalAuthProvider represents an external authentication provider
type ExternalAuthProvider struct {
	ID              string
	Name            string
	IssuerURL       string
	ClientID        string
	IssuerAudiences []string
	ClaimMappings   *ClaimMappings
	ConsoleClientID string
}

// Create creates a new external auth provider
func (s *service) Create(ctx context.Context, clusterID string, config Config) (*ExternalAuthProvider, error) {
	s.logger.InfoContext(ctx, "creating external auth provider",
		slog.String("cluster_id", clusterID),
		slog.String("name", config.Name),
		slog.String("issuer_url", config.IssuerURL))

	// Validate issuer URL
	if err := validateIssuerURL(config.IssuerURL); err != nil {
		return nil, fmt.Errorf("invalid issuer URL: %w", err)
	}

	// Build the external auth config
	builder := cmv1.NewExternalAuthConfig().
		ID(config.Name)

	// Build external auth builder
	externalAuthBuilder := cmv1.NewExternalAuth().
		Issuer(cmv1.NewTokenIssuer().
			URL(config.IssuerURL).
			Audiences(config.IssuerAudiences...))

	// Add claim mappings if provided
	if config.ClaimMappings != nil {
		claimsBuilder := cmv1.NewExternalAuthClaim()

		if config.ClaimMappings.Username != nil {
			claimsBuilder.Username(cmv1.NewUsernameClaimMapping().
				Claim(config.ClaimMappings.Username.Claim).
				PrefixPolicy("add").
				Prefix(config.ClaimMappings.Username.Prefix))
		}

		if config.ClaimMappings.Email != nil {
			claimsBuilder.Email(cmv1.NewEmailClaimMapping().
				Claim(config.ClaimMappings.Email.Claim))
		}

		if config.ClaimMappings.Name != nil {
			claimsBuilder.Name(cmv1.NewNameClaimMapping().
				Claim(config.ClaimMappings.Name.Claim))
		}

		if config.ClaimMappings.Groups != nil {
			claimsBuilder.Groups(cmv1.NewGroupsClaimMapping().
				Claim(config.ClaimMappings.Groups.Claim))
		}

		if config.ClaimMappings.PreferredUsername != nil {
			claimsBuilder.PreferredUsername(cmv1.NewPreferredUsernameClaimMapping().
				Claim(config.ClaimMappings.PreferredUsername.Claim))
		}

		externalAuthBuilder.Claim(claimsBuilder)
	}

	// Add clients
	clientsBuilder := cmv1.NewExternalAuthClientConfig()

	// Component client
	componentBuilder := cmv1.NewAuthClientComponent().
		ClientID(config.ClientID)
	if config.ClientSecret != "" {
		componentBuilder.ClientSecret(config.ClientSecret)
	}
	clientsBuilder.Component(componentBuilder)

	// Console client if provided
	if config.ConsoleClientID != "" {
		consoleBuilder := cmv1.NewAuthClientComponent().
			ClientID(config.ConsoleClientID)
		if config.ConsoleClientSecret != "" {
			consoleBuilder.ClientSecret(config.ConsoleClientSecret)
		}
		clientsBuilder.Component(consoleBuilder)
	}

	externalAuthBuilder.Clients(clientsBuilder)
	builder.ExternalAuths(externalAuthBuilder)

	externalAuthConfig, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build external auth config: %w", err)
	}

	// Create via API
	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		ExternalConfiguration().
		ExternalAuths().
		Add().Body(externalAuthConfig).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create external auth provider: %w", err)
	}

	result := convertExternalAuthConfig(response.Body())
	s.logger.InfoContext(ctx, "external auth provider created successfully",
		slog.String("name", result.Name))

	return result, nil
}

// List lists all external auth providers for a cluster
func (s *service) List(ctx context.Context, clusterID string) ([]*ExternalAuthProvider, error) {
	s.logger.InfoContext(ctx, "listing external auth providers", slog.String("cluster_id", clusterID))

	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		ExternalConfiguration().
		ExternalAuths().
		List().
		Page(1).
		Size(100).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list external auth providers: %w", err)
	}

	providers := make([]*ExternalAuthProvider, 0, response.Total())
	response.Items().Each(func(config *cmv1.ExternalAuthConfig) bool {
		providers = append(providers, convertExternalAuthConfig(config))
		return true
	})

	s.logger.InfoContext(ctx, "found external auth providers", slog.Int("count", len(providers)))
	return providers, nil
}

// Get retrieves a specific external auth provider
func (s *service) Get(ctx context.Context, clusterID, providerName string) (*ExternalAuthProvider, error) {
	s.logger.InfoContext(ctx, "getting external auth provider",
		slog.String("cluster_id", clusterID),
		slog.String("provider", providerName))

	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		ExternalConfiguration().
		ExternalAuths().
		ExternalAuth(providerName).
		Get().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get external auth provider: %w", err)
	}

	return convertExternalAuthConfig(response.Body()), nil
}

// Delete deletes an external auth provider
func (s *service) Delete(ctx context.Context, clusterID, providerName string) error {
	s.logger.InfoContext(ctx, "deleting external auth provider",
		slog.String("cluster_id", clusterID),
		slog.String("provider", providerName))

	_, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		ExternalConfiguration().
		ExternalAuths().
		ExternalAuth(providerName).
		Delete().
		SendContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete external auth provider: %w", err)
	}

	s.logger.InfoContext(ctx, "external auth provider deleted successfully",
		slog.String("provider", providerName))
	return nil
}

// IsSupported checks if external auth is supported for the cluster
func (s *service) IsSupported(ctx context.Context, clusterID string) (bool, error) {
	s.logger.InfoContext(ctx, "checking external auth support", slog.String("cluster_id", clusterID))

	// Get cluster details
	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Get().
		SendContext(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get cluster: %w", err)
	}

	cluster := response.Body()

	// External auth is only supported for HCP clusters
	if !cluster.Hypershift().Enabled() {
		return false, nil
	}

	// Check if external auth is already configured
	if cluster.ExternalAuthConfig() != nil && cluster.ExternalAuthConfig().Enabled() {
		s.logger.InfoContext(ctx, "external auth already configured for cluster")
	}

	return true, nil
}

// Helper functions

func validateIssuerURL(issuerURL string) error {
	if issuerURL == "" {
		return fmt.Errorf("issuer URL is required")
	}

	u, err := url.Parse(issuerURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if u.Scheme != "https" {
		return fmt.Errorf("issuer URL must use HTTPS")
	}

	if u.Host == "" {
		return fmt.Errorf("issuer URL must have a valid host")
	}

	// URL should not end with a trailing slash
	if strings.HasSuffix(issuerURL, "/") {
		return fmt.Errorf("issuer URL should not end with a trailing slash")
	}

	return nil
}

func convertExternalAuthConfig(config *cmv1.ExternalAuthConfig) *ExternalAuthProvider {
	if config == nil {
		return nil
	}

	provider := &ExternalAuthProvider{
		ID:   config.ID(),
		Name: config.ID(), // ID and Name are the same in the API
	}

	// Get the first external auth (there should only be one per config)
	if config.ExternalAuths() != nil && len(config.ExternalAuths()) > 0 {
		externalAuth := config.ExternalAuths()[0]

		if externalAuth.Issuer() != nil {
			provider.IssuerURL = externalAuth.Issuer().URL()
			provider.IssuerAudiences = externalAuth.Issuer().Audiences()
		}

		if externalAuth.Clients() != nil && externalAuth.Clients().Component() != nil {
			provider.ClientID = externalAuth.Clients().Component().ClientID()
		}

		// Convert claim mappings
		if externalAuth.Claim() != nil {
			provider.ClaimMappings = &ClaimMappings{}

			if externalAuth.Claim().Username() != nil {
				provider.ClaimMappings.Username = &ClaimMapping{
					Claim:  externalAuth.Claim().Username().Claim(),
					Prefix: externalAuth.Claim().Username().Prefix(),
				}
			}

			if externalAuth.Claim().Email() != nil {
				provider.ClaimMappings.Email = &ClaimMapping{
					Claim: externalAuth.Claim().Email().Claim(),
				}
			}

			if externalAuth.Claim().Name() != nil {
				provider.ClaimMappings.Name = &ClaimMapping{
					Claim: externalAuth.Claim().Name().Claim(),
				}
			}

			if externalAuth.Claim().Groups() != nil {
				provider.ClaimMappings.Groups = &ClaimMapping{
					Claim: externalAuth.Claim().Groups().Claim(),
				}
			}

			if externalAuth.Claim().PreferredUsername() != nil {
				provider.ClaimMappings.PreferredUsername = &ClaimMapping{
					Claim: externalAuth.Claim().PreferredUsername().Claim(),
				}
			}
		}
	}

	return provider
}

// FormatClaimMappings formats claim mappings for display
func FormatClaimMappings(mappings *ClaimMappings) string {
	if mappings == nil {
		return "Default mappings"
	}

	var parts []string
	if mappings.Username != nil {
		parts = append(parts, fmt.Sprintf("username:%s", mappings.Username.Claim))
	}
	if mappings.Email != nil {
		parts = append(parts, fmt.Sprintf("email:%s", mappings.Email.Claim))
	}
	if mappings.Groups != nil {
		parts = append(parts, fmt.Sprintf("groups:%s", mappings.Groups.Claim))
	}

	if len(parts) == 0 {
		return "Default mappings"
	}

	return strings.Join(parts, ", ")
}
