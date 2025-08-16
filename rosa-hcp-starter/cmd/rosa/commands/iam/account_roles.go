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

type accountRolesOptions struct {
	prefix              string
	region              string
	permissionsBoundary string
	path                string
	tags                map[string]string
	force               bool
	dryRun              bool
	interactive         bool
}

// NewCreateAccountRolesCommand creates the account-roles command
func NewCreateAccountRolesCommand(logger *slog.Logger) *cobra.Command {
	opts := &accountRolesOptions{}
	
	cmd := &cobra.Command{
		Use:     "account-roles",
		Aliases: []string{"accountroles"},
		Short:   "Create account-wide IAM roles for ROSA HCP",
		Long: `Create account-wide IAM roles required for ROSA HCP clusters.

This command creates three IAM roles:
  - Installer: Used by Red Hat to create and manage cluster resources
  - Support: Used by Red Hat support to troubleshoot issues
  - Worker: Used by EC2 instances running as worker nodes

These roles are required before creating your first ROSA HCP cluster.`,
		Example: `  # Create default account roles
  rosa create account-roles

  # Create account roles with custom prefix
  rosa create account-roles --prefix MyOrg-HCP

  # Create account roles with permissions boundary
  rosa create account-roles --permissions-boundary arn:aws:iam::123456789012:policy/boundary

  # Interactive mode
  rosa create account-roles --interactive`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateAccountRoles(cmd.Context(), logger, opts)
		},
	}
	
	cmd.Flags().StringVar(&opts.prefix, "prefix", iam.DefaultPrefix, "Prefix for role names")
	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region")
	cmd.Flags().StringVar(&opts.permissionsBoundary, "permissions-boundary", "", "ARN of permissions boundary policy")
	cmd.Flags().StringVar(&opts.path, "path", "", "IAM path for roles (e.g., /rosa-hcp/)")
	cmd.Flags().StringToStringVar(&opts.tags, "tags", nil, "Additional tags for roles (key=value)")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Force creation, replacing existing roles")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Show what would be created without creating")
	cmd.Flags().BoolVarP(&opts.interactive, "interactive", "i", false, "Interactive mode")
	
	return cmd
}

