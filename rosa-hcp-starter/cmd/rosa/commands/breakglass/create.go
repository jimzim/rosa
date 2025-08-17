package breakglass

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/breakglass"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// CreateOptions contains options for creating a break-glass credential
type CreateOptions struct {
	ClusterName    string
	Username       string
	Expiration     string
	Description    string
	ShowKubeconfig bool
	SaveKubeconfig string
	Interactive    bool
}

// NewCreateCommand creates the break-glass-credential create command
func NewCreateCommand(logger *slog.Logger) *cobra.Command {
	opts := &CreateOptions{}

	cmd := &cobra.Command{
		Use:     "break-glass-credential",
		Aliases: []string{"break-glass-credentials", "breakglass"},
		Short:   "Create a break-glass credential for emergency cluster access",
		Long: `Create a break-glass credential for emergency access to a cluster when 
external authentication is unavailable or misconfigured.

Break-glass credentials provide temporary emergency access and should be:
- Created only when needed
- Revoked after use
- Monitored for usage

Requirements:
- Cluster must be HCP (Hosted Control Plane)
- External authentication must be configured`,
		Example: `  # Create a break-glass credential interactively
  rosa create break-glass-credential --cluster my-cluster

  # Create with specific username and expiration
  rosa create break-glass-credential --cluster my-cluster \
    --username emergency-admin \
    --expiration 24h

  # Create and save kubeconfig to file
  rosa create break-glass-credential --cluster my-cluster \
    --save-kubeconfig ~/emergency-kubeconfig`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateBreakGlass(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.StringVar(&opts.Username, "username", "", "Username for the credential (default: generated)")
	flags.StringVar(&opts.Expiration, "expiration", "24h", "Expiration duration (e.g., 24h, 7d)")
	flags.StringVar(&opts.Description, "description", "", "Description of why this credential was created")
	flags.BoolVar(&opts.ShowKubeconfig, "kubeconfig", true, "Display the kubeconfig after creation")
	flags.StringVar(&opts.SaveKubeconfig, "save-kubeconfig", "", "Save kubeconfig to specified file")
	flags.BoolVarP(&opts.Interactive, "interactive", "i", false, "Interactive mode")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runCreateBreakGlass(ctx context.Context, logger *slog.Logger, opts *CreateOptions) error {
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

	// Create Break-glass service
	bgSvc, err := breakglass.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create break-glass service: %w", err)
	}

	// Check if supported
	supported, reason, err := bgSvc.IsSupported(ctx, cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to check break-glass support: %w", err)
	}
	if !supported {
		writer.Error(reason)
		if reason == "Break-glass credentials require external authentication to be configured first" {
			writer.Info("\nTo configure external authentication:")
			fmt.Printf("  rosa create external-auth-provider --cluster %s\n", opts.ClusterName)
		}
		return fmt.Errorf("break-glass credentials not available")
	}

	// Interactive mode
	if opts.Interactive {
		// Username
		if opts.Username == "" {
			defaultUsername := fmt.Sprintf("emergency-%d", time.Now().Unix())
			err = huh.NewInput().
				Title("Username").
				Description("Username for the break-glass credential").
				Placeholder(defaultUsername).
				Value(&opts.Username).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get username: %w", err)
			}
			if opts.Username == "" {
				opts.Username = defaultUsername
			}
		}

		// Expiration
		var expirationChoice string
		err = huh.NewSelect[string]().
			Title("Expiration").
			Description("How long should this credential remain valid?").
			Options(
				huh.NewOption("1 hour", "1h"),
				huh.NewOption("4 hours", "4h"),
				huh.NewOption("12 hours", "12h"),
				huh.NewOption("24 hours (default)", "24h"),
				huh.NewOption("48 hours", "48h"),
				huh.NewOption("7 days", "168h"),
			).
			Value(&expirationChoice).
			Run()
		if err != nil {
			return fmt.Errorf("failed to get expiration: %w", err)
		}
		if expirationChoice != "" {
			opts.Expiration = expirationChoice
		}

		// Description
		if opts.Description == "" {
			err = huh.NewText().
				Title("Description (optional)").
				Description("Reason for creating this emergency credential").
				Placeholder("Emergency maintenance due to auth provider outage").
				Value(&opts.Description).
				CharLimit(200).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get description: %w", err)
			}
		}

		// Save kubeconfig
		if opts.SaveKubeconfig == "" {
			var saveToFile bool
			err = huh.NewConfirm().
				Title("Save kubeconfig to file?").
				Description("Save the kubeconfig to a file for later use").
				Value(&saveToFile).
				Run()
			if err != nil {
				return fmt.Errorf("failed to get save option: %w", err)
			}

			if saveToFile {
				defaultPath := fmt.Sprintf("emergency-kubeconfig-%s-%d", cluster.Name(), time.Now().Unix())
				err = huh.NewInput().
					Title("File path").
					Placeholder(defaultPath).
					Value(&opts.SaveKubeconfig).
					Run()
				if err != nil {
					return fmt.Errorf("failed to get file path: %w", err)
				}
				if opts.SaveKubeconfig == "" {
					opts.SaveKubeconfig = defaultPath
				}
			}
		}
	}

	// Parse expiration
	duration, err := time.ParseDuration(opts.Expiration)
	if err != nil {
		return fmt.Errorf("invalid expiration duration: %w", err)
	}
	expirationTime := time.Now().Add(duration)

	// Generate username if not provided
	if opts.Username == "" {
		opts.Username = fmt.Sprintf("emergency-%d", time.Now().Unix())
	}

	// Prepare configuration
	bgConfig := breakglass.Config{
		Username:       opts.Username,
		ExpirationTime: expirationTime,
		Description:    opts.Description,
	}

	// Display configuration
	writer.Title("Creating Break-glass Credential")
	writer.KeyValue(map[string]string{
		"Cluster":  cluster.Name(),
		"Username": bgConfig.Username,
		"Expires":  expirationTime.Format(time.RFC3339),
		"Duration": opts.Expiration,
	})
	if bgConfig.Description != "" {
		writer.KeyValue(map[string]string{
			"Description": bgConfig.Description,
		})
	}

	// Create the credential
	writer.Info("\nCreating break-glass credential...")
	credential, err := bgSvc.Create(ctx, cluster.ID(), bgConfig)
	if err != nil {
		return fmt.Errorf("failed to create break-glass credential: %w", err)
	}

	writer.Success("Break-glass credential created successfully")
	writer.KeyValue(map[string]string{
		"Credential ID": credential.ID,
		"Status":        breakglass.FormatStatus(credential.Status),
	})

	// Get kubeconfig
	if opts.ShowKubeconfig || opts.SaveKubeconfig != "" {
		writer.Info("\nRetrieving kubeconfig...")
		kubeconfig, err := bgSvc.GetKubeconfig(ctx, cluster.ID(), credential.ID)
		if err != nil {
			writer.Warning("Failed to retrieve kubeconfig immediately: %v", err)
			writer.Info("You can retrieve it later with:")
			fmt.Printf("  rosa describe break-glass-credential %s --cluster %s --kubeconfig\n",
				credential.ID, opts.ClusterName)
		} else {
			// Save to file if requested
			if opts.SaveKubeconfig != "" {
				err = os.WriteFile(opts.SaveKubeconfig, []byte(kubeconfig), 0600)
				if err != nil {
					return fmt.Errorf("failed to save kubeconfig: %w", err)
				}
				writer.Success("Kubeconfig saved to: %s", opts.SaveKubeconfig)
				writer.Info("\nTo use this kubeconfig:")
				fmt.Printf("  export KUBECONFIG=%s\n", opts.SaveKubeconfig)
				fmt.Printf("  kubectl get nodes\n")
			}

			// Display kubeconfig if requested
			if opts.ShowKubeconfig && opts.SaveKubeconfig == "" {
				writer.Info("\n" + strings.Repeat("=", 60))
				writer.Info("KUBECONFIG:")
				writer.Info(strings.Repeat("=", 60))
				fmt.Println(kubeconfig)
				writer.Info(strings.Repeat("=", 60))
			}
		}
	}

	// Display important information
	writer.Warning("\n⚠️  IMPORTANT SECURITY NOTICE:")
	writer.Warning("• This credential provides FULL CLUSTER ADMIN access")
	writer.Warning("• It will expire at: %s", expirationTime.Format(time.RFC3339))
	writer.Warning("• All actions are logged for audit purposes")
	writer.Warning("• Revoke immediately after use:")
	fmt.Printf("    rosa revoke break-glass-credential %s --cluster %s\n",
		credential.ID, opts.ClusterName)

	writer.Info("\n📋 Available Commands:")
	fmt.Printf("  # View credential details:\n")
	fmt.Printf("  rosa describe break-glass-credential %s --cluster %s\n\n",
		credential.ID, opts.ClusterName)

	fmt.Printf("  # List all break-glass credentials:\n")
	fmt.Printf("  rosa list break-glass-credentials --cluster %s\n\n", opts.ClusterName)

	fmt.Printf("  # Revoke this credential:\n")
	fmt.Printf("  rosa revoke break-glass-credential %s --cluster %s\n",
		credential.ID, opts.ClusterName)

	return nil
}
