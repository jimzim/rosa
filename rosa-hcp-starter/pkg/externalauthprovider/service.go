package externalauthprovider

import (
	"context"
	"fmt"
	"log/slog"

	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// Service provides External Authentication Provider operations
type Service interface {
	Create(ctx context.Context, clusterID string, config Config) (*ExternalAuthProvider, error)
	List(ctx context.Context, clusterID string) ([]*ExternalAuthProvider, error)
	Get(ctx context.Context, clusterID, providerName string) (*ExternalAuthProvider, error)
	Delete(ctx context.Context, clusterID, providerName string) error
}

type service struct {
	logger *slog.Logger
	ocm    *sdk.Connection
}

// NewService creates a new External Authentication Provider service
func NewService(ctx context.Context, logger *slog.Logger, ocm *sdk.Connection) (Service, error) {
	return &service{
		logger: logger,
		ocm:    ocm,
	}, nil
}

// Config holds external auth provider configuration
type Config struct {
	Name      string
	IssuerURL string
	ClientID  string
	ClientSecret string
	Claims    ClaimsConfig
}

// ClaimsConfig holds claim mappings
type ClaimsConfig struct {
	Username          string
	Email             string
	Name              string
	Groups            string
	PreferredUsername string
}

// ExternalAuthProvider represents an external authentication provider
type ExternalAuthProvider struct {
	Name      string
	IssuerURL string
	ClientID  string
	Claims    ClaimsConfig
}

// Create creates a new external auth provider
func (s *service) Create(ctx context.Context, clusterID string, config Config) (*ExternalAuthProvider, error) {
	s.logger.InfoContext(ctx, "creating external auth provider",
		slog.String("cluster", clusterID),
		slog.String("name", config.Name))

	// Get cluster to verify it exists and supports external auth
	clusterResp, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Get().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	cluster := clusterResp.Body()
	
	// Check if HCP cluster
	if cluster.Hypershift() == nil || !cluster.Hypershift().Enabled() {
		return nil, fmt.Errorf("external auth providers are only supported for HCP clusters")
	}
	
	// Check if external auth is enabled
	if cluster.ExternalAuthConfig() == nil || !cluster.ExternalAuthConfig().Enabled() {
		return nil, fmt.Errorf("external authentication is not enabled for this cluster. Create cluster with --external-auth-providers-enabled")
	}

	// Build the external auth provider (NOT external auth config)
	// This matches what the original ROSA CLI does
	externalAuthBuilder := cmv1.NewExternalAuth()
	externalAuthBuilder.ID(config.Name)
	
	// Set issuer with audiences
	tokenIssuerBuilder := cmv1.NewTokenIssuer()
	tokenIssuerBuilder.URL(config.IssuerURL)
	if config.ClientID != "" {
		tokenIssuerBuilder.Audiences(config.ClientID)
	}
	externalAuthBuilder.Issuer(tokenIssuerBuilder)
	
	// Set claim mappings if provided
	if config.Claims.Username != "" || config.Claims.Groups != "" {
		claimMappingsBuilder := cmv1.NewExternalAuthClaim()
		
		// Map username claim
		if config.Claims.Username != "" {
			usernameMappingBuilder := cmv1.NewUsernameClaim()
			usernameMappingBuilder.Claim(config.Claims.Username)
			usernameMappingBuilder.PrefixPolicy("") // No prefix by default
			claimMappingsBuilder.Mappings(cmv1.NewTokenClaimMappings().UserName(usernameMappingBuilder))
		}
		
		// Map groups claim
		if config.Claims.Groups != "" {
			groupsMappingBuilder := cmv1.NewGroupsClaim()
			groupsMappingBuilder.Claim(config.Claims.Groups)
			claimMappingsBuilder.Mappings(cmv1.NewTokenClaimMappings().Groups(groupsMappingBuilder))
		}
		
		externalAuthBuilder.Claim(claimMappingsBuilder)
	}
	
	// Set console client if provided
	if config.ClientID != "" && config.ClientSecret != "" {
		clientConfigBuilder := cmv1.NewExternalAuthClientConfig()
		clientConfigBuilder.ID(config.ClientID)
		// Note: ClientSecret might need different handling
		clientConfigBuilder.Secret(config.ClientSecret)
		externalAuthBuilder.Clients(clientConfigBuilder)
	}

	externalAuth, err := externalAuthBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build external auth: %w", err)
	}

	// Add the external auth provider to the cluster's external auth config
	// This is the correct API path according to the original ROSA CLI
	resp, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		ExternalAuthConfig().
		ExternalAuths().
		Add().
		Body(externalAuth).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create external auth provider: %w", err)
	}

	createdAuth := resp.Body()
	s.logger.InfoContext(ctx, "external auth provider created successfully")

	return &ExternalAuthProvider{
		Name:      createdAuth.ID(),
		IssuerURL: config.IssuerURL,
		ClientID:  config.ClientID,
		Claims:    config.Claims,
	}, nil
}

