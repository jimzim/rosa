package cluster

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/pkg/cluster"
)

// NewCommand creates the cluster command with all subcommands
func NewCommand(svc *cluster.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Create and manage ROSA clusters",
		Long:  "Create and manage Red Hat OpenShift Service on AWS (ROSA) clusters with Hosted Control Planes.",
	}

	// Add subcommands
	cmd.AddCommand(
		NewCreateCommand(svc),
		NewDeleteCommand(svc),
		NewDescribeCommand(svc),
		NewListCommand(svc),
		NewEditCommand(svc),
	)

	return cmd
}

// Placeholder commands - to be implemented
func NewDeleteCommand(svc *cluster.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "Delete a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("Delete cluster - to be implemented")
			return nil
		},
	}
}

func NewDescribeCommand(svc *cluster.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "describe",
		Short: "Describe a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("Describe cluster - to be implemented")
			return nil
		},
	}
}

func NewListCommand(svc *cluster.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List clusters",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("List clusters - to be implemented")
			return nil
		},
	}
}

func NewEditCommand(svc *cluster.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Edit a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("Edit cluster - to be implemented")
			return nil
		},
	}
}
