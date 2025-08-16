package admin

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"strings"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa-hcp/pkg/api"
)

// Service provides admin user operations
type Service struct {
	ocm    api.Client
	logger *slog.Logger
}

// NewService creates a new admin service
func NewService(ocm api.Client, logger *slog.Logger) *Service {
	return &Service{
		ocm:    ocm,
		logger: logger.With("service", "admin"),
	}
}

// AdminUser represents a cluster admin user
type AdminUser struct {
	Username  string
	Password  string
	ClusterID string
	CreatedAt time.Time
	ExpiresAt *time.Time
}

// CreateOptions contains options for creating an admin user
type CreateOptions struct {
	ClusterID string
	Username  string
	Password  string
	ExpiresIn time.Duration
}

// Create creates a cluster admin user
func (s *Service) Create(ctx context.Context, opts CreateOptions) (*AdminUser, error) {
	s.logger.InfoContext(ctx, "creating admin user",
		slog.String("cluster", opts.ClusterID),
		slog.String("username", opts.Username))

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	// Generate password if not provided
	password := opts.Password
	if password == "" {
		password = generateSecurePassword()
	}

	// Build HTPasswd identity provider
	htpasswdBuilder := cmv1.NewHTPasswdIdentityProvider().
		Users(cmv1.NewHTPasswdUserList().
			Items(cmv1.NewHTPasswdUser().
				Username(opts.Username).
				Password(password)))

	// Build the identity provider
	idpBuilder := cmv1.NewIdentityProvider().
		Type(cmv1.IdentityProviderTypeHtpasswd).
		Name("cluster-admin").
		MappingMethod(cmv1.IdentityProviderMappingMethodClaim).
		Htpasswd(htpasswdBuilder)

	idp, err := idpBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build identity provider: %w", err)
	}

	// Add the IDP to the cluster
	_, err = conn.ClustersMgmt().V1().
		Clusters().
		Cluster(opts.ClusterID).
		IdentityProviders().
		Add().
		Body(idp).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity provider: %w", err)
	}

	// Note: In ROSA HCP, the HTPasswd IDP with the cluster-admin name
	// automatically gets cluster-admin privileges. The user/group
	// management APIs are not available in HCP clusters.

	adminUser := &AdminUser{
		Username:  opts.Username,
		Password:  password,
		ClusterID: opts.ClusterID,
		CreatedAt: time.Now(),
	}

	if opts.ExpiresIn > 0 {
		expiresAt := time.Now().Add(opts.ExpiresIn)
		adminUser.ExpiresAt = &expiresAt
	}

	s.logger.InfoContext(ctx, "admin user created successfully",
		slog.String("username", opts.Username))

	return adminUser, nil
}

// Delete deletes a cluster admin user
func (s *Service) Delete(ctx context.Context, clusterID, username string) error {
	s.logger.InfoContext(ctx, "deleting admin user",
		slog.String("cluster", clusterID),
		slog.String("username", username))

	conn := s.ocm.GetConnection()
	if conn == nil {
		return fmt.Errorf("no connection available")
	}

	// In HCP, we just need to remove the HTPasswd IDP
	idps, err := conn.ClustersMgmt().V1().
		Clusters().
		Cluster(clusterID).
		IdentityProviders().
		List().
		SendContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to list identity providers: %w", err)
	}

	idps.Items().Each(func(idp *cmv1.IdentityProvider) bool {
		if idp.Name() == "cluster-admin" && idp.Type() == cmv1.IdentityProviderTypeHtpasswd {
			_, delErr := conn.ClustersMgmt().V1().
				Clusters().
				Cluster(clusterID).
				IdentityProviders().
				IdentityProvider(idp.ID()).
				Delete().
				SendContext(ctx)
			if delErr != nil {
				s.logger.WarnContext(ctx, "failed to delete identity provider",
					slog.String("error", delErr.Error()))
			}
		}
		return true
	})

	s.logger.InfoContext(ctx, "admin user deleted successfully",
		slog.String("username", username))

	return nil
}

// List lists admin users for a cluster
func (s *Service) List(ctx context.Context, clusterID string) ([]*AdminUser, error) {
	s.logger.InfoContext(ctx, "listing admin users",
		slog.String("cluster", clusterID))

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	// In HCP, we look for HTPasswd IDPs which are typically admin users
	idps, err := conn.ClustersMgmt().V1().
		Clusters().
		Cluster(clusterID).
		IdentityProviders().
		List().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list identity providers: %w", err)
	}

	var adminUsers []*AdminUser

	idps.Items().Each(func(idp *cmv1.IdentityProvider) bool {
		if idp.Type() == cmv1.IdentityProviderTypeHtpasswd && idp.Name() == "cluster-admin" {
			// HTPasswd IDP represents an admin user
			if idp.Htpasswd() != nil && idp.Htpasswd().Users() != nil {
				idp.Htpasswd().Users().Each(func(user *cmv1.HTPasswdUser) bool {
					adminUsers = append(adminUsers, &AdminUser{
						Username:  user.Username(),
						ClusterID: clusterID,
						CreatedAt: time.Now(), // We don't have the actual creation time
					})
					return true
				})
			}
		}
		return true
	})

	return adminUsers, nil
}

// generateSecurePassword generates a secure random password
func generateSecurePassword() string {
	const (
		length = 20
		chars  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?"
	)

	// Ensure we have at least one of each type
	password := []byte{
		'A' + byte(randInt(26)), // Uppercase
		'a' + byte(randInt(26)), // Lowercase
		'0' + byte(randInt(10)), // Digit
		"!@#$%^&*"[randInt(8)],  // Special char
	}

	// Fill the rest randomly
	for len(password) < length {
		password = append(password, chars[randInt(len(chars))])
	}

	// Shuffle the password
	for i := len(password) - 1; i > 0; i-- {
		j := randInt(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password)
}

func randInt(max int) int {
	b := make([]byte, 1)
	rand.Read(b)
	return int(b[0]) % max
}

// GetClusterAPIURL gets the API URL for a cluster
func (s *Service) GetClusterAPIURL(ctx context.Context, clusterID string) (string, error) {
	conn := s.ocm.GetConnection()
	if conn == nil {
		return "", fmt.Errorf("no connection available")
	}

	cluster, err := conn.ClustersMgmt().V1().
		Clusters().
		Cluster(clusterID).
		Get().
		SendContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster: %w", err)
	}

	apiURL := cluster.Body().API().URL()
	if apiURL == "" {
		return "", fmt.Errorf("cluster API URL not available yet")
	}

	return apiURL, nil
}

// GetClusterConsoleURL gets the console URL for a cluster
func (s *Service) GetClusterConsoleURL(ctx context.Context, clusterID string) (string, error) {
	conn := s.ocm.GetConnection()
	if conn == nil {
		return "", fmt.Errorf("no connection available")
	}

	cluster, err := conn.ClustersMgmt().V1().
		Clusters().
		Cluster(clusterID).
		Get().
		SendContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster: %w", err)
	}

	consoleURL := cluster.Body().Console().URL()
	if consoleURL == "" {
		// Try to construct it from API URL
		apiURL := cluster.Body().API().URL()
		if apiURL != "" {
			// Replace api with console
			consoleURL = strings.Replace(apiURL, "api.", "console-openshift-console.apps.", 1)
			consoleURL = strings.Replace(consoleURL, ":6443", "", 1)
		}
	}

	if consoleURL == "" {
		return "", fmt.Errorf("cluster console URL not available yet")
	}

	return consoleURL, nil
}
