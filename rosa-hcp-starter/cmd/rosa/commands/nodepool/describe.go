package nodepool

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/pkg/nodepool"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// NewDescribeCommand creates the nodepool describe command
func NewDescribeCommand(svc *nodepool.Service) *cobra.Command {
	var clusterID string

	cmd := &cobra.Command{
		Use:   "describe NODEPOOL",
		Short: "Describe a node pool",
		Long:  "Display detailed information about a node pool.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodePoolID := args[0]

			if clusterID == "" {
				return fmt.Errorf("cluster ID is required")
			}

			np, err := svc.Get(cmd.Context(), clusterID, nodePoolID)
			if err != nil {
				return fmt.Errorf("failed to get node pool: %w", err)
			}

			// Display node pool details
			output.Info("Node Pool Details:")
			writer := output.NewWriter(output.FormatText)

			info := map[string]string{
				"ID":            np.ID,
				"Name":          np.Name,
				"State":         np.State,
				"Instance Type": np.InstanceType,
				"Disk Size":     fmt.Sprintf("%d GiB", np.DiskSize),
				"Version":       np.Version,
				"Auto Repair":   fmt.Sprintf("%v", np.AutoRepair),
			}

			if np.Autoscaling {
				info["Autoscaling"] = fmt.Sprintf("Enabled (%d-%d nodes)", np.MinReplicas, np.MaxReplicas)
				info["Current Replicas"] = fmt.Sprintf("%d", np.Replicas)
			} else {
				info["Replicas"] = fmt.Sprintf("%d", np.Replicas)
			}

			if np.Subnet != "" {
				info["Subnet"] = np.Subnet
			}

			writer.KeyValue(info)

			// Display labels
			if len(np.Labels) > 0 {
				output.Info("\nLabels:")
				for k, v := range np.Labels {
					fmt.Printf("  %s: %s\n", k, v)
				}
			}

			// Display taints
			if len(np.Taints) > 0 {
				output.Info("\nTaints:")
				for _, taint := range np.Taints {
					fmt.Printf("  %s=%s:%s\n", taint.Key, taint.Value, taint.Effect)
				}
			}

			// Display tuning configs
			if len(np.TuningConfigs) > 0 {
				output.Info("\nTuning Configs:")
				fmt.Printf("  %s\n", strings.Join(np.TuningConfigs, ", "))
			}

			// Display status message if any
			if np.Message != "" {
				output.Info("\nStatus Message:")
				fmt.Printf("  %s\n", np.Message)
			}

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&clusterID, "cluster", "", "Cluster ID or name (required)")

	cmd.MarkFlagRequired("cluster")

	return cmd
}
