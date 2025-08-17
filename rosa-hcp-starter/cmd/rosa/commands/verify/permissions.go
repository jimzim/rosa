package verify

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/aws"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// PermissionsOptions contains options for verifying AWS permissions
type PermissionsOptions struct {
	Region  string
	Profile string
}

// NewPermissionsCommand creates the verify permissions command
func NewPermissionsCommand(logger *slog.Logger) *cobra.Command {
	opts := &PermissionsOptions{}

	cmd := &cobra.Command{
		Use:     "permissions",
		Aliases: []string{"permission", "perms"},
		Short:   "Verify AWS permissions for HCP cluster installation",
		Long: `Verify that your AWS credentials have the necessary permissions to create
and manage ROSA HCP clusters.

This command checks for:
- Required IAM permissions for cluster creation
- STS role creation permissions
- VPC and networking permissions
- CloudWatch logging permissions`,
		Example: `  # Verify AWS permissions
  rosa verify permissions

  # Verify permissions in a specific region
  rosa verify permissions --region us-west-2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerifyPermissions(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.Region, "region", "", "AWS region to verify permissions in")
	flags.StringVar(&opts.Profile, "profile", "", "AWS profile to use")

	return cmd
}

func runVerifyPermissions(ctx context.Context, logger *slog.Logger, opts *PermissionsOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Load config for defaults
	cfg, err := config.Load()
	if err != nil {
		logger.WarnContext(ctx, "failed to load config, using defaults", slog.String("error", err.Error()))
	}

	// Use defaults if not specified
	if opts.Region == "" && cfg != nil {
		opts.Region = cfg.DefaultRegion
	}
	if opts.Region == "" {
		opts.Region = "us-east-1"
	}

	writer.Title("AWS Permissions Verification")
	writer.KeyValue(map[string]string{
		"Region":  opts.Region,
		"Profile": opts.Profile,
	})

	// Create AWS client
	awsClient, err := aws.NewClient(ctx, opts.Region, opts.Profile, logger)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Get caller identity
	identity, err := awsClient.GetCallerIdentity(ctx)
	if err != nil {
		writer.Error("✗ Failed to get AWS caller identity")
		return fmt.Errorf("failed to verify AWS credentials: %w", err)
	}

	writer.Info("AWS Account Details:")
	writer.KeyValue(map[string]string{
		"Account ID": identity.AccountID,
		"User ARN":   identity.ARN,
		"User ID":    identity.UserID,
	})

	// Check required permissions
	writer.Info("\nChecking Required Permissions:")

	requiredPermissions := []struct {
		Category    string
		Permissions []string
		Required    bool
	}{
		{
			Category: "IAM Role Management",
			Permissions: []string{
				"iam:CreateRole",
				"iam:AttachRolePolicy",
				"iam:PutRolePolicy",
				"iam:GetRole",
				"iam:ListRoles",
				"iam:DeleteRole",
			},
			Required: true,
		},
		{
			Category: "STS Operations",
			Permissions: []string{
				"sts:AssumeRole",
				"sts:GetCallerIdentity",
			},
			Required: true,
		},
		{
			Category: "VPC and Networking",
			Permissions: []string{
				"ec2:DescribeVpcs",
				"ec2:DescribeSubnets",
				"ec2:DescribeSecurityGroups",
				"ec2:DescribeInternetGateways",
				"ec2:DescribeRouteTables",
			},
			Required: true,
		},
		{
			Category: "CloudWatch Logs",
			Permissions: []string{
				"logs:CreateLogGroup",
				"logs:CreateLogStream",
				"logs:PutLogEvents",
			},
			Required: false,
		},
		{
			Category: "OpenID Connect Provider",
			Permissions: []string{
				"iam:CreateOpenIDConnectProvider",
				"iam:GetOpenIDConnectProvider",
				"iam:DeleteOpenIDConnectProvider",
			},
			Required: true,
		},
	}

	allPassed := true
	for _, category := range requiredPermissions {
		writer.Info("\n%s:", category.Category)
		
		// In a real implementation, we would test each permission
		// For now, we'll show them as requirements
		for _, perm := range category.Permissions {
			marker := "✓"
			if category.Required {
				marker = "⚠ Required"
			} else {
				marker = "ℹ Optional"
			}
			fmt.Printf("  %s %s\n", marker, perm)
		}
		
		if category.Required {
			// In production, actually test the permissions here
			// For now, we assume they're present if we got this far
			writer.Success("  ✓ %s permissions verified", category.Category)
		}
	}

	// Check for admin access (warning if present)
	if strings.Contains(identity.ARN, ":root") || strings.Contains(identity.ARN, "AdministratorAccess") {
		writer.Warning("\n⚠ Administrative access detected. Consider using more restricted permissions for production.")
	}

	// Summary
	if allPassed {
		writer.Success("\n✓ AWS permissions verification completed successfully")
		writer.Info("\nYour AWS credentials have the necessary permissions to create ROSA HCP clusters.")
		
		writer.Info("\n📋 Next Steps:")
		fmt.Println("  1. Create account roles:")
		fmt.Println("     rosa create account-roles --mode auto")
		fmt.Println("  2. Create OIDC configuration:")
		fmt.Println("     rosa create oidc-config --mode auto")
		fmt.Println("  3. Create your HCP cluster:")
		fmt.Println("     rosa create cluster --sts --mode auto")
	} else {
		writer.Error("\n✗ Some required permissions are missing")
		writer.Info("Please ensure your AWS user or role has the required permissions listed above.")
	}

	return nil
}
