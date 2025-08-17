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
	RevokedAt      *time.Time
	Description    string
}

// Create creates a new break-glass credential
func (s *service) Create(ctx context.Context, clusterID string, config Config) (*BreakGlassCredential, error) {
	s.logger.InfoContext(ctx, "creating break-glass credential",
		slog.String("cluster_id", clusterID),
		slog.String("username", config.Username))

	// Build the break-glass credential
	builder := cmv1.NewBreakGlassCredential()

	if config.Username != "" {
		builder.Username(config.Username)
	}

	if !config.ExpirationTime.IsZero() {
		builder.ExpirationTimestamp(config.ExpirationTime)
	}

	if config.Description != "" {
		// Note: Description field might not be available in the SDK
		// This is a placeholder for when it becomes available
	}

	credential, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build break-glass credential: %w", err)
	}

	// Create via API
	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		BreakGlassCredentials().
		Add().Body(credential).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create break-glass credential: %w", err)
	}

	result := convertBreakGlassCredential(response.Body())
	s.logger.InfoContext(ctx, "break-glass credential created successfully",
		slog.String("id", result.ID))

	return result, nil
}

// List lists all break-glass credentials for a cluster
func (s *service) List(ctx context.Context, clusterID string) ([]*BreakGlassCredential, error) {
	s.logger.InfoContext(ctx, "listing break-glass credentials", slog.String("cluster_id", clusterID))

	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		BreakGlassCredentials().
		List().
		Page(1).
		Size(100).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list break-glass credentials: %w", err)
	}

	credentials := make([]*BreakGlassCredential, 0, response.Total())
	response.Items().Each(func(cred *cmv1.BreakGlassCredential) bool {
		credentials = append(credentials, convertBreakGlassCredential(cred))
		return true
	})

	s.logger.InfoContext(ctx, "found break-glass credentials", slog.Int("count", len(credentials)))
	return credentials, nil
}

// Get retrieves a specific break-glass credential
func (s *service) Get(ctx context.Context, clusterID, credentialID string) (*BreakGlassCredential, error) {
	s.logger.InfoContext(ctx, "getting break-glass credential",
		slog.String("cluster_id", clusterID),
		slog.String("credential_id", credentialID))

	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		BreakGlassCredentials().
		BreakGlassCredential(credentialID).
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
		slog.String("cluster_id", clusterID),
		slog.String("credential_id", credentialID))

	// First check if the credential exists and is valid
	credential, err := s.Get(ctx, clusterID, credentialID)
	if err != nil {
		return "", err
	}

	// Check status
	if credential.Status == "revoked" {
		return "", fmt.Errorf("break-glass credential has been revoked")
	}
	if credential.Status == "expired" {
		return "", fmt.Errorf("break-glass credential has expired")
	}
	if credential.Status == "awaiting_revocation" {
		return "", fmt.Errorf("break-glass credential is awaiting revocation")
	}

	// Poll for kubeconfig (it may take a moment to generate)
	maxAttempts := 30 // 30 seconds timeout
	pollInterval := time.Second

	for i := 0; i < maxAttempts; i++ {
		// Try to get the credential with kubeconfig
		response, err := s.ocm.ClustersMgmt().V1().
			Clusters().Cluster(clusterID).
			BreakGlassCredentials().
			BreakGlassCredential(credentialID).
			Get().
			SendContext(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get break-glass credential: %w", err)
		}

		if response.Body().Kubeconfig() != "" {
			return response.Body().Kubeconfig(), nil
		}

		// Check if status changed to issued
		if response.Body().Status() == cmv1.BreakGlassCredentialStatusIssued {
			if response.Body().Kubeconfig() != "" {
				return response.Body().Kubeconfig(), nil
			}
		}

		s.logger.DebugContext(ctx, "waiting for kubeconfig to be generated",
			slog.Int("attempt", i+1),
			slog.String("status", string(response.Body().Status())))

		time.Sleep(pollInterval)
	}

	return "", fmt.Errorf("timeout waiting for kubeconfig to be generated")
}

