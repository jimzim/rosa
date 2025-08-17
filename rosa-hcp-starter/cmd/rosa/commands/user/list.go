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

// ListOptions contains options for listing users
type ListOptions struct {
	ClusterName  string
	OutputFormat string
}

// NewListCommand creates the list users command
func NewListCommand(logger *slog.Logger) *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:     "users",
		Aliases: []string{"user"},
		Short:   "List cluster users with administrative roles",
		Long: `List all users who have been granted administrative roles on the cluster.

Note: This command does not work with clusters using external authentication.
For external auth clusters, user information is managed in your identity provider.`,
		Example: `  # List all users with admin roles
  rosa list users --cluster my-cluster

  # Output in JSON format
  rosa list users --cluster my-cluster --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListUsers(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format (text, json, yaml)")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runListUsers(ctx context.Context, logger *slog.Logger, opts *ListOptions) error {
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

	// List users
	users, err := userSvc.ListUsers(ctx, cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	// Display results based on output format
	var writer *output.Writer

	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		writer = output.NewWriter(output.FormatJSON)
		return writer.Print(users)
	case "yaml":
		writer = output.NewWriter(output.FormatYAML)
		return writer.Print(users)
	default:
		writer = output.NewWriter(output.FormatText)

		if len(users) == 0 {
			writer.Info("No users with administrative roles found for cluster '%s'", opts.ClusterName)
			writer.Info("\nTo grant a role to a user:")
			fmt.Printf("  rosa grant user cluster-admin --user <username> --cluster %s\n", opts.ClusterName)
			return nil
		}

		writer.Title(fmt.Sprintf("Users with Administrative Roles - Cluster '%s'", cluster.Name()))

		// Count by role
		clusterAdminCount := 0
		dedicatedAdminCount := 0
		for _, u := range users {
			for _, role := range u.Roles {
				if role == "cluster-admin" {
					clusterAdminCount++
				}
				if role == "dedicated-admin" {
					dedicatedAdminCount++
				}
			}
		}

		// Display as table
		fmt.Printf("\n%-30s %-40s\n", "USERNAME", "ROLES")
		fmt.Println(strings.Repeat("-", 70))

		for _, u := range users {
			rolesStr := strings.Join(u.Roles, ", ")
			fmt.Printf("%-30s %-40s\n", u.Username, rolesStr)
		}
		fmt.Println()

		// Summary
		writer.Info("Summary:")
		writer.KeyValue(map[string]string{
			"Total Users":      fmt.Sprintf("%d", len(users)),
			"Cluster Admins":   fmt.Sprintf("%d", clusterAdminCount),
			"Dedicated Admins": fmt.Sprintf("%d", dedicatedAdminCount),
		})

		// Available commands
		writer.Info("\n📋 Available Commands:")
		writer.Info("  # Grant a role to a user:")
		fmt.Printf("  rosa grant user cluster-admin --user <username> --cluster %s\n", opts.ClusterName)

		writer.Info("\n  # Revoke a role from a user:")
		fmt.Printf("  rosa revoke user cluster-admin --user <username> --cluster %s\n", opts.ClusterName)

		if cluster.Hypershift().Enabled() {
			writer.Info("\n💡 Tip: For HCP clusters, consider using external authentication:")
			fmt.Printf("  rosa create external-auth-provider --cluster %s\n", opts.ClusterName)
		}
	}

	return nil
}
