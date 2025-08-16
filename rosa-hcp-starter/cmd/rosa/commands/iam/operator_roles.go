package iam

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	
	"github.com/openshift/rosa-hcp/pkg/iam"
	"github.com/openshift/rosa-hcp/pkg/output"
)

type operatorRolesOptions struct {
	prefix       string
	region       string
	clusterID    string
	oidcEndpoint string
	accountRolePrefix string
	permissionsBoundary string
	path         string
	tags         map[string]string
	force        bool
	dryRun       bool
	interactive  bool
}

// NewCreateOperatorRolesCommand creates the operator-roles command
func NewCreateOperatorRolesCommand(logger *slog.Logger) *cobra.Command {
	opts := &operatorRolesOptions{}
	
	cmd := &cobra.Command{
		Use:     "operator-roles",
		Aliases: []string{"operatorroles"},
		Short:   "Create cluster-specific operator IAM roles",
		Long: `Create cluster-specific operator IAM roles for ROSA HCP.

These roles are created per cluster and allow various Kubernetes operators
to interact with AWS services. The roles trust the cluster's OIDC provider.

Operator roles include:
  - Ingress Operator: Manages load balancers and DNS
  - CSI Driver: Manages persistent volumes
  - Image Registry: Manages S3 buckets for container images
  - Cloud Network Config: Manages network configuration
  - Control Plane Operator: Manages control plane resources
  - KMS Provider: Manages encryption keys
  - Kube Controller Manager: Core Kubernetes controller`,
		Example: `  # Create operator roles for a cluster
  rosa create operator-roles --cluster my-cluster --oidc-endpoint https://oidc.example.com

  # Create with custom prefix
  rosa create operator-roles --cluster my-cluster --oidc-endpoint https://oidc.example.com --prefix MyOrg

  # Interactive mode
  rosa create operator-roles --interactive`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateOperatorRoles(cmd.Context(), logger, opts)
		},
	}
	
	cmd.Flags().StringVar(&opts.prefix, "prefix", "ManagedOpenShift", "Prefix for operator role names")
	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region")
	cmd.Flags().StringVarP(&opts.clusterID, "cluster", "c", "", "Cluster ID or name")
	cmd.Flags().StringVar(&opts.oidcEndpoint, "oidc-endpoint", "", "OIDC provider endpoint URL")
	cmd.Flags().StringVar(&opts.accountRolePrefix, "account-role-prefix", "", "Prefix of account roles to link with")
	cmd.Flags().StringVar(&opts.permissionsBoundary, "permissions-boundary", "", "ARN of permissions boundary policy")
	cmd.Flags().StringVar(&opts.path, "path", "", "IAM path for roles")
	cmd.Flags().StringToStringVar(&opts.tags, "tags", nil, "Additional tags for roles")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Force creation, replacing existing roles")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Show what would be created without creating")
	cmd.Flags().BoolVarP(&opts.interactive, "interactive", "i", false, "Interactive mode")
	
	return cmd
}

func runCreateOperatorRoles(ctx context.Context, logger *slog.Logger, opts *operatorRolesOptions) error {
	writer := output.NewWriter(output.FormatText)
	
	// Interactive mode
	if opts.interactive {
		if err := promptForOperatorRolesOptions(ctx, opts); err != nil {
			return err
		}
	}
	
	// Validate options
	if opts.region == "" {
		return fmt.Errorf("AWS region is required")
	}
	if opts.clusterID == "" {
		return fmt.Errorf("cluster ID is required")
	}
	if opts.oidcEndpoint == "" {
		return fmt.Errorf("OIDC endpoint is required")
	}
	
	// Create IAM service
	iamService, err := iam.NewService(ctx, opts.region, logger)
	if err != nil {
		return fmt.Errorf("failed to create IAM service: %w", err)
	}
	
	// Get caller identity
	identity, err := iamService.GetCallerIdentity(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AWS identity: %w", err)
	}
	
	writer.Title("Operator Roles Configuration")
	writer.KeyValue(map[string]string{
		"AWS Account":          identity.AccountID,
		"Region":              opts.region,
		"Cluster":             opts.clusterID,
		"OIDC Endpoint":       opts.oidcEndpoint,
		"Role Prefix":         opts.prefix,
		"Permissions Boundary": valueOrDefault(opts.permissionsBoundary, "None"),
		"Path":                valueOrDefault(opts.path, "/"),
	})
	
	if opts.dryRun {
		writer.Info("Dry run mode - no roles will be created")
		fmt.Println()
		writer.Title("Operator roles to be created:")
		operators := []string{
			"ingress-operator",
			"cluster-csi-drivers",
			"cloud-network-config-controller",
			"kube-controller-manager",
			"kms-provider",
			"control-plane-operator",
			"image-registry-operator",
		}
		var roleNames []string
		for _, op := range operators {
			roleName := fmt.Sprintf("%s-%s-%s", opts.prefix, opts.clusterID, op)
			if len(roleName) > 64 {
				roleName = roleName[:64] + "..."
			}
			roleNames = append(roleNames, roleName)
		}
		writer.List(roleNames)
		return nil
	}
	
	// Create the operator roles
	writer.Info("Creating operator roles...")
	
	config := iam.RoleConfig{
		Prefix:              opts.prefix,
		AccountID:           identity.AccountID,
		Region:              opts.region,
		PermissionsBoundary: opts.permissionsBoundary,
		Path:                opts.path,
		Tags:                opts.tags,
	}
	
	roles, err := iamService.CreateOperatorRoles(ctx, config, opts.oidcEndpoint, opts.clusterID)
	if err != nil {
		return fmt.Errorf("failed to create operator roles: %w", err)
	}
	
	// Display results
	writer.Success("Operator roles created successfully!")
	fmt.Println()
	
	writer.Title("Created Operator Roles")
	headers := []string{"OPERATOR", "ROLE NAME", "ROLE ARN"}
	var rows [][]string
	
	for _, role := range roles {
		operatorName := strings.TrimPrefix(role.RoleType, "Operator:")
		rows = append(rows, []string{
			operatorName,
			truncateString(role.RoleName, 30),
			truncateARN(role.RoleARN, 50),
		})
	}
	
	writer.Table(headers, rows)
	
	fmt.Println()
	writer.Info("Next steps:")
	fmt.Println("Your operator roles are ready to be used with the cluster.")
	fmt.Println("The cluster will automatically use these roles for its operators.")
	
	return nil
}

