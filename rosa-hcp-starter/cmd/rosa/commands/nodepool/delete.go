package nodepool

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/pkg/nodepool"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// NewDeleteCommand creates the nodepool delete command
func NewDeleteCommand(svc *nodepool.Service) *cobra.Command {
	var clusterID string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete NODEPOOL",
		Short: "Delete a node pool",
		Long:  "Delete a node pool from a ROSA HCP cluster.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodePoolID := args[0]

			if clusterID == "" {
				return fmt.Errorf("cluster ID is required")
			}

			// Confirm deletion
			if !yes {
				fmt.Printf("Are you sure you want to delete node pool '%s'? (y/N): ", nodePoolID)
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Deletion cancelled")
					return nil
				}
			}

			progress := output.NewProgress(fmt.Sprintf("Deleting node pool '%s'", nodePoolID))
			progress.Start()
			defer progress.Stop()

			err := svc.Delete(cmd.Context(), clusterID, nodePoolID)
			if err != nil {
				progress.Stop()
				return fmt.Errorf("failed to delete node pool: %w", err)
			}

			progress.Success(fmt.Sprintf("Node pool '%s' deletion initiated", nodePoolID))
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&clusterID, "cluster", "", "Cluster ID or name (required)")
	flags.BoolVarP(&yes, "yes", "y", false, "Skip confirmation")

	cmd.MarkFlagRequired("cluster")

	return cmd
}