func runCreateAccountRoles(ctx context.Context, logger *slog.Logger, opts *accountRolesOptions) error {
	writer := output.NewWriter(output.FormatText)
	
	// Interactive mode
	if opts.interactive {
		if err := promptForAccountRolesOptions(ctx, opts); err != nil {
			return err
		}
	}
	
	// Validate options
	if opts.region == "" {
		return fmt.Errorf("AWS region is required")
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
	
	writer.Title("Account Roles Configuration")
	writer.KeyValue(map[string]string{
		"AWS Account":          identity.AccountID,
		"Region":              opts.region,
		"Role Prefix":         opts.prefix,
		"Permissions Boundary": valueOrDefault(opts.permissionsBoundary, "None"),
		"Path":                valueOrDefault(opts.path, "/"),
	})
	
	if opts.dryRun {
		writer.Info("Dry run mode - no roles will be created")
		fmt.Println()
		writer.Title("Roles to be created:")
		roleNames := []string{
			fmt.Sprintf("%s-Installer", opts.prefix),
			fmt.Sprintf("%s-Support", opts.prefix),
			fmt.Sprintf("%s-Worker", opts.prefix),
		}
		writer.List(roleNames)
		return nil
	}
	
	// Check if roles already exist
	existingRoles, _ := iamService.ListAccountRoles(ctx, opts.prefix)
	if len(existingRoles) > 0 && !opts.force {
		writer.Warning("Account roles with prefix '%s' already exist", opts.prefix)
		
		var overwrite bool
		err := huh.NewConfirm().
			Title("Overwrite existing roles?").
			Description("This will update the existing roles with new policies").
			Value(&overwrite).
			Run()
		if err != nil {
			return err
		}
		
		if !overwrite {
			writer.Info("Creation cancelled")
			return nil
		}
	}
	
	// Create the roles
	writer.Info("Creating account roles...")
	
	config := iam.RoleConfig{
		Prefix:              opts.prefix,
		AccountID:           identity.AccountID,
		Region:              opts.region,
		PermissionsBoundary: opts.permissionsBoundary,
		Path:                opts.path,
		Tags:                opts.tags,
	}
	
	roles, err := iamService.CreateAccountRoles(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create account roles: %w", err)
	}
	
	// Display results
	writer.Success("Account roles created successfully!")
	fmt.Println()
	
	writer.Title("Created Roles")
	headers := []string{"ROLE TYPE", "ROLE NAME", "ROLE ARN"}
	var rows [][]string
	
	for _, role := range roles {
		rows = append(rows, []string{
			role.RoleType,
			role.RoleName,
			role.RoleARN,
		})
	}
	
	writer.Table(headers, rows)
	
	fmt.Println()
	writer.Info("Next steps:")
	fmt.Println("1. Create OIDC configuration:")
	fmt.Printf("   rosa create oidc-config\n")
	fmt.Println("2. Create your first cluster:")
	fmt.Printf("   rosa cluster create --name mycluster --sts --role-arn %s\n", roles[0].RoleARN)
	
	return nil
}

// NewDeleteAccountRolesCommand creates the delete account-roles command
func NewDeleteAccountRolesCommand(logger *slog.Logger) *cobra.Command {
	opts := &accountRolesOptions{}
	
	cmd := &cobra.Command{
		Use:   "account-roles",
		Short: "Delete account-wide IAM roles",
		Long:  `Delete account-wide IAM roles created for ROSA HCP clusters.`,
		Example: `  # Delete account roles with default prefix
  rosa delete account-roles

  # Delete account roles with custom prefix
  rosa delete account-roles --prefix MyOrg-HCP`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteAccountRoles(cmd.Context(), logger, opts)
		},
	}
	
	cmd.Flags().StringVar(&opts.prefix, "prefix", iam.DefaultPrefix, "Prefix of roles to delete")
	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region")
	cmd.Flags().BoolVarP(&opts.force, "yes", "y", false, "Skip confirmation prompt")
	
	return cmd
}

func runDeleteAccountRoles(ctx context.Context, logger *slog.Logger, opts *accountRolesOptions) error {
	writer := output.NewWriter(output.FormatText)
	
	if opts.region == "" {
		return fmt.Errorf("AWS region is required")
	}
	
	// Create IAM service
	iamService, err := iam.NewService(ctx, opts.region, logger)
	if err != nil {
		return fmt.Errorf("failed to create IAM service: %w", err)
	}
	
	// List roles to be deleted
	roles, err := iamService.ListAccountRoles(ctx, opts.prefix)
	if err != nil {
		return fmt.Errorf("failed to list account roles: %w", err)
	}
	
	if len(roles) == 0 {
		writer.Info("No account roles found with prefix '%s'", opts.prefix)
		return nil
	}
	
	// Show what will be deleted
	writer.Title("Account roles to be deleted")
	var roleNames []string
	for _, role := range roles {
		roleNames = append(roleNames, role.RoleName)
	}
	writer.List(roleNames)
	
	// Confirm deletion
	if !opts.force {
		writer.Warning("This action cannot be undone!")
		
		var confirm bool
		err := huh.NewConfirm().
			Title("Delete account roles?").
			Description(fmt.Sprintf("Delete %d account roles with prefix '%s'?", len(roles), opts.prefix)).
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
	writer.Info("Deleting account roles...")
	err = iamService.DeleteAccountRoles(ctx, opts.prefix)
	if err != nil {
		return fmt.Errorf("failed to delete account roles: %w", err)
	}
	
	writer.Success("Account roles deleted successfully")
	return nil
}

// NewListAccountRolesCommand creates the list account-roles command  
func NewListAccountRolesCommand(logger *slog.Logger) *cobra.Command {
	opts := &accountRolesOptions{}
	
	cmd := &cobra.Command{
		Use:     "account-roles",
		Aliases: []string{"accountroles"},
		Short:   "List account-wide IAM roles",
		Long:    `List account-wide IAM roles created for ROSA HCP clusters.`,
		Example: `  # List account roles with default prefix
  rosa list account-roles

  # List account roles with custom prefix
  rosa list account-roles --prefix MyOrg-HCP`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListAccountRoles(cmd.Context(), logger, opts)
		},
	}
	
	cmd.Flags().StringVar(&opts.prefix, "prefix", iam.DefaultPrefix, "Prefix of roles to list")
	cmd.Flags().StringVar(&opts.region, "region", "", "AWS region")
	
	return cmd
}

