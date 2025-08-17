package ingress

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/ingress"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// DeleteOptions contains options for deleting an ingress
type DeleteOptions struct {
	ClusterName string
	IngressID   string
	Yes         bool
}

// NewDeleteCommand creates the ingress delete command
func NewDeleteCommand(logger *slog.Logger) *cobra.Command {
	opts := &DeleteOptions{}

	cmd := &cobra.Command{
		Use:   "ingress INGRESS_ID",
		Short: "Delete an ingress from a cluster",
		Long: `Delete a custom ingress (load balancer) from a cluster.

Note: The default ingress cannot be deleted.`,
		Example: `  # Delete an ingress
  rosa delete ingress apps2 --cluster my-cluster

  # Delete without confirmation prompt
  rosa delete ingress apps2 --cluster my-cluster --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IngressID = args[0]
			return runDeleteIngress(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runDeleteIngress(ctx context.Context, logger *slog.Logger, opts *DeleteOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Validate ingress ID
	if !ingress.IngressKeyRE.MatchString(opts.IngressID) {
		return fmt.Errorf("ingress ID '%s' must be 3-5 lowercase letters or digits", opts.IngressID)
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

	// Get cluster
	clusterResp, err := apiClient.GetCluster(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	cluster := clusterResp.Body()

	// Create Ingress service
	ingressSvc, err := ingress.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create ingress service: %w", err)
	}

	// Get the ingress to show details and check if it's default
	existingIngress, err := ingressSvc.Get(ctx, cluster.ID(), opts.IngressID)
	if err != nil {
		return fmt.Errorf("failed to get ingress: %w", err)
	}

	// Check if this is the default ingress
	if existingIngress.Default {
		return fmt.Errorf("cannot delete the default ingress")
	}

	writer.Title("Delete Ingress")
	writer.KeyValue(map[string]string{
		"Cluster":        cluster.Name(),
		"Ingress ID":     existingIngress.ID,
		"Listening":      string(existingIngress.Listening),
		"Load Balancer":  string(existingIngress.LoadBalancerType),
	})

	if len(existingIngress.RouteSelectors) > 0 {
		writer.KeyValue(map[string]string{
			"Route Selectors": ingress.FormatRouteSelectors(existingIngress.RouteSelectors),
		})
	}

	// Confirm deletion
	if !opts.Yes {
		var confirm bool
		err = huh.NewConfirm().
			Title("Are you sure you want to delete this ingress?").
			Description("This action cannot be undone.").
			Value(&confirm).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirm {
			writer.Info("Deletion cancelled")
			return nil
		}
	}

	// Delete the ingress
	writer.Info("Deleting ingress...")
	err = ingressSvc.Delete(ctx, cluster.ID(), opts.IngressID)
	if err != nil {
		return fmt.Errorf("failed to delete ingress: %w", err)
	}

	writer.Success("Ingress '%s' deleted successfully", opts.IngressID)
	writer.Info("\nNote: It may take a few minutes for the load balancer to be fully removed from AWS.")

	return nil
}