// NewDeleteOperatorRolesCommand creates the delete operator-roles command
func NewDeleteOperatorRolesCommand(logger *slog.Logger) *cobra.Command {
	opts := &operatorRolesOptions{}
	
	cmd := &cobra.Command{
		Use:   "operator-roles",
		Short: "Delete cluster-specific operator IAM roles",
		Long:  `Delete cluster-specific operator IAM roles created for a ROSA HCP cluster.`,
		Example: `  # Delete operator roles for a cluster
  rosa delete operator-roles --cluster my-cluster --prefix ManagedOpenShift

  # Delete with confirmation
  rosa delete operator-roles --cluster my-cluster --prefix ManagedOpenShift --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteOperatorRoles(cmd.Context(), logger, opts)
		},
	}
	
	cmd.Flags().StringVar(&opts.prefix, "prefix", "ManagedOpenShift", "Prefix of operator roles")
	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region")
	cmd.Flags().StringVarP(&opts.clusterID, "cluster", "c", "", "Cluster ID or name")
	cmd.Flags().BoolVarP(&opts.force, "yes", "y", false, "Skip confirmation prompt")
	
	return cmd
}

func runDeleteOperatorRoles(ctx context.Context, logger *slog.Logger, opts *operatorRolesOptions) error {
	writer := output.NewWriter(output.FormatText)
	
	if opts.region == "" {
		return fmt.Errorf("AWS region is required")
	}
	if opts.clusterID == "" {
		return fmt.Errorf("cluster ID is required")
	}
	
	// Create IAM service
	iamService, err := iam.NewService(ctx, opts.region, logger)
	if err != nil {
		return fmt.Errorf("failed to create IAM service: %w", err)
	}
	
	// Show what will be deleted
	writer.Title("Operator roles to be deleted")
	writer.KeyValue(map[string]string{
		"Cluster": opts.clusterID,
		"Prefix":  opts.prefix,
		"Pattern": fmt.Sprintf("%s-%s-*", opts.prefix, opts.clusterID),
	})
	
	// Confirm deletion
	if !opts.force {
		writer.Warning("This action cannot be undone!")
		
		var confirm bool
		err := huh.NewConfirm().
			Title("Delete operator roles?").
			Description(fmt.Sprintf("Delete all operator roles for cluster '%s'?", opts.clusterID)).
			Value(&confirm).
			Run()
		if err != nil {
			return err
		}
		
		if !confirm {
			writer.Info("Deletion cancelled")
			return nil
		}
	}
	
	// Delete the roles
	writer.Info("Deleting operator roles...")
	err = iamService.DeleteOperatorRoles(ctx, opts.prefix, opts.clusterID)
	if err != nil {
		return fmt.Errorf("failed to delete operator roles: %w", err)
	}
	
	writer.Success("Operator roles deleted successfully")
	return nil
}

func promptForOperatorRolesOptions(ctx context.Context, opts *operatorRolesOptions) error {
	// Prompt for region
	if opts.region == "" {
		regions := []string{
			"us-east-1", "us-east-2", "us-west-1", "us-west-2",
			"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1",
			"ap-northeast-1", "ap-northeast-2", "ap-southeast-1", "ap-southeast-2",
		}
		
		var region string
		err := huh.NewSelect[string]().
			Title("Select AWS Region").
			Options(huh.NewOptions(regions...)...).
			Value(&region).
			Run()
		if err != nil {
			return err
		}
		opts.region = region
	}
	
	// Prompt for cluster ID
	if opts.clusterID == "" {
		err := huh.NewInput().
			Title("Cluster ID").
			Description("ID or name of the cluster").
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("cluster ID cannot be empty")
				}
				return nil
			}).
			Value(&opts.clusterID).
			Run()
		if err != nil {
			return err
		}
	}
	
	// Prompt for OIDC endpoint
	if opts.oidcEndpoint == "" {
		err := huh.NewInput().
			Title("OIDC Endpoint").
			Description("OIDC provider endpoint URL (e.g., oidc.example.com)").
			Placeholder("oidc.example.com or https://oidc.example.com").
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("OIDC endpoint cannot be empty")
				}
				return nil
			}).
			Value(&opts.oidcEndpoint).
			Run()
		if err != nil {
			return err
		}
	}
	
	// Prompt for custom prefix
	var customPrefix bool
	err := huh.NewConfirm().
		Title("Use custom role prefix?").
		Description("Default prefix is 'ManagedOpenShift'").
		Value(&customPrefix).
		Run()
	if err != nil {
		return err
	}
	
	if customPrefix {
		err := huh.NewInput().
			Title("Role Prefix").
			Description("Prefix for operator role names").
			Value(&opts.prefix).
			Run()
		if err != nil {
			return err
		}
	}
	
	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
