package user

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// Service provides User management operations
type Service interface {
	GrantRole(ctx context.Context, clusterID, username, role string) error
	RevokeRole(ctx context.Context, clusterID, username, role string) error
	ListUsers(ctx context.Context, clusterID string) ([]*User, error)
	GetUser(ctx context.Context, clusterID, username string) (*User, error)
}

type service struct {
	logger *slog.Logger
	ocm    *sdk.Connection
}

// NewService creates a new User management service
func NewService(ctx context.Context, logger *slog.Logger, ocm *sdk.Connection) (Service, error) {
	return &service{
		logger: logger,
		ocm:    ocm,
	}, nil
}

// User represents a cluster user with their roles
type User struct {
	Username string
	Groups   []string
	Roles    []string
}

const (
	// Role names in OpenShift
	ClusterAdminRole   = "cluster-admins"
	DedicatedAdminRole = "dedicated-admins"

	// Display names
	ClusterAdminDisplay   = "cluster-admin"
	DedicatedAdminDisplay = "dedicated-admin"
)

// GrantRole grants a role to a user
func (s *service) GrantRole(ctx context.Context, clusterID, username, role string) error {
	s.logger.InfoContext(ctx, "granting role to user",
		slog.String("cluster_id", clusterID),
		slog.String("username", username),
		slog.String("role", role))

	// Validate username
	if !isValidUsername(username) {
		return fmt.Errorf("invalid username: must contain only letters, numbers, dashes, and underscores")
	}

	// Normalize role name
	groupName := normalizeRole(role)
	if groupName == "" {
		return fmt.Errorf("invalid role: must be 'cluster-admin' or 'dedicated-admin'")
	}

	// Check if cluster exists and is ready
	clusterResp, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Get().
		SendContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	cluster := clusterResp.Body()
	if cluster.State() != cmv1.ClusterStateReady && cluster.State() != cmv1.ClusterStateHibernating {
		return fmt.Errorf("cluster is not ready (current state: %s)", cluster.State())
	}

	// Check if external auth is configured (users can't be managed directly with external auth)
	if cluster.ExternalAuthConfig() != nil && cluster.ExternalAuthConfig().Enabled() {
		return fmt.Errorf("user management is not supported for clusters with external authentication. " +
			"Users and roles should be managed in your external identity provider")
	}

	// Build the user
	user, err := cmv1.NewUser().
		ID(username).
		Build()
	if err != nil {
		return fmt.Errorf("failed to build user: %w", err)
	}

	// Add user to the group
	_, err = s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Groups().Group(groupName).
		Users().
		Add().
		Body(user).
		SendContext(ctx)
	if err != nil {
		// Check if user already exists in group
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("user '%s' already has role '%s'", username, role)
		}
		return fmt.Errorf("failed to grant role: %w", err)
	}

	s.logger.InfoContext(ctx, "role granted successfully",
		slog.String("username", username),
		slog.String("role", role))

	return nil
}

// RevokeRole revokes a role from a user
func (s *service) RevokeRole(ctx context.Context, clusterID, username, role string) error {
	s.logger.InfoContext(ctx, "revoking role from user",
		slog.String("cluster_id", clusterID),
		slog.String("username", username),
		slog.String("role", role))

	// Normalize role name
	groupName := normalizeRole(role)
	if groupName == "" {
		return fmt.Errorf("invalid role: must be 'cluster-admin' or 'dedicated-admin'")
	}

	// Check if cluster exists
	clusterResp, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Get().
		SendContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	cluster := clusterResp.Body()

	// Check if external auth is configured
	if cluster.ExternalAuthConfig() != nil && cluster.ExternalAuthConfig().Enabled() {
		return fmt.Errorf("user management is not supported for clusters with external authentication")
	}

	// Remove user from the group
	_, err = s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Groups().Group(groupName).
		Users().User(username).
		Delete().
		SendContext(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("user '%s' does not have role '%s'", username, role)
		}
		return fmt.Errorf("failed to revoke role: %w", err)
	}

	s.logger.InfoContext(ctx, "role revoked successfully",
		slog.String("username", username),
		slog.String("role", role))

	return nil
}

