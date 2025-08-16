package admin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/admin"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
)

type createAdminOptions struct {
	clusterName string
	username    string
	password    string
	expiresIn   string
	interactive bool
}

// NewCreateCommand creates the admin create command
func NewCreateCommand(logger *slog.Logger) *cobra.Command {
	opts := &createAdminOptions{}

	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Create a cluster-admin user",
		Long: `Create a cluster-admin user with full administrative access to the cluster.

This command creates an HTPasswd identity provider with a single admin user
that has cluster-admin privileges. This is useful for initial cluster access
and emergency administrative tasks.`,
		Example: `  # Create admin user for a cluster
  rosa create admin --cluster my-cluster

  # Create admin with custom username
  rosa create admin --cluster my-cluster --username myadmin

  # Create admin with expiration
  rosa create admin --cluster my-cluster --expires-in 24h

  # Interactive mode
  rosa create admin --interactive`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateAdmin(cmd.Context(), logger, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.clusterName, "cluster", "c", "", "Name or ID of the cluster")
	cmd.Flags().StringVar(&opts.username, "username", "cluster-admin", "Username for the admin user")
	cmd.Flags().StringVar(&opts.password, "password", "", "Password for the admin user (auto-generated if not provided)")
	cmd.Flags().StringVar(&opts.expiresIn, "expires-in", "", "Expiration time (e.g., 24h, 7d)")
	cmd.Flags().BoolVarP(&opts.interactive, "interactive", "i", false, "Interactive mode")

	return cmd
}

