package logs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// UninstallOptions contains options for viewing uninstallation logs
type UninstallOptions struct {
	ClusterName string
	Tail        int
	Watch       bool
}

// NewUninstallCommand creates the logs uninstall command
func NewUninstallCommand(logger *slog.Logger) *cobra.Command {
	opts := &UninstallOptions{}

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Show cluster uninstallation logs",
		Long: `Show uninstallation logs for a ROSA HCP cluster.

This command displays the uninstallation progress and any errors that occur
during cluster deletion.`,
		Example: `  # Show last 100 uninstall log lines for a cluster
  rosa logs uninstall --cluster my-cluster --tail 100

  # Watch uninstallation logs in real-time
  rosa logs uninstall --cluster my-cluster --watch`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstallLogs(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.IntVar(&opts.Tail, "tail", 2000, "Number of lines to show from the end of the log")
	flags.BoolVarP(&opts.Watch, "watch", "w", false, "Watch for new log entries")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runUninstallLogs(ctx context.Context, logger *slog.Logger, opts *UninstallOptions) error {
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

	// Get cluster
	clusterResp, err := apiClient.GetCluster(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	cluster := clusterResp.Body()

	// Check cluster state
	switch cluster.State() {
	case cmv1.ClusterStateUninstalling:
		writer.Info("Cluster '%s' is currently uninstalling...", cluster.Name())
	case cmv1.ClusterStateInstalling, cmv1.ClusterStatePending, cmv1.ClusterStateWaiting:
		writer.Error("Cluster '%s' is in '%s' state and no uninstallation logs are available", 
			cluster.Name(), cluster.State())
		return fmt.Errorf("no uninstallation logs available for cluster in %s state", cluster.State())
	case cmv1.ClusterStateReady:
		writer.Warning("Cluster '%s' is not being uninstalled (state: ready)", cluster.Name())
		if !opts.Watch {
			return fmt.Errorf("cluster is not being uninstalled")
		}
	}

	// Get uninstallation logs
	logResponse, err := apiClient.GetConnection().ClustersMgmt().V1().
		Clusters().Cluster(cluster.ID()).
		Logs().Uninstall().
		Get().
		Tail(opts.Tail).
		SendContext(ctx)
	if err != nil {
		// Check if logs are not yet available
		if cluster.State() != cmv1.ClusterStateUninstalling {
			writer.Info("Uninstallation logs are not available for cluster in '%s' state", cluster.State())
			return nil
		}
		return fmt.Errorf("failed to get uninstallation logs: %w", err)
	}

	// Display logs
	logs := logResponse.Body().Content()
	if logs != "" {
		fmt.Println(logs)
	} else {
		writer.Info("No uninstallation logs available yet")
	}

	// Watch mode
	if opts.Watch && cluster.State() == cmv1.ClusterStateUninstalling {
		writer.Info("\nWatching for new log entries... (Press Ctrl+C to stop)")
		writer.Info("Checking for updates every 5 seconds...")

		lastContent := logs
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				// Get cluster state
				clusterResp, err := apiClient.GetCluster(ctx, opts.ClusterName)
				if err != nil {
					logger.WarnContext(ctx, "failed to check cluster state", slog.String("error", err.Error()))
					continue
				}
				cluster = clusterResp.Body()

				// Check if uninstallation is complete
				if cluster.State() != cmv1.ClusterStateUninstalling {
					writer.Success("\n✓ Cluster uninstallation completed")
					return nil
				}

				// Get latest logs
				logResponse, err := apiClient.GetConnection().ClustersMgmt().V1().
					Clusters().Cluster(cluster.ID()).
					Logs().Uninstall().
					Get().
					SendContext(ctx)
				if err != nil {
					logger.WarnContext(ctx, "failed to get logs", slog.String("error", err.Error()))
					continue
				}

				newContent := logResponse.Body().Content()
				if newContent != lastContent {
					// Print only the new content
					if len(newContent) > len(lastContent) {
						fmt.Print(newContent[len(lastContent):])
					}
					lastContent = newContent
				}
			}
		}
	}

	return nil
}
