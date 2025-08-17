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

// EditOptions contains options for editing an ingress
type EditOptions struct {
	ClusterName    string
	IngressID      string
	Private        *bool
	LBType         string
	RouteSelectors string
	ClearSelectors bool
	Interactive    bool
}

// NewEditCommand creates the ingress edit command
func NewEditCommand(logger *slog.Logger) *cobra.Command {
	opts := &EditOptions{}
	var privateFlag, publicFlag bool

	cmd := &cobra.Command{
		Use:   "ingress INGRESS_ID",
		Short: "Edit an ingress for a cluster",
		Long: `Edit an existing ingress (load balancer) for a cluster.

You can modify the listening method (private/public), load balancer type,
and route selectors. The default ingress can be edited but not deleted.`,
		Example: `  # Make an ingress private
  rosa edit ingress apps2 --cluster my-cluster --private

  # Change load balancer type to NLB
  rosa edit ingress apps2 --cluster my-cluster --lb-type nlb

  # Update route selectors
  rosa edit ingress apps2 --cluster my-cluster --route-selectors app=backend

  # Clear route selectors
  rosa edit ingress apps2 --cluster my-cluster --clear-selectors`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IngressID = args[0]
			
			// Handle private/public flags
			if privateFlag && publicFlag {
				return fmt.Errorf("cannot specify both --private and --public")
			}
			if privateFlag {
				private := true
				opts.Private = &private
			} else if publicFlag {
				private := false
				opts.Private = &private
			}
			
			return runEditIngress(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.BoolVar(&privateFlag, "private", false, "Make the ingress private (internal)")
	flags.BoolVar(&publicFlag, "public", false, "Make the ingress public (external)")
	flags.StringVar(&opts.LBType, "lb-type", "", "Load balancer type (classic or nlb)")
	flags.StringVar(&opts.RouteSelectors, "route-selectors", "", "Route selectors (comma-separated key=value pairs)")
	flags.BoolVar(&opts.ClearSelectors, "clear-selectors", false, "Clear all route selectors")
	flags.BoolVarP(&opts.Interactive, "interactive", "i", false, "Interactive mode")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runEditIngress(ctx context.Context, logger *slog.Logger, opts *EditOptions) error {
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

	// Get existing ingress
	existingIngress, err := ingressSvc.Get(ctx, cluster.ID(), opts.IngressID)
	if err != nil {
		return fmt.Errorf("failed to get ingress: %w", err)
	}

	// Interactive mode
	if opts.Interactive {
		// Private/Public
		var listening int
		if existingIngress.Listening == ingress.ListeningInternal {
			listening = 1
		}
		
		err = huh.NewSelect[int]().
			Title("Listening method").
			Options(
				huh.NewOption("Public (External)", 0),
				huh.NewOption("Private (Internal)", 1),
			).
			Value(&listening).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get listening method: %w", err)
		}
		
		private := (listening == 1)
		opts.Private = &private

		// Load balancer type
		var lbTypeIdx int
		if existingIngress.LoadBalancerType == ingress.LoadBalancerNLB {
			lbTypeIdx = 1
		}
		
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
		
		lbTypes := []string{"classic", "nlb"}
		opts.LBType = lbTypes[lbTypeIdx]

		// Route selectors
		currentSelectors := ingress.FormatRouteSelectors(existingIngress.RouteSelectors)
		err = huh.NewInput().
			Title("Route selectors").
			Description("Comma-separated key=value pairs (leave empty to keep current)").
			Placeholder(currentSelectors).
			Value(&opts.RouteSelectors).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get route selectors: %w", err)
		}
		
		// Ask about clearing if they left it empty and there were existing selectors
		if opts.RouteSelectors == "" && len(existingIngress.RouteSelectors) > 0 {
			err = huh.NewConfirm().
				Title("Clear route selectors?").
				Description("Do you want to clear the existing route selectors?").
				Value(&opts.ClearSelectors).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get clear selectors option: %w", err)
			}
		}
	}

	// Check if any changes were requested
	if opts.Private == nil && opts.LBType == "" && opts.RouteSelectors == "" && !opts.ClearSelectors {
		if !opts.Interactive {
			writer.Warning("No changes specified. Use flags or --interactive mode to specify changes.")
			return nil
		}
	}

	// Prepare update config
	updateConfig := ingress.UpdateConfig{
		Private: opts.Private,
	}

	// Set load balancer type if specified
	if opts.LBType != "" {
		if opts.LBType != "classic" && opts.LBType != "nlb" {
			return fmt.Errorf("load balancer type must be 'classic' or 'nlb'")
		}
		lbType := ingress.LoadBalancerType(opts.LBType)
		updateConfig.LoadBalancerType = &lbType
	}

	// Handle route selectors
	if opts.ClearSelectors {
		// Clear selectors
		updateConfig.RouteSelectors = make(map[string]string)
	} else if opts.RouteSelectors != "" {
		// Parse and set new selectors
		routeSelectors, err := ingress.ParseRouteSelectors(opts.RouteSelectors)
		if err != nil {
			return fmt.Errorf("failed to parse route selectors: %w", err)
		}
		updateConfig.RouteSelectors = routeSelectors
	}

	// Display update summary
	writer.Title("Updating Ingress")
	writer.KeyValue(map[string]string{
		"Cluster": cluster.Name(),
		"Ingress": opts.IngressID,
	})

	// Show what's changing
	writer.Info("Changes:")
	changeCount := 0
	
	if opts.Private != nil {
		oldListening := string(existingIngress.Listening)
		newListening := "external"
		if *opts.Private {
			newListening = "internal"
		}
		fmt.Printf("  Listening: %s → %s\n", oldListening, newListening)
		changeCount++
	}
	
	if updateConfig.LoadBalancerType != nil {
		fmt.Printf("  Load Balancer: %s → %s\n", existingIngress.LoadBalancerType, *updateConfig.LoadBalancerType)
		changeCount++
	}
	
	if updateConfig.RouteSelectors != nil {
		oldSelectors := ingress.FormatRouteSelectors(existingIngress.RouteSelectors)
		if oldSelectors == "" {
			oldSelectors = "(none)"
		}
		newSelectors := ingress.FormatRouteSelectors(updateConfig.RouteSelectors)
		if newSelectors == "" {
			newSelectors = "(none)"
		}
		fmt.Printf("  Route Selectors: %s → %s\n", oldSelectors, newSelectors)
		changeCount++
	}
	
	if changeCount == 0 {
		writer.Info("No changes to apply")
		return nil
	}

	// Update the ingress
	updatedIngress, err := ingressSvc.Update(ctx, cluster.ID(), opts.IngressID, updateConfig)
	if err != nil {
		return fmt.Errorf("failed to update ingress: %w", err)
	}

	writer.Success("Ingress '%s' updated successfully", updatedIngress.ID)
	
	// Display updated ingress details
	writer.Info("\nUpdated Ingress Details:")
	writer.KeyValue(map[string]string{
		"ID":               updatedIngress.ID,
		"Default":          fmt.Sprintf("%t", updatedIngress.Default),
		"Listening":        string(updatedIngress.Listening),
		"Load Balancer":    string(updatedIngress.LoadBalancerType),
	})

	if updatedIngress.DNSName != "" {
		writer.KeyValue(map[string]string{"DNS Name": updatedIngress.DNSName})
	}

	if len(updatedIngress.RouteSelectors) > 0 {
		writer.KeyValue(map[string]string{"Route Selectors": ingress.FormatRouteSelectors(updatedIngress.RouteSelectors)})
	}

	writer.Info("\nChanges may take a few minutes to propagate.")

	return nil
}
