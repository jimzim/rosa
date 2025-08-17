package kubeletconfig

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/kubeletconfig"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// CreateOptions contains options for creating a KubeletConfig
type CreateOptions struct {
	ClusterName  string
	Name         string
	PodPidsLimit int
	Interactive  bool
	DryRun       bool
}

// NewCreateCommand creates the kubeletconfig create command
func NewCreateCommand(logger *slog.Logger) *cobra.Command {
	opts := &CreateOptions{}

	cmd := &cobra.Command{
		Use:   "kubeletconfig",
		Short: "Create a KubeletConfig for a cluster",
		Long: `Create a KubeletConfig to customize kubelet settings for node pools.

This command creates a named KubeletConfig that can be attached to node pools
during creation. Currently supports configuring the pod PIDs limit.`,
		Example: `  # Create a kubeletconfig with high pod density
  rosa create kubeletconfig --cluster my-cluster --name high-density --pod-pids-limit 10000

  # Interactive mode
  rosa create kubeletconfig --cluster my-cluster --interactive`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateKubeletConfig(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster")
	flags.StringVar(&opts.Name, "name", "", "Name of the KubeletConfig (required for HCP)")
	flags.IntVar(&opts.PodPidsLimit, "pod-pids-limit", 4096, "Maximum number of PIDs per pod")
	flags.BoolVarP(&opts.Interactive, "interactive", "i", false, "Interactive mode")
	flags.BoolVar(&opts.DryRun, "dry-run", false, "Show what would be created without creating it")

	return cmd
}

func runCreateKubeletConfig(ctx context.Context, logger *slog.Logger, opts *CreateOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if cluster is specified
	if opts.ClusterName == "" {
		if !opts.Interactive {
			return fmt.Errorf("cluster name is required")
		}
		// In interactive mode, prompt for cluster
		err = huh.NewInput().
			Title("Cluster name").
			Description("Enter the name or ID of the cluster").
			Value(&opts.ClusterName).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("cluster name cannot be empty")
				}
				return nil
			}).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get cluster name: %w", err)
		}
	}

	// Interactive mode for other options
	if opts.Interactive {
		if opts.Name == "" {
			err = huh.NewInput().
				Title("KubeletConfig name").
				Description("Enter a name for this KubeletConfig").
				Value(&opts.Name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name cannot be empty for HCP clusters")
					}
					return nil
				}).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get name: %w", err)
			}
		}

		// For integer input, we need to handle it differently with huh
		var pidLimitStr string
		pidLimitStr = fmt.Sprintf("%d", opts.PodPidsLimit)
		err = huh.NewInput().
			Title("Pod PIDs limit").
			Description("Maximum number of PIDs per pod (minimum 4096)").
			Value(&pidLimitStr).
			Validate(func(s string) error {
				var val int
				if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
					return fmt.Errorf("must be a valid number")
				}
				if val < 4096 {
					return fmt.Errorf("pod-pids-limit must be at least 4096")
				}
				return nil
			}).
			Run()
		if err == nil {
			fmt.Sscanf(pidLimitStr, "%d", &opts.PodPidsLimit)
		}
		if err != nil {
			return fmt.Errorf("failed to get pod-pids-limit: %w", err)
		}
	}

	// Validate options
	if opts.Name == "" {
		return fmt.Errorf("name is required for KubeletConfig in HCP clusters")
	}

	if opts.PodPidsLimit < 4096 {
		return fmt.Errorf("pod-pids-limit must be at least 4096, got %d", opts.PodPidsLimit)
	}

	// Create API client
	apiClient, err := api.NewClient(ctx, api.Config{
		URL:   cfg.APIURL,
		Token: cfg.Token,
	})
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Get cluster to verify it exists and is HCP
	clusterResp, err := apiClient.GetCluster(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	cluster := clusterResp.Body()

	// Check if cluster is HCP
	if !cluster.Hypershift().Enabled() {
		return fmt.Errorf("KubeletConfig with named configurations is only supported for HCP clusters")
	}

	writer.Title("Creating KubeletConfig")
	writer.KeyValue(map[string]string{
		"Cluster":        cluster.Name(),
		"Name":           opts.Name,
		"Pod PIDs Limit": fmt.Sprintf("%d", opts.PodPidsLimit),
	})

	if opts.DryRun {
		writer.Warning("Dry run mode - no changes will be made")
		writer.Success("Would create KubeletConfig '%s'", opts.Name)
		return nil
	}

	// Create KubeletConfig service
	kcSvc, err := kubeletconfig.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create kubeletconfig service: %w", err)
	}

	// Create the KubeletConfig
	kc, err := kcSvc.Create(ctx, cluster.ID(), kubeletconfig.Config{
		Name:         opts.Name,
		PodPidsLimit: opts.PodPidsLimit,
	})
	if err != nil {
		return fmt.Errorf("failed to create kubeletconfig: %w", err)
	}

	writer.Success("KubeletConfig '%s' created successfully", kc.Name)
	writer.Info("To use this config with a node pool:")
	fmt.Printf("  rosa nodepool create --cluster %s --kubelet-configs %s ...\n", opts.ClusterName, kc.Name)

	return nil
}
