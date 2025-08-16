package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/cluster"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// EditOptions contains options for editing a cluster
type EditOptions struct {
	ClusterName string

	// Scaling options
	MinReplicas *int
	MaxReplicas *int

	// Network options
	Private  *bool
	ProxyURL string
	NoProxy  string

	// Display options
	DisplayName string

	// Labels and tags
	Labels map[string]string
	Tags   map[string]string

	// Monitoring
	DisableWorkloadMonitoring *bool

	// Interactive mode
	Interactive bool
	DryRun      bool
}

// NewEditClusterCommand creates the cluster edit command
func NewEditClusterCommand(logger *slog.Logger) *cobra.Command {
	opts := &EditOptions{
		Labels: make(map[string]string),
		Tags:   make(map[string]string),
	}

	// Initialize flag variables
	var minReplicas, maxReplicas int
	var private, disableWorkloadMonitoring bool

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit cluster properties",
		Long: `Edit properties of an existing ROSA HCP cluster.

This command allows you to modify various cluster settings such as:
- Scaling configuration (min/max replicas)
- Network settings (private, proxy)
- Display name and labels
- Monitoring configuration`,
		Example: `  # Scale cluster compute nodes
  rosa edit cluster --cluster my-cluster --min-replicas 3 --max-replicas 10

  # Make cluster private
  rosa edit cluster --cluster my-cluster --private

  # Update display name
  rosa edit cluster --cluster my-cluster --display-name "Production Cluster"

  # Add labels
  rosa edit cluster --cluster my-cluster --labels env=prod,team=platform

  # Interactive mode
  rosa edit cluster --cluster my-cluster --interactive`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set the pointer values if flags were changed
			if cmd.Flags().Changed("min-replicas") {
				opts.MinReplicas = &minReplicas
			}
			if cmd.Flags().Changed("max-replicas") {
				opts.MaxReplicas = &maxReplicas
			}
			if cmd.Flags().Changed("private") {
				opts.Private = &private
			}
			if cmd.Flags().Changed("disable-workload-monitoring") {
				opts.DisableWorkloadMonitoring = &disableWorkloadMonitoring
			}
			return runEditCluster(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster")

	// Scaling flags
	flags.IntVar(&minReplicas, "min-replicas", 0, "Minimum number of compute nodes")
	flags.IntVar(&maxReplicas, "max-replicas", 0, "Maximum number of compute nodes")

	// Network flags
	flags.BoolVar(&private, "private", false, "Restrict API endpoint to private access")
	flags.StringVar(&opts.ProxyURL, "http-proxy", "", "HTTP proxy URL")
	flags.StringVar(&opts.NoProxy, "no-proxy", "", "Comma-separated list of destinations to bypass proxy")

	// Display flags
	flags.StringVar(&opts.DisplayName, "display-name", "", "Display name for the cluster")

	// Labels and tags
	flags.StringToStringVar(&opts.Labels, "labels", nil, "Labels to apply to the cluster (key=value)")
	flags.StringToStringVar(&opts.Tags, "tags", nil, "AWS tags to apply to cluster resources (key=value)")

	// Monitoring flags
	flags.BoolVar(&disableWorkloadMonitoring, "disable-workload-monitoring", false, "Disable workload monitoring")

	// Operational flags
	flags.BoolVarP(&opts.Interactive, "interactive", "i", false, "Interactive mode")
	flags.BoolVar(&opts.DryRun, "dry-run", false, "Show what would be changed without making changes")

	return cmd
}

func runEditCluster(ctx context.Context, logger *slog.Logger, opts *EditOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Interactive mode
	if opts.Interactive {
		if err := promptForEditOptions(ctx, opts); err != nil {
			return err
		}
	}

	// Validate options
	if opts.ClusterName == "" {
		return fmt.Errorf("cluster name is required")
	}

	// Check if any changes were requested
	if !hasChanges(opts) {
		return fmt.Errorf("no changes specified")
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

	// Create cluster service
	clusterService := cluster.NewService(apiClient, nil, logger)

	// Get current cluster state
	writer.Info("Fetching cluster details...")
	currentCluster, err := clusterService.Get(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Show current state and proposed changes
	writer.Title("Current Cluster Configuration")
	writer.KeyValue(map[string]string{
		"Name":         currentCluster.Name,
		"ID":           currentCluster.ID,
		"State":        currentCluster.State,
		"Display Name": valueOrDefaultStr(currentCluster.DisplayName, currentCluster.Name),
	})

	fmt.Println()
	writer.Title("Proposed Changes")

	changes := make(map[string]string)

	if opts.MinReplicas != nil && *opts.MinReplicas > 0 {
		changes["Min Replicas"] = fmt.Sprintf("%d → %d", currentCluster.MinReplicas, *opts.MinReplicas)
	}
	if opts.MaxReplicas != nil && *opts.MaxReplicas > 0 {
		changes["Max Replicas"] = fmt.Sprintf("%d → %d", currentCluster.MaxReplicas, *opts.MaxReplicas)
	}
	if opts.Private != nil {
		changes["Private"] = fmt.Sprintf("%v → %v", currentCluster.Private, *opts.Private)
	}
	if opts.DisplayName != "" && opts.DisplayName != currentCluster.DisplayName {
		changes["Display Name"] = fmt.Sprintf("%s → %s",
			valueOrDefaultStr(currentCluster.DisplayName, "none"),
			opts.DisplayName)
	}
	if len(opts.Labels) > 0 {
		changes["Labels"] = formatMapChange(currentCluster.Labels, opts.Labels)
	}
	if len(opts.Tags) > 0 {
		changes["AWS Tags"] = formatMapChange(currentCluster.Tags, opts.Tags)
	}
	if opts.DisableWorkloadMonitoring != nil {
		changes["Workload Monitoring"] = fmt.Sprintf("%v → %v",
			!currentCluster.DisableWorkloadMonitoring,
			!*opts.DisableWorkloadMonitoring)
	}

	if len(changes) == 0 {
		writer.Info("No changes to apply")
		return nil
	}

	writer.KeyValue(changes)

	if opts.DryRun {
		fmt.Println()
		writer.Info("DRY RUN: No changes will be made")
		return nil
	}

	// Confirm changes
	fmt.Println()
	var confirm bool
	err = huh.NewConfirm().
		Title("Apply changes?").
		Description("Do you want to apply these changes to the cluster?").
		Value(&confirm).
		Run()
	if err != nil {
		return err
	}

	if !confirm {
		writer.Info("Edit cancelled")
		return nil
	}

	// Apply changes
	writer.Info("Applying changes to cluster...")

	// Build update request
	updateOpts := cluster.UpdateOptions{
		ClusterID:                 currentCluster.ID,
		DisplayName:               opts.DisplayName,
		Labels:                    opts.Labels,
		Tags:                      opts.Tags,
		MinReplicas:               opts.MinReplicas,
		MaxReplicas:               opts.MaxReplicas,
		Private:                   opts.Private,
		ProxyURL:                  opts.ProxyURL,
		NoProxy:                   opts.NoProxy,
		DisableWorkloadMonitoring: opts.DisableWorkloadMonitoring,
	}

	err = clusterService.Update(ctx, updateOpts)
	if err != nil {
		return fmt.Errorf("failed to update cluster: %w", err)
	}

	writer.Success("Cluster updated successfully!")

	// Show updated configuration
	fmt.Println()
	writer.Info("Fetching updated cluster details...")
	updatedCluster, err := clusterService.Get(ctx, opts.ClusterName)
	if err != nil {
		logger.Warn("Could not fetch updated cluster details", "error", err)
	} else {
		writer.Title("Updated Cluster Configuration")
		writer.KeyValue(map[string]string{
			"Name":         updatedCluster.Name,
			"Display Name": valueOrDefaultStr(updatedCluster.DisplayName, updatedCluster.Name),
			"Min Replicas": fmt.Sprintf("%d", updatedCluster.MinReplicas),
			"Max Replicas": fmt.Sprintf("%d", updatedCluster.MaxReplicas),
			"Private":      fmt.Sprintf("%v", updatedCluster.Private),
			"State":        updatedCluster.State,
		})
	}

	return nil
}

func hasChanges(opts *EditOptions) bool {
	return (opts.MinReplicas != nil && *opts.MinReplicas > 0) ||
		(opts.MaxReplicas != nil && *opts.MaxReplicas > 0) ||
		opts.Private != nil ||
		opts.ProxyURL != "" ||
		opts.NoProxy != "" ||
		opts.DisplayName != "" ||
		len(opts.Labels) > 0 ||
		len(opts.Tags) > 0 ||
		opts.DisableWorkloadMonitoring != nil
}

func formatMapChange(current, new map[string]string) string {
	if len(current) == 0 && len(new) > 0 {
		return fmt.Sprintf("none → %s", formatMap(new))
	}
	if len(current) > 0 && len(new) > 0 {
		return fmt.Sprintf("%s → %s", formatMap(current), formatMap(new))
	}
	return formatMap(new)
}

func formatMap(m map[string]string) string {
	var pairs []string
	for k, v := range m {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ", ")
}

func promptForEditOptions(ctx context.Context, opts *EditOptions) error {
	// Prompt for cluster name
	if opts.ClusterName == "" {
		err := huh.NewInput().
			Title("Cluster Name").
			Description("Name or ID of the cluster to edit").
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

	// Ask what to edit
	editChoices := []string{
		"Scaling (min/max replicas)",
		"Display name",
		"Network settings",
		"Labels",
		"AWS tags",
		"Monitoring",
	}

	var selectedEdits []string
	err := huh.NewMultiSelect[string]().
		Title("What would you like to edit?").
		Options(huh.NewOptions(editChoices...)...).
		Value(&selectedEdits).
		Run()
	if err != nil {
		return err
	}

	// Prompt for specific changes based on selection
	for _, choice := range selectedEdits {
		switch choice {
		case "Scaling (min/max replicas)":
			var minReplicasStr string
			err := huh.NewInput().
				Title("Minimum Replicas").
				Description("Minimum number of compute nodes (0 to skip)").
				Value(&minReplicasStr).
				Run()
			if err != nil {
				return err
			}
			if minReplicasStr != "" && minReplicasStr != "0" {
				var minReplicas int
				fmt.Sscanf(minReplicasStr, "%d", &minReplicas)
				if minReplicas > 0 {
					opts.MinReplicas = &minReplicas
				}
			}

			var maxReplicasStr string
			err = huh.NewInput().
				Title("Maximum Replicas").
				Description("Maximum number of compute nodes (0 to skip)").
				Value(&maxReplicasStr).
				Run()
			if err != nil {
				return err
			}
			if maxReplicasStr != "" && maxReplicasStr != "0" {
				var maxReplicas int
				fmt.Sscanf(maxReplicasStr, "%d", &maxReplicas)
				if maxReplicas > 0 {
					opts.MaxReplicas = &maxReplicas
				}
			}

		case "Display name":
			err := huh.NewInput().
				Title("Display Name").
				Description("New display name for the cluster").
				Value(&opts.DisplayName).
				Run()
			if err != nil {
				return err
			}

		case "Network settings":
			var makePrivate bool
			err := huh.NewConfirm().
				Title("Make cluster private?").
				Description("Restrict API endpoint to private access").
				Value(&makePrivate).
				Run()
			if err != nil {
				return err
			}
			opts.Private = &makePrivate

		case "Labels":
			var labelsStr string
			err := huh.NewInput().
				Title("Labels").
				Description("Comma-separated key=value pairs (e.g., env=prod,team=platform)").
				Value(&labelsStr).
				Run()
			if err != nil {
				return err
			}
			if labelsStr != "" {
				opts.Labels = parseKeyValuePairs(labelsStr)
			}

		case "AWS tags":
			var tagsStr string
			err := huh.NewInput().
				Title("AWS Tags").
				Description("Comma-separated key=value pairs (e.g., Owner=team,Project=app)").
				Value(&tagsStr).
				Run()
			if err != nil {
				return err
			}
			if tagsStr != "" {
				opts.Tags = parseKeyValuePairs(tagsStr)
			}

		case "Monitoring":
			var disableMonitoring bool
			err := huh.NewConfirm().
				Title("Disable workload monitoring?").
				Description("Disable workload monitoring to save resources").
				Value(&disableMonitoring).
				Run()
			if err != nil {
				return err
			}
			opts.DisableWorkloadMonitoring = &disableMonitoring
		}
	}

	return nil
}

func parseKeyValuePairs(input string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}

func valueOrDefaultStr(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
