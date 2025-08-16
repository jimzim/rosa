package nodepool

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/pkg/nodepool"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// EditOptions contains options for editing a node pool
type EditOptions struct {
	ClusterID   string
	Replicas    int
	MinReplicas int
	MaxReplicas int
	AutoRepair  *bool
}

// NewEditCommand creates the nodepool edit command
func NewEditCommand(svc *nodepool.Service) *cobra.Command {
	opts := &EditOptions{}
	var enableAutoRepair, disableAutoRepair bool

	cmd := &cobra.Command{
		Use:   "edit NODEPOOL",
		Short: "Edit a node pool",
		Long:  "Edit configuration of an existing node pool.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodePoolID := args[0]

			if opts.ClusterID == "" {
				return fmt.Errorf("cluster ID is required")
			}

			// Handle auto-repair flags
			if enableAutoRepair && disableAutoRepair {
				return fmt.Errorf("cannot enable and disable auto-repair at the same time")
			}
			if enableAutoRepair {
				enabled := true
				opts.AutoRepair = &enabled
			}
			if disableAutoRepair {
				disabled := false
				opts.AutoRepair = &disabled
			}

			// Check if any changes were specified
			if !cmd.Flags().Changed("replicas") &&
				!cmd.Flags().Changed("min-replicas") &&
				!cmd.Flags().Changed("max-replicas") &&
				opts.AutoRepair == nil {
				return fmt.Errorf("no changes specified")
			}

			progress := output.NewProgress(fmt.Sprintf("Updating node pool '%s'", nodePoolID))
			progress.Start()
			defer progress.Stop()

			updateConfig := nodepool.UpdateConfig{
				Replicas:    opts.Replicas,
				MinReplicas: opts.MinReplicas,
				MaxReplicas: opts.MaxReplicas,
				AutoRepair:  opts.AutoRepair,
			}

			np, err := svc.Update(cmd.Context(), opts.ClusterID, nodePoolID, updateConfig)
			if err != nil {
				progress.Stop()
				return fmt.Errorf("failed to update node pool: %w", err)
			}

			progress.Success(fmt.Sprintf("Node pool '%s' updated successfully", np.Name))

			// Display updated configuration
			output.Info("\nUpdated Configuration:")
			writer := output.NewWriter(output.FormatText)

			info := map[string]string{
				"Name": np.Name,
			}

			if np.Autoscaling {
				info["Autoscaling"] = fmt.Sprintf("%d-%d nodes", np.MinReplicas, np.MaxReplicas)
			} else {
				info["Replicas"] = fmt.Sprintf("%d", np.Replicas)
			}

			if opts.AutoRepair != nil {
				info["Auto Repair"] = fmt.Sprintf("%v", *opts.AutoRepair)
			}

			writer.KeyValue(info)

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.ClusterID, "cluster", "", "Cluster ID or name (required)")
	flags.IntVar(&opts.Replicas, "replicas", 0, "Number of nodes (disables autoscaling)")
	flags.IntVar(&opts.MinReplicas, "min-replicas", 0, "Minimum nodes for autoscaling")
	flags.IntVar(&opts.MaxReplicas, "max-replicas", 0, "Maximum nodes for autoscaling")
	flags.BoolVar(&enableAutoRepair, "enable-auto-repair", false, "Enable auto-repair")
	flags.BoolVar(&disableAutoRepair, "disable-auto-repair", false, "Disable auto-repair")

	cmd.MarkFlagRequired("cluster")

	return cmd
}