func runCreateAdmin(ctx context.Context, logger *slog.Logger, opts *createAdminOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Interactive mode
	if opts.interactive {
		if err := promptForAdminOptions(ctx, opts); err != nil {
			return err
		}
	}

	// Validate options
	if opts.clusterName == "" {
		return fmt.Errorf("cluster name is required")
	}

	// Parse expiration
	var expiresIn time.Duration
	if opts.expiresIn != "" {
		var err error
		expiresIn, err = time.ParseDuration(opts.expiresIn)
		if err != nil {
			return fmt.Errorf("invalid expiration duration: %w", err)
		}
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

	// Create admin service
	adminService := admin.NewService(apiClient, logger)

	// Get cluster ID (in case user provided name instead of ID)
	// For now, we'll assume they provide the ID
	clusterID := opts.clusterName

	writer.Info("Creating cluster-admin user...")

	// Create the admin user
	adminUser, err := adminService.Create(ctx, admin.CreateOptions{
		ClusterID: clusterID,
		Username:  opts.username,
		Password:  opts.password,
		ExpiresIn: expiresIn,
	})
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	// Get cluster URLs
	apiURL, err := adminService.GetClusterAPIURL(ctx, clusterID)
	if err != nil {
		logger.Warn("Could not get cluster API URL", "error", err)
		apiURL = "Not available yet"
	}

	consoleURL, err := adminService.GetClusterConsoleURL(ctx, clusterID)
	if err != nil {
		logger.Warn("Could not get cluster console URL", "error", err)
		consoleURL = "Not available yet"
	}

	// Display results
	writer.Success("Admin user created successfully!")
	fmt.Println()

	writer.Title("Cluster Admin Credentials")
	writer.KeyValue(map[string]string{
		"Username": adminUser.Username,
		"Password": adminUser.Password,
	})

	if adminUser.ExpiresAt != nil {
		writer.KeyValue(map[string]string{
			"Expires": adminUser.ExpiresAt.Format(time.RFC3339),
		})
	}

	fmt.Println()
	writer.Title("Cluster URLs")
	writer.KeyValue(map[string]string{
		"API URL":     apiURL,
		"Console URL": consoleURL,
	})

	fmt.Println()
	writer.Info("To login to the cluster:")
	fmt.Printf("  oc login %s --username %s --password %s\n", apiURL, adminUser.Username, adminUser.Password)
	fmt.Println()
	writer.Info("To access the web console:")
	fmt.Printf("  Open %s in your browser\n", consoleURL)
	fmt.Printf("  Login with username: %s\n", adminUser.Username)

	writer.Warning("Please save these credentials securely. The password cannot be retrieved later.")

	return nil
}

// NewDeleteCommand creates the admin delete command
func NewDeleteCommand(logger *slog.Logger) *cobra.Command {
	opts := &createAdminOptions{}

	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Delete a cluster-admin user",
		Long:  `Delete a cluster-admin user from the cluster.`,
		Example: `  # Delete admin user from a cluster
  rosa delete admin --cluster my-cluster --username cluster-admin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteAdmin(cmd.Context(), logger, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.clusterName, "cluster", "c", "", "Name or ID of the cluster")
	cmd.Flags().StringVar(&opts.username, "username", "cluster-admin", "Username of the admin user to delete")

	return cmd
}

func runDeleteAdmin(ctx context.Context, logger *slog.Logger, opts *createAdminOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Validate options
	if opts.clusterName == "" {
		return fmt.Errorf("cluster name is required")
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

	// Create admin service
	adminService := admin.NewService(apiClient, logger)

	// Confirm deletion
	var confirm bool
	err = huh.NewConfirm().
		Title("Delete admin user?").
		Description(fmt.Sprintf("Delete admin user '%s' from cluster '%s'?", opts.username, opts.clusterName)).
		Value(&confirm).
		Run()
	if err != nil {
		return err
	}

	if !confirm {
		writer.Info("Deletion cancelled")
		return nil
	}

	// Delete the admin user
	writer.Info("Deleting admin user...")
	err = adminService.Delete(ctx, opts.clusterName, opts.username)
	if err != nil {
		return fmt.Errorf("failed to delete admin user: %w", err)
	}

	writer.Success(fmt.Sprintf("Admin user '%s' deleted successfully", opts.username))

	return nil
}

// NewListCommand creates the admin list command
func NewListCommand(logger *slog.Logger) *cobra.Command {
	opts := &createAdminOptions{}

	cmd := &cobra.Command{
		Use:   "admins",
		Short: "List cluster-admin users",
		Long:  `List all cluster-admin users for a cluster.`,
		Example: `  # List admin users for a cluster
  rosa list admins --cluster my-cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListAdmins(cmd.Context(), logger, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.clusterName, "cluster", "c", "", "Name or ID of the cluster")

	return cmd
}

func runListAdmins(ctx context.Context, logger *slog.Logger, opts *createAdminOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Validate options
	if opts.clusterName == "" {
		return fmt.Errorf("cluster name is required")
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

	// Create admin service
	adminService := admin.NewService(apiClient, logger)

	// List admin users
	writer.Info("Fetching admin users...")
	admins, err := adminService.List(ctx, opts.clusterName)
	if err != nil {
		return fmt.Errorf("failed to list admin users: %w", err)
	}

	if len(admins) == 0 {
		writer.Info("No admin users found for cluster " + opts.clusterName)
		return nil
	}

	// Display results
	writer.Title(fmt.Sprintf("Admin Users for cluster %s", opts.clusterName))
	fmt.Println()

	headers := []string{"USERNAME", "CREATED"}
	var rows [][]string

	for _, admin := range admins {
		rows = append(rows, []string{
			admin.Username,
			admin.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	writer.Table(headers, rows)

	return nil
}

func promptForAdminOptions(ctx context.Context, opts *createAdminOptions) error {
	// Prompt for cluster name
	if opts.clusterName == "" {
		err := huh.NewInput().
			Title("Cluster Name").
			Description("Name or ID of the cluster").
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("cluster name cannot be empty")
				}
				return nil
			}).
			Value(&opts.clusterName).
			Run()
		if err != nil {
			return err
		}
	}

	// Prompt for username
	var customUsername bool
	err := huh.NewConfirm().
		Title("Use custom username?").
		Description("Default username is 'cluster-admin'").
		Value(&customUsername).
		Run()
	if err != nil {
		return err
	}

	if customUsername {
		err := huh.NewInput().
			Title("Admin Username").
			Description("Username for the admin user").
			Value(&opts.username).
			Run()
		if err != nil {
			return err
		}
	} else {
		opts.username = "cluster-admin"
	}

	// Prompt for password
	var customPassword bool
	err = huh.NewConfirm().
		Title("Set custom password?").
		Description("A secure password will be generated if not set").
		Value(&customPassword).
		Run()
	if err != nil {
		return err
	}

	if customPassword {
		err := huh.NewInput().
			Title("Admin Password").
			Description("Password for the admin user").
			Value(&opts.password).
			Run()
		if err != nil {
			return err
		}
	}

	// Prompt for expiration
	var setExpiration bool
	err = huh.NewConfirm().
		Title("Set expiration?").
		Description("Admin user can expire after a certain time").
		Value(&setExpiration).
		Run()
	if err != nil {
		return err
	}

	if setExpiration {
		expirationOptions := []string{
			"1h", "6h", "12h", "24h", "7d", "30d",
		}

		err := huh.NewSelect[string]().
			Title("Expiration Time").
			Options(huh.NewOptions(expirationOptions...)...).
			Value(&opts.expiresIn).
			Run()
		if err != nil {
			return err
		}
	}

	return nil
}
