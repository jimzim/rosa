package version

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
	"github.com/openshift/rosa-hcp/pkg/version"
)

type listVersionsOptions struct {
	channelGroup string
	hcpOnly      bool
	showAll      bool
}

// NewListCommand creates the list versions command
func NewListCommand(logger *slog.Logger) *cobra.Command {
	opts := &listVersionsOptions{}

	cmd := &cobra.Command{
		Use:   "versions",
		Short: "List available OpenShift versions",
		Long: `List available OpenShift versions for ROSA HCP clusters.

This command shows all OpenShift versions that can be used when creating
new clusters or upgrading existing ones. Versions are grouped by channel
and sorted by version number.`,
		Example: `  # List all available HCP versions
  rosa list versions

  # List versions for a specific channel group
  rosa list versions --channel-group stable

  # Show all versions including unavailable ones
  rosa list versions --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListVersions(cmd.Context(), logger, opts)
		},
	}

	cmd.Flags().StringVar(&opts.channelGroup, "channel-group", "", "Filter by channel group (stable, candidate, fast, nightly)")
	cmd.Flags().BoolVar(&opts.hcpOnly, "hcp-only", true, "Show only HCP-compatible versions")
	cmd.Flags().BoolVar(&opts.showAll, "all", false, "Show all versions including unavailable ones")

	return cmd
}

func runListVersions(ctx context.Context, logger *slog.Logger, opts *listVersionsOptions) error {
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

	// Create version service
	versionService := version.NewService(apiClient, logger)

	writer.Info("Fetching available OpenShift versions...")

	// List versions
	versions, err := versionService.List(ctx, version.ListOptions{
		ChannelGroup: opts.channelGroup,
		HCPOnly:      opts.hcpOnly,
		Available:    !opts.showAll,
	})
	if err != nil {
		return fmt.Errorf("failed to list versions: %w", err)
	}

	if len(versions) == 0 {
		writer.Info("No versions found matching the criteria")
		return nil
	}

	// Group versions by channel
	versionsByChannel := make(map[string][]*version.Version)
	for _, v := range versions {
		channel := v.ChannelGroup
		if channel == "" {
			channel = "unknown"
		}
		versionsByChannel[channel] = append(versionsByChannel[channel], v)
	}

	// Display versions by channel
	for channel, channelVersions := range versionsByChannel {
		writer.Title(fmt.Sprintf("Channel: %s", channel))
		fmt.Println()

		headers := []string{"VERSION", "DEFAULT", "AVAILABLE", "HCP COMPATIBLE", "UPGRADES AVAILABLE"}
		var rows [][]string

		for _, v := range channelVersions {
			// Format version display
			versionStr := v.Version
			if v.HCPDefault {
				versionStr += " ★" // Star for HCP default
			}

			defaultStr := "No"
			if v.Default {
				defaultStr = "Yes"
			}

			availableStr := "No"
			if v.Available {
				availableStr = "Yes"
			}

			hcpStr := "No"
			if v.HCPAvailable {
				hcpStr = "Yes"
			}

			upgradesStr := fmt.Sprintf("%d", len(v.UpgradesTo))

			rows = append(rows, []string{
				versionStr,
				defaultStr,
				availableStr,
				hcpStr,
				upgradesStr,
			})
		}

		writer.Table(headers, rows)
		fmt.Println()
	}

	// Show legend
	writer.Info("Legend:")
	fmt.Println("  ★ = Default version for HCP clusters")
	fmt.Println()

	// Show summary
	totalCount := len(versions)
	availableCount := 0
	hcpCount := 0
	for _, v := range versions {
		if v.Available {
			availableCount++
		}
		if v.HCPAvailable {
			hcpCount++
		}
	}

	writer.KeyValue(map[string]string{
		"Total Versions": fmt.Sprintf("%d", totalCount),
		"Available":      fmt.Sprintf("%d", availableCount),
		"HCP Compatible": fmt.Sprintf("%d", hcpCount),
	})

	// Show how to use versions
	fmt.Println()
	writer.Info("To create a cluster with a specific version:")
	fmt.Println("  rosa cluster create --version <version> ...")
	fmt.Println()
	writer.Info("To upgrade a cluster:")
	fmt.Println("  rosa upgrade cluster --cluster <name> --version <version>")

	return nil
}

// NewUpgradePathsCommand creates the command to list upgrade paths for a cluster
func NewUpgradePathsCommand(logger *slog.Logger) *cobra.Command {
	var clusterName string

	cmd := &cobra.Command{
		Use:   "upgrades",
		Short: "List available upgrades for a cluster",
		Long: `List available upgrade versions for a specific ROSA HCP cluster.

This command shows all OpenShift versions that your cluster can be upgraded to,
based on the current version and upgrade compatibility matrix.`,
		Example: `  # List available upgrades for a cluster
  rosa list upgrades --cluster my-cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListUpgradePaths(cmd.Context(), logger, clusterName)
		},
	}

	cmd.Flags().StringVarP(&clusterName, "cluster", "c", "", "Name or ID of the cluster")
	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runListUpgradePaths(ctx context.Context, logger *slog.Logger, clusterName string) error {
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

	// Create version service
	versionService := version.NewService(apiClient, logger)

	writer.Info(fmt.Sprintf("Fetching available upgrades for cluster '%s'...", clusterName))

	// Get upgrade paths
	upgrades, err := versionService.GetUpgradePaths(ctx, clusterName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cluster '%s' not found", clusterName)
		}
		return fmt.Errorf("failed to get upgrade paths: %w", err)
	}

	if len(upgrades) == 0 {
		writer.Info("No upgrades available for this cluster")
		writer.Info("The cluster may already be on the latest version")
		return nil
	}

	// Display available upgrades
	writer.Title(fmt.Sprintf("Available Upgrades for cluster '%s'", clusterName))
	fmt.Println()

	headers := []string{"VERSION", "CHANNEL", "AVAILABLE", "NOTES"}
	var rows [][]string

	for _, upgrade := range upgrades {
		availableStr := "Yes"
		if !upgrade.Available {
			availableStr = "No (not yet released)"
		}

		notes := ""
		if upgrade.HCPDefault {
			notes = "Recommended"
		}

		rows = append(rows, []string{
			upgrade.Version,
			upgrade.ChannelGroup,
			availableStr,
			notes,
		})
	}

	writer.Table(headers, rows)

	fmt.Println()
	writer.Info("To upgrade your cluster:")
	fmt.Printf("  rosa upgrade cluster --cluster %s --version <version>\n", clusterName)
	fmt.Println()
	writer.Info("To schedule an upgrade:")
	fmt.Printf("  rosa upgrade cluster --cluster %s --version <version> --schedule-date <date> --schedule-time <time>\n", clusterName)

	return nil
}
