package addon

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/addon"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// ListOptions contains options for listing add-ons
type ListOptions struct {
	ClusterName  string
	OutputFormat string
	ShowAll      bool
}

// NewListCommand creates the addon list command
func NewListCommand(logger *slog.Logger) *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:     "addons",
		Aliases: []string{"addon"},
		Short:   "List add-ons for a cluster",
		Long: `List all available and installed add-ons for a cluster.

Shows the installation state of each add-on and whether it's available
for installation on the specified cluster.`,
		Example: `  # List all add-ons for a cluster
  rosa list addons --cluster my-cluster

  # Show all available add-ons (not cluster-specific)
  rosa list addons --all

  # Output in JSON format
  rosa list addons --cluster my-cluster --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListAddons(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster")
	flags.StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format (text, json, yaml)")
	flags.BoolVar(&opts.ShowAll, "all", false, "Show all available add-ons (not cluster-specific)")

	return cmd
}

func runListAddons(ctx context.Context, logger *slog.Logger, opts *ListOptions) error {
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

	// Create Add-on service
	addonSvc, err := addon.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create add-on service: %w", err)
	}

	var writer *output.Writer

	// If --all flag is set, show all available add-ons
	if opts.ShowAll {
		available, err := addonSvc.ListAvailable(ctx)
		if err != nil {
			return fmt.Errorf("failed to list available add-ons: %w", err)
		}

		switch strings.ToLower(opts.OutputFormat) {
		case "json":
			writer = output.NewWriter(output.FormatJSON)
			return writer.Print(available)
		case "yaml":
			writer = output.NewWriter(output.FormatYAML)
			return writer.Print(available)
		default:
			writer = output.NewWriter(output.FormatText)
			writer.Title("Available Add-ons")

			if len(available) == 0 {
				writer.Info("No add-ons available")
				return nil
			}

			// Display as table
			fmt.Printf("%-30s %-40s %-10s %-8s\n", "ID", "NAME", "VERSION", "STATUS")
			fmt.Println(strings.Repeat("-", 90))

			for _, a := range available {
				status := "Available"
				if !a.Available {
					status = "Disabled"
				}
				if a.RequiresSTS {
					status += " (STS)"
				}
				
				version := a.Version
				if version == "" {
					version = "-"
				}

				fmt.Printf("%-30s %-40s %-10s %-8s\n", 
					a.ID, 
					truncate(a.Name, 40),
					truncate(version, 10),
					status)
			}
			fmt.Println()

			writer.Info("To install an add-on:")
			fmt.Println("  rosa install addon <addon-id> --cluster <cluster-name>")
			
			return nil
		}
	}

	// Cluster-specific add-ons list
	if opts.ClusterName == "" {
		return fmt.Errorf("either --cluster or --all flag is required")
	}

	// Get cluster
	clusterResp, err := apiClient.GetCluster(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	cluster := clusterResp.Body()

	// List cluster add-ons
	clusterAddons, err := addonSvc.List(ctx, cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to list cluster add-ons: %w", err)
	}

	// Display results based on output format
	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		writer = output.NewWriter(output.FormatJSON)
		return writer.Print(clusterAddons)
	case "yaml":
		writer = output.NewWriter(output.FormatYAML)
		return writer.Print(clusterAddons)
	default:
		writer = output.NewWriter(output.FormatText)
		writer.Title(fmt.Sprintf("Add-ons for cluster '%s'", cluster.Name()))

		if len(clusterAddons) == 0 {
			writer.Info("No add-ons found")
			return nil
		}

		// Count installed add-ons
		installedCount := 0
		for _, ca := range clusterAddons {
			if ca.State != addon.StateNotInstalled {
				installedCount++
			}
		}

		writer.KeyValue(map[string]string{
			"Total Available": fmt.Sprintf("%d", len(clusterAddons)),
			"Installed":       fmt.Sprintf("%d", installedCount),
		})
		fmt.Println()

		// Display as table
		fmt.Printf("%-30s %-40s %-15s %-10s\n", "ID", "NAME", "STATE", "AVAILABLE")
		fmt.Println(strings.Repeat("-", 100))

		// Show installed add-ons first
		for _, ca := range clusterAddons {
			if ca.State != addon.StateNotInstalled {
				displayAddonRow(ca)
			}
		}

		// Then show available but not installed
		hasAvailable := false
		for _, ca := range clusterAddons {
			if ca.State == addon.StateNotInstalled && ca.Available {
				if !hasAvailable {
					fmt.Println(strings.Repeat("-", 100))
					hasAvailable = true
				}
				displayAddonRow(ca)
			}
		}

		fmt.Println()

		// Show helpful commands
		if installedCount > 0 {
			writer.Info("To uninstall an add-on:")
			fmt.Println("  rosa uninstall addon <addon-id> --cluster " + opts.ClusterName)
		}

		hasAvailableToInstall := false
		for _, ca := range clusterAddons {
			if ca.State == addon.StateNotInstalled && ca.Available {
				hasAvailableToInstall = true
				break
			}
		}

		if hasAvailableToInstall {
			writer.Info("\nTo install an add-on:")
			fmt.Println("  rosa install addon <addon-id> --cluster " + opts.ClusterName)
		}

		// Check cluster info for STS
		isSTS, _, _ := addon.GetClusterInfo(cluster)
		if isSTS {
			writer.Info("\nNote: This is an STS cluster. Some add-ons may require additional IAM roles.")
		}
	}

	return nil
}

func displayAddonRow(ca *addon.ClusterAddOn) {
	stateDisplay := string(ca.State)
	// Add color/emphasis for certain states
	switch ca.State {
	case addon.StateReady:
		stateDisplay = string(ca.State)
	case addon.StateInstalling:
		stateDisplay = string(ca.State) + "..."
	case addon.StateDeleting:
		stateDisplay = string(ca.State) + "..."
	case addon.StateFailed:
		stateDisplay = "FAILED"
	}

	available := "Yes"
	if !ca.Available {
		available = "No"
	}

	fmt.Printf("%-30s %-40s %-15s %-10s\n",
		ca.ID,
		truncate(ca.Name, 40),
		stateDisplay,
		available)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
