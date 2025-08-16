package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift/rosa-hcp/cmd/rosa/commands/auth"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/cluster"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/iam"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/network"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/nodepool"
	"github.com/openshift/rosa-hcp/cmd/rosa/commands/oidc"
	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/aws"
	clusterSvc "github.com/openshift/rosa-hcp/pkg/cluster"
	nodepoolSvc "github.com/openshift/rosa-hcp/pkg/nodepool"
	oidcSvc "github.com/openshift/rosa-hcp/pkg/oidc"
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
		auth.NewLoginCommand(),
		auth.NewWhoAmICommand(),
		NewCompletionCommand(),
		NewDocsCommand(),
	)

	return rootCmd
}

// Services holds all service instances
type Services struct {
	API      api.Client
	AWS      aws.Client
	Cluster  *clusterSvc.Service
	NodePool *nodepoolSvc.Service
	OIDC     *oidcSvc.Service
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
	awsClient, err := aws.NewClient(ctx, aws.Config{
		Region:  profile.Region,
		Profile: profile.Name,
		RoleARN: profile.STS.RoleARN,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Create services
	return &Services{
		API:      apiClient,
		AWS:      awsClient,
		Cluster:  clusterSvc.NewService(apiClient, awsClient, logger),
		NodePool: nodepoolSvc.NewService(apiClient, awsClient, logger),
		OIDC:     oidcSvc.NewService(apiClient, awsClient, logger),
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
