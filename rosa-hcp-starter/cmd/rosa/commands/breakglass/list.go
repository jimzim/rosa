package breakglass

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/breakglass"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// ListOptions contains options for listing break-glass credentials
type ListOptions struct {
	ClusterName  string
	OutputFormat string
}

// NewListCommand creates the break-glass-credentials list command
func NewListCommand(logger *slog.Logger) *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:     "break-glass-credentials",
		Aliases: []string{"break-glass-credential", "breakglass"},
		Short:   "List break-glass credentials for a cluster",
		Long:    "List all break-glass credentials configured for emergency access to a cluster.",
		Example: `  # List all break-glass credentials for a cluster
  rosa list break-glass-credentials --cluster my-cluster

  # Output in JSON format
  rosa list break-glass-credentials --cluster my-cluster --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListBreakGlass(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format (text, json, yaml)")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runListBreakGlass(ctx context.Context, logger *slog.Logger, opts *ListOptions) error {
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

	// Create Break-glass service
	bgSvc, err := breakglass.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create break-glass service: %w", err)
	}

	// Check if supported
	supported, reason, err := bgSvc.IsSupported(ctx, cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to check break-glass support: %w", err)
	}
	if !supported {
		writer := output.NewWriter(output.FormatText)
		writer.Error(reason)
		return fmt.Errorf("break-glass credentials not available")
	}

	// List credentials
	credentials, err := bgSvc.List(ctx, cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to list break-glass credentials: %w", err)
	}

	// Display results based on output format
	var writer *output.Writer

	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		writer = output.NewWriter(output.FormatJSON)
		return writer.Print(credentials)
	case "yaml":
		writer = output.NewWriter(output.FormatYAML)
		return writer.Print(credentials)
	default:
		writer = output.NewWriter(output.FormatText)

		if len(credentials) == 0 {
			writer.Info("No break-glass credentials found for cluster '%s'", opts.ClusterName)
			writer.Info("\nTo create a break-glass credential:")
			fmt.Printf("  rosa create break-glass-credential --cluster %s\n", opts.ClusterName)
			return nil
		}

		writer.Title(fmt.Sprintf("Break-glass Credentials for cluster '%s'", cluster.Name()))

		// Count active vs inactive
		activeCount := 0
		for _, cred := range credentials {
			if cred.IsActive() {
				activeCount++
			}
		}

		if activeCount > 0 {
			writer.Warning("⚠️  %d ACTIVE emergency credential(s) found", activeCount)
		}
		fmt.Println()

		// Display as table
		fmt.Printf("%-20s %-20s %-15s %-25s %-25s\n",
			"ID", "USERNAME", "STATUS", "CREATED", "EXPIRES")
		fmt.Println(strings.Repeat("-", 105))

		for _, cred := range credentials {
			// Format times
			createdStr := cred.CreatedAt.Format("2006-01-02 15:04:05")
			expiresStr := "N/A"
			if !cred.ExpirationTime.IsZero() {
				if cred.Status == "revoked" {
					expiresStr = "Revoked"
				} else if time.Now().After(cred.ExpirationTime) {
					expiresStr = "Expired"
				} else {
					// Show time remaining
					remaining := time.Until(cred.ExpirationTime)
					if remaining.Hours() > 24 {
						expiresStr = fmt.Sprintf("%.0f days", remaining.Hours()/24)
					} else if remaining.Hours() > 1 {
						expiresStr = fmt.Sprintf("%.0f hours", remaining.Hours())
					} else {
						expiresStr = fmt.Sprintf("%.0f minutes", remaining.Minutes())
					}
				}
			}

			statusStr := breakglass.FormatStatus(cred.Status)

			fmt.Printf("%-20s %-20s %-15s %-25s %-25s\n",
				truncate(cred.ID, 20),
				truncate(cred.Username, 20),
				statusStr,
				createdStr,
				expiresStr)

			// Show description if present
			if cred.Description != "" {
				fmt.Printf("  Description: %s\n", cred.Description)
			}
		}
		fmt.Println()

		// Show statistics
		writer.Info("Summary:")
		writer.KeyValue(map[string]string{
			"Total Credentials": fmt.Sprintf("%d", len(credentials)),
			"Active":            fmt.Sprintf("%d", activeCount),
			"Expired/Revoked":   fmt.Sprintf("%d", len(credentials)-activeCount),
		})

		// Show helpful commands
		if activeCount > 0 {
			writer.Warning("\n⚠️  Active credentials should be revoked after use:")
			for _, cred := range credentials {
				if cred.IsActive() {
					fmt.Printf("  rosa revoke break-glass-credential %s --cluster %s\n",
						cred.ID, opts.ClusterName)
				}
			}
		}

		writer.Info("\nAvailable Commands:")
		writer.Info("  # View details of a specific credential:")
		fmt.Printf("  rosa describe break-glass-credential <id> --cluster %s\n", opts.ClusterName)

		writer.Info("\n  # Revoke all credentials:")
		fmt.Printf("  rosa revoke break-glass-credentials --all --cluster %s\n", opts.ClusterName)

		writer.Info("\n  # Create a new credential:")
		fmt.Printf("  rosa create break-glass-credential --cluster %s\n", opts.ClusterName)
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