// ListUsers lists all users with administrative roles
func (s *service) ListUsers(ctx context.Context, clusterID string) ([]*User, error) {
	s.logger.InfoContext(ctx, "listing users", slog.String("cluster_id", clusterID))

	// Check if cluster exists
	clusterResp, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Get().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	cluster := clusterResp.Body()

	// Check if external auth is configured
	if cluster.ExternalAuthConfig() != nil && cluster.ExternalAuthConfig().Enabled() {
		return nil, fmt.Errorf("listing users is not supported for clusters with external authentication")
	}

	// Get users from each admin group
	userMap := make(map[string]*User)

	// Get cluster-admins
	clusterAdmins, err := s.getUsersFromGroup(ctx, clusterID, ClusterAdminRole)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to get cluster-admins", slog.String("error", err.Error()))
	} else {
		for _, username := range clusterAdmins {
			if userMap[username] == nil {
				userMap[username] = &User{
					Username: username,
					Groups:   []string{},
					Roles:    []string{},
				}
			}
			userMap[username].Groups = append(userMap[username].Groups, ClusterAdminRole)
			userMap[username].Roles = append(userMap[username].Roles, ClusterAdminDisplay)
		}
	}

	// Get dedicated-admins
	dedicatedAdmins, err := s.getUsersFromGroup(ctx, clusterID, DedicatedAdminRole)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to get dedicated-admins", slog.String("error", err.Error()))
	} else {
		for _, username := range dedicatedAdmins {
			if userMap[username] == nil {
				userMap[username] = &User{
					Username: username,
					Groups:   []string{},
					Roles:    []string{},
				}
			}
			userMap[username].Groups = append(userMap[username].Groups, DedicatedAdminRole)
			userMap[username].Roles = append(userMap[username].Roles, DedicatedAdminDisplay)
		}
	}

	// Convert map to slice
	users := make([]*User, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, user)
	}

	s.logger.InfoContext(ctx, "found users", slog.Int("count", len(users)))
	return users, nil
}

// GetUser gets information about a specific user
func (s *service) GetUser(ctx context.Context, clusterID, username string) (*User, error) {
	s.logger.InfoContext(ctx, "getting user",
		slog.String("cluster_id", clusterID),
		slog.String("username", username))

	user := &User{
		Username: username,
		Groups:   []string{},
		Roles:    []string{},
	}

	// Check cluster-admins group
	isClusterAdmin, err := s.isUserInGroup(ctx, clusterID, username, ClusterAdminRole)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to check cluster-admins", slog.String("error", err.Error()))
	} else if isClusterAdmin {
		user.Groups = append(user.Groups, ClusterAdminRole)
		user.Roles = append(user.Roles, ClusterAdminDisplay)
	}

	// Check dedicated-admins group
	isDedicatedAdmin, err := s.isUserInGroup(ctx, clusterID, username, DedicatedAdminRole)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to check dedicated-admins", slog.String("error", err.Error()))
	} else if isDedicatedAdmin {
		user.Groups = append(user.Groups, DedicatedAdminRole)
		user.Roles = append(user.Roles, DedicatedAdminDisplay)
	}

	if len(user.Roles) == 0 {
		return nil, fmt.Errorf("user '%s' has no administrative roles", username)
	}

	return user, nil
}

// Helper functions

func (s *service) getUsersFromGroup(ctx context.Context, clusterID, groupName string) ([]string, error) {
	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Groups().Group(groupName).
		Users().
		List().
		Page(1).
		Size(1000).
		SendContext(ctx)
	if err != nil {
		return nil, err
	}

	usernames := make([]string, 0, response.Total())
	response.Items().Each(func(user *cmv1.User) bool {
		usernames = append(usernames, user.ID())
		return true
	})

	return usernames, nil
}

func (s *service) isUserInGroup(ctx context.Context, clusterID, username, groupName string) (bool, error) {
	_, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Groups().Group(groupName).
		Users().User(username).
		Get().
		SendContext(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isValidUsername(username string) bool {
	if username == "" {
		return false
	}
	for _, ch := range username {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}

func normalizeRole(role string) string {
	switch strings.ToLower(role) {
	case "cluster-admin", "cluster-admins", "clusteradmin":
		return ClusterAdminRole
	case "dedicated-admin", "dedicated-admins", "dedicatedadmin":
		return DedicatedAdminRole
	default:
		return ""
	}
}
