package nodepool

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/pkg/errors"
	"github.com/openshift/rosa-hcp/pkg/nodepool"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// CreateOptions contains options for creating a node pool
type CreateOptions struct {
	ClusterID      string
	Name           string
	Replicas       int
	MinReplicas    int
	MaxReplicas    int
	InstanceType   string
	DiskSize       int
	Labels         map[string]string
	Taints         []string
	Version        string
	Subnet         string
	AutoRepair     bool
	Autoscaling    bool
	KubeletConfigs []string
	TuningConfigs  []string
}

// NewCreateCommand creates the nodepool create command
func NewCreateCommand(svc *nodepool.Service) *cobra.Command {
	opts := &CreateOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a node pool",
		Long: `Create a new node pool for a ROSA HCP cluster.

Node pools in HCP clusters can have different versions and configurations,
allowing for greater flexibility in managing your compute resources.`,
		Example: `  # Create a node pool with specific instance type
  rosa nodepool create \
    --cluster my-cluster \
    --name gpu-nodes \
    --replicas 3 \
    --instance-type g4dn.xlarge
    
  # Create an autoscaling node pool
  rosa nodepool create \
    --cluster my-cluster \
    --name autoscale-pool \
    --min-replicas 2 \
    --max-replicas 10 \
    --instance-type m5.xlarge`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd.Context(), svc, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.ClusterID, "cluster", "", "Cluster ID or name (required)")
	flags.StringVar(&opts.Name, "name", "", "Name of the node pool (required)")
	flags.IntVar(&opts.Replicas, "replicas", 2, "Number of nodes")
	flags.IntVar(&opts.MinReplicas, "min-replicas", 0, "Minimum nodes for autoscaling")
	flags.IntVar(&opts.MaxReplicas, "max-replicas", 0, "Maximum nodes for autoscaling")
	flags.StringVar(&opts.InstanceType, "instance-type", "m5.xlarge", "EC2 instance type")
	flags.IntVar(&opts.DiskSize, "disk-size", 120, "Root disk size in GiB")
	flags.StringToStringVar(&opts.Labels, "labels", nil, "Labels for nodes (key=value)")
	flags.StringSliceVar(&opts.Taints, "taints", nil, "Taints for nodes (key=value:effect)")
	flags.StringVar(&opts.Version, "version", "", "OpenShift version (defaults to cluster version)")
	flags.StringVar(&opts.Subnet, "subnet", "", "AWS subnet ID")
	flags.BoolVar(&opts.AutoRepair, "auto-repair", true, "Enable auto-repair")
	flags.StringSliceVar(&opts.KubeletConfigs, "kubelet-configs", nil, "KubeletConfigs to apply")
	flags.StringSliceVar(&opts.TuningConfigs, "tuning-configs", nil, "TuningConfigs to apply")

	cmd.MarkFlagRequired("cluster")
	cmd.MarkFlagRequired("name")

	return cmd
}

