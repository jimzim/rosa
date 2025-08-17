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

// InstallOptions contains options for viewing installation logs
type InstallOptions struct {
	ClusterName string
	Tail        int
	Watch       bool
}

// NewInstallCommand creates the logs install command
func NewInstallCommand(logger *slog.Logger) *cobra.Command {
	opts := &InstallOptions{}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Show cluster installation logs",
		Long: `Show installation logs for a ROSA HCP cluster.

This command displays the installation progress and any errors that occur
during cluster creation.`,
		Example: `  # Show last 100 install log lines for a cluster
  rosa logs install --cluster my-cluster --tail 100

  # Watch installation logs in real-time
  rosa logs install --cluster my-cluster --watch`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstallLogs(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.IntVar(&opts.Tail, "tail", 2000, "Number of lines to show from the end of the log")
	flags.BoolVarP(&opts.Watch, "watch", "w", false, "Watch for new log entries")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runInstallLogs(ctx context.Context, logger *slog.Logger, opts *InstallOptions) error {
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
	case cmv1.ClusterStateReady:
		writer.Success("Cluster '%s' has been successfully installed", cluster.Name())
		return nil
	case cmv1.ClusterStatePending:
		writer.Info("Cluster '%s' is pending installation. Logs will be available shortly...", cluster.Name())
	case cmv1.ClusterStateInstalling:
		writer.Info("Cluster '%s' is currently installing...", cluster.Name())
	case cmv1.ClusterStateError:
		writer.Error("Cluster '%s' failed to install", cluster.Name())
	case cmv1.ClusterStateUninstalling:
		writer.Warning("Cluster '%s' is being uninstalled", cluster.Name())
		return fmt.Errorf("cluster is being uninstalled")
	}

	// Get installation logs
	logResponse, err := apiClient.GetConnection().ClustersMgmt().V1().
		Clusters().Cluster(cluster.ID()).
		Logs().Install().
		Get().
		Tail(opts.Tail).
		SendContext(ctx)
	if err != nil {
		// Check if logs are not yet available
		if cluster.State() == cmv1.ClusterStatePending {
			writer.Info("Installation logs are not yet available. Please wait a few minutes and try again.")
			return nil
		}
		return fmt.Errorf("failed to get installation logs: %w", err)
	}

	// Display logs
	logs := logResponse.Body().Content()
	if logs != "" {
		fmt.Println(logs)
	} else {
		writer.Info("No installation logs available yet")
	}

	// Watch mode
	if opts.Watch && cluster.State() != cmv1.ClusterStateReady {
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

				// Check if installation is complete
				if cluster.State() == cmv1.ClusterStateReady {
					writer.Success("\n✓ Cluster installation completed successfully!")
					return nil
				}

				// Get latest logs
				logResponse, err := apiClient.GetConnection().ClustersMgmt().V1().
					Clusters().Cluster(cluster.ID()).
					Logs().Install().
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
