package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/cluster"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// DescribeOptions contains options for describing a cluster
type DescribeOptions struct {
	ClusterName string
	ShowRoles   bool
}

// NewDescribeCommand creates the cluster describe command
func NewDescribeCommand(logger *slog.Logger) *cobra.Command {
	opts := &DescribeOptions{}

	cmd := &cobra.Command{
		Use:     "cluster",
		Short:   "Show details of a ROSA HCP cluster",
		Long:    "Display detailed information about a ROSA cluster with Hosted Control Planes.",
		Example: `  # Describe a cluster
  rosa describe cluster --cluster my-cluster

  # Show cluster with IAM role details
  rosa describe cluster --cluster my-cluster --get-role-policy-bindings`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDescribeCluster(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster (required)")
	flags.BoolVar(&opts.ShowRoles, "get-role-policy-bindings", false, "Show attached policies for STS roles")

	cmd.MarkFlagRequired("cluster")

	return cmd
}

func runDescribeCluster(ctx context.Context, logger *slog.Logger, opts *DescribeOptions) error {
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

	// Create cluster service
	clusterSvc, err := cluster.NewService(ctx, logger, apiClient.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create cluster service: %w", err)
	}

	// Get the cluster
	clusterInfo, err := clusterSvc.Get(ctx, opts.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Display cluster header
	writer.Title(fmt.Sprintf("Cluster: %s", clusterInfo.Name))
	
	// Basic Information
	writer.Info("📋 Basic Information")
	basicInfo := map[string]string{
		"ID":             clusterInfo.ID,
		"Name":           clusterInfo.Name,
		"Domain":         clusterInfo.DomainPrefix + ".openshiftapps.com",
		"Status":         clusterInfo.State,
		"Region":         clusterInfo.Region,
		"Multi-AZ":       fmt.Sprintf("%v", clusterInfo.MultiAZ),
		"OpenShift":      clusterInfo.Version,
		"HCP":            "Enabled",
	}
	writer.KeyValue(basicInfo)

	// Console and API URLs
	writer.Info("\n🌐 Access URLs")
	accessInfo := map[string]string{
		"Console URL":  clusterInfo.ConsoleURL,
		"API URL":      clusterInfo.APIURL,
	}
	writer.KeyValue(accessInfo)

	// Network Configuration
	writer.Info("\n🔌 Network Configuration")
	networkInfo := map[string]string{
		"Type":           clusterInfo.NetworkType,
		"Private":        fmt.Sprintf("%v", clusterInfo.Private),
		"Service CIDR":   clusterInfo.ServiceCIDR,
		"Pod CIDR":       clusterInfo.PodCIDR,
		"Machine CIDR":   clusterInfo.MachineCIDR,
	}
	
	if len(clusterInfo.SubnetIDs) > 0 {
		networkInfo["Subnets"] = strings.Join(clusterInfo.SubnetIDs, ", ")
	}
	
	writer.KeyValue(networkInfo)

	// Compute Configuration
	writer.Info("\n💻 Compute Configuration")
	computeInfo := map[string]string{
		"Compute Nodes":  fmt.Sprintf("%d", clusterInfo.ComputeNodes),
		"Instance Type":  clusterInfo.ComputeMachineType,
	}
	
	if clusterInfo.Autoscaling {
		computeInfo["Autoscaling"] = "Enabled"
		computeInfo["Min Replicas"] = fmt.Sprintf("%d", clusterInfo.MinReplicas)
		computeInfo["Max Replicas"] = fmt.Sprintf("%d", clusterInfo.MaxReplicas)
	} else {
		computeInfo["Autoscaling"] = "Disabled"
	}
	
	writer.KeyValue(computeInfo)

	// AWS Configuration (if STS)
	if clusterInfo.AWS != nil {
		writer.Info("\n☁️ AWS Configuration")
		awsInfo := map[string]string{
			"Account ID":        clusterInfo.AWS.AccountID,
			"STS":               "Enabled",
			"Installer Role":    clusterInfo.AWS.InstallerRoleARN,
			"Support Role":      clusterInfo.AWS.SupportRoleARN,
			"Worker Role":       clusterInfo.AWS.WorkerRoleARN,
		}
		
		if clusterInfo.AWS.AuditLogRoleARN != "" {
			awsInfo["Audit Log Role"] = clusterInfo.AWS.AuditLogRoleARN
		}
		
		writer.KeyValue(awsInfo)
		
		// Show OIDC endpoint
		if clusterInfo.AWS.OIDCEndpointURL != "" {
			writer.Info("\n🔐 OIDC Configuration")
			writer.KeyValue(map[string]string{
				"OIDC Endpoint": clusterInfo.AWS.OIDCEndpointURL,
			})
		}
	}

	// Proxy Configuration
	if clusterInfo.ProxyURL != "" {
		writer.Info("\n🔒 Proxy Configuration")
		proxyInfo := map[string]string{
			"HTTP Proxy": clusterInfo.ProxyURL,
		}
		if clusterInfo.NoProxy != "" {
			proxyInfo["No Proxy"] = clusterInfo.NoProxy
		}
		writer.KeyValue(proxyInfo)
	}

	// Tags
	if len(clusterInfo.Tags) > 0 {
		writer.Info("\n🏷️ Tags")
		writer.KeyValue(clusterInfo.Tags)
	}

	// Status Messages
	switch clusterInfo.State {
	case "ready":
		writer.Success("\n✓ Cluster is ready and operational")
		
		// Show login command
		writer.Info("\n📋 Login Command:")
		fmt.Printf("  oc login %s --username <user> --password <password>\n", clusterInfo.APIURL)
		
	case "installing":
		writer.Info("\n⏳ Cluster is being installed...")
		writer.Info("Check installation progress with:")
		fmt.Printf("  rosa logs install --cluster %s --watch\n", clusterInfo.Name)
		
	case "error":
		writer.Error("\n✗ Cluster is in error state")
		if clusterInfo.StateDescription != "" {
			writer.Error("Error: %s", clusterInfo.StateDescription)
		}
		
	case "uninstalling":
		writer.Warning("\n⚠ Cluster is being uninstalled")
		writer.Info("Check uninstallation progress with:")
		fmt.Printf("  rosa logs uninstall --cluster %s --watch\n", clusterInfo.Name)
	}

	// Show available actions
	writer.Info("\n📋 Available Actions:")
	fmt.Println("  # Edit cluster configuration:")
	fmt.Printf("  rosa edit cluster --cluster %s\n", clusterInfo.Name)
	fmt.Println("\n  # Upgrade cluster:")
	fmt.Printf("  rosa upgrade cluster --cluster %s\n", clusterInfo.Name)
	fmt.Println("\n  # Create admin user:")
	fmt.Printf("  rosa create admin --cluster %s\n", clusterInfo.Name)
	fmt.Println("\n  # Manage NodePools:")
	fmt.Printf("  rosa list nodepools --cluster %s\n", clusterInfo.Name)

	// Show role policy bindings if requested
	if opts.ShowRoles && clusterInfo.AWS != nil {
		writer.Info("\n📋 IAM Role Policy Bindings:")
		fmt.Println("  (This would show detailed IAM role policies - not implemented in this version)")
	}

	return nil
}
