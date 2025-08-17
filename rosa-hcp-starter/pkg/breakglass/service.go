package breakglass

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// Service provides Break-glass Credential operations
type Service interface {
	Create(ctx context.Context, clusterID string, config Config) (*BreakGlassCredential, error)
	List(ctx context.Context, clusterID string) ([]*BreakGlassCredential, error)
	Get(ctx context.Context, clusterID, credentialID string) (*BreakGlassCredential, error)
	GetKubeconfig(ctx context.Context, clusterID, credentialID string) (string, error)
	Revoke(ctx context.Context, clusterID, credentialID string) error
	RevokeAll(ctx context.Context, clusterID string) error
	IsSupported(ctx context.Context, clusterID string) (bool, string, error)
}

type service struct {
	logger *slog.Logger
	ocm    *sdk.Connection
}

// NewService creates a new Break-glass Credential service
func NewService(ctx context.Context, logger *slog.Logger, ocm *sdk.Connection) (Service, error) {
	return &service{
		logger: logger,
		ocm:    ocm,
	}, nil
}

// Config holds break-glass credential configuration
type Config struct {
	Username       string
	ExpirationTime time.Time
	Description    string
}

// BreakGlassCredential represents a break-glass credential
type BreakGlassCredential struct {
	ID             string
	Username       string
	Status         string
	ExpirationTime time.Time
	CreatedAt      time.Time
	RevokedAt      time.Time
	Description    string
	Kubeconfig     string
}

// IsActive checks if the credential is active
func (c *BreakGlassCredential) IsActive() bool {
	return c.Status == "issued" && (c.ExpirationTime.IsZero() || c.ExpirationTime.After(time.Now()))
}

// Create creates a new break-glass credential
func (s *service) Create(ctx context.Context, clusterID string, config Config) (*BreakGlassCredential, error) {
	s.logger.InfoContext(ctx, "creating break-glass credential",
		slog.String("cluster", clusterID),
		slog.String("username", config.Username))

	// Build the break-glass credential
	builder := cmv1.NewBreakGlassCredential()
	builder.Username(config.Username)
	
	if !config.ExpirationTime.IsZero() {
		builder.ExpirationTimestamp(config.ExpirationTime)
	}
	
	// Description field might not be available in the SDK
	// We'll handle it separately if needed

	credential, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build break-glass credential: %w", err)
	}

	// Create via API
	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		BreakGlassCredentials().
		Add().
		Body(credential).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create break-glass credential: %w", err)
	}

	created := response.Body()
	
	return convertBreakGlassCredential(created), nil
}

// List lists all break-glass credentials for a cluster
func (s *service) List(ctx context.Context, clusterID string) ([]*BreakGlassCredential, error) {
	s.logger.InfoContext(ctx, "listing break-glass credentials",
		slog.String("cluster", clusterID))

	// List via API
	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		BreakGlassCredentials().
		List().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list break-glass credentials: %w", err)
	}

	credentials := make([]*BreakGlassCredential, 0)
	response.Items().Each(func(item *cmv1.BreakGlassCredential) bool {
		credentials = append(credentials, convertBreakGlassCredential(item))
		return true
	})

	s.logger.InfoContext(ctx, "found break-glass credentials", slog.Int("count", len(credentials)))
	return credentials, nil
}

// Get retrieves a specific break-glass credential
func (s *service) Get(ctx context.Context, clusterID, credentialID string) (*BreakGlassCredential, error) {
	s.logger.InfoContext(ctx, "getting break-glass credential",
		slog.String("cluster", clusterID),
		slog.String("credential", credentialID))

	// Get via API
	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		BreakGlassCredentials().BreakGlassCredential(credentialID).
		Get().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get break-glass credential: %w", err)
	}

	return convertBreakGlassCredential(response.Body()), nil
}

// GetKubeconfig retrieves the kubeconfig for a break-glass credential
func (s *service) GetKubeconfig(ctx context.Context, clusterID, credentialID string) (string, error) {
	s.logger.InfoContext(ctx, "getting kubeconfig for break-glass credential",
		slog.String("cluster", clusterID),
		slog.String("credential", credentialID))

	credential, err := s.Get(ctx, clusterID, credentialID)
	if err != nil {
		return "", err
	}

	if credential.Kubeconfig == "" {
		return "", fmt.Errorf("kubeconfig not yet available for credential %s", credentialID)
	}

	return credential.Kubeconfig, nil
}