// Revoke revokes a specific break-glass credential
func (s *service) Revoke(ctx context.Context, clusterID, credentialID string) error {
	s.logger.InfoContext(ctx, "revoking break-glass credential",
		slog.String("cluster_id", clusterID),
		slog.String("credential_id", credentialID))

	// Send revocation request
	_, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		BreakGlassCredentials().
		BreakGlassCredential(credentialID).
		Delete().
		SendContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke break-glass credential: %w", err)
	}

	s.logger.InfoContext(ctx, "break-glass credential revoked successfully",
		slog.String("credential_id", credentialID))
	return nil
}

// RevokeAll revokes all break-glass credentials for a cluster
func (s *service) RevokeAll(ctx context.Context, clusterID string) error {
	s.logger.InfoContext(ctx, "revoking all break-glass credentials", slog.String("cluster_id", clusterID))

	// List all credentials
	credentials, err := s.List(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("failed to list credentials: %w", err)
	}

	// Revoke each one
	var revokeErrors []error
	for _, cred := range credentials {
		if cred.Status != "revoked" {
			err := s.Revoke(ctx, clusterID, cred.ID)
			if err != nil {
				s.logger.ErrorContext(ctx, "failed to revoke credential",
					slog.String("credential_id", cred.ID),
					slog.String("error", err.Error()))
				revokeErrors = append(revokeErrors, err)
			}
		}
	}

	if len(revokeErrors) > 0 {
		return fmt.Errorf("failed to revoke %d credentials", len(revokeErrors))
	}

	s.logger.InfoContext(ctx, "all break-glass credentials revoked successfully")
	return nil
}

// IsSupported checks if break-glass credentials are supported for the cluster
func (s *service) IsSupported(ctx context.Context, clusterID string) (bool, string, error) {
	s.logger.InfoContext(ctx, "checking break-glass credential support", slog.String("cluster_id", clusterID))

	// Get cluster details
	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Get().
		SendContext(ctx)
	if err != nil {
		return false, "", fmt.Errorf("failed to get cluster: %w", err)
	}

	cluster := response.Body()

	// Break-glass credentials are only supported for HCP clusters
	if !cluster.Hypershift().Enabled() {
		return false, "Break-glass credentials are only supported for HCP (Hosted Control Plane) clusters", nil
	}

	// Check if external auth is configured (required for break-glass)
	if cluster.ExternalAuthConfig() == nil || !cluster.ExternalAuthConfig().Enabled() {
		return false, "Break-glass credentials require external authentication to be configured first", nil
	}

	return true, "", nil
}

// Helper functions

func convertBreakGlassCredential(cred *cmv1.BreakGlassCredential) *BreakGlassCredential {
	if cred == nil {
		return nil
	}

	bgc := &BreakGlassCredential{
		ID:       cred.ID(),
		Username: cred.Username(),
	}

	// Convert status
	switch cred.Status() {
	case cmv1.BreakGlassCredentialStatusIssued:
		bgc.Status = "issued"
	case cmv1.BreakGlassCredentialStatusExpired:
		bgc.Status = "expired"
	case cmv1.BreakGlassCredentialStatusRevoked:
		bgc.Status = "revoked"
	case cmv1.BreakGlassCredentialStatusAwaitingRevocation:
		bgc.Status = "awaiting_revocation"
	default:
		bgc.Status = "pending"
	}

	// Set timestamps
	if cred.ExpirationTimestamp() != nil && !cred.ExpirationTimestamp().IsZero() {
		bgc.ExpirationTime = *cred.ExpirationTimestamp()
	}

	if cred.CreationTimestamp() != nil && !cred.CreationTimestamp().IsZero() {
		bgc.CreatedAt = *cred.CreationTimestamp()
	}

	if cred.RevocationTimestamp() != nil && !cred.RevocationTimestamp().IsZero() {
		t := *cred.RevocationTimestamp()
		bgc.RevokedAt = &t
	}

	return bgc
}

// FormatStatus returns a formatted status string with color hints
func FormatStatus(status string) string {
	switch status {
	case "issued":
		return "✓ ACTIVE"
	case "expired":
		return "⚠ EXPIRED"
	case "revoked":
		return "✗ REVOKED"
	case "awaiting_revocation":
		return "⏳ REVOKING"
	default:
		return "⏳ PENDING"
	}
}

// IsActive returns true if the credential is currently usable
func (c *BreakGlassCredential) IsActive() bool {
	return c.Status == "issued" && time.Now().Before(c.ExpirationTime)
}
