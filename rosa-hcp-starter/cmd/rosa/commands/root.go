package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift/rosa-hcp/cmd/rosa/commands/addon"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/admin"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/auth"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/breakglass"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/cluster"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/dnsdomain"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/externalauthprovider"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/iam"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/idp"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/ingress"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/instance"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/kubeletconfig"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/logs"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/network"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/nodepool"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/oidc"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/region"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/tuningconfig"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/user"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/verify"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/version"
	"github.com/openshift/rosa-hcp/internal/config"
	addonSvc "github.com/openshift/rosa-hcp/pkg/addon"
	adminSvc "github.com/openshift/rosa-hcp/pkg/admin"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/aws"
	breakglassSvc "github.com/openshift/rosa-hcp/pkg/breakglass"
	clusterSvc "github.com/openshift/rosa-hcp/pkg/cluster"
	extAuthSvc "github.com/openshift/rosa-hcp/pkg/externalauthprovider"
	iamSvc "github.com/openshift/rosa-hcp/pkg/iam"
	idpSvc "github.com/openshift/rosa-hcp/pkg/idp"
	ingressSvc "github.com/openshift/rosa-hcp/pkg/ingress"
	instanceSvc "github.com/openshift/rosa-hcp/pkg/instance"
	kubeletconfigSvc "github.com/openshift/rosa-hcp/pkg/kubeletconfig"
	networkSvc "github.com/openshift/rosa-hcp/pkg/network"
	nodepoolSvc "github.com/openshift/rosa-hcp/pkg/nodepool"
	oidcSvc "github.com/openshift/rosa-hcp/pkg/oidc"
	regionSvc "github.com/openshift/rosa-hcp/pkg/region"
	tuningconfigSvc "github.com/openshift/rosa-hcp/pkg/tuningconfig"
	userSvc "github.com/openshift/rosa-hcp/pkg/user"
	versionSvc "github.com/openshift/rosa-hcp/pkg/version"
)

// GlobalOptions contains flags that apply to all commands
type GlobalOptions struct {
	Debug   bool
	Profile string
	Region  string
	Output  string
}

