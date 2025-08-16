package nodepool

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/pkg/nodepool"
)

// NewListCommand creates the nodepool list command
func NewListCommand(svc *nodepool.Service) *cobra.Command {
	var clusterID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List node pools",
		Long:  "List all node pools for a ROSA HCP cluster.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return fmt.Errorf("cluster ID is required")
			}

			nodePools, err := svc.List(cmd.Context(), clusterID)
			if err != nil {
				return fmt.Errorf("failed to list node pools: %w", err)
			}

			if len(nodePools) == 0 {
				fmt.Printf("No node pools found for cluster '%s'\n", clusterID)
				return nil
			}

			// Print table header
			fmt.Printf("%-20s %-15s %-10s %-15s %-15s %-10s\n",
				"NAME", "ID", "STATE", "REPLICAS", "INSTANCE TYPE", "VERSION")
			fmt.Println(strings.Repeat("-", 90))

			// Print node pools
			for _, np := range nodePools {
				replicas := fmt.Sprintf("%d", np.Replicas)
				if np.Autoscaling {
					replicas = fmt.Sprintf("%d-%d", np.MinReplicas, np.MaxReplicas)
				}

				fmt.Printf("%-20s %-15s %-10s %-15s %-15s %-10s\n",
					np.Name,
					np.ID,
					np.State,
					replicas,
					np.InstanceType,
					np.Version,
				)
			}

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&clusterID, "cluster", "", "Cluster ID or name (required)")

	cmd.MarkFlagRequired("cluster")

	return cmd
}
