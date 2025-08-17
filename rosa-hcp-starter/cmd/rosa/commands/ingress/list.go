package ingress

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/ingress"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// ListOptions contains options for listing ingresses
type ListOptions struct {
	ClusterName  string
	OutputFormat string
}

// NewListCommand creates the ingress list command
func NewListCommand(logger *slog.Logger) *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:     "ingresses",
		Aliases: []string{"ingress"},
		Short:   "List ingresses for a cluster",
		Long: `List all ingresses (load balancers) configured for a cluster.

Each cluster has a default ingress and may have additional custom ingresses.`,
		Example: `  # List all ingresses for a cluster
  rosa list ingresses --cluster my-cluster

  # Output in JSON format
  rosa list ingresses --cluster my-cluster --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListIngresses(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format (text, json, yaml)")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runListIngresses(ctx context.Context, logger *slog.Logger, opts *ListOptions) error {
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

	// Get cluster to verify it exists
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

	// List ingresses
	ingresses, err := ingressSvc.List(ctx, cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to list ingresses: %w", err)
	}

	// Display results based on output format
	var writer *output.Writer

	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		writer = output.NewWriter(output.FormatJSON)
		return writer.Print(ingresses)
	case "yaml":
		writer = output.NewWriter(output.FormatYAML)
		return writer.Print(ingresses)
	default:
		writer = output.NewWriter(output.FormatText)
		if len(ingresses) == 0 {
			writer.Info("No ingresses found for cluster '%s'", opts.ClusterName)
			return nil
		}

		writer.Title(fmt.Sprintf("Ingresses for cluster '%s'", cluster.Name()))
		
		// Display as table
		fmt.Printf("%-10s %-8s %-10s %-10s %-8s %s\n", 
			"ID", "DEFAULT", "LISTENING", "LB TYPE", "PRIVATE", "ROUTE SELECTORS")
		fmt.Println(strings.Repeat("-", 80))
		
		for _, ing := range ingresses {
			isPrivate := ing.Listening == ingress.ListeningInternal
			routeSelectors := ingress.FormatRouteSelectors(ing.RouteSelectors)
			if routeSelectors == "" {
				routeSelectors = "-"
			}
			
			fmt.Printf("%-10s %-8t %-10s %-10s %-8t %s\n", 
				ing.ID, 
				ing.Default,
				ing.Listening,
				ing.LoadBalancerType,
				isPrivate,
				routeSelectors)
			
			// Show DNS name on a separate line if available
			if ing.DNSName != "" {
				fmt.Printf("  DNS: %s\n", ing.DNSName)
			}
		}
		fmt.Println()

		// Show helpful commands
		if len(ingresses) > 1 {
			writer.Info("To get more details about a specific ingress:")
			fmt.Printf("  rosa describe ingress <ingress-id> --cluster %s\n", opts.ClusterName)
			
			// Find non-default ingresses
			for _, ing := range ingresses {
				if !ing.Default {
					writer.Info("\nTo edit or delete a custom ingress:")
					fmt.Printf("  rosa edit ingress %s --cluster %s\n", ing.ID, opts.ClusterName)
					fmt.Printf("  rosa delete ingress %s --cluster %s\n", ing.ID, opts.ClusterName)
					break
				}
			}
		}
	}

	return nil
}
