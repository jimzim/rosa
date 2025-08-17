package user

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
	"github.com/openshift/rosa-hcp/pkg/user"
)

// GrantOptions contains options for granting roles to users
type GrantOptions struct {
	ClusterName string
	Username    string
	Role        string
}

// NewGrantCommand creates the grant user command
func NewGrantCommand(logger *slog.Logger) *cobra.Command {
	opts := &GrantOptions{}

	cmd := &cobra.Command{
		Use:   "user ROLE",
		Short: "Grant role to a user",
		Long: `Grant administrative role to a user on a cluster.

Available roles:
- cluster-admin: Full cluster administrator access
- dedicated-admin: Admin access with some restrictions

Note: This command does not work with clusters using external authentication.
For external auth clusters, manage users in your identity provider.`,
		Example: `  # Grant cluster-admin role to a user
  rosa grant user cluster-admin --user alice --cluster my-cluster

  # Grant dedicated-admin role to a user
  rosa grant user dedicated-admin --user bob --cluster my-cluster`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Role = args[0]
			return runGrantUser(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVarP(&opts.Username, "user", "u", "", "Username to grant the role to (required)")

	cmd.MarkFlagRequired("cluster")
	cmd.MarkFlagRequired("user")

	return cmd
}

func runGrantUser(ctx context.Context, logger *slog.Logger, opts *GrantOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Validate role
	if opts.Role != "cluster-admin" && opts.Role != "dedicated-admin" {
		return fmt.Errorf("invalid role '%s': must be 'cluster-admin' or 'dedicated-admin'", opts.Role)
	}

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

	// Create User service
	userSvc, err := user.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create user service: %w", err)
	}

	writer.Title("Grant Role to User")
	writer.KeyValue(map[string]string{
		"Cluster":  cluster.Name(),
		"Username": opts.Username,
		"Role":     opts.Role,
	})

	// Grant the role
	writer.Info("Granting role...")
	err = userSvc.GrantRole(ctx, cluster.ID(), opts.Username, opts.Role)
	if err != nil {
		return fmt.Errorf("failed to grant role: %w", err)
	}

	writer.Success("Role '%s' granted to user '%s' successfully", opts.Role, opts.Username)

	// Show next steps
	writer.Info("\n📋 Next Steps:")
	writer.Info("1. The user can now log in with their credentials")
	writer.Info("2. View all users with administrative roles:")
	fmt.Printf("   rosa list users --cluster %s\n", opts.ClusterName)
	writer.Info("\n3. To revoke this role later:")
	fmt.Printf("   rosa revoke user %s --user %s --cluster %s\n",
		opts.Role, opts.Username, opts.ClusterName)

	return nil
}
