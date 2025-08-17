package addon

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/addon"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// InstallOptions contains options for installing an add-on
type InstallOptions struct {
	ClusterName      string
	AddonID          string
	BillingModel     string
	BillingAccountID string
	Parameters       map[string]string
	Yes              bool
	Interactive      bool
}

// NewInstallCommand creates the addon install command
func NewInstallCommand(logger *slog.Logger) *cobra.Command {
	opts := &InstallOptions{
		Parameters: make(map[string]string),
	}

	cmd := &cobra.Command{
		Use:   "addon ADDON_ID",
		Short: "Install an add-on to a cluster",
		Long: `Install a Red Hat managed add-on to a cluster.

Add-ons extend cluster functionality with additional operators and services
like logging, monitoring, or service mesh capabilities.`,
		Example: `  # Install the cluster-logging-operator add-on
  rosa install addon cluster-logging-operator --cluster my-cluster

  # Install with marketplace billing
  rosa install addon my-addon --cluster my-cluster --billing-model marketplace-aws

  # Install with parameters
  rosa install addon my-addon --cluster my-cluster --param key1=value1 --param key2=value2`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.AddonID = args[0]
			}
			return runInstallAddon(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVar(&opts.BillingModel, "billing-model", "standard", "Billing model (standard, marketplace, marketplace-aws)")
	flags.StringVar(&opts.BillingAccountID, "billing-account-id", "", "Billing account ID for marketplace billing")
	flags.StringToStringVar(&opts.Parameters, "param", nil, "Add-on parameters (can be specified multiple times)")
	flags.BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")
	flags.BoolVarP(&opts.Interactive, "interactive", "i", false, "Interactive mode")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runInstallAddon(ctx context.Context, logger *slog.Logger, opts *InstallOptions) error {
	writer := output.NewWriter(output.FormatText)

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

	// Check cluster state
	if cluster.State() != "ready" {
		return fmt.Errorf("cluster '%s' is not ready (current state: %s)", opts.ClusterName, cluster.State())
	}

	// Create Add-on service
	addonSvc, err := addon.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create add-on service: %w", err)
	}

	// Interactive mode - select add-on if not specified
	if opts.Interactive || opts.AddonID == "" {
		// List available add-ons
		available, err := addonSvc.ListAvailable(ctx)
		if err != nil {
			return fmt.Errorf("failed to list available add-ons: %w", err)
		}

		// Filter only enabled add-ons
		var enabledAddons []*addon.AddOn
		var addonOptions []huh.Option[string]
		for _, a := range available {
			if a.Available {
				enabledAddons = append(enabledAddons, a)
				desc := a.Name
				if a.RequiresSTS {
					desc += " (requires STS)"
				}
				addonOptions = append(addonOptions, huh.NewOption(desc, a.ID))
			}
		}

		if len(addonOptions) == 0 {
			return fmt.Errorf("no add-ons available for installation")
		}

		// Select add-on
		err = huh.NewSelect[string]().
			Title("Select add-on to install").
			Options(addonOptions...).
			Value(&opts.AddonID).
			Run()
		if err != nil {
			return fmt.Errorf("failed to select add-on: %w", err)
		}
	}

	// Get add-on details
	addonDetails, err := addonSvc.GetAddOn(ctx, opts.AddonID)
	if err != nil {
		return fmt.Errorf("failed to get add-on details: %w", err)
	}

	if !addonDetails.Available {
		return fmt.Errorf("add-on '%s' is not available for installation", opts.AddonID)
	}

	// Check if already installed
	existing, _ := addonSvc.Get(ctx, cluster.ID(), opts.AddonID)
	if existing != nil {
		return fmt.Errorf("add-on '%s' is already installed on cluster '%s' (state: %s)", 
			opts.AddonID, opts.ClusterName, existing.State)
	}

	// Get cluster info for STS
	isSTS, operatorRolePrefix, accountID := addon.GetClusterInfo(cluster)

	// Interactive mode for billing
	if opts.Interactive {
		billingOptions := []huh.Option[string]{
			huh.NewOption("Standard", "standard"),
			huh.NewOption("AWS Marketplace", "marketplace-aws"),
		}
		
		err = huh.NewSelect[string]().
			Title("Select billing model").
			Options(billingOptions...).
			Value(&opts.BillingModel).
			Run()
		if err != nil {
			return fmt.Errorf("failed to select billing model: %w", err)
		}

		if strings.Contains(opts.BillingModel, "marketplace") {
			err = huh.NewInput().
				Title("Billing account ID (optional)").
				Description("AWS Marketplace account ID for billing").
				Value(&opts.BillingAccountID).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get billing account ID: %w", err)
			}
		}
	}

	// Display installation summary
	writer.Title("Add-on Installation Summary")
	writer.KeyValue(map[string]string{
		"Cluster":        cluster.Name(),
		"Add-on ID":      addonDetails.ID,
		"Add-on Name":    addonDetails.Name,
		"Version":        addonDetails.Version,
		"Billing Model":  opts.BillingModel,
	})

	if opts.BillingAccountID != "" {
		writer.KeyValue(map[string]string{"Billing Account": opts.BillingAccountID})
	}

	if len(opts.Parameters) > 0 {
		writer.KeyValue(map[string]string{"Parameters": addon.FormatParameters(opts.Parameters)})
	}

	// Show STS warning if applicable
	if addonDetails.RequiresSTS {
		if !isSTS {
			return fmt.Errorf("add-on '%s' requires STS authentication, but cluster is not STS-enabled", opts.AddonID)
		}
		
		writer.Warning("This add-on requires STS authentication and will need access to AWS resources.")
		writer.Info("Account ID: %s", accountID)
		
		if len(addonDetails.CredentialsRequests) > 0 {
			writer.Info("Required IAM policies:")
			for _, cr := range addonDetails.CredentialsRequests {
				fmt.Printf("  - %s\n", cr)
			}
		}

		// In a real implementation, we would create the operator roles here
		// For now, we'll just show a message
		writer.Warning("\nNote: Operator IAM roles must be created for this add-on to function properly.")
		writer.Info("Run the following command to create the required roles:")
		fmt.Printf("  rosa create operator-roles --cluster %s --addon %s\n", opts.ClusterName, opts.AddonID)
	}

	// Confirm installation
	if !opts.Yes {
		var confirm bool
		err = huh.NewConfirm().
			Title(fmt.Sprintf("Install add-on '%s' on cluster '%s'?", addonDetails.Name, opts.ClusterName)).
			Value(&confirm).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirm {
			writer.Info("Installation cancelled")
			return nil
		}
	}

	// Prepare installation config
	installConfig := addon.InstallConfig{
		AddonID:            opts.AddonID,
		BillingModel:       addon.BillingModel(opts.BillingModel),
		BillingAccountID:   opts.BillingAccountID,
		Parameters:         opts.Parameters,
		OperatorRolePrefix: operatorRolePrefix,
	}

	// Install the add-on
	writer.Info("Installing add-on...")
	installation, err := addonSvc.Install(ctx, cluster.ID(), installConfig)
	if err != nil {
		return fmt.Errorf("failed to install add-on: %w", err)
	}

	writer.Success("Add-on '%s' installation initiated", installation.AddonName)
	
	// Display installation details
	writer.Info("\nInstallation Details:")
	writer.KeyValue(map[string]string{
		"Installation ID": installation.ID,
		"State":          string(installation.State),
	})

	if installation.StateDescription != "" {
		writer.KeyValue(map[string]string{"Description": installation.StateDescription})
	}

	writer.Info("\nThe add-on installation may take several minutes to complete.")
	writer.Info("To check the installation status:")
	fmt.Printf("  rosa list addons --cluster %s\n", opts.ClusterName)
	fmt.Printf("  rosa describe addon %s --cluster %s\n", opts.AddonID, opts.ClusterName)

	if addonDetails.DocsLink != "" {
		writer.Info("\nDocumentation: %s", addonDetails.DocsLink)
	}

	return nil
}