// NewRootCommand creates the root command
func NewRootCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger) *cobra.Command {
	globalOpts := &GlobalOptions{}

	rootCmd := &cobra.Command{
		Use:   "rosa",
		Short: "ROSA CLI for managing Red Hat OpenShift Service on AWS with Hosted Control Planes",
		Long: `ROSA (Red Hat OpenShift Service on AWS) CLI is a command-line tool for 
creating and managing ROSA clusters with Hosted Control Planes (HCP).

This tool simplifies cluster lifecycle management, node pool operations, and 
configuration of ROSA HCP clusters running on AWS infrastructure.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Set up logging level based on debug flag
			if globalOpts.Debug {
				logger = logger.With(slog.String("debug", "enabled"))
			}

			// Validate output format
			switch globalOpts.Output {
			case "text", "json", "yaml":
				// Valid formats
			default:
				return fmt.Errorf("invalid output format: %s (must be text, json, or yaml)", globalOpts.Output)
			}

			return nil
		},
	}

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&globalOpts.Debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringVar(&globalOpts.Profile, "profile", "", "Use a specific profile from config file")
	rootCmd.PersistentFlags().StringVar(&globalOpts.Region, "region", "", "AWS region (overrides profile setting)")
	rootCmd.PersistentFlags().StringVarP(&globalOpts.Output, "output", "o", "text", "Output format (text|json|yaml)")

	// Add all subcommands
	rootCmd.AddCommand(
		NewClusterCommand(ctx, cfg, logger, globalOpts),
		NewNodePoolCommand(ctx, cfg, logger, globalOpts),
		NewCreateCommand(ctx, cfg, logger, globalOpts),
		NewListCommand(ctx, cfg, logger, globalOpts),
		NewDeleteCommand(ctx, cfg, logger, globalOpts),
		NewEditCommand(ctx, cfg, logger, globalOpts),
		NewDescribeCommand(ctx, cfg, logger, globalOpts),
		NewUpgradeCommand(ctx, cfg, logger, globalOpts),
		NewInstallCommand(ctx, cfg, logger, globalOpts),
		NewUninstallCommand(ctx, cfg, logger, globalOpts),
		NewLogsCommand(ctx, cfg, logger, globalOpts),
		NewVerifyCommand(ctx, cfg, logger, globalOpts),
		NewGrantCommand(ctx, cfg, logger, globalOpts),
		NewRevokeCommand(ctx, cfg, logger, globalOpts),
		auth.NewLoginCommand(),
		auth.NewWhoAmICommand(),
		NewCompletionCommand(),
		NewDocsCommand(),
	)

	return rootCmd
}

// Services holds all service instances
type Services struct {
	API           api.Client
	AWS           aws.Client
	Cluster       *clusterSvc.Service
	NodePool      *nodepoolSvc.Service
	OIDC          *oidcSvc.Service
	Admin         *adminSvc.Service
	IDP           *idpSvc.Service
	Version       *versionSvc.Service
	IAM           iamSvc.Service
	Network       networkSvc.Service
	InstanceSvc   instanceSvc.Service
	RegionSvc     regionSvc.Service
	KubeletConfig kubeletconfigSvc.Service
	TuningConfig  tuningconfigSvc.Service
	Ingress       ingressSvc.Service
	Addon         addonSvc.Service
	ExternalAuth  extAuthSvc.Service
	BreakGlass    breakglassSvc.Service
	User          userSvc.Service
}

// Instance returns the instance service
func (s *Services) Instance() instanceSvc.Service {
	return s.InstanceSvc
}

// Region returns the region service
func (s *Services) Region() regionSvc.Service {
	return s.RegionSvc
}

// initializeServices creates all service instances with proper dependencies
func initializeServices(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) (*Services, error) {
	// Determine active profile
	profile := cfg.GetProfile(opts.Profile)
	if opts.Region != "" {
		profile.Region = opts.Region
	}

	// Create API client
	apiClient, err := api.NewClient(ctx, api.Config{
		URL:   cfg.APIURL,
		Token: cfg.Token,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Create AWS client
	awsClient, err := aws.NewClient(ctx, profile.Region, profile.Name, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Create IAM service
	iamService, err := iamSvc.NewService(ctx, profile.Region, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM service: %w", err)
	}

	// Create Network service
	networkService, err := networkSvc.NewService(ctx, profile.Region, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Network service: %w", err)
	}

	// Create Instance service
	instanceService, err := instanceSvc.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to create Instance service: %w", err)
	}

	// Create Region service
	regionService, err := regionSvc.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to create Region service: %w", err)
	}

	// Create KubeletConfig service
	kubeletConfigService, err := kubeletconfigSvc.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to create KubeletConfig service: %w", err)
	}

	// Create TuningConfig service
	tuningConfigService, err := tuningconfigSvc.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to create TuningConfig service: %w", err)
	}

	// Create Ingress service
	ingressService, err := ingressSvc.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to create Ingress service: %w", err)
	}

	// Create Addon service
	addonService, err := addonSvc.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to create Addon service: %w", err)
	}

	// Create ExternalAuth service
	externalAuthService, err := extAuthSvc.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to create ExternalAuth service: %w", err)
	}

	// Create BreakGlass service
	breakGlassService, err := breakglassSvc.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to create BreakGlass service: %w", err)
	}

	// Create User service
	userService, err := userSvc.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to create User service: %w", err)
	}

	// Create services
	return &Services{
		API:           apiClient,
		AWS:           awsClient,
		Cluster:       clusterSvc.NewService(apiClient, awsClient, logger),
		NodePool:      nodepoolSvc.NewService(apiClient, awsClient, logger),
		OIDC:          oidcSvc.NewService(apiClient, awsClient, logger),
		Admin:         adminSvc.NewService(apiClient, logger),
		IDP:           idpSvc.NewService(apiClient, logger),
		Version:       versionSvc.NewService(apiClient, logger),
		IAM:           iamService,
		Network:       networkService,
		InstanceSvc:   instanceService,
		RegionSvc:     regionService,
		KubeletConfig: kubeletConfigService,
		TuningConfig:  tuningConfigService,
		Ingress:       ingressService,
		Addon:         addonService,
		ExternalAuth:  externalAuthService,
		BreakGlass:    breakGlassService,
		User:          userService,
	}, nil
}

// NewClusterCommand creates the cluster command
func NewClusterCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	// Import the cluster package commands
	clusterCmd := &cobra.Command{
		Use:   "cluster",
		Short: "Create and manage ROSA clusters",
		Long:  "Create and manage Red Hat OpenShift Service on AWS (ROSA) clusters with Hosted Control Planes.",
	}

	// Function to lazily initialize services
	getService := func() (*clusterSvc.Service, error) {
		services, err := initializeServices(ctx, cfg, logger, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize services: %w\n\nPlease ensure ROSA_TOKEN environment variable is set or run 'rosa login'")
		}
		return services.Cluster, nil
	}

	// Add the actual implementation commands
	clusterCmd.AddCommand(
		NewClusterCreateCommand(getService),
		NewClusterDeleteCommand(getService),
		NewClusterListCommand(getService),
		NewClusterDescribeCommand(getService),
		NewClusterEditCommand(logger),
		NewClusterUpgradeCommand(logger),
	)

	return clusterCmd
}

// NewClusterCreateCommand wraps the actual cluster create command
func NewClusterCreateCommand(getService func() (*clusterSvc.Service, error)) *cobra.Command {
	// We'll create the actual command without a service first
	cmd := cluster.NewCreateCommand(nil)

	// Override the RunE to inject the service
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Initialize service when the command actually runs
		service, err := getService()
		if err != nil {
			return err
		}

		// Create a new command with the service and run it directly
		// The flags are already bound to this command, so we just need to
		// inject the service and call the original RunE
		actualCmd := cluster.NewCreateCommand(service)

		// Copy all flag values from the wrapper command to the actual command
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if actualCmd.Flags().Lookup(f.Name) != nil {
				actualCmd.Flags().Set(f.Name, f.Value.String())
			}
		})

		// Execute the actual command's RunE with the current command context
		return actualCmd.RunE(cmd, args)
	}

	return cmd
}

// NewClusterDeleteCommand creates the cluster delete command
func NewClusterDeleteCommand(getService func() (*clusterSvc.Service, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "delete CLUSTER",
		Short: "Delete a cluster",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := getService()
			if err != nil {
				return err
			}

			clusterKey := ""
			if len(args) > 0 {
				clusterKey = args[0]
			} else {
				// Get cluster key from flag
				clusterKey, _ = cmd.Flags().GetString("cluster")
			}

			if clusterKey == "" {
				return fmt.Errorf("cluster name or ID required")
			}

			// Confirm deletion
			if !cmd.Flags().Changed("yes") {
				fmt.Printf("Are you sure you want to delete cluster '%s'? (y/N): ", clusterKey)
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Deletion cancelled")
					return nil
				}
			}

			fmt.Printf("Deleting cluster '%s'...\n", clusterKey)
			err = service.Delete(cmd.Context(), clusterKey)
			if err != nil {
				return fmt.Errorf("failed to delete cluster: %w", err)
			}

			fmt.Printf("Cluster '%s' deletion initiated\n", clusterKey)
			return nil
		},
	}
}

// NewClusterListCommand creates the cluster list command
func NewClusterListCommand(getService func() (*clusterSvc.Service, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List clusters",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := getService()
			if err != nil {
				return err
			}

			clusters, err := service.List(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to list clusters: %w", err)
			}

			if len(clusters) == 0 {
				fmt.Println("No clusters found")
				return nil
			}

			// Print table header
			fmt.Printf("%-20s %-15s %-10s %-15s %-10s\n", "NAME", "ID", "STATE", "REGION", "VERSION")
			fmt.Println(strings.Repeat("-", 80))

			// Print clusters
			for _, cluster := range clusters {
				fmt.Printf("%-20s %-15s %-10s %-15s %-10s\n",
					cluster.Name,
					cluster.ID,
					cluster.State,
					cluster.Region,
					cluster.Version,
				)
			}

			return nil
		},
	}
}

// NewClusterDescribeCommand creates the cluster describe command
func NewClusterDescribeCommand(getService func() (*clusterSvc.Service, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "describe CLUSTER",
		Short: "Describe a cluster",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := getService()
			if err != nil {
				return err
			}

			clusterKey := ""
			if len(args) > 0 {
				clusterKey = args[0]
			} else {
				clusterKey, _ = cmd.Flags().GetString("cluster")
			}

			if clusterKey == "" {
				return fmt.Errorf("cluster name or ID required")
			}

			cluster, err := service.Get(cmd.Context(), clusterKey)
			if err != nil {
				return fmt.Errorf("failed to get cluster: %w", err)
			}

			// Print cluster details
			fmt.Println("Cluster Details:")
			fmt.Printf("  Name:        %s\n", cluster.Name)
			fmt.Printf("  ID:          %s\n", cluster.ID)
			fmt.Printf("  State:       %s\n", cluster.State)
			fmt.Printf("  Region:      %s\n", cluster.Region)
			fmt.Printf("  Version:     %s\n", cluster.Version)
			fmt.Printf("  Created:     %s\n", cluster.CreatedAt.Format("2006-01-02 15:04:05"))
			if cluster.APIURL != "" {
				fmt.Printf("  API URL:     %s\n", cluster.APIURL)
			}
			if cluster.ConsoleURL != "" {
				fmt.Printf("  Console URL: %s\n", cluster.ConsoleURL)
			}

			return nil
		},
	}
}

// NewClusterEditCommand creates the cluster edit command
func NewClusterEditCommand(logger *slog.Logger) *cobra.Command {
	return cluster.NewEditClusterCommand(logger)
}

// NewClusterUpgradeCommand creates the cluster upgrade command
func NewClusterUpgradeCommand(logger *slog.Logger) *cobra.Command {
	return cluster.NewUpgradeCommand(logger)
}

// NewNodePoolCommand creates the nodepool command
func NewNodePoolCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	// Function to lazily initialize services
	getService := func() (*nodepoolSvc.Service, error) {
		services, err := initializeServices(ctx, cfg, logger, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize services: %w\n\nPlease ensure ROSA_TOKEN environment variable is set or run 'rosa login'")
		}
		return services.NodePool, nil
	}

	// Create wrapper commands that inject the service
	cmd := &cobra.Command{
		Use:     "nodepool",
		Aliases: []string{"nodepools", "node-pool", "node-pools", "np"},
		Short:   "Create and manage node pools",
		Long:    "Create and manage node pools for ROSA HCP clusters.",
	}

	// Add subcommands with lazy service initialization
	cmd.AddCommand(
		wrapNodePoolCommand(nodepool.NewCreateCommand, getService),
		wrapNodePoolCommand(nodepool.NewDeleteCommand, getService),
		wrapNodePoolCommand(nodepool.NewListCommand, getService),
		wrapNodePoolCommand(nodepool.NewDescribeCommand, getService),
		wrapNodePoolCommand(nodepool.NewEditCommand, getService),
	)

	return cmd
}

// wrapNodePoolCommand wraps a nodepool command with lazy service initialization
func wrapNodePoolCommand(cmdFunc func(*nodepoolSvc.Service) *cobra.Command, getService func() (*nodepoolSvc.Service, error)) *cobra.Command {
	// Create command without service first
	cmd := cmdFunc(nil)

	// Override RunE to inject the service
	originalRunE := cmd.RunE
	if originalRunE != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			// Initialize service when the command actually runs
			service, err := getService()
			if err != nil {
				return err
			}

			// Create a new command with the service
			actualCmd := cmdFunc(service)
			// Copy flags
			actualCmd.Flags().VisitAll(func(f *pflag.Flag) {
				if cmd.Flags().Changed(f.Name) {
					actualCmd.Flags().Set(f.Name, f.Value.String())
				}
			})

			// Execute the actual command
			return actualCmd.RunE(cmd, args)
		}
	}

	return cmd
}

// NewCreateCommand creates the create command with subcommands
func NewCreateCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create HCP resources",
		Long:  "Create ROSA HCP resources such as OIDC configurations and account roles.",
	}

	// Function to lazily initialize services
	getOIDCService := func() (*oidcSvc.Service, error) {
		services, err := initializeServices(ctx, cfg, logger, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize services: %w\n\nPlease ensure ROSA_TOKEN environment variable is set or run 'rosa login'")
		}
		return services.OIDC, nil
	}

	// Add OIDC config create command
	oidcCmd := oidc.NewCreateCommand(nil)
	oidcCmd.RunE = func(cmd *cobra.Command, args []string) error {
		service, err := getOIDCService()
		if err != nil {
			return err
		}

		// Create actual command with service
		actualCmd := oidc.NewCreateCommand(service)
		// Copy flags
		actualCmd.Flags().VisitAll(func(f *pflag.Flag) {
			if cmd.Flags().Changed(f.Name) {
				actualCmd.Flags().Set(f.Name, f.Value.String())
			}
		})

		return actualCmd.RunE(cmd, args)
	}

	cmd.AddCommand(oidcCmd)

	// Add network create command
	cmd.AddCommand(network.NewCreateCommand(logger))

	// Add IAM account-roles command
	cmd.AddCommand(iam.NewCreateAccountRolesCommand(logger))

	// Add IAM operator-roles command
	cmd.AddCommand(iam.NewCreateOperatorRolesCommand(logger))

	// Add admin user command
	cmd.AddCommand(admin.NewCreateCommand(logger))

	// Add identity provider command
	cmd.AddCommand(idp.NewCreateCommand(logger))

	// Add KubeletConfig command
	cmd.AddCommand(kubeletconfig.NewCreateCommand(logger))

	// Add TuningConfig command
	cmd.AddCommand(tuningconfig.NewCreateCommand(logger))

	// Add Ingress command
	cmd.AddCommand(ingress.NewCreateCommand(logger))

	// Add external auth provider create command
	cmd.AddCommand(externalauthprovider.NewCreateCommand(logger))

	// Add break-glass credential create command
	cmd.AddCommand(breakglass.NewCreateCommand(logger))

	// Add DNS domain create command
	cmd.AddCommand(dnsdomain.NewCreateCommand(logger))

	return cmd
}

// NewListCommand creates the list command with subcommands
func NewListCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List HCP resources",
		Long:  "List ROSA HCP resources such as clusters, networks, and node pools.",
	}

	// Add network list command
	cmd.AddCommand(network.NewListCommand(logger))

	// Add IAM account-roles list command
	cmd.AddCommand(iam.NewListAccountRolesCommand(logger))

	// Add admin list command
	cmd.AddCommand(admin.NewListCommand(logger))

	// Add IdP list command
	cmd.AddCommand(idp.NewListCommand(logger))

	// Add versions list command
	cmd.AddCommand(version.NewListCommand(logger))

	// Add upgrades list command
	cmd.AddCommand(version.NewUpgradePathsCommand(logger))

	// Add instance types list command
	cmd.AddCommand(instance.NewListCommand(logger))

	// Add regions list command
	cmd.AddCommand(region.NewListCommand(logger))

	// Add kubeletconfigs list command
	cmd.AddCommand(kubeletconfig.NewListCommand(logger))

	// Add tuning-configs list command
	cmd.AddCommand(tuningconfig.NewListCommand(logger))

	// Add ingresses list command
	cmd.AddCommand(ingress.NewListCommand(logger))

	// Add addons list command
	cmd.AddCommand(addon.NewListCommand(logger))

	// Add external auth providers list command
	cmd.AddCommand(externalauthprovider.NewListCommand(logger))

	// Add break-glass credentials list command
	cmd.AddCommand(breakglass.NewListCommand(logger))

	// Add users list command
	cmd.AddCommand(user.NewListCommand(logger))

	// Add DNS domains list command
	cmd.AddCommand(dnsdomain.NewListCommand(logger))

	return cmd
}

// NewDeleteCommand creates the delete command with subcommands
func NewDeleteCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete HCP resources",
		Long:  "Delete ROSA HCP resources such as clusters, networks, and node pools.",
	}

	// Add network delete command
	cmd.AddCommand(network.NewDeleteCommand(logger))

	// Add IAM account-roles delete command
	cmd.AddCommand(iam.NewDeleteAccountRolesCommand(logger))

	// Add IAM operator-roles delete command
	cmd.AddCommand(iam.NewDeleteOperatorRolesCommand(logger))

	// Add admin delete command
	cmd.AddCommand(admin.NewDeleteCommand(logger))

	// Add IdP delete command
	cmd.AddCommand(idp.NewDeleteCommand(logger))

	// Add upgrade cancel command
	cmd.AddCommand(cluster.NewCancelUpgradeCommand(logger))

	// Add kubeletconfig delete command
	cmd.AddCommand(kubeletconfig.NewDeleteCommand(logger))

	// Add tuning-config delete command
	cmd.AddCommand(tuningconfig.NewDeleteCommand(logger))

	// Add ingress delete command
	cmd.AddCommand(ingress.NewDeleteCommand(logger))

	// Add external auth provider delete command
	cmd.AddCommand(externalauthprovider.NewDeleteCommand(logger))

	// Add DNS domain delete command
	cmd.AddCommand(dnsdomain.NewDeleteCommand(logger))

	return cmd
}

// NewEditCommand creates the edit command with subcommands
func NewEditCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit HCP resources",
		Long:  "Edit ROSA HCP resources such as clusters, ingresses, and node pools.",
	}

	// Add ingress edit command
	cmd.AddCommand(ingress.NewEditCommand(logger))

	// Add other edit commands that might already exist

	return cmd
}

// NewDescribeCommand creates the describe command with subcommands
func NewDescribeCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describe HCP resources",
		Long:  "Show detailed information about ROSA HCP resources.",
	}

	// Add ingress describe command
	cmd.AddCommand(ingress.NewDescribeCommand(logger))

	// Add external auth provider describe command
	cmd.AddCommand(externalauthprovider.NewDescribeCommand(logger))

	// Add break-glass credential describe command
	cmd.AddCommand(breakglass.NewDescribeCommand(logger))

	// Add cluster describe command
	cmd.AddCommand(cluster.NewDescribeCommand(logger))

	// Add kubeletconfig describe command
	cmd.AddCommand(kubeletconfig.NewDescribeCommand(logger))

	// Add tuning-config describe command
	cmd.AddCommand(tuningconfig.NewDescribeCommand(logger))

	// Add DNS domain describe command
	cmd.AddCommand(dnsdomain.NewDescribeCommand(logger))

	return cmd
}

// NewInstallCommand creates the install command with subcommands
func NewInstallCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install add-ons and components",
		Long:  "Install Red Hat managed add-ons and components to ROSA HCP clusters.",
	}

	// Add addon install command
	cmd.AddCommand(addon.NewInstallCommand(logger))

	return cmd
}

// NewUninstallCommand creates the uninstall command with subcommands
func NewUninstallCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall add-ons and components",
		Long:  "Uninstall Red Hat managed add-ons and components from ROSA HCP clusters.",
	}

	// Add addon uninstall command
	cmd.AddCommand(addon.NewUninstallCommand(logger))

	return cmd
}

// NewVerifyCommand creates the verify command with subcommands
func NewVerifyCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify resources are configured correctly",
		Long:  "Verify resources are configured correctly for ROSA HCP cluster installation.",
	}

	// Add network verify command
	cmd.AddCommand(verify.NewNetworkCommand(logger))

	// Add permissions verify command
	cmd.AddCommand(verify.NewPermissionsCommand(logger))

	// Add quota verify command
	cmd.AddCommand(verify.NewQuotaCommand(logger))

	return cmd
}

// NewGrantCommand creates the grant command with subcommands
func NewGrantCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grant",
		Short: "Grant permissions to users",
		Long:  "Grant roles and permissions to users on ROSA HCP clusters.",
	}

	// Add grant user command
	cmd.AddCommand(user.NewGrantCommand(logger))

	return cmd
}

// NewRevokeCommand creates the revoke command with subcommands
func NewRevokeCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke permissions and credentials",
		Long:  "Revoke roles, permissions, and emergency credentials from ROSA HCP clusters.",
	}

	// Add revoke user command
	cmd.AddCommand(user.NewRevokeCommand(logger))

	// Add revoke break-glass credentials command
	cmd.AddCommand(breakglass.NewRevokeCommand(logger))

	return cmd
}

// NewUpgradeCommand creates the upgrade command with subcommands
func NewUpgradeCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade clusters and components",
		Long:  "Upgrade ROSA HCP clusters and their components to newer versions.",
	}

	// Add cluster upgrade command
	cmd.AddCommand(cluster.NewUpgradeCommand(logger))

	return cmd
}

// NewLogsCommand creates the logs command with subcommands
func NewLogsCommand(ctx context.Context, cfg *config.Config, logger *slog.Logger, opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View cluster logs",
		Long:  "View installation, uninstallation, and audit logs for ROSA HCP clusters.",
	}

	// Add install logs command
	cmd.AddCommand(logs.NewInstallCommand(logger))

	// Add uninstall logs command
	cmd.AddCommand(logs.NewUninstallCommand(logger))

	return cmd
}

// NewLoginCommand creates the login command
func NewLoginCommand(client api.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Log in to your Red Hat account",
		Long:  "Authenticate with Red Hat OpenShift Cluster Manager to manage ROSA clusters",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Implementation would go here
			fmt.Println("Login functionality to be implemented")
			return nil
		},
	}
}

// NewWhoAmICommand creates the whoami command
func NewWhoAmICommand(client api.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Display current login information",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Implementation would go here
			fmt.Println("WhoAmI functionality to be implemented")
			return nil
		},
	}
}

// NewCompletionCommand creates the completion command
func NewCompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for rosa CLI.

To load completions:

Bash:
  $ source <(rosa completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ rosa completion bash > /etc/bash_completion.d/rosa
  # macOS:
  $ rosa completion bash > /usr/local/etc/bash_completion.d/rosa

Zsh:
  $ source <(rosa completion zsh)
  # To load completions for each session, execute once:
  $ rosa completion zsh > "${fpath[1]}/_rosa"

Fish:
  $ rosa completion fish | source
  # To load completions for each session, execute once:
  $ rosa completion fish > ~/.config/fish/completions/rosa.fish

PowerShell:
  PS> rosa completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> rosa completion powershell > rosa.ps1
  # and source this file from your PowerShell profile.
`,
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
}

// NewDocsCommand creates the docs command
func NewDocsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "docs",
		Short: "Open ROSA documentation in your browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Implementation would open browser to docs
			fmt.Println("Opening ROSA HCP documentation...")
			return nil
		},
	}
}
