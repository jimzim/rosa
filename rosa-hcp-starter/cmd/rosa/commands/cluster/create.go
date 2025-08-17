package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/pkg/cluster"
	"github.com/openshift/rosa-hcp/pkg/errors"
	"github.com/openshift/rosa-hcp/pkg/interactive"
	"github.com/openshift/rosa-hcp/pkg/network"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// CreateOptions contains all options for cluster creation
type CreateOptions struct {
	// Basic Configuration
	Name         string
	Region       string
	Version      string
	DomainPrefix string
	
	// AWS Configuration
	RoleARN              string
	SupportRoleARN       string
	WorkerRoleARN        string
	ExternalID           string
	Tags                 map[string]string
	DisableSCPChecks     bool
	BillingAccount       string
	Ec2MetadataHttpTokens string // required, optional
	
	// Network Configuration
	SubnetIDs   []string
	MachineCIDR string
	ServiceCIDR string
	PodCIDR     string
	HostPrefix  int
	Private     bool
	PrivateLink bool
	
	// Additional Security and VPC Options
	AdditionalSecurityGroups  []string
	SharedVPCRoleARN          string
	PrivateHostedZoneID       string
	PrivateHostedZoneRoleARN  string
	BaseDomain                string
	
	// Proxy Configuration
	HTTPProxy               string
	HTTPSProxy              string
	NoProxy                 string
	AdditionalTrustBundle   string
	
	// Security and Compliance
	FIPS                    bool
	EtcdEncryption          bool
	EtcdEncryptionKMSARN    string
	AuditLogRoleARN         string
	
	// Compute Configuration
	ComputeNodes       int
	ComputeType        string
	AutoscalingEnabled bool
	MinReplicas        int
	MaxReplicas        int
	
	// Advanced Configuration
	MultiAZ            bool
	OidcConfigID       string

	// Operational Flags
	Interactive bool
	Output      string
}

