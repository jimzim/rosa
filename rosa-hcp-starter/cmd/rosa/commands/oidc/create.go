package oidc

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/pkg/oidc"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// CreateOptions contains options for creating OIDC config
type CreateOptions struct {
	Managed          bool
	BucketName       string
	PrivateKeySecret string
	IssuerURL        string
	Region           string
	RoleARN          string
	Prefix           string
}

// NewCreateCommand creates the OIDC config create command
func NewCreateCommand(svc *oidc.Service) *cobra.Command {
	opts := &CreateOptions{}

	cmd := &cobra.Command{
		Use:   "oidc-config",
		Short: "Create OIDC configuration",
		Long: `Create an OpenID Connect (OIDC) configuration for STS authentication.

OIDC configuration is required for ROSA HCP clusters to authenticate with AWS using STS.
You can create either a managed (Red Hat hosted) or unmanaged (self-hosted) OIDC configuration.`,
		Example: `  # Create managed OIDC config (recommended)
  rosa create oidc-config --managed
  
  # Create unmanaged OIDC config with custom S3 bucket
  rosa create oidc-config \
    --bucket-name my-oidc-bucket \
    --region us-west-2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd.Context(), svc, opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.Managed, "managed", true, "Use Red Hat managed OIDC configuration (recommended)")
	flags.StringVar(&opts.BucketName, "bucket-name", "", "S3 bucket name for unmanaged OIDC")
	flags.StringVar(&opts.PrivateKeySecret, "private-key-secret", "", "AWS Secrets Manager ARN for private key")
	flags.StringVar(&opts.IssuerURL, "issuer-url", "", "Custom issuer URL for unmanaged OIDC")
	flags.StringVar(&opts.Region, "region", "", "AWS region for resources")
	flags.StringVar(&opts.RoleARN, "role-arn", "", "IAM role ARN for OIDC provider")
	flags.StringVar(&opts.Prefix, "prefix", "", "Prefix for OIDC resources")

	return cmd
}

func runCreate(ctx context.Context, svc *oidc.Service, opts *CreateOptions) error {
	// Validate options
	if err := validateCreateOptions(opts); err != nil {
		return err
	}

	progress := output.NewProgress("Creating OIDC configuration")
	progress.Start()
	defer progress.Stop()

	var config *oidc.Config
	var err error

	if opts.Managed {
		// Create managed OIDC config
		config, err = svc.CreateManaged(ctx, oidc.ManagedConfig{
			Region: opts.Region,
		})
		if err != nil {
			progress.Stop()
			return fmt.Errorf("failed to create managed OIDC config: %w", err)
		}
	} else {
		// Create unmanaged OIDC config
		config, err = svc.CreateUnmanaged(ctx, oidc.UnmanagedConfig{
			BucketName:       opts.BucketName,
			PrivateKeySecret: opts.PrivateKeySecret,
			IssuerURL:        opts.IssuerURL,
			Region:           opts.Region,
			RoleARN:          opts.RoleARN,
		})
		if err != nil {
			progress.Stop()
			return fmt.Errorf("failed to create unmanaged OIDC config: %w", err)
		}
	}

	progress.Success("OIDC configuration created successfully")

	// Display OIDC config details
	output.Info("\nOIDC Configuration:")
	writer := output.NewWriter(output.FormatText)

	configInfo := map[string]string{
		"ID":         config.ID,
		"Type":       config.Type,
		"Issuer URL": config.IssuerURL,
		"Region":     config.Region,
	}

	if !opts.Managed {
		configInfo["S3 Bucket"] = config.BucketName
		if config.PrivateKeySecret != "" {
			configInfo["Private Key"] = config.PrivateKeySecret
		}
	}

	writer.KeyValue(configInfo)

	// Display next steps with better integration guidance
	output.Info("\nNext Steps:")
	fmt.Println("  1. Create account roles (if not already created):")
	fmt.Println("     rosa create account-roles --region " + config.Region)
	fmt.Println()
	fmt.Println("  2. Create operator roles for your cluster:")
	fmt.Printf("     rosa create operator-roles --cluster <cluster-name> --oidc-endpoint %s\n", config.IssuerURL)
	fmt.Println()
	fmt.Println("  3. Create a cluster with this OIDC configuration:")
	fmt.Printf("     rosa cluster create --name <cluster-name> --oidc-config-id %s --role-arn <installer-role-arn> --subnet-ids <subnet-ids>\n", config.ID)
	
	// Additional helpful information
	output.Info("\nImportant:")
	fmt.Println("  • The OIDC endpoint URL for operator roles is: " + config.IssuerURL)
	fmt.Printf("  • Use --oidc-config-id %s when creating clusters\n", config.ID)
	fmt.Println("  • This OIDC configuration can be reused for multiple clusters")

	return nil
}

func validateCreateOptions(opts *CreateOptions) error {
	if !opts.Managed {
		if opts.BucketName == "" {
			return fmt.Errorf("bucket name is required for unmanaged OIDC configuration")
		}

		// Validate bucket name
		if len(opts.BucketName) < 3 || len(opts.BucketName) > 63 {
			return fmt.Errorf("bucket name must be between 3 and 63 characters")
		}

		if !isValidBucketName(opts.BucketName) {
			return fmt.Errorf("invalid bucket name: must contain only lowercase letters, numbers, and hyphens")
		}
	}

	return nil
}

func isValidBucketName(name string) bool {
	// S3 bucket naming rules
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '.') {
			return false
		}
	}

	// Cannot start or end with hyphen
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return false
	}

	// Cannot have consecutive periods or hyphens
	if strings.Contains(name, "..") || strings.Contains(name, "--") {
		return false
	}

	return true
}
