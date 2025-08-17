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

// DescribeOptions contains options for describing break-glass credentials
type DescribeOptions struct {
	ClusterName  string
	CredentialID string
}

// NewDescribeCommand creates the break-glass credential describe command
func NewDescribeCommand(logger *slog.Logger) *cobra.Command {
	opts := &DescribeOptions{}

	cmd := &cobra.Command{
		Use:     "break-glass-credential",
		Aliases: []string{"breakglasscredential"},
		Short:   "Show details of a break-glass credential",
		Long:    "Display detailed information about a specific break-glass credential.",
		Example: `  # Describe a break-glass credential
  rosa describe break-glass-credential --cluster=mycluster --credential-id=abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDescribeBreakGlass(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVar(&opts.CredentialID, "credential-id", "", "ID of the credential to describe (required)")

	cmd.MarkFlagRequired("cluster")
	cmd.MarkFlagRequired("credential-id")

	return cmd
}

func runDescribeBreakGlass(ctx context.Context, logger *slog.Logger, opts *DescribeOptions) error {
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

	// Create break-glass service
	bgSvc, err := breakglassSvc.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create break-glass service: %w", err)
	}

	// Get the credential
	credential, err := bgSvc.Get(ctx, opts.ClusterName, opts.CredentialID)
	if err != nil {
		return fmt.Errorf("failed to get break-glass credential: %w", err)
	}

	// Display credential details
	writer.Title("Break-glass Credential Details")
	
	details := map[string]string{
		"ID":              credential.ID,
		"Cluster":         opts.ClusterName,
		"Username":        credential.Username,
		"Status":          credential.Status,
		"Created":         breakglassSvc.FormatTime(credential.CreatedAt),
	}
	
	if !credential.ExpirationTime.IsZero() {
		details["Expires"] = breakglassSvc.FormatTime(credential.ExpirationTime)
	}
	
	if !credential.RevokedAt.IsZero() {
		details["Revoked"] = breakglassSvc.FormatTime(credential.RevokedAt)
	}
	
	if credential.Description != "" {
		details["Description"] = credential.Description
	}
	
	writer.KeyValue(details)

	// Show status-specific information
	switch credential.Status {
	case "issued":
		if credential.IsActive() {
			writer.Success("✓ Credential is active and can be used for emergency access")
			
			// Check if kubeconfig is available
			if credential.Kubeconfig != "" {
				writer.Info("\n📋 Kubeconfig is available. To retrieve it:")
				fmt.Printf("  rosa get break-glass-credential --cluster %s --credential-id %s --kubeconfig\n", 
					opts.ClusterName, opts.CredentialID)
			}
		} else {
			writer.Warning("⚠ Credential has been issued but may have expired")
		}
	case "pending":
		writer.Info("⏳ Credential is pending issuance. Please wait a few moments.")
	case "revoked":
		writer.Error("✗ Credential has been revoked and cannot be used")
	case "expired":
		writer.Warning("⚠ Credential has expired and cannot be used")
	default:
		writer.Info("Status: %s", credential.Status)
	}

	// Show commands for actions
	if credential.Status == "issued" && credential.IsActive() {
		writer.Info("\n📋 Available Actions:")
		fmt.Printf("  # Revoke this credential:\n")
		fmt.Printf("  rosa revoke break-glass-credential --cluster %s --credential-id %s\n", 
			opts.ClusterName, opts.CredentialID)
	}

	return nil
}