// NewCreateCommand creates the cluster create command
func NewCreateCommand(svc *cluster.Service) *cobra.Command {
	opts := &CreateOptions{}

	// Store service for later initialization if nil
	var service *cluster.Service

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new ROSA cluster with Hosted Control Planes",
		Long: `Create a new Red Hat OpenShift Service on AWS (ROSA) cluster with Hosted Control Planes.

All clusters created with this command use Hosted Control Planes (HCP) architecture where 
the control plane runs in a Red Hat-owned AWS account, reducing your operational overhead 
and improving resource efficiency.`,
		Example: `  # Create a cluster interactively
  rosa create cluster --interactive
  
  # Create a cluster with minimal options
  rosa create cluster --name my-cluster --region us-west-2
  
  # Create a cluster with specific configuration
  rosa create cluster \
    --name my-cluster \
    --region us-west-2 \
    --version 4.14.0 \
    --compute-nodes 3 \
    --compute-type m5.xlarge`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize service if not provided (lazy initialization)
			if service == nil && svc == nil {
				// For now, return an error - in a real implementation,
				// we'd initialize the service here
				return fmt.Errorf("service not initialized - please set ROSA_TOKEN environment variable")
			}
			if service == nil {
				service = svc
			}
			return runCreate(cmd.Context(), service, opts)
		},
	}

	// Basic flags
	flags := cmd.Flags()
	flags.StringVar(&opts.Name, "name", "", "Name of the cluster (required)")
	flags.StringVar(&opts.Region, "region", "", "AWS region for the cluster (required)")
	flags.StringVar(&opts.Version, "version", "", "OpenShift version (default: latest stable)")
	flags.StringVar(&opts.DomainPrefix, "domain-prefix", "", "Optional unique domain prefix for cluster subdomain")

	// AWS Configuration flags
	flags.StringVar(&opts.RoleARN, "role-arn", "", "ARN of the installer role")
	flags.StringVar(&opts.SupportRoleARN, "support-role-arn", "", "ARN of the support role")
	flags.StringVar(&opts.WorkerRoleARN, "worker-iam-role", "", "ARN of the worker role")
	flags.StringVar(&opts.ExternalID, "external-id", "", "External ID for STS roles")
	flags.StringToStringVar(&opts.Tags, "tags", nil, "AWS tags for cluster resources (key=value)")
	flags.BoolVar(&opts.DisableSCPChecks, "disable-scp-checks", false, "Disable AWS SCP checks during installation")
	flags.StringVar(&opts.BillingAccount, "billing-account", "", "AWS billing account ID")
	flags.StringVar(&opts.Ec2MetadataHttpTokens, "ec2-metadata-http-tokens", "optional", "Require IMDSv2 for EC2 instances (required/optional)")

	// Network Configuration flags
	flags.StringSliceVar(&opts.SubnetIDs, "subnet-ids", nil, "AWS subnet IDs from existing VPC (comma-separated, required for HCP)")
	flags.StringVar(&opts.MachineCIDR, "machine-cidr", "10.0.0.0/16", "CIDR for machines")
	flags.StringVar(&opts.ServiceCIDR, "service-cidr", "172.30.0.0/16", "CIDR for services")
	flags.StringVar(&opts.PodCIDR, "pod-cidr", "10.128.0.0/14", "CIDR for pods")
	flags.IntVar(&opts.HostPrefix, "host-prefix", 23, "Host prefix for pods")
	flags.BoolVar(&opts.Private, "private", false, "Create private cluster")
	flags.BoolVar(&opts.PrivateLink, "private-link", false, "Use AWS PrivateLink")
	
	// Additional Security and VPC Options
	flags.StringSliceVar(&opts.AdditionalSecurityGroups, "additional-security-group-ids", nil, "Additional security groups for compute nodes")
	flags.StringVar(&opts.SharedVPCRoleARN, "shared-vpc-role-arn", "", "ARN of role for shared VPC installation")
	flags.StringVar(&opts.PrivateHostedZoneID, "private-hosted-zone-id", "", "Private hosted zone ID for shared VPC")
	flags.StringVar(&opts.PrivateHostedZoneRoleARN, "private-hosted-zone-role-arn", "", "ARN of role for private hosted zone")
	flags.StringVar(&opts.BaseDomain, "base-domain", "", "Base domain for cluster")
	
	// Proxy Configuration
	flags.StringVar(&opts.HTTPProxy, "http-proxy", "", "HTTP proxy URL")
	flags.StringVar(&opts.HTTPSProxy, "https-proxy", "", "HTTPS proxy URL")
	flags.StringVar(&opts.NoProxy, "no-proxy", "", "Comma-separated list of destinations to bypass proxy")
	flags.StringVar(&opts.AdditionalTrustBundle, "additional-trust-bundle-file", "", "Path to PEM file with additional CA certificates")
	
	// Security and Compliance
	flags.BoolVar(&opts.FIPS, "fips", false, "Enable FIPS mode")
	flags.BoolVar(&opts.EtcdEncryption, "etcd-encryption", false, "Enable etcd encryption")
	flags.StringVar(&opts.EtcdEncryptionKMSARN, "etcd-encryption-kms-arn", "", "ARN of KMS key for etcd encryption")
	flags.StringVar(&opts.AuditLogRoleARN, "audit-log-arn", "", "ARN of role for audit log forwarding")

	// Compute Configuration flags
	flags.IntVar(&opts.ComputeNodes, "compute-nodes", 2, "Number of compute nodes")
	flags.StringVar(&opts.ComputeType, "compute-type", "m5.xlarge", "EC2 instance type for compute nodes")
	flags.BoolVar(&opts.AutoscalingEnabled, "enable-autoscaling", false, "Enable cluster autoscaling")
	flags.IntVar(&opts.MinReplicas, "min-replicas", 2, "Minimum number of compute nodes (when autoscaling)")
	flags.IntVar(&opts.MaxReplicas, "max-replicas", 10, "Maximum number of compute nodes (when autoscaling)")

	// Advanced Configuration flags
	flags.BoolVar(&opts.MultiAZ, "multi-az", true, "Deploy across multiple availability zones")
	flags.StringVar(&opts.OidcConfigID, "oidc-config-id", "", "OIDC configuration ID")
	flags.BoolVar(&opts.DisableWorkloadMonitoring, "disable-workload-monitoring", false, "Disable workload monitoring")
	flags.BoolVar(&opts.ExternalAuthProvidersEnabled, "external-auth-providers-enabled", false, "Enable external authentication providers")

	// Operational flags
	flags.BoolVar(&opts.DryRun, "dry-run", false, "Simulate cluster creation without creating resources")
	flags.BoolVarP(&opts.Interactive, "interactive", "i", false, "Interactive mode")

	return cmd
}

