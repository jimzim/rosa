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

// CreateOptions contains options for creating an ingress
type CreateOptions struct {
	ClusterName    string
	ID             string
	Private        bool
	LBType         string
	RouteSelectors string
	Interactive    bool
}

// NewCreateCommand creates the ingress create command
func NewCreateCommand(logger *slog.Logger) *cobra.Command {
	opts := &CreateOptions{}

	cmd := &cobra.Command{
		Use:   "ingress",
		Short: "Create an additional ingress for a cluster",
		Long: `Create an additional ingress (load balancer) for a cluster.

Each cluster has a default ingress that cannot be deleted. You can create
additional ingresses to handle different traffic patterns or routing requirements.`,
		Example: `  # Create a private ingress
  rosa create ingress --cluster my-cluster --private

  # Create an ingress with NLB load balancer
  rosa create ingress --cluster my-cluster --lb-type nlb

  # Create an ingress with route selectors
  rosa create ingress --cluster my-cluster --route-selectors app=frontend,tier=web`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateIngress(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVar(&opts.ID, "id", "", "ID for the ingress (3-5 lowercase letters/digits, optional)")
	flags.BoolVar(&opts.Private, "private", false, "Create a private (internal) ingress")
	flags.StringVar(&opts.LBType, "lb-type", "classic", "Load balancer type (classic or nlb)")
	flags.StringVar(&opts.RouteSelectors, "route-selectors", "", "Route selectors (comma-separated key=value pairs)")
	flags.BoolVarP(&opts.Interactive, "interactive", "i", false, "Interactive mode")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runCreateIngress(ctx context.Context, logger *slog.Logger, opts *CreateOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Interactive mode
	if opts.Interactive {
		// Cluster name
		if opts.ClusterName == "" {
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

		// Ingress ID (optional)
		err = huh.NewInput().
			Title("Ingress ID (optional)").
			Description("3-5 lowercase letters or digits (leave empty for auto-generated)").
			Value(&opts.ID).
			Validate(func(s string) error {
				if s != "" && !ingress.IngressKeyRE.MatchString(s) {
					return fmt.Errorf("must be 3-5 lowercase letters or digits")
				}
				return nil
			}).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get ingress ID: %w", err)
		}

		// Private/Public
		err = huh.NewConfirm().
			Title("Private ingress?").
			Description("Should this ingress be private (internal)?").
			Value(&opts.Private).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get private setting: %w", err)
		}

		// Load balancer type
		var lbTypeIdx int
		lbTypes := []string{"classic", "nlb"}
		err = huh.NewSelect[int]().
			Title("Load balancer type").
			Options(
				huh.NewOption("Classic", 0),
				huh.NewOption("Network Load Balancer (NLB)", 1),
			).
			Value(&lbTypeIdx).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get load balancer type: %w", err)
		}
		opts.LBType = lbTypes[lbTypeIdx]

		// Route selectors (optional)
		err = huh.NewInput().
			Title("Route selectors (optional)").
			Description("Comma-separated key=value pairs (e.g., app=frontend,tier=web)").
			Value(&opts.RouteSelectors).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get route selectors: %w", err)
		}
	}

	// Validate options
	if opts.ID != "" && !ingress.IngressKeyRE.MatchString(opts.ID) {
		return fmt.Errorf("ingress ID must be 3-5 lowercase letters or digits")
	}

	if opts.LBType != "classic" && opts.LBType != "nlb" {
		return fmt.Errorf("load balancer type must be 'classic' or 'nlb'")
	}

	// Parse route selectors
	var routeSelectors map[string]string
	if opts.RouteSelectors != "" {
		routeSelectors, err = ingress.ParseRouteSelectors(opts.RouteSelectors)
		if err != nil {
			return fmt.Errorf("failed to parse route selectors: %w", err)
		}
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

	// Check if cluster is HCP
	if !cluster.Hypershift().Enabled() {
		// Ingress works for both Classic and HCP, but we're HCP-focused
		writer.Warning("Note: This CLI is optimized for HCP clusters")
	}

	// Create Ingress service
	ingressSvc, err := ingress.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create ingress service: %w", err)
	}

	// Prepare create config
	createConfig := ingress.CreateConfig{
		ID:               opts.ID,
		Private:          opts.Private,
		LoadBalancerType: ingress.LoadBalancerType(opts.LBType),
		RouteSelectors:   routeSelectors,
	}

	writer.Title("Creating Ingress")
	writer.KeyValue(map[string]string{
		"Cluster":           cluster.Name(),
		"Private":           fmt.Sprintf("%t", opts.Private),
		"Load Balancer":     opts.LBType,
	})
	
	if opts.ID != "" {
		writer.KeyValue(map[string]string{"ID": opts.ID})
	}
	
	if len(routeSelectors) > 0 {
		writer.KeyValue(map[string]string{"Route Selectors": ingress.FormatRouteSelectors(routeSelectors)})
	}

	// Create the ingress
	createdIngress, err := ingressSvc.Create(ctx, cluster.ID(), createConfig)
	if err != nil {
		return fmt.Errorf("failed to create ingress: %w", err)
	}

	writer.Success("Ingress '%s' created successfully", createdIngress.ID)
	
	// Display ingress details
	writer.Info("\nIngress Details:")
	writer.KeyValue(map[string]string{
		"ID":               createdIngress.ID,
		"Default":          fmt.Sprintf("%t", createdIngress.Default),
		"Listening":        string(createdIngress.Listening),
		"Load Balancer":    string(createdIngress.LoadBalancerType),
	})

	if createdIngress.DNSName != "" {
		writer.KeyValue(map[string]string{"DNS Name": createdIngress.DNSName})
	}

	if len(createdIngress.RouteSelectors) > 0 {
		writer.KeyValue(map[string]string{"Route Selectors": ingress.FormatRouteSelectors(createdIngress.RouteSelectors)})
	}

	writer.Info("\nThe ingress may take a few minutes to provision. To check status:")
	fmt.Printf("  rosa describe ingress %s --cluster %s\n", createdIngress.ID, opts.ClusterName)

	return nil
}
