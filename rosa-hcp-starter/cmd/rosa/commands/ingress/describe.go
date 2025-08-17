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

// DescribeOptions contains options for describing an ingress
type DescribeOptions struct {
	ClusterName  string
	IngressID    string
	OutputFormat string
}

// NewDescribeCommand creates the ingress describe command
func NewDescribeCommand(logger *slog.Logger) *cobra.Command {
	opts := &DescribeOptions{}

	cmd := &cobra.Command{
		Use:   "ingress INGRESS_ID",
		Short: "Describe an ingress for a cluster",
		Long:  `Show detailed information about a specific ingress (load balancer) for a cluster.`,
		Example: `  # Describe an ingress
  rosa describe ingress apps --cluster my-cluster

  # Output in JSON format
  rosa describe ingress apps --cluster my-cluster --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IngressID = args[0]
			return runDescribeIngress(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format (text, json, yaml)")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runDescribeIngress(ctx context.Context, logger *slog.Logger, opts *DescribeOptions) error {
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

	// Get the ingress
	ing, err := ingressSvc.Get(ctx, cluster.ID(), opts.IngressID)
	if err != nil {
		return fmt.Errorf("failed to get ingress: %w", err)
	}

	// Display results based on output format
	var writer *output.Writer

	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		writer = output.NewWriter(output.FormatJSON)
		return writer.Print(ing)
	case "yaml":
		writer = output.NewWriter(output.FormatYAML)
		return writer.Print(ing)
	default:
		writer = output.NewWriter(output.FormatText)
		
		writer.Title(fmt.Sprintf("Ingress '%s' for cluster '%s'", ing.ID, cluster.Name()))
		
		// Basic information
		writer.KeyValue(map[string]string{
			"ID":              ing.ID,
			"Default":         fmt.Sprintf("%t", ing.Default),
			"Listening":       string(ing.Listening),
			"Private":         fmt.Sprintf("%t", ing.Listening == ingress.ListeningInternal),
			"Load Balancer":   string(ing.LoadBalancerType),
		})

		// DNS information
		if ing.DNSName != "" {
			writer.KeyValue(map[string]string{
				"DNS Name": ing.DNSName,
			})
			
			// Show example URLs
			writer.Info("\nAccess URLs:")
			protocol := "https"
			if ing.Listening == ingress.ListeningInternal {
				writer.Info("  (Internal access only)")
			}
			fmt.Printf("  Console:   %s://console-openshift-console.%s\n", protocol, ing.DNSName)
			fmt.Printf("  OAuth:     %s://oauth-openshift.%s\n", protocol, ing.DNSName)
			fmt.Printf("  Routes:    %s://<route-name>.%s\n", protocol, ing.DNSName)
		}

		// Route selectors
		if len(ing.RouteSelectors) > 0 {
			writer.Info("\nRoute Selectors:")
			for key, value := range ing.RouteSelectors {
				fmt.Printf("  %s: %s\n", key, value)
			}
			writer.Info("\nOnly routes matching these selectors will be exposed through this ingress.")
		} else {
			writer.Info("\nRoute Selectors: (none)")
			if !ing.Default {
				writer.Info("All routes will be exposed through this ingress (no filtering).")
			}
		}

		// Status and actions
		writer.Info("\nStatus:")
		if ing.Default {
			writer.Info("  This is the default ingress and cannot be deleted.")
			writer.Info("  You can edit its configuration using:")
			fmt.Printf("    rosa edit ingress %s --cluster %s\n", ing.ID, opts.ClusterName)
		} else {
			writer.Info("  This is a custom ingress.")
			writer.Info("  Available actions:")
			fmt.Printf("    rosa edit ingress %s --cluster %s\n", ing.ID, opts.ClusterName)
			fmt.Printf("    rosa delete ingress %s --cluster %s\n", ing.ID, opts.ClusterName)
		}

		// Tips based on configuration
		if ing.LoadBalancerType == ingress.LoadBalancerClassic {
			writer.Info("\nNote: Using Classic Load Balancer. Consider NLB for better performance:")
			fmt.Printf("  rosa edit ingress %s --cluster %s --lb-type nlb\n", ing.ID, opts.ClusterName)
		}

		if ing.Listening == ingress.ListeningExternal && !ing.Default {
			writer.Info("\nSecurity tip: If this ingress is for internal services, consider making it private:")
			fmt.Printf("  rosa edit ingress %s --cluster %s --private\n", ing.ID, opts.ClusterName)
		}
	}

	return nil
}