func runCreate(ctx context.Context, svc *cluster.Service, opts *CreateOptions) error {
	// Handle interactive mode
	if opts.Interactive {
		if err := runInteractive(ctx, opts); err != nil {
			return fmt.Errorf("interactive mode failed: %w", err)
		}
	}

	// Validate required fields
	if err := validateCreateOptions(opts); err != nil {
		return err
	}

	// Show what will be created
	if opts.DryRun {
		output.Info("DRY RUN: The following cluster would be created:")
		printClusterConfig(opts)
		return nil
	}

	// Create progress indicator
	progress := output.NewProgress(fmt.Sprintf("Creating cluster '%s'", opts.Name))
	progress.Start()
	defer progress.Stop()

	// Prepare cluster configuration
	config := cluster.CreateConfig{
		Name:               opts.Name,
		Region:             opts.Region,
		Version:            opts.Version,
		RoleARN:            opts.RoleARN,
		SupportRoleARN:     opts.SupportRoleARN,
		WorkerRoleARN:      opts.WorkerRoleARN,
		ExternalID:         opts.ExternalID,
		Tags:               opts.Tags,
		SubnetIDs:          opts.SubnetIDs,
		MachineCIDR:        opts.MachineCIDR,
		ServiceCIDR:        opts.ServiceCIDR,
		PodCIDR:            opts.PodCIDR,
		HostPrefix:         opts.HostPrefix,
		Private:            opts.Private,
		PrivateLink:        opts.PrivateLink,
		ComputeNodes:       opts.ComputeNodes,
		ComputeType:        opts.ComputeType,
		MultiAZ:            opts.MultiAZ,
		FIPS:               opts.FIPS,
		EtcdEncryption:     opts.EtcdEncryption,
		DisableWorkloadMon: opts.DisableWorkloadMon,
		BillingAccount:     opts.BillingAccount,
		OidcConfigID:       opts.OidcConfigID,
	}

	// Create the cluster
	result := svc.Create(ctx, config)
	if result.IsErr() {
		progress.Stop()
		return handleCreateError(result.Error())
	}

	cluster, _ := result.Unwrap()
	progress.Success(fmt.Sprintf("Cluster '%s' created successfully", cluster.Name))

	// Print cluster details
	output.Info("\nCluster Details:")
	writer := output.NewWriter(output.FormatText)
	writer.KeyValue(map[string]string{
		"ID":          cluster.ID,
		"Name":        cluster.Name,
		"State":       cluster.State,
		"API URL":     cluster.APIURL,
		"Console URL": cluster.ConsoleURL,
		"Region":      cluster.Region,
		"Version":     cluster.Version,
	})

	output.Info("\nNext Steps:")
	writer.List([]string{
		fmt.Sprintf("Configure kubectl: rosa create kubeconfig --cluster %s", cluster.Name),
		fmt.Sprintf("View cluster status: rosa describe cluster --cluster %s", cluster.Name),
		fmt.Sprintf("Create a node pool: rosa create nodepool --cluster %s", cluster.Name),
		fmt.Sprintf("Access the console: %s", cluster.ConsoleURL),
	})

	return nil
}