// List lists all external auth providers for a cluster
func (s *service) List(ctx context.Context, clusterID string) ([]*ExternalAuthProvider, error) {
	s.logger.InfoContext(ctx, "listing external auth providers",
		slog.String("cluster", clusterID))

	// List all external auth providers from the correct API path
	resp, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		ExternalAuthConfig().
		ExternalAuths().
		List().
		Page(1).
		Size(-1).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list external auth providers: %w", err)
	}

	items := resp.Items()
	var providers []*ExternalAuthProvider
	
	items.Each(func(auth *cmv1.ExternalAuth) bool {
		provider := &ExternalAuthProvider{
			Name: auth.ID(),
		}
		
		// Extract issuer info
		if auth.Issuer() != nil {
			provider.IssuerURL = auth.Issuer().URL()
			if len(auth.Issuer().Audiences()) > 0 {
				provider.ClientID = auth.Issuer().Audiences()[0]
			}
		}
		
		// Extract claim mappings
		if auth.Claim() != nil && auth.Claim().Mappings() != nil {
			mappings := auth.Claim().Mappings()
			if mappings.UserName() != nil {
				provider.Claims.Username = mappings.UserName().Claim()
			}
			if mappings.Groups() != nil {
				provider.Claims.Groups = mappings.Groups().Claim()
			}
		}
		
		providers = append(providers, provider)
		return true
	})

	s.logger.InfoContext(ctx, "found external auth providers", slog.Int("count", len(providers)))
	return providers, nil
}

// Get retrieves a specific external auth provider
func (s *service) Get(ctx context.Context, clusterID, providerName string) (*ExternalAuthProvider, error) {
	s.logger.InfoContext(ctx, "getting external auth provider",
		slog.String("cluster", clusterID),
		slog.String("provider", providerName))

	// Get the specific external auth provider from the correct API path
	resp, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		ExternalAuthConfig().
		ExternalAuths().
		ExternalAuth(providerName).
		Get().
		SendContext(ctx)
	if err != nil {
		if resp != nil && resp.Status() == 404 {
			return nil, fmt.Errorf("external auth provider '%s' not found", providerName)
		}
		return nil, fmt.Errorf("failed to get external auth provider: %w", err)
	}

	auth := resp.Body()
	provider := &ExternalAuthProvider{
		Name: auth.ID(),
	}
	
	// Extract issuer info
	if auth.Issuer() != nil {
		provider.IssuerURL = auth.Issuer().URL()
		if len(auth.Issuer().Audiences()) > 0 {
			provider.ClientID = auth.Issuer().Audiences()[0]
		}
	}
	
	// Extract claim mappings
	if auth.Claim() != nil && auth.Claim().Mappings() != nil {
		mappings := auth.Claim().Mappings()
		if mappings.UserName() != nil {
			provider.Claims.Username = mappings.UserName().Claim()
		}
		if mappings.Groups() != nil {
			provider.Claims.Groups = mappings.Groups().Claim()
		}
	}

	return provider, nil
}

// Delete deletes an external auth provider
func (s *service) Delete(ctx context.Context, clusterID, providerName string) error {
	s.logger.InfoContext(ctx, "deleting external auth provider",
		slog.String("cluster", clusterID),
		slog.String("provider", providerName))

	// Delete the specific external auth provider using the correct API path
	resp, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		ExternalAuthConfig().
		ExternalAuths().
		ExternalAuth(providerName).
		Delete().
		SendContext(ctx)
	if err != nil {
		if resp != nil && resp.Status() == 404 {
			return fmt.Errorf("external auth provider '%s' not found", providerName)
		}
		return fmt.Errorf("failed to delete external auth provider: %w", err)
	}

	s.logger.InfoContext(ctx, "external auth provider deleted successfully")
	return nil
}