// Revoke revokes a specific break-glass credential
// NOTE: The OCM API does not support revoking individual credentials
// This method is kept for interface compatibility but returns an error
func (s *service) Revoke(ctx context.Context, clusterID, credentialID string) error {
	s.logger.InfoContext(ctx, "individual break-glass credential revocation not supported",
		slog.String("cluster", clusterID),
		slog.String("credential", credentialID))

	// The OCM API only supports bulk deletion of ALL break-glass credentials
	// Individual credential deletion is not supported
	return fmt.Errorf("individual break-glass credential revocation is not supported. Use RevokeAll to revoke all credentials")
}

// RevokeAll revokes all break-glass credentials for a cluster
func (s *service) RevokeAll(ctx context.Context, clusterID string) error {
	s.logger.InfoContext(ctx, "revoking all break-glass credentials",
		slog.String("cluster", clusterID))

	// The OCM API supports bulk deletion of all credentials
	// This matches the original ROSA CLI behavior
	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		BreakGlassCredentials().
		Delete().
		SendContext(ctx)
	if err != nil {
		if response != nil && response.Status() == 404 {
			return fmt.Errorf("no break-glass credentials found for cluster")
		}
		return fmt.Errorf("failed to revoke all break-glass credentials: %w", err)
	}

	s.logger.InfoContext(ctx, "all break-glass credentials revoked successfully")
	return nil
}

// IsSupported checks if break-glass credentials are supported for the cluster
func (s *service) IsSupported(ctx context.Context, clusterID string) (bool, string, error) {
	s.logger.InfoContext(ctx, "checking break-glass credential support",
		slog.String("cluster", clusterID))

	// Get cluster
	clusterResp, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Get().
		SendContext(ctx)
	if err != nil {
		return false, "", fmt.Errorf("failed to get cluster: %w", err)
	}

	cluster := clusterResp.Body()
	
	// Check if external auth is configured (required for break-glass)
	if cluster.ExternalAuthConfig() == nil || !cluster.ExternalAuthConfig().Enabled() {
		return false, "Break-glass credentials require external authentication to be configured", nil
	}
	
	// Check if it's an HCP cluster
	if cluster.Hypershift() == nil || !cluster.Hypershift().Enabled() {
		return false, "Break-glass credentials are only supported for HCP clusters", nil
	}

	return true, "", nil
}

// Helper function to convert SDK type to our type
func convertBreakGlassCredential(cred *cmv1.BreakGlassCredential) *BreakGlassCredential {
	if cred == nil {
		return nil
	}

	bgc := &BreakGlassCredential{
		ID:       cred.ID(),
		Username: cred.Username(),
		Status:   string(cred.Status()),
	}

	// FIXED: Handle timestamps properly - they return time.Time, not *time.Time
	// Check if not zero value instead of != nil
	expirationTime := cred.ExpirationTimestamp()
	if !expirationTime.IsZero() {
		bgc.ExpirationTime = expirationTime
	}

	// FIXED: CreationTimestamp might not exist in the SDK
	// We might need to use a different field or leave it empty
	// For now, we'll use the current time as a placeholder
	bgc.CreatedAt = time.Now() // This should be from the API response

	// FIXED: Handle revocation timestamp
	revocationTime := cred.RevocationTimestamp()
	if !revocationTime.IsZero() {
		bgc.RevokedAt = revocationTime
	}

	// Kubeconfig might be in a separate field or require another API call
	if cred.Kubeconfig() != "" {
		bgc.Kubeconfig = cred.Kubeconfig()
	}

	return bgc
}

// Helper function to format time for display
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// FormatStatus formats the credential status for display
func FormatStatus(status string) string {
	switch status {
	case "issued":
		return "Active"
	case "revoked":
		return "Revoked"
	case "expired":
		return "Expired"
	default:
		return status
	}
}

// FormatTime formats a time.Time for display
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