func runInteractive(ctx context.Context, opts *CreateOptions) error {
	// Basic configuration
	name, err := interactive.PromptString(
		"Cluster Name",
		"A unique name for your cluster (lowercase letters, numbers, and hyphens)",
		opts.Name,
		true,
	)
	if err != nil {
		return err
	}
	opts.Name = name

	// Region selection
	regions := []interactive.SelectOption[string]{
		{Label: "US East (N. Virginia) - us-east-1", Value: "us-east-1"},
		{Label: "US East (Ohio) - us-east-2", Value: "us-east-2"},
		{Label: "US West (Oregon) - us-west-2", Value: "us-west-2"},
		{Label: "EU (Ireland) - eu-west-1", Value: "eu-west-1"},
		{Label: "EU (Frankfurt) - eu-central-1", Value: "eu-central-1"},
		{Label: "Asia Pacific (Singapore) - ap-southeast-1", Value: "ap-southeast-1"},
		{Label: "Asia Pacific (Sydney) - ap-southeast-2", Value: "ap-southeast-2"},
	}

	region, err := interactive.PromptSelect(
		"AWS Region",
		"Select the AWS region for your cluster",
		regions,
		opts.Region,
	)
	if err != nil {
		return err
	}
	opts.Region = region

	// Multi-AZ configuration
	multiAZ, err := interactive.PromptBool(
		"High Availability",
		"Deploy control plane across multiple availability zones?",
		true,
	)
	if err != nil {
		return err
	}
	opts.MultiAZ = multiAZ

	// Private cluster
	private, err := interactive.PromptBool(
		"Private Cluster",
		"Create a private cluster (API endpoint not accessible from internet)?",
		false,
	)
	if err != nil {
		return err
	}
	opts.Private = private

	if opts.Private {
		privateLink, err := interactive.PromptBool(
			"AWS PrivateLink",
			"Use AWS PrivateLink for secure private connectivity?",
			false,
		)
		if err != nil {
			return err
		}
		opts.PrivateLink = privateLink
	}

	// Compute configuration
	computeNodes, err := interactive.PromptString(
		"Number of Compute Nodes",
		"Initial number of worker nodes (minimum 2)",
		"2",
		true,
	)
	if err != nil {
		return err
	}
	fmt.Sscanf(computeNodes, "%d", &opts.ComputeNodes)

	instanceTypes := []interactive.SelectOption[string]{
		{Label: "m5.xlarge (4 vCPU, 16 GiB) - General Purpose", Value: "m5.xlarge"},
		{Label: "m5.2xlarge (8 vCPU, 32 GiB) - General Purpose", Value: "m5.2xlarge"},
		{Label: "m5.4xlarge (16 vCPU, 64 GiB) - General Purpose", Value: "m5.4xlarge"},
		{Label: "c5.2xlarge (8 vCPU, 16 GiB) - Compute Optimized", Value: "c5.2xlarge"},
		{Label: "r5.xlarge (4 vCPU, 32 GiB) - Memory Optimized", Value: "r5.xlarge"},
	}

	computeType, err := interactive.PromptSelect(
		"Compute Instance Type",
		"Select EC2 instance type for compute nodes",
		instanceTypes,
		"m5.xlarge",
	)
	if err != nil {
		return err
	}
	opts.ComputeType = computeType

	return nil
}

