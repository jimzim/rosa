package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/cluster"
	"github.com/openshift/rosa-hcp/pkg/output"
	"github.com/openshift/rosa-hcp/pkg/version"
)

// UpgradeOptions contains options for upgrading a cluster
type UpgradeOptions struct {
	ClusterName string
	Version     string

	// Scheduling options
	ScheduleDate string
	ScheduleTime string
	NextRun      bool

	// Control options
	DryRun      bool
	Interactive bool
}

// NewUpgradeCommand creates the cluster upgrade command
func NewUpgradeCommand(logger *slog.Logger) *cobra.Command {
	opts := &UpgradeOptions{}

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade a ROSA HCP cluster",
		Long: `Upgrade a ROSA HCP cluster to a newer OpenShift version.

This command allows you to upgrade your cluster immediately or schedule an
upgrade for a future time. Use 'rosa list versions' to see available versions
and 'rosa list upgrades --cluster <name>' to see available upgrade paths.`,
		Example: `  # Upgrade cluster immediately
  rosa upgrade cluster --cluster my-cluster --version 4.14.5

  # Schedule an upgrade
  rosa upgrade cluster --cluster my-cluster --version 4.14.5 \
    --schedule-date 2024-01-15 --schedule-time 02:00

  # Interactive mode to select version
  rosa upgrade cluster --cluster my-cluster --interactive

  # Dry run to see what would happen
  rosa upgrade cluster --cluster my-cluster --version 4.14.5 --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgradeCluster(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster")
	flags.StringVar(&opts.Version, "version", "", "OpenShift version to upgrade to")

	// Scheduling flags
	flags.StringVar(&opts.ScheduleDate, "schedule-date", "", "Schedule date for upgrade (YYYY-MM-DD)")
	flags.StringVar(&opts.ScheduleTime, "schedule-time", "", "Schedule time for upgrade in UTC (HH:MM)")
	flags.BoolVar(&opts.NextRun, "next-run", false, "Schedule upgrade for next maintenance window")

	// Control flags
	flags.BoolVar(&opts.DryRun, "dry-run", false, "Show what would be upgraded without making changes")
	flags.BoolVarP(&opts.Interactive, "interactive", "i", false, "Interactive mode")

	return cmd
}

func runUpgradeCluster(ctx context.Context, logger *slog.Logger, opts *UpgradeOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Interactive mode
	if opts.Interactive {
		if err := promptForUpgradeOptions(ctx, opts); err != nil {
			return err
		}
	}

	// Validate options
	if opts.ClusterName == "" {
		return fmt.Errorf("cluster name is required")
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

	// Create services
	clusterService := cluster.NewService(apiClient, nil, logger)
	versionService := version.NewService(apiClient, logger)

	// Get current cluster state
	writer.Info("Fetching cluster details...")
	currentCluster, err := clusterService.Get(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// If no version specified, show available upgrades
	if opts.Version == "" && !opts.Interactive {
		writer.Title(fmt.Sprintf("Available upgrades for cluster '%s'", currentCluster.Name))
		fmt.Println()

		upgrades, err := versionService.GetUpgradePaths(ctx, currentCluster.ID)
		if err != nil {
			return fmt.Errorf("failed to get upgrade paths: %w", err)
		}

		if len(upgrades) == 0 {
			writer.Info("No upgrades available. The cluster may already be on the latest version.")
			return nil
		}

		headers := []string{"VERSION", "CHANNEL", "AVAILABLE"}
		var rows [][]string

		for _, upgrade := range upgrades {
			availableStr := "Yes"
			if !upgrade.Available {
				availableStr = "No"
			}

			rows = append(rows, []string{
				upgrade.Version,
				upgrade.ChannelGroup,
				availableStr,
			})
		}

		writer.Table(headers, rows)

		fmt.Println()
		writer.Info("To upgrade, run:")
		fmt.Printf("  rosa upgrade cluster --cluster %s --version <version>\n", opts.ClusterName)

		return nil
	}

	// Parse schedule if provided
	var scheduleTime *time.Time
	if opts.ScheduleDate != "" || opts.ScheduleTime != "" {
		if opts.ScheduleDate == "" || opts.ScheduleTime == "" {
			return fmt.Errorf("both --schedule-date and --schedule-time are required for scheduled upgrades")
		}

		// Parse the schedule
		scheduleStr := fmt.Sprintf("%s %s", opts.ScheduleDate, opts.ScheduleTime)
		parsedTime, err := time.Parse("2006-01-02 15:04", scheduleStr)
		if err != nil {
			return fmt.Errorf("invalid schedule format: %w", err)
		}

		// Ensure it's in the future
		if parsedTime.Before(time.Now()) {
			return fmt.Errorf("scheduled time must be in the future")
		}

		scheduleTime = &parsedTime
	}

	// Show upgrade details
	writer.Title("Upgrade Details")
	writer.KeyValue(map[string]string{
		"Cluster":         currentCluster.Name,
		"Current Version": currentCluster.Version,
		"Target Version":  opts.Version,
		"Upgrade Type":    getUpgradeType(currentCluster.Version, opts.Version),
	})

	if scheduleTime != nil {
		writer.KeyValue(map[string]string{
			"Scheduled For": scheduleTime.Format(time.RFC3339),
		})
	} else {
		writer.KeyValue(map[string]string{
			"Timing": "Immediate",
		})
	}

	if opts.DryRun {
		fmt.Println()
		writer.Info("DRY RUN: No changes will be made")
		return nil
	}

	// Confirm upgrade
	fmt.Println()
	writer.Warning("Cluster upgrades cannot be rolled back once started.")

	var confirm bool
	err = huh.NewConfirm().
		Title("Proceed with upgrade?").
		Description(fmt.Sprintf("Upgrade cluster '%s' from %s to %s?",
			currentCluster.Name, currentCluster.Version, opts.Version)).
		Value(&confirm).
		Run()
	if err != nil {
		return err
	}

	if !confirm {
		writer.Info("Upgrade cancelled")
		return nil
	}

	// Perform upgrade
	writer.Info("Initiating cluster upgrade...")

	upgradeOpts := cluster.UpgradeOptions{
		ClusterID:    currentCluster.ID,
		Version:      opts.Version,
		ScheduleTime: scheduleTime,
	}

	upgradeID, err := clusterService.Upgrade(ctx, upgradeOpts)
	if err != nil {
		return fmt.Errorf("failed to initiate upgrade: %w", err)
	}

	writer.Success("Cluster upgrade initiated successfully!")

	// Show upgrade status
	fmt.Println()
	writer.Title("Upgrade Information")
	writer.KeyValue(map[string]string{
		"Upgrade ID":     upgradeID,
		"Cluster":        currentCluster.Name,
		"Target Version": opts.Version,
	})

	if scheduleTime != nil {
		writer.KeyValue(map[string]string{
			"Scheduled For": scheduleTime.Format(time.RFC3339),
			"Status":        "Scheduled",
		})

		fmt.Println()
		writer.Info("The upgrade has been scheduled.")
		writer.Info("To cancel the scheduled upgrade:")
		fmt.Printf("  rosa delete upgrade --cluster %s --upgrade-id %s\n", opts.ClusterName, upgradeID)
	} else {
		writer.KeyValue(map[string]string{
			"Status": "In Progress",
		})

		fmt.Println()
		writer.Info("The upgrade is now in progress. This may take 30-60 minutes.")
		writer.Info("To check upgrade status:")
		fmt.Printf("  rosa describe upgrade --cluster %s\n", opts.ClusterName)
	}

	return nil
}

// CancelUpgradeCommand creates the command to cancel a scheduled upgrade
func NewCancelUpgradeCommand(logger *slog.Logger) *cobra.Command {
	var clusterName, upgradeID string

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Cancel a scheduled cluster upgrade",
		Long:  `Cancel a scheduled cluster upgrade. Only scheduled upgrades that haven't started can be cancelled.`,
		Example: `  # Cancel a scheduled upgrade
  rosa delete upgrade --cluster my-cluster --upgrade-id abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCancelUpgrade(cmd.Context(), logger, clusterName, upgradeID)
		},
	}

	cmd.Flags().StringVarP(&clusterName, "cluster", "c", "", "Name or ID of the cluster")
	cmd.Flags().StringVar(&upgradeID, "upgrade-id", "", "ID of the upgrade to cancel")
	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runCancelUpgrade(ctx context.Context, logger *slog.Logger, clusterName, upgradeID string) error {
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

	// Create cluster service
	clusterService := cluster.NewService(apiClient, nil, logger)

	// Confirm cancellation
	var confirm bool
	err = huh.NewConfirm().
		Title("Cancel upgrade?").
		Description(fmt.Sprintf("Cancel scheduled upgrade for cluster '%s'?", clusterName)).
		Value(&confirm).
		Run()
	if err != nil {
		return err
	}

	if !confirm {
		writer.Info("Cancellation aborted")
		return nil
	}

	// Cancel the upgrade
	writer.Info("Cancelling upgrade...")
	err = clusterService.CancelUpgrade(ctx, clusterName, upgradeID)
	if err != nil {
		return fmt.Errorf("failed to cancel upgrade: %w", err)
	}

	writer.Success("Upgrade cancelled successfully")

	return nil
}

func promptForUpgradeOptions(ctx context.Context, opts *UpgradeOptions) error {
	// Prompt for cluster name
	if opts.ClusterName == "" {
		err := huh.NewInput().
			Title("Cluster Name").
			Description("Name or ID of the cluster to upgrade").
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("cluster name cannot be empty")
				}
				return nil
			}).
			Value(&opts.ClusterName).
			Run()
		if err != nil {
			return err
		}
	}

	// TODO: Fetch available versions and let user select
	// For now, just prompt for version string
	if opts.Version == "" {
		err := huh.NewInput().
			Title("Target Version").
			Description("OpenShift version to upgrade to (e.g., 4.14.5)").
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("version cannot be empty")
				}
				return nil
			}).
			Value(&opts.Version).
			Run()
		if err != nil {
			return err
		}
	}

	// Ask about scheduling
	var schedule bool
	err := huh.NewConfirm().
		Title("Schedule upgrade?").
		Description("Schedule the upgrade for a specific time?").
		Value(&schedule).
		Run()
	if err != nil {
		return err
	}

	if schedule {
		// Get schedule date
		err := huh.NewInput().
			Title("Schedule Date").
			Description("Date for upgrade (YYYY-MM-DD)").
			Placeholder(time.Now().AddDate(0, 0, 1).Format("2006-01-02")).
			Value(&opts.ScheduleDate).
			Run()
		if err != nil {
			return err
		}

		// Get schedule time
		err = huh.NewInput().
			Title("Schedule Time").
			Description("Time for upgrade in UTC (HH:MM)").
			Placeholder("02:00").
			Value(&opts.ScheduleTime).
			Run()
		if err != nil {
			return err
		}
	}

	return nil
}

func getUpgradeType(currentVersion, targetVersion string) string {
	// Remove version prefixes
	current := strings.TrimPrefix(currentVersion, "openshift-v")
	target := strings.TrimPrefix(targetVersion, "openshift-v")

	// Parse versions
	currentParts := strings.Split(current, ".")
	targetParts := strings.Split(target, ".")

	if len(currentParts) < 2 || len(targetParts) < 2 {
		return "Version Change"
	}

	// Check major version
	if currentParts[0] != targetParts[0] {
		return "Major Upgrade"
	}

	// Check minor version
	if currentParts[1] != targetParts[1] {
		return "Minor Upgrade"
	}

	return "Patch Upgrade"
}
