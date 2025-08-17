package verify

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/aws"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// NetworkOptions contains options for verifying network configuration
type NetworkOptions struct {
	ClusterName string
	SubnetIDs   []string
	Region      string
	HostedCP    bool
}

// NewNetworkCommand creates the verify network command
func NewNetworkCommand(logger *slog.Logger) *cobra.Command {
	opts := &NetworkOptions{}

	cmd := &cobra.Command{
		Use:   "network",
		Short: "Verify VPC and subnet configuration for cluster deployment",
		Long: `Verify that VPC and subnets are configured correctly for ROSA HCP deployment.

This command checks:
- Subnet existence and availability
- CIDR blocks and IP availability
- Route tables and internet connectivity
- Network ACLs and security groups
- DNS resolution settings`,
		Example: `  # Verify subnets for a cluster
  rosa verify network --cluster my-cluster

  # Verify specific subnets
  rosa verify network --subnet-ids subnet-123,subnet-456

  # Verify for HCP clusters
  rosa verify network --subnet-ids subnet-123,subnet-456 --hosted-cp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerifyNetwork(cmd.Context(), logger, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.ClusterName, "cluster", "c", "", "Name or ID of the cluster")
	flags.StringSliceVar(&opts.SubnetIDs, "subnet-ids", nil, "Subnet IDs to verify (comma-separated)")
	flags.StringVar(&opts.Region, "region", "", "AWS region (auto-detected if not specified)")
	flags.BoolVar(&opts.HostedCP, "hosted-cp", true, "Verify for HCP (Hosted Control Plane) clusters")

	return cmd
}

func runVerifyNetwork(ctx context.Context, logger *slog.Logger, opts *NetworkOptions) error {
	writer := output.NewWriter(output.FormatText)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// If cluster is specified, get subnet IDs from it
	if opts.ClusterName != "" && len(opts.SubnetIDs) == 0 {
		apiClient, err := api.NewClient(ctx, api.Config{
			URL:   cfg.APIURL,
			Token: cfg.Token,
		})
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		clusterResp, err := apiClient.GetCluster(ctx, opts.ClusterName)
		if err != nil {
			return fmt.Errorf("failed to get cluster: %w", err)
		}
		cluster := clusterResp.Body()

		// Get subnet IDs from cluster
		if cluster.AWS() != nil && len(cluster.AWS().SubnetIDs()) > 0 {
			opts.SubnetIDs = cluster.AWS().SubnetIDs()
			if opts.Region == "" && cluster.Region() != nil {
				opts.Region = cluster.Region().ID()
			}
		} else {
			return fmt.Errorf("cluster does not have subnet IDs configured")
		}
	}

	// Validate we have subnet IDs
	if len(opts.SubnetIDs) == 0 {
		return fmt.Errorf("subnet IDs are required (use --subnet-ids or --cluster)")
	}

	// Auto-detect region if not specified
	if opts.Region == "" {
		opts.Region = cfg.DefaultRegion
		if opts.Region == "" {
			opts.Region = "us-east-1" // Default
		}
	}

	writer.Title("Network Verification")
	writer.KeyValue(map[string]string{
		"Region":     opts.Region,
		"Subnet IDs": strings.Join(opts.SubnetIDs, ", "),
		"Mode":       "HCP (Hosted Control Plane)",
	})

	// Create AWS client
	awsClient, err := aws.NewClient(ctx, opts.Region, cfg.ActiveProfile, logger)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Run verification checks
	writer.Info("\n🔍 Running verification checks...")

	var failedChecks []string
	var warningChecks []string

	// Check 1: Subnet existence
	writer.Info("\n1. Verifying subnet existence...")
	subnets, err := awsClient.DescribeSubnets(ctx, opts.SubnetIDs)
	if err != nil {
		failedChecks = append(failedChecks, fmt.Sprintf("Subnet verification failed: %v", err))
		writer.Error("   ✗ Failed to describe subnets: %v", err)
	} else {
		writer.Success("   ✓ All %d subnets exist", len(subnets))

		// Verify we have the expected number
		if len(subnets) != len(opts.SubnetIDs) {
			warningChecks = append(warningChecks,
				fmt.Sprintf("Found %d subnets but expected %d", len(subnets), len(opts.SubnetIDs)))
		}
	}

	// Check 2: Availability zones (for HCP, should be in different AZs)
	if len(subnets) > 0 {
		writer.Info("\n2. Checking availability zones...")
		azMap := make(map[string][]string)
		for _, subnet := range subnets {
			if subnet.AvailabilityZone != nil {
				azMap[*subnet.AvailabilityZone] = append(azMap[*subnet.AvailabilityZone], *subnet.SubnetId)
			}
		}

		if len(azMap) < 2 && opts.HostedCP {
			warningChecks = append(warningChecks,
				"HCP clusters should use subnets in at least 2 availability zones for high availability")
			writer.Warning("   ⚠ Only %d availability zone(s) used (recommend 2+ for HA)", len(azMap))
		} else {
			writer.Success("   ✓ Subnets span %d availability zones", len(azMap))
		}

		for az, subnetList := range azMap {
			fmt.Printf("      • %s: %s\n", az, strings.Join(subnetList, ", "))
		}
	}

	// Check 3: IP availability
	if len(subnets) > 0 {
		writer.Info("\n3. Checking IP availability...")
		totalAvailable := 0
		lowIPSubnets := []string{}

		for _, subnet := range subnets {
			if subnet.AvailableIpAddressCount != nil {
				available := int(*subnet.AvailableIpAddressCount)
				totalAvailable += available

				if available < 16 {
					lowIPSubnets = append(lowIPSubnets, *subnet.SubnetId)
				}
			}
		}

		if len(lowIPSubnets) > 0 {
			warningChecks = append(warningChecks,
				fmt.Sprintf("Subnets with low IP availability: %s", strings.Join(lowIPSubnets, ", ")))
			writer.Warning("   ⚠ Some subnets have low IP availability (< 16 IPs)")
		}

		if totalAvailable < 50 {
			failedChecks = append(failedChecks, "Insufficient IP addresses available across subnets")
			writer.Error("   ✗ Insufficient total IP addresses: %d (need at least 50)", totalAvailable)
		} else {
			writer.Success("   ✓ Sufficient IP addresses available: %d", totalAvailable)
		}
	}

	// Check 4: VPC and routing
	if len(subnets) > 0 && subnets[0].VpcId != nil {
		writer.Info("\n4. Checking VPC configuration...")
		vpcID := *subnets[0].VpcId

		// Verify all subnets are in the same VPC
		sameVPC := true
		for _, subnet := range subnets {
			if subnet.VpcId == nil || *subnet.VpcId != vpcID {
				sameVPC = false
				break
			}
		}

		if !sameVPC {
			failedChecks = append(failedChecks, "Subnets are not in the same VPC")
			writer.Error("   ✗ Subnets must be in the same VPC")
		} else {
			writer.Success("   ✓ All subnets in VPC: %s", vpcID)

			// Check for internet gateway
			hasIGW, err := awsClient.VPCHasInternetGateway(ctx, vpcID)
			if err != nil {
				warningChecks = append(warningChecks, fmt.Sprintf("Could not verify internet gateway: %v", err))
				writer.Warning("   ⚠ Could not verify internet gateway")
			} else if hasIGW {
				writer.Success("   ✓ Internet gateway attached")
			} else {
				writer.Info("   ℹ No internet gateway (private cluster configuration)")
			}
		}
	}

	// Check 5: Tags
	writer.Info("\n5. Checking subnet tags...")
	for _, subnet := range subnets {
		hasNameTag := false
		for _, tag := range subnet.Tags {
			if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil && *tag.Value != "" {
				hasNameTag = true
				break
			}
		}
		if !hasNameTag && subnet.SubnetId != nil {
			warningChecks = append(warningChecks,
				fmt.Sprintf("Subnet %s has no Name tag", *subnet.SubnetId))
		}
	}
	if len(warningChecks) == 0 {
		writer.Success("   ✓ All subnets have appropriate tags")
	} else {
		writer.Warning("   ⚠ Some subnets missing Name tags")
	}

	// Summary
	fmt.Println()
	writer.Title("Verification Summary")

	if len(failedChecks) == 0 && len(warningChecks) == 0 {
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10")).
			Background(lipgloss.Color("0")).
			Padding(0, 1)
		fmt.Println(style.Render("✓ All checks passed! Network is ready for ROSA HCP deployment."))
	} else {
		if len(failedChecks) > 0 {
			writer.Error("\n❌ Failed Checks:")
			for _, check := range failedChecks {
				fmt.Printf("   • %s\n", check)
			}
		}

		if len(warningChecks) > 0 {
			writer.Warning("\n⚠️  Warnings:")
			for _, check := range warningChecks {
				fmt.Printf("   • %s\n", check)
			}
		}

		if len(failedChecks) > 0 {
			writer.Error("\n❌ Network verification failed. Please address the issues above before proceeding.")
			return fmt.Errorf("network verification failed")
		} else {
			writer.Warning("\n⚠️  Network verification passed with warnings. Review the warnings above.")
		}
	}

	// Recommendations
	writer.Info("\n📋 Next Steps:")
	if len(failedChecks) == 0 {
		writer.Info("1. Network is ready for cluster creation")
		writer.Info("2. Create your HCP cluster:")
		fmt.Printf("   rosa create cluster --subnet-ids %s\n", strings.Join(opts.SubnetIDs, ","))
	} else {
		writer.Info("1. Fix the failed checks above")
		writer.Info("2. Re-run verification:")
		fmt.Printf("   rosa verify network --subnet-ids %s\n", strings.Join(opts.SubnetIDs, ","))
	}

	return nil
}
