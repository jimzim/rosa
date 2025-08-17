package breakglass

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	breakglassSvc "github.com/openshift/rosa-hcp/pkg/breakglass"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// RevokeOptions contains options for revoking break-glass credentials
type RevokeOptions struct {
	ClusterName   string
	CredentialID  string
	RevokeAll     bool
}

// NewRevokeCommand creates the break-glass credential revoke command
func NewRevokeCommand(logger *slog.Logger) *cobra.Command {
	opts := &RevokeOptions{}

	cmd := &cobra.Command{
		Use:     "break-glass-credential",
		Aliases: []string{"break-glass-credentials", "breakglasscredential", "breakglasscredentials"},
		Short:   "Revoke break-glass credentials",
		Long:    "Revoke break-glass credentials from a cluster. Either revoke a specific credential by ID or all credentials.",
		Example: `  # Revoke a specific break-glass credential
  rosa revoke break-glass-credential --cluster=mycluster --credential-id=abc123

  # Revoke all break-glass credentials
  rosa revoke break-glass-credential --cluster=mycluster --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRevokeBreakGlass(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVar(&opts.CredentialID, "credential-id", "", "ID of the specific credential to revoke")
	flags.BoolVar(&opts.RevokeAll, "all", false, "Revoke all break-glass credentials")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runRevokeBreakGlass(ctx context.Context, logger *slog.Logger, opts *RevokeOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Validate options
	if !opts.RevokeAll && opts.CredentialID == "" {
		return fmt.Errorf("either --credential-id or --all must be specified")
	}
	if opts.RevokeAll && opts.CredentialID != "" {
		return fmt.Errorf("cannot specify both --credential-id and --all")
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

	// Create break-glass service
	bgSvc, err := breakglassSvc.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create break-glass service: %w", err)
	}

	// Get cluster to verify it exists and has external auth
	clusterResp, err := apiClient.GetCluster(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	cluster := clusterResp.Body()

	// Check if external auth is enabled
	if cluster.ExternalAuthConfig() == nil || !cluster.ExternalAuthConfig().Enabled() {
		return fmt.Errorf("break-glass credentials are only supported for clusters with external authentication enabled")
	}

	if opts.RevokeAll {
		// List all credentials
		credentials, err := bgSvc.List(ctx, opts.ClusterName)
		if err != nil {
			return fmt.Errorf("failed to list break-glass credentials: %w", err)
		}

		if len(credentials) == 0 {
			writer.Info("No break-glass credentials found for cluster '%s'", opts.ClusterName)
			return nil
		}

		writer.Warning("This will revoke ALL %d break-glass credentials for cluster '%s'", len(credentials), opts.ClusterName)
		writer.Info("Revoking all break-glass credentials...")

		// Revoke all credentials
		for _, cred := range credentials {
			if cred.Status == "revoked" {
				continue // Skip already revoked
			}
			err = bgSvc.Revoke(ctx, opts.ClusterName, cred.ID)
			if err != nil {
				writer.Error("Failed to revoke credential %s: %v", cred.ID, err)
			} else {
				writer.Info("Revoked credential %s (%s)", cred.ID, cred.Username)
			}
		}

		writer.Success("Successfully requested revocation for all break-glass credentials from cluster '%s'", opts.ClusterName)
	} else {
		// Revoke specific credential
		writer.Warning("This will revoke break-glass credential '%s' from cluster '%s'", opts.CredentialID, opts.ClusterName)
		
		// Get the credential to verify it exists
		credential, err := bgSvc.Get(ctx, opts.ClusterName, opts.CredentialID)
		if err != nil {
			return fmt.Errorf("failed to get credential: %w", err)
		}

		if credential.Status == "revoked" {
			writer.Info("Credential '%s' is already revoked", opts.CredentialID)
			return nil
		}

		writer.Info("Revoking break-glass credential '%s' (%s)...", opts.CredentialID, credential.Username)
		err = bgSvc.Revoke(ctx, opts.ClusterName, opts.CredentialID)
		if err != nil {
			return fmt.Errorf("failed to revoke credential: %w", err)
		}

		writer.Success("Successfully revoked break-glass credential '%s' from cluster '%s'", opts.CredentialID, opts.ClusterName)
	}

	return nil
}
