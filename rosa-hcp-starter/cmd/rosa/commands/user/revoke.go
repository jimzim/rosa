package user

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
	"github.com/openshift/rosa-hcp/pkg/user"
)

// RevokeOptions contains options for revoking user roles
type RevokeOptions struct {
	ClusterName string
	Username    string
	Role        string
}

// NewRevokeCommand creates the user revoke command
func NewRevokeCommand(logger *slog.Logger) *cobra.Command {
	opts := &RevokeOptions{}

	cmd := &cobra.Command{
		Use:     "user ROLE",
		Aliases: []string{"role"},
		Short:   "Revoke role from users",
		Long:    "Revoke role from cluster user",
		Example: `  # Revoke cluster-admin role from a user
  rosa revoke user cluster-admins --user=myusername --cluster=mycluster

  # Revoke dedicated-admin role from a user
  rosa revoke user dedicated-admins --user=myusername --cluster=mycluster`,
		Args: func(_ *cobra.Command, argv []string) error {
			if len(argv) != 1 {
				return fmt.Errorf("expected exactly one argument: the role to revoke (cluster-admins or dedicated-admins)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Role = args[0]
			return runRevokeUser(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVarP(&opts.Username, "user", "u", "", "Username to revoke the role from (required)")

	cmd.MarkFlagRequired("cluster")
	cmd.MarkFlagRequired("user")

	return cmd
}

func runRevokeUser(ctx context.Context, logger *slog.Logger, opts *RevokeOptions) error {
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

	// Create user service
	userSvc, err := user.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create user service: %w", err)
	}

	// Normalize role name (allow aliases)
	role := opts.Role
	validRoles := []string{"cluster-admins", "dedicated-admins"}
	validAliases := []string{"cluster-admin", "dedicated-admin"}
	
	// Check if it's an alias
	for _, alias := range validAliases {
		if role == alias {
			role = fmt.Sprintf("%ss", role) // Add 's' to make it plural
			break
		}
	}

	// Validate role
	isValid := false
	for _, validRole := range validRoles {
		if role == validRole {
			isValid = true
			break
		}
	}
	
	if !isValid {
		return fmt.Errorf("invalid role: %s. Expected one of: %s", opts.Role, strings.Join(validRoles, ", "))
	}

	// Validate username
	if !isValidUsername(opts.Username) {
		return fmt.Errorf("username '%s' isn't valid: it must contain only letters, digits, dashes and underscores", opts.Username)
	}

	// Check if user exists with the role
	users, err := userSvc.ListUsers(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	found := false
	for _, u := range users {
		if u.Username == opts.Username {
			for _, r := range u.Roles {
				if r == role {
					found = true
					break
				}
			}
		}
	}

	if !found {
		writer.Warning("User '%s' does not have role '%s' on cluster '%s'", opts.Username, role, opts.ClusterName)
		return nil
	}

	// Confirm revocation
	writer.Warning("This will revoke role '%s' from user '%s' on cluster '%s'", role, opts.Username, opts.ClusterName)
	
	// Revoke the role
	writer.Info("Revoking role '%s' from user '%s'...", role, opts.Username)
	err = userSvc.RevokeRole(ctx, opts.ClusterName, opts.Username, role)
	if err != nil {
		return fmt.Errorf("failed to revoke role: %w", err)
	}

	writer.Success("Successfully revoked role '%s' from user '%s' on cluster '%s'", role, opts.Username, opts.ClusterName)
	return nil
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
