package idp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/idp"
	"github.com/openshift/rosa-hcp/pkg/output"
)

type createIDPOptions struct {
	clusterName string
	idpName     string
	idpType     string
	interactive bool

	// GitHub options
	githubClientID     string
	githubClientSecret string
	githubOrgs         []string
	githubTeams        []string
	githubHostname     string

	// Google options
	googleClientID     string
	googleClientSecret string
	googleHostedDomain string

	// GitLab options
	gitlabClientID     string
	gitlabClientSecret string
	gitlabURL          string
	gitlabCA           string
}

// NewCreateCommand creates the IdP create command
func NewCreateCommand(logger *slog.Logger) *cobra.Command {
	opts := &createIDPOptions{}

	cmd := &cobra.Command{
		Use:   "idp",
		Short: "Create an identity provider",
		Long: `Create an identity provider for user authentication.

Supported identity providers:
- GitHub: Authenticate using GitHub accounts
- Google: Authenticate using Google accounts
- GitLab: Authenticate using GitLab accounts
- LDAP: Authenticate using LDAP directory (coming soon)
- OpenID: Authenticate using OpenID Connect (coming soon)`,
		Example: `  # Create GitHub IdP
  rosa create idp --cluster my-cluster --type github \
    --github-client-id <id> --github-client-secret <secret>

  # Create Google IdP with hosted domain
  rosa create idp --cluster my-cluster --type google \
    --google-client-id <id> --google-client-secret <secret> \
    --google-hosted-domain example.com

  # Interactive mode
  rosa create idp --cluster my-cluster --interactive`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateIDP(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.clusterName, "cluster", "c", "", "Name or ID of the cluster")
	flags.StringVar(&opts.idpName, "name", "", "Name for the identity provider")
	flags.StringVar(&opts.idpType, "type", "", "Type of identity provider (github, google, gitlab)")
	flags.BoolVarP(&opts.interactive, "interactive", "i", false, "Interactive mode")

	// GitHub flags
	flags.StringVar(&opts.githubClientID, "github-client-id", "", "GitHub OAuth app client ID")
	flags.StringVar(&opts.githubClientSecret, "github-client-secret", "", "GitHub OAuth app client secret")
	flags.StringSliceVar(&opts.githubOrgs, "github-organizations", []string{}, "GitHub organizations to restrict access")
	flags.StringSliceVar(&opts.githubTeams, "github-teams", []string{}, "GitHub teams to restrict access")
	flags.StringVar(&opts.githubHostname, "github-hostname", "", "GitHub Enterprise hostname")

	// Google flags
	flags.StringVar(&opts.googleClientID, "google-client-id", "", "Google OAuth client ID")
	flags.StringVar(&opts.googleClientSecret, "google-client-secret", "", "Google OAuth client secret")
	flags.StringVar(&opts.googleHostedDomain, "google-hosted-domain", "", "Google Workspace domain to restrict access")

	// GitLab flags
	flags.StringVar(&opts.gitlabClientID, "gitlab-client-id", "", "GitLab OAuth app client ID")
	flags.StringVar(&opts.gitlabClientSecret, "gitlab-client-secret", "", "GitLab OAuth app client secret")
	flags.StringVar(&opts.gitlabURL, "gitlab-url", "https://gitlab.com", "GitLab instance URL")
	flags.StringVar(&opts.gitlabCA, "gitlab-ca", "", "CA certificate for self-hosted GitLab")

	return cmd
}

func runCreateIDP(ctx context.Context, logger *slog.Logger, opts *createIDPOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Interactive mode
	if opts.interactive {
		if err := promptForIDPOptions(ctx, opts); err != nil {
			return err
		}
	}

	// Validate options
	if opts.clusterName == "" {
		return fmt.Errorf("cluster name is required")
	}
	if opts.idpType == "" {
		return fmt.Errorf("IdP type is required")
	}
	if opts.idpName == "" {
		opts.idpName = fmt.Sprintf("%s-idp", opts.idpType)
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

	// Create IdP service
	idpService := idp.NewService(apiClient, logger)

	writer.Info(fmt.Sprintf("Creating %s identity provider...", opts.idpType))

	var createdIDP *idp.IdentityProvider

	switch strings.ToLower(opts.idpType) {
	case "github":
		if opts.githubClientID == "" || opts.githubClientSecret == "" {
			return fmt.Errorf("GitHub client ID and secret are required")
		}

		config := idp.GitHubConfig{
			ClientID:      opts.githubClientID,
			ClientSecret:  opts.githubClientSecret,
			Organizations: opts.githubOrgs,
			Teams:         opts.githubTeams,
			Hostname:      opts.githubHostname,
		}

		createdIDP, err = idpService.CreateGitHubIDP(ctx, opts.clusterName, opts.idpName, config)

	case "google":
		if opts.googleClientID == "" || opts.googleClientSecret == "" {
			return fmt.Errorf("Google client ID and secret are required")
		}

		config := idp.GoogleConfig{
			ClientID:     opts.googleClientID,
			ClientSecret: opts.googleClientSecret,
			HostedDomain: opts.googleHostedDomain,
		}

		createdIDP, err = idpService.CreateGoogleIDP(ctx, opts.clusterName, opts.idpName, config)

	case "gitlab":
		if opts.gitlabClientID == "" || opts.gitlabClientSecret == "" {
			return fmt.Errorf("GitLab client ID and secret are required")
		}

		config := idp.GitLabConfig{
			ClientID:     opts.gitlabClientID,
			ClientSecret: opts.gitlabClientSecret,
			URL:          opts.gitlabURL,
			CA:           opts.gitlabCA,
		}

		createdIDP, err = idpService.CreateGitLabIDP(ctx, opts.clusterName, opts.idpName, config)

	default:
		return fmt.Errorf("unsupported IdP type: %s", opts.idpType)
	}

	if err != nil {
		return fmt.Errorf("failed to create IdP: %w", err)
	}

	writer.Success("Identity provider created successfully!")

	// Display IdP details
	fmt.Println()
	writer.Title("Identity Provider Details")
	writer.KeyValue(map[string]string{
		"Name":    createdIDP.Name,
		"Type":    createdIDP.Type,
		"ID":      createdIDP.ID,
		"Cluster": opts.clusterName,
	})

	// Show OAuth callback URL
	fmt.Println()
	writer.Info("OAuth Callback URL:")
	fmt.Printf("  https://oauth-openshift.apps.<cluster-domain>/oauth2callback/%s\n", opts.idpName)
	fmt.Println()
	writer.Info("Add this callback URL to your OAuth application settings.")

	return nil
}

// NewDeleteCommand creates the IdP delete command
func NewDeleteCommand(logger *slog.Logger) *cobra.Command {
	var clusterName, idpName string

	cmd := &cobra.Command{
		Use:   "idp",
		Short: "Delete an identity provider",
		Long:  `Delete an identity provider from a cluster.`,
		Example: `  # Delete an IdP
  rosa delete idp --cluster my-cluster --name github-idp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteIDP(cmd.Context(), logger, clusterName, idpName)
		},
	}

	cmd.Flags().StringVarP(&clusterName, "cluster", "c", "", "Name or ID of the cluster")
	cmd.Flags().StringVar(&idpName, "name", "", "Name of the identity provider to delete")
	cmd.MarkFlagRequired("cluster")
	cmd.MarkFlagRequired("name")

	return cmd
}

func runDeleteIDP(ctx context.Context, logger *slog.Logger, clusterName, idpName string) error {
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

	// Create IdP service
	idpService := idp.NewService(apiClient, logger)

	// Confirm deletion
	var confirm bool
	err = huh.NewConfirm().
		Title("Delete identity provider?").
		Description(fmt.Sprintf("Delete IdP '%s' from cluster '%s'?", idpName, clusterName)).
		Value(&confirm).
		Run()
	if err != nil {
		return err
	}

	if !confirm {
		writer.Info("Deletion cancelled")
		return nil
	}

	// Delete the IdP
	writer.Info("Deleting identity provider...")
	err = idpService.Delete(ctx, clusterName, idpName)
	if err != nil {
		return fmt.Errorf("failed to delete IdP: %w", err)
	}

	writer.Success(fmt.Sprintf("Identity provider '%s' deleted successfully", idpName))

	return nil
}

// NewListCommand creates the IdP list command
func NewListCommand(logger *slog.Logger) *cobra.Command {
	var clusterName string

	cmd := &cobra.Command{
		Use:   "idps",
		Short: "List identity providers",
		Long:  `List all identity providers for a cluster.`,
		Example: `  # List IdPs for a cluster
  rosa list idps --cluster my-cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListIDPs(cmd.Context(), logger, clusterName)
		},
	}

	cmd.Flags().StringVarP(&clusterName, "cluster", "c", "", "Name or ID of the cluster")
	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runListIDPs(ctx context.Context, logger *slog.Logger, clusterName string) error {
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

	// Create IdP service
	idpService := idp.NewService(apiClient, logger)

	// List IdPs
	writer.Info("Fetching identity providers...")
	idps, err := idpService.List(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to list IdPs: %w", err)
	}

	if len(idps) == 0 {
		writer.Info(fmt.Sprintf("No identity providers found for cluster '%s'", clusterName))
		return nil
	}

	// Display results
	writer.Title(fmt.Sprintf("Identity Providers for cluster '%s'", clusterName))
	fmt.Println()

	headers := []string{"NAME", "TYPE", "MAPPING", "LOGIN", "CHALLENGE"}
	var rows [][]string

	for _, idp := range idps {
		loginStr := "No"
		if idp.Login {
			loginStr = "Yes"
		}

		challengeStr := "No"
		if idp.Challenge {
			challengeStr = "Yes"
		}

		rows = append(rows, []string{
			idp.Name,
			idp.Type,
			idp.MappingMethod,
			loginStr,
			challengeStr,
		})
	}

	writer.Table(headers, rows)

	return nil
}

func promptForIDPOptions(ctx context.Context, opts *createIDPOptions) error {
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

	// Select IdP type
	if opts.idpType == "" {
		idpTypes := []string{"github", "google", "gitlab"}

		err := huh.NewSelect[string]().
			Title("Identity Provider Type").
			Options(huh.NewOptions(idpTypes...)...).
			Value(&opts.idpType).
			Run()
		if err != nil {
			return err
		}
	}

	// Prompt for IdP name
	if opts.idpName == "" {
		defaultName := fmt.Sprintf("%s-idp", opts.idpType)
		err := huh.NewInput().
			Title("Identity Provider Name").
			Description("Name for the identity provider").
			Placeholder(defaultName).
			Value(&opts.idpName).
			Run()
		if err != nil {
			return err
		}
		if opts.idpName == "" {
			opts.idpName = defaultName
		}
	}

	// Type-specific prompts
	switch opts.idpType {
	case "github":
		if opts.githubClientID == "" {
			err := huh.NewInput().
				Title("GitHub Client ID").
				Description("OAuth app client ID from GitHub").
				Value(&opts.githubClientID).
				Run()
			if err != nil {
				return err
			}
		}

		if opts.githubClientSecret == "" {
			err := huh.NewInput().
				Title("GitHub Client Secret").
				Description("OAuth app client secret from GitHub").
				Value(&opts.githubClientSecret).
				Run()
			if err != nil {
				return err
			}
		}

		// Ask about organizations
		var restrictOrgs bool
		err := huh.NewConfirm().
			Title("Restrict to organizations?").
			Description("Limit access to specific GitHub organizations").
			Value(&restrictOrgs).
			Run()
		if err != nil {
			return err
		}

		if restrictOrgs {
			var orgsStr string
			err := huh.NewInput().
				Title("GitHub Organizations").
				Description("Comma-separated list of organizations").
				Value(&orgsStr).
				Run()
			if err != nil {
				return err
			}
			if orgsStr != "" {
				opts.githubOrgs = strings.Split(orgsStr, ",")
			}
		}

	case "google":
		if opts.googleClientID == "" {
			err := huh.NewInput().
				Title("Google Client ID").
				Description("OAuth client ID from Google").
				Value(&opts.googleClientID).
				Run()
			if err != nil {
				return err
			}
		}

		if opts.googleClientSecret == "" {
			err := huh.NewInput().
				Title("Google Client Secret").
				Description("OAuth client secret from Google").
				Value(&opts.googleClientSecret).
				Run()
			if err != nil {
				return err
			}
		}

		// Ask about hosted domain
		var restrictDomain bool
		err := huh.NewConfirm().
			Title("Restrict to Google Workspace domain?").
			Description("Limit access to a specific Google Workspace domain").
			Value(&restrictDomain).
			Run()
		if err != nil {
			return err
		}

		if restrictDomain {
			err := huh.NewInput().
				Title("Google Workspace Domain").
				Description("Domain to restrict access (e.g., example.com)").
				Value(&opts.googleHostedDomain).
				Run()
			if err != nil {
				return err
			}
		}

	case "gitlab":
		if opts.gitlabURL == "" {
			opts.gitlabURL = "https://gitlab.com"
		}

		var customGitLab bool
		err := huh.NewConfirm().
			Title("Use self-hosted GitLab?").
			Description("Connect to a self-hosted GitLab instance").
			Value(&customGitLab).
			Run()
		if err != nil {
			return err
		}

		if customGitLab {
			err := huh.NewInput().
				Title("GitLab URL").
				Description("URL of your GitLab instance").
				Value(&opts.gitlabURL).
				Run()
			if err != nil {
				return err
			}
		}

		if opts.gitlabClientID == "" {
			err := huh.NewInput().
				Title("GitLab Client ID").
				Description("OAuth app client ID from GitLab").
				Value(&opts.gitlabClientID).
				Run()
			if err != nil {
				return err
			}
		}

		if opts.gitlabClientSecret == "" {
			err := huh.NewInput().
				Title("GitLab Client Secret").
				Description("OAuth app client secret from GitLab").
				Value(&opts.gitlabClientSecret).
				Run()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
