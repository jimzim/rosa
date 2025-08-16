package nodepool

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/pkg/nodepool"
)

// NewCommand creates the nodepool command with all subcommands
func NewCommand(svc *nodepool.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "nodepool",
		Aliases: []string{"nodepools", "node-pool", "node-pools", "np"},
		Short:   "Create and manage node pools",
		Long:    "Create and manage node pools for ROSA HCP clusters.",
	}

	// Add subcommands
	cmd.AddCommand(
		NewCreateCommand(svc),
		NewDeleteCommand(svc),
		NewListCommand(svc),
		NewDescribeCommand(svc),
		NewEditCommand(svc),
	)

	return cmd
}
