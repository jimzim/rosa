package idp

import (
	"context"
	"fmt"
	"log/slog"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa-hcp/pkg/api"
)

// Service provides identity provider operations
type Service struct {
	ocm    api.Client
	logger *slog.Logger
}

// NewService creates a new IdP service
func NewService(ocm api.Client, logger *slog.Logger) *Service {
	return &Service{
		ocm:    ocm,
		logger: logger.With("service", "idp"),
	}
}

// IdentityProvider represents an identity provider
type IdentityProvider struct {
	ID            string
	Name          string
	Type          string
	MappingMethod string
	ClusterID     string
	Challenge     bool
	Login         bool
}

// GitHubConfig contains GitHub IdP configuration
type GitHubConfig struct {
	ClientID      string
	ClientSecret  string
	Organizations []string
	Teams         []string
	Hostname      string
}

// GoogleConfig contains Google IdP configuration
type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	HostedDomain string
}

// GitLabConfig contains GitLab IdP configuration
type GitLabConfig struct {
	ClientID     string
	ClientSecret string
	URL          string
	CA           string
}

// LDAPConfig contains LDAP IdP configuration
type LDAPConfig struct {
	URL          string
	BindDN       string
	BindPassword string
	Insecure     bool
	CA           string
	Attributes   LDAPAttributes
}

// LDAPAttributes contains LDAP attribute mappings
type LDAPAttributes struct {
	ID                []string
	Email             []string
	Name              []string
	PreferredUsername []string
}

// CreateGitHubIDP creates a GitHub identity provider
func (s *Service) CreateGitHubIDP(ctx context.Context, clusterID, name string, config GitHubConfig) (*IdentityProvider, error) {
	s.logger.InfoContext(ctx, "creating GitHub IdP",
		slog.String("cluster", clusterID),
		slog.String("name", name))

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	// Build GitHub provider
	githubBuilder := cmv1.NewGithubIdentityProvider().
		ClientID(config.ClientID).
		ClientSecret(config.ClientSecret)

	if len(config.Organizations) > 0 {
		githubBuilder.Organizations(config.Organizations...)
	}
	if len(config.Teams) > 0 {
		githubBuilder.Teams(config.Teams...)
	}
	if config.Hostname != "" {
		githubBuilder.Hostname(config.Hostname)
	}

	// Build identity provider
	idpBuilder := cmv1.NewIdentityProvider().
		Type(cmv1.IdentityProviderTypeGithub).
		Name(name).
		MappingMethod(cmv1.IdentityProviderMappingMethodClaim).
		Github(githubBuilder)

	idp, err := idpBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build IdP: %w", err)
	}

	// Create the IdP
	response, err := conn.ClustersMgmt().V1().
		Clusters().
		Cluster(clusterID).
		IdentityProviders().
		Add().
		Body(idp).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create IdP: %w", err)
	}

	created := response.Body()

	s.logger.InfoContext(ctx, "GitHub IdP created successfully",
		slog.String("id", created.ID()),
		slog.String("name", created.Name()))

	return &IdentityProvider{
		ID:            created.ID(),
		Name:          created.Name(),
		Type:          "GitHub",
		MappingMethod: string(created.MappingMethod()),
		ClusterID:     clusterID,
		Challenge:     created.Challenge(),
		Login:         created.Login(),
	}, nil
}

// CreateGoogleIDP creates a Google identity provider
func (s *Service) CreateGoogleIDP(ctx context.Context, clusterID, name string, config GoogleConfig) (*IdentityProvider, error) {
	s.logger.InfoContext(ctx, "creating Google IdP",
		slog.String("cluster", clusterID),
		slog.String("name", name))

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	// Build Google provider
	googleBuilder := cmv1.NewGoogleIdentityProvider().
		ClientID(config.ClientID).
		ClientSecret(config.ClientSecret)

	if config.HostedDomain != "" {
		googleBuilder.HostedDomain(config.HostedDomain)
	}

	// Build identity provider
	idpBuilder := cmv1.NewIdentityProvider().
		Type(cmv1.IdentityProviderTypeGoogle).
		Name(name).
		MappingMethod(cmv1.IdentityProviderMappingMethodClaim).
		Google(googleBuilder)

	idp, err := idpBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build IdP: %w", err)
	}

	// Create the IdP
	response, err := conn.ClustersMgmt().V1().
		Clusters().
		Cluster(clusterID).
		IdentityProviders().
		Add().
		Body(idp).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create IdP: %w", err)
	}

	created := response.Body()

	s.logger.InfoContext(ctx, "Google IdP created successfully",
		slog.String("id", created.ID()),
		slog.String("name", created.Name()))

	return &IdentityProvider{
		ID:            created.ID(),
		Name:          created.Name(),
		Type:          "Google",
		MappingMethod: string(created.MappingMethod()),
		ClusterID:     clusterID,
		Challenge:     created.Challenge(),
		Login:         created.Login(),
	}, nil
}