func runListAccountRoles(ctx context.Context, logger *slog.Logger, opts *accountRolesOptions) error {
	writer := output.NewWriter(output.FormatText)
	
	if opts.region == "" {
		return fmt.Errorf("AWS region is required")
	}
	
	// Create IAM service
	iamService, err := iam.NewService(ctx, opts.region, logger)
	if err != nil {
		return fmt.Errorf("failed to create IAM service: %w", err)
	}
	
	// List roles
	writer.Info("Fetching account roles...")
	roles, err := iamService.ListAccountRoles(ctx, opts.prefix)
	if err != nil {
		return fmt.Errorf("failed to list account roles: %w", err)
	}
	
	if len(roles) == 0 {
		writer.Info("No account roles found with prefix '%s'", opts.prefix)
		return nil
	}
	
	// Display roles
	writer.Title(fmt.Sprintf("Account Roles (prefix: %s)", opts.prefix))
	fmt.Println()
	
	headers := []string{"ROLE TYPE", "ROLE NAME", "ROLE ARN", "POLICIES"}
	var rows [][]string
	
	for _, role := range roles {
		policyCount := fmt.Sprintf("%d attached", len(role.Policies))
		rows = append(rows, []string{
			role.RoleType,
			role.RoleName,
			truncateARN(role.RoleARN, 50),
			policyCount,
		})
	}
	
	writer.Table(headers, rows)
	
	// Validate roles
	fmt.Println()
	writer.Info("Validating roles...")
	err = iamService.ValidateAccountRoles(ctx, opts.prefix)
	if err != nil {
		writer.Error("Validation failed: %s", err)
		return nil
	}
	
	writer.Success("All required account roles are present and valid")
	return nil
}

func promptForAccountRolesOptions(ctx context.Context, opts *accountRolesOptions) error {
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
	
	// Prompt for prefix
	var customPrefix bool
	err := huh.NewConfirm().
		Title("Use custom role prefix?").
		Description(fmt.Sprintf("Default prefix is '%s'", iam.DefaultPrefix)).
		Value(&customPrefix).
		Run()
	if err != nil {
		return err
	}
	
	if customPrefix {
		err := huh.NewInput().
			Title("Role Prefix").
			Description("Prefix for all role names").
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("prefix cannot be empty")
				}
				if len(s) > 32 {
					return fmt.Errorf("prefix too long (max 32 characters)")
				}
				return nil
			}).
			Value(&opts.prefix).
			Run()
		if err != nil {
			return err
		}
	}
	
	// Prompt for permissions boundary
	var usePermissionsBoundary bool
	err = huh.NewConfirm().
		Title("Use permissions boundary?").
		Description("Restrict maximum permissions for the roles").
		Value(&usePermissionsBoundary).
		Run()
	if err != nil {
		return err
	}
	
	if usePermissionsBoundary {
		err := huh.NewInput().
			Title("Permissions Boundary ARN").
			Description("ARN of the permissions boundary policy").
			Placeholder("arn:aws:iam::123456789012:policy/boundary").
			Value(&opts.permissionsBoundary).
			Run()
		if err != nil {
			return err
		}
	}
	
	return nil
}

func valueOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func truncateARN(arn string, maxLen int) string {
	if len(arn) <= maxLen {
		return arn
	}
	// Show beginning and end of ARN
	parts := strings.Split(arn, ":")
	if len(parts) >= 6 {
		// Show account and role name
		account := parts[4]
		rolePart := parts[5]
		if strings.Contains(rolePart, "/") {
			roleNameParts := strings.Split(rolePart, "/")
			roleName := roleNameParts[len(roleNameParts)-1]
			return fmt.Sprintf("...%s:role/%s", account, roleName)
		}
	}
	return arn[:maxLen-3] + "..."
}