func runCreate(ctx context.Context, svc *nodepool.Service, opts *CreateOptions) error {
	// Validate options
	if err := validateCreateOptions(opts); err != nil {
		return err
	}

	// Determine if autoscaling is enabled
	opts.Autoscaling = opts.MinReplicas > 0 && opts.MaxReplicas > 0

	// Show what will be created
	output.Info("Creating node pool:")
	writer := output.NewWriter(output.FormatText)

	poolInfo := map[string]string{
		"Cluster":       opts.ClusterID,
		"Name":          opts.Name,
		"Instance Type": opts.InstanceType,
		"Disk Size":     fmt.Sprintf("%d GiB", opts.DiskSize),
		"Auto Repair":   fmt.Sprintf("%v", opts.AutoRepair),
	}

	if opts.Autoscaling {
		poolInfo["Autoscaling"] = fmt.Sprintf("%d-%d nodes", opts.MinReplicas, opts.MaxReplicas)
	} else {
		poolInfo["Replicas"] = fmt.Sprintf("%d", opts.Replicas)
	}

	if opts.Version != "" {
		poolInfo["Version"] = opts.Version
	}

	if opts.Subnet != "" {
		poolInfo["Subnet"] = opts.Subnet
	}

	if len(opts.Labels) > 0 {
		var labels []string
		for k, v := range opts.Labels {
			labels = append(labels, fmt.Sprintf("%s=%s", k, v))
		}
		poolInfo["Labels"] = strings.Join(labels, ", ")
	}

	if len(opts.Taints) > 0 {
		poolInfo["Taints"] = strings.Join(opts.Taints, ", ")
	}

	if len(opts.TuningConfigs) > 0 {
		poolInfo["Tuning Configs"] = strings.Join(opts.TuningConfigs, ", ")
	}

	writer.KeyValue(poolInfo)

	// Create progress indicator
	progress := output.NewProgress(fmt.Sprintf("Creating node pool '%s'", opts.Name))
	progress.Start()
	defer progress.Stop()

	// Create the node pool
	config := nodepool.CreateConfig{
		ClusterID:      opts.ClusterID,
		Name:           opts.Name,
		Replicas:       opts.Replicas,
		MinReplicas:    opts.MinReplicas,
		MaxReplicas:    opts.MaxReplicas,
		InstanceType:   opts.InstanceType,
		DiskSize:       opts.DiskSize,
		Labels:         opts.Labels,
		Taints:         parseTaints(opts.Taints),
		Version:        opts.Version,
		Subnet:         opts.Subnet,
		AutoRepair:     opts.AutoRepair,
		Autoscaling:    opts.Autoscaling,
		KubeletConfigs: opts.KubeletConfigs,
		TuningConfigs:  opts.TuningConfigs,
	}

	np, err := svc.Create(ctx, config)
	if err != nil {
		progress.Stop()
		return fmt.Errorf("failed to create node pool: %w", err)
	}

	progress.Success(fmt.Sprintf("Node pool '%s' created successfully", np.Name))

	// Display node pool details
	output.Info("\nNode Pool Details:")
	npInfo := map[string]string{
		"ID":            np.ID,
		"Name":          np.Name,
		"State":         np.State,
		"Instance Type": np.InstanceType,
	}

	if np.Autoscaling {
		npInfo["Autoscaling"] = fmt.Sprintf("%d-%d nodes", np.MinReplicas, np.MaxReplicas)
	} else {
		npInfo["Replicas"] = fmt.Sprintf("%d", np.Replicas)
	}

	writer.KeyValue(npInfo)

	// Show KubeletConfigs if present
	if len(np.KubeletConfigs) > 0 {
		output.Info("\nKubeletConfigs:")
		for _, kc := range np.KubeletConfigs {
			fmt.Printf("  - %s\n", kc)
		}
	}

	// Show TuningConfigs if present
	if len(np.TuningConfigs) > 0 {
		output.Info("\nTuningConfigs:")
		for _, tc := range np.TuningConfigs {
			fmt.Printf("  - %s\n", tc)
		}
	}

	output.Info("\nThe node pool is being created. This may take several minutes.")
	output.Info("To check the status, run:")
	fmt.Printf("  rosa nodepool describe %s --cluster %s\n", np.Name, opts.ClusterID)

	return nil
}

func validateCreateOptions(opts *CreateOptions) error {
	var errs []string

	if opts.ClusterID == "" {
		errs = append(errs, "cluster ID is required")
	}

	if opts.Name == "" {
		errs = append(errs, "node pool name is required")
	}

	// Validate autoscaling
	if opts.MinReplicas > 0 || opts.MaxReplicas > 0 {
		if opts.MinReplicas <= 0 {
			errs = append(errs, "min-replicas must be greater than 0 for autoscaling")
		}
		if opts.MaxReplicas <= 0 {
			errs = append(errs, "max-replicas must be greater than 0 for autoscaling")
		}
		if opts.MinReplicas > opts.MaxReplicas {
			errs = append(errs, "min-replicas cannot be greater than max-replicas")
		}
	} else if opts.Replicas < 1 {
		errs = append(errs, "replicas must be at least 1")
	}

	if opts.DiskSize < 30 {
		errs = append(errs, "disk size must be at least 30 GiB")
	}

	if len(errs) > 0 {
		return errors.Validation("nodepool.create", fmt.Errorf(strings.Join(errs, "; ")))
	}

	return nil
}

// parseTaints parses taint strings into structured format
func parseTaints(taints []string) []nodepool.Taint {
	var result []nodepool.Taint
	for _, taint := range taints {
		// Parse format: key=value:effect
		parts := strings.Split(taint, ":")
		if len(parts) != 2 {
			continue
		}

		kvParts := strings.Split(parts[0], "=")
		if len(kvParts) != 2 {
			continue
		}

		result = append(result, nodepool.Taint{
			Key:    kvParts[0],
			Value:  kvParts[1],
			Effect: parts[1],
		})
	}
	return result
}