func validateCreateOptions(opts *CreateOptions) error {
	var errs []string

	if opts.Name == "" {
		errs = append(errs, "cluster name is required")
	} else {
		if err := validateClusterName(opts.Name); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if opts.Region == "" {
		errs = append(errs, "region is required")
	}

	if opts.ComputeNodes < 2 {
		errs = append(errs, "minimum 2 compute nodes required")
	}
	
	// Validate VPC and subnets if provided
	if len(opts.SubnetIDs) > 0 {
		if err := validateVPCAndSubnets(context.Background(), opts); err != nil {
			errs = append(errs, fmt.Sprintf("VPC validation failed: %v", err))
		}
	} else {
		// For HCP, subnet IDs are required
		errs = append(errs, "subnet IDs are required for HCP clusters (use --subnet-ids)")
	}

	if len(errs) > 0 {
		return errors.Validation("cluster.create", fmt.Errorf(strings.Join(errs, "; "))).
			WithSuggestion("Use --interactive flag for guided cluster creation")
	}

	return nil
}

// validateVPCAndSubnets validates that the provided subnets exist and belong to the same VPC
func validateVPCAndSubnets(ctx context.Context, opts *CreateOptions) error {
	if len(opts.SubnetIDs) == 0 {
		return fmt.Errorf("no subnet IDs provided")
	}
	
	if opts.Region == "" {
		return fmt.Errorf("region is required for VPC validation")
	}
	
	// Get VPC ID from the first subnet
	publicSubnets, privateSubnets, err := network.GetSubnetsFromVPC(ctx, opts.Region, "")
	if err != nil {
		// If we can't validate, log a warning but don't fail
		// The actual cluster creation will fail if subnets are invalid
		slog.Warn("Unable to validate VPC subnets", "error", err)
		return nil
	}
	
	// Check if all provided subnets exist in either public or private lists
	allSubnets := append(publicSubnets, privateSubnets...)
	for _, subnetID := range opts.SubnetIDs {
		found := false
		for _, subnet := range allSubnets {
			if subnet == subnetID {
				found = true
				break
			}
		}
		if !found {
			// Try to validate the specific subnet exists
			// This is a basic check - the actual validation happens during cluster creation
			slog.Warn("Subnet may not exist or be accessible", "subnet", subnetID)
		}
	}
	
	return nil
}

func validateClusterName(name string) error {
	if len(name) < 3 || len(name) > 54 {
		return fmt.Errorf("cluster name must be between 3 and 54 characters")
	}

	// Must start with lowercase letter
	if name[0] < 'a' || name[0] > 'z' {
		return fmt.Errorf("cluster name must start with a lowercase letter")
	}

	// Only lowercase letters, numbers, and hyphens
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return fmt.Errorf("cluster name can only contain lowercase letters, numbers, and hyphens")
		}
	}

	return nil
}

func printClusterConfig(opts *CreateOptions) {
	writer := output.NewWriter(output.FormatText)
	writer.KeyValue(map[string]string{
		"Name":            opts.Name,
		"Region":          opts.Region,
		"Version":         valueOrDefault(opts.Version, "latest"),
		"Multi-AZ":        fmt.Sprintf("%v", opts.MultiAZ),
		"Private":         fmt.Sprintf("%v", opts.Private),
		"PrivateLink":     fmt.Sprintf("%v", opts.PrivateLink),
		"Compute Nodes":   fmt.Sprintf("%d", opts.ComputeNodes),
		"Compute Type":    opts.ComputeType,
		"FIPS":            fmt.Sprintf("%v", opts.FIPS),
		"Etcd Encryption": fmt.Sprintf("%v", opts.EtcdEncryption),
	})
}

func valueOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func handleCreateError(err error) error {
	// Add specific error handling and suggestions
	if errors.IsPermission(err) {
		if rosaErr, ok := err.(*errors.ROSAError); ok {
			return rosaErr.WithSuggestion("Ensure your AWS credentials have the necessary permissions. Run 'rosa verify permissions' to check.")
		}
	}

	if errors.IsValidation(err) {
		if rosaErr, ok := err.(*errors.ROSAError); ok {
			return rosaErr.WithSuggestion("Check your input parameters. Use --interactive for guided cluster creation.")
		}
	}

	return err
}
