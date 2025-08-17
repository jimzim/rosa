package tuningconfig

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
	"github.com/openshift/rosa-hcp/pkg/tuningconfig"
)

// CreateOptions contains options for creating a TuningConfig
type CreateOptions struct {
	ClusterName    string
	Name           string
	SpecPath       string
	CreateExample  bool
	Interactive    bool
	DryRun         bool
}

// NewCreateCommand creates the tuning-config create command
func NewCreateCommand(logger *slog.Logger) *cobra.Command {
	opts := &CreateOptions{}

	cmd := &cobra.Command{
		Use:   "tuning-config",
		Short: "Create a TuningConfig for a cluster",
		Long: `Create a TuningConfig to customize kernel and OS settings for node pools.

This command creates a named TuningConfig that can be attached to node pools
during creation. TuningConfigs use TuneD profiles to apply kernel parameters
and other OS-level optimizations.

Note: TuningConfigs are only supported on HCP (Hosted Control Plane) clusters.`,
		Example: `  # Create a tuning config from a spec file
  rosa create tuning-config --cluster my-cluster --name database-tuning --spec-path tuning.yaml

  # Create an example spec file
  rosa create tuning-config --create-example tuning-example.yaml

  # Interactive mode
  rosa create tuning-config --cluster my-cluster --interactive`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateTuningConfig(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster")
	flags.StringVar(&opts.Name, "name", "", "Name of the TuningConfig")
	flags.StringVar(&opts.SpecPath, "spec-path", "", "Path to the TuneD spec file (JSON or YAML)")
	flags.BoolVar(&opts.CreateExample, "create-example", false, "Create an example spec file and exit")
	flags.BoolVarP(&opts.Interactive, "interactive", "i", false, "Interactive mode")
	flags.BoolVar(&opts.DryRun, "dry-run", false, "Show what would be created without creating it")

	return cmd
}

func runCreateTuningConfig(ctx context.Context, logger *slog.Logger, opts *CreateOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Handle --create-example flag
	if opts.CreateExample {
		examplePath := "tuning-example.yaml"
		if opts.SpecPath != "" {
			examplePath = opts.SpecPath
		}

		err := createExampleSpecFile(examplePath)
		if err != nil {
			return fmt.Errorf("failed to create example spec file: %w", err)
		}

		writer.Success("Created example tuning config spec file: %s", examplePath)
		writer.Info("Edit this file and then run:")
		fmt.Printf("  rosa create tuning-config --cluster <cluster> --name <name> --spec-path %s\n", examplePath)
		return nil
	}

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
				Title("TuningConfig name").
				Description("Enter a name for this TuningConfig").
				Value(&opts.Name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name cannot be empty")
					}
					return nil
				}).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get name: %w", err)
			}
		}

		if opts.SpecPath == "" {
			err = huh.NewInput().
				Title("Spec file path").
				Description("Path to the TuneD spec file (JSON or YAML)").
				Value(&opts.SpecPath).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("spec path cannot be empty")
					}
					if _, err := os.Stat(s); err != nil {
						return fmt.Errorf("spec file does not exist: %s", s)
					}
					return nil
				}).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get spec path: %w", err)
			}
		}
	}

	// Validate options
	if opts.Name == "" {
		return fmt.Errorf("name is required")
	}

	if opts.SpecPath == "" {
		return fmt.Errorf("spec-path is required")
	}

	// Check if spec file exists
	if _, err := os.Stat(opts.SpecPath); err != nil {
		return fmt.Errorf("spec file does not exist: %s", opts.SpecPath)
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

	// Check if cluster is HCP (TuningConfigs are HCP-only)
	if !cluster.Hypershift().Enabled() {
		return fmt.Errorf("TuningConfigs are only supported for HCP clusters")
	}

	writer.Title("Creating TuningConfig")
	writer.KeyValue(map[string]string{
		"Cluster":   cluster.Name(),
		"Name":      opts.Name,
		"Spec File": opts.SpecPath,
	})

	if opts.DryRun {
		writer.Warning("Dry run mode - no changes will be made")
		writer.Success("Would create TuningConfig '%s'", opts.Name)
		return nil
	}

	// Create TuningConfig service
	tcSvc, err := tuningconfig.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create tuningconfig service: %w", err)
	}

	// Create the TuningConfig
	tc, err := tcSvc.Create(ctx, cluster.ID(), tuningconfig.Config{
		Name:     opts.Name,
		SpecPath: opts.SpecPath,
	})
	if err != nil {
		return fmt.Errorf("failed to create tuningconfig: %w", err)
	}

	writer.Success("TuningConfig '%s' created successfully", tc.Name)
	writer.Info("To use this config with a node pool:")
	fmt.Printf("  rosa nodepool create --cluster %s --tuning-configs %s ...\n", opts.ClusterName, tc.Name)

	return nil
}

// createExampleSpecFile creates an example tuning config spec file
func createExampleSpecFile(path string) error {
	exampleSpec := `profile:
- name: custom-profile
  data: |
    [main]
    summary=Custom OpenShift profile for database workloads
    include=openshift-node
    
    [sysctl]
    # Optimize for database workloads
    vm.dirty_ratio="20"
    vm.dirty_background_ratio="5"
    vm.swappiness="10"
    
    # Network optimizations
    net.core.somaxconn="4096"
    net.ipv4.tcp_max_syn_backlog="8192"
    
    # Kernel scheduler optimizations
    kernel.sched_migration_cost_ns="5000000"
    kernel.numa_balancing="1"

recommend:
- priority: 20
  profile: custom-profile
  # Optional: Add match conditions to apply only to specific nodes
  # match:
  # - label: node-role.kubernetes.io/worker
  #   type: pod
`

	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Write the file
	if err := os.WriteFile(path, []byte(exampleSpec), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