// CreateGitLabIDP creates a GitLab identity provider
func (s *Service) CreateGitLabIDP(ctx context.Context, clusterID, name string, config GitLabConfig) (*IdentityProvider, error) {
	s.logger.InfoContext(ctx, "creating GitLab IdP",
		slog.String("cluster", clusterID),
		slog.String("name", name))

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	// Build GitLab provider
	gitlabBuilder := cmv1.NewGitlabIdentityProvider().
		ClientID(config.ClientID).
		ClientSecret(config.ClientSecret).
		URL(config.URL)

	if config.CA != "" {
		gitlabBuilder.CA(config.CA)
	}

	// Build identity provider
	idpBuilder := cmv1.NewIdentityProvider().
		Type(cmv1.IdentityProviderTypeGitlab).
		Name(name).
		MappingMethod(cmv1.IdentityProviderMappingMethodClaim).
		Gitlab(gitlabBuilder)

	idp, err := idpBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build IdP: %w", err)
	}

	// Create the IdP
	response, err := conn.ClustersMgmt().V1().
		Clusters().
		Cluster(clusterID).
		IdentityProviders().
		Add().
		Body(idp).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create IdP: %w", err)
	}

	created := response.Body()

	s.logger.InfoContext(ctx, "GitLab IdP created successfully",
		slog.String("id", created.ID()),
		slog.String("name", created.Name()))

	return &IdentityProvider{
		ID:            created.ID(),
		Name:          created.Name(),
		Type:          "GitLab",
		MappingMethod: string(created.MappingMethod()),
		ClusterID:     clusterID,
		Challenge:     created.Challenge(),
		Login:         created.Login(),
	}, nil
}

// List lists all identity providers for a cluster
func (s *Service) List(ctx context.Context, clusterID string) ([]*IdentityProvider, error) {
	s.logger.InfoContext(ctx, "listing IdPs",
		slog.String("cluster", clusterID))

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	response, err := conn.ClustersMgmt().V1().
		Clusters().
		Cluster(clusterID).
		IdentityProviders().
		List().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list IdPs: %w", err)
	}

	var idps []*IdentityProvider
	response.Items().Each(func(idp *cmv1.IdentityProvider) bool {
		provider := &IdentityProvider{
			ID:            idp.ID(),
			Name:          idp.Name(),
			Type:          string(idp.Type()),
			MappingMethod: string(idp.MappingMethod()),
			ClusterID:     clusterID,
			Challenge:     idp.Challenge(),
			Login:         idp.Login(),
		}
		idps = append(idps, provider)
		return true
	})

	s.logger.InfoContext(ctx, "listed IdPs",
		slog.Int("count", len(idps)))

	return idps, nil
}

// Delete deletes an identity provider
func (s *Service) Delete(ctx context.Context, clusterID, idpName string) error {
	s.logger.InfoContext(ctx, "deleting IdP",
		slog.String("cluster", clusterID),
		slog.String("name", idpName))

	conn := s.ocm.GetConnection()
	if conn == nil {
		return fmt.Errorf("no connection available")
	}

	// Find the IdP by name
	idps, err := s.List(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("failed to list IdPs: %w", err)
	}

	var idpID string
	for _, idp := range idps {
		if idp.Name == idpName {
			idpID = idp.ID
			break
		}
	}

	if idpID == "" {
		return fmt.Errorf("IdP '%s' not found", idpName)
	}

	// Delete the IdP
	_, err = conn.ClustersMgmt().V1().
		Clusters().
		Cluster(clusterID).
		IdentityProviders().
		IdentityProvider(idpID).
		Delete().
		SendContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete IdP: %w", err)
	}

	s.logger.InfoContext(ctx, "IdP deleted successfully",
		slog.String("name", idpName))

	return nil
}
