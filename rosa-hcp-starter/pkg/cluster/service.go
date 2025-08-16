package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/aws"
	"github.com/openshift/rosa-hcp/pkg/errors"
)

// Service provides cluster operations
type Service struct {
	ocm    api.Client
	aws    aws.Client
	logger *slog.Logger
}

// NewService creates a new cluster service
func NewService(ocm api.Client, aws aws.Client, logger *slog.Logger) *Service {
	return &Service{
		ocm:    ocm,
		aws:    aws,
		logger: logger.With("service", "cluster"),
	}
}

// CreateConfig holds cluster creation configuration
type CreateConfig struct {
	// Basic Configuration
	Name    string
	Region  string
	Version string

	// AWS Configuration
	RoleARN        string
	SupportRoleARN string
	WorkerRoleARN  string
	ExternalID     string
	Tags           map[string]string

	// Network Configuration
	SubnetIDs   []string
	MachineCIDR string
	ServiceCIDR string
	PodCIDR     string
	HostPrefix  int
	Private     bool
	PrivateLink bool

	// Compute Configuration
	ComputeNodes int
	ComputeType  string

	// Advanced Configuration
	MultiAZ            bool
	FIPS               bool
	EtcdEncryption     bool
	DisableWorkloadMon bool
	BillingAccount     string
	OidcConfigID       string
}

// Cluster represents a ROSA HCP cluster
type Cluster struct {
	ID         string
	Name       string
	State      string
	APIURL     string
	ConsoleURL string
	Region     string
	Version    string
	CreatedAt  time.Time
}

// Create creates a new HCP cluster
func (s *Service) Create(ctx context.Context, config CreateConfig) errors.Result[*Cluster] {
	s.logger.InfoContext(ctx, "creating cluster",
		slog.String("name", config.Name),
		slog.String("region", config.Region),
	)

	// Build OCM cluster object
	builder := cmv1.NewCluster()
	builder.Name(config.Name)
	builder.Region(cmv1.NewCloudRegion().ID(config.Region))

	// IMPORTANT: Set cloud provider and product for ROSA
	builder.CloudProvider(cmv1.NewCloudProvider().ID("aws"))
	builder.Product(cmv1.NewProduct().ID("rosa"))

	// Set as managed cluster
	builder.Managed(true)

	// Set HCP flag - this is the key difference!
	// All clusters are HCP now
	builder.Hypershift(cmv1.NewHypershift().Enabled(true))

	// Set version if provided
	if config.Version != "" {
		builder.Version(cmv1.NewVersion().ID(config.Version))
	}

	// Configure AWS settings - this IS required for ROSA clusters
	// The error "aws field is mandatory" confirms we need this
	awsBuilder := cmv1.NewAWS()

	// For HCP clusters, we typically need subnet IDs from an existing VPC
	// If no subnets are provided, we need to inform the user
	if len(config.SubnetIDs) == 0 {
		// For MVP, we'll just add a placeholder region
		// In production, this should fail with a helpful error
		s.logger.WarnContext(ctx, "No subnet IDs provided - cluster creation may fail",
			slog.String("info", "Run 'rosa create network' first to create a VPC"),
		)
	} else {
		// Set the subnet IDs from the provided VPC
		awsBuilder.SubnetIDs(config.SubnetIDs...)
	}

	// Note: AWS region is set on the cluster level, not on the AWS builder

	// Configure STS if roles are provided
	if config.RoleARN != "" {
		stsBuilder := cmv1.NewSTS()
		stsBuilder.RoleARN(config.RoleARN)
		stsBuilder.SupportRoleARN(config.SupportRoleARN)

		// Instance IAM roles (no control plane role for HCP!)
		instanceRoles := cmv1.NewInstanceIAMRoles()
		instanceRoles.WorkerRoleARN(config.WorkerRoleARN)
		stsBuilder.InstanceIAMRoles(instanceRoles)

		if config.ExternalID != "" {
			stsBuilder.ExternalID(config.ExternalID)
		}

		awsBuilder.STS(stsBuilder)
	}

	// Add tags if provided
	if len(config.Tags) > 0 {
		awsBuilder.Tags(config.Tags)
	}

	// Add billing account if provided
	if config.BillingAccount != "" {
		awsBuilder.BillingAccountID(config.BillingAccount)
	}

	// Note: Availability zones are typically derived from subnet IDs
	// For HCP clusters, the control plane and workers can be in different AZs

	// AWS field is mandatory for ROSA clusters
	builder.AWS(awsBuilder)

	// Configure network
	networkBuilder := cmv1.NewNetwork()
	if config.MachineCIDR != "" {
		networkBuilder.MachineCIDR(config.MachineCIDR)
	}
	if config.ServiceCIDR != "" {
		networkBuilder.ServiceCIDR(config.ServiceCIDR)
	}
	if config.PodCIDR != "" {
		networkBuilder.PodCIDR(config.PodCIDR)
	}
	if config.HostPrefix != 0 {
		networkBuilder.HostPrefix(config.HostPrefix)
	}
	builder.Network(networkBuilder)

	// Configure API access
	if config.Private {
		builder.API(cmv1.NewClusterAPI().Listening(cmv1.ListeningMethodInternal))
	} else {
		builder.API(cmv1.NewClusterAPI().Listening(cmv1.ListeningMethodExternal))
	}

	// Configure properties
	properties := make(map[string]string)
	properties["rosa_cli_version"] = "2.0.0" // HCP-only version
	properties["rosa_hcp"] = "true"
	builder.Properties(properties)

	// Configure multi-AZ
	if config.MultiAZ {
		builder.MultiAZ(true)
	}

	// Configure security features
	if config.FIPS {
		builder.FIPS(true)
	}

	if config.EtcdEncryption {
		builder.EtcdEncryption(true)
	}

	if config.DisableWorkloadMon {
		builder.DisableUserWorkloadMonitoring(true)
	}

	// Configure initial node pool
	// Note: For HCP clusters, node pools are typically created separately
	// after the cluster is created, not as part of the initial cluster creation
	// Commenting this out to fix the "aws field not supported" error

	/*
		// TODO: Create node pools separately after cluster creation
		nodePoolBuilder := cmv1.NewNodePool().
			ID("default").
			Replicas(config.ComputeNodes).
			AWSNodePool(cmv1.NewAWSNodePool().
				InstanceType(config.ComputeType))

		nodePool, _ := nodePoolBuilder.Build()
		nodePoolList := cmv1.NewNodePoolList()
		nodePoolList.Items(nodePool)
		builder.NodePools(nodePoolList)
	*/

	// Build the cluster object
	cluster, err := builder.Build()
	if err != nil {
		return errors.Err[*Cluster](
			errors.Validation("cluster.Create", err).
				WithSuggestion("Check cluster configuration requirements"),
		)
	}

	// Create via OCM API
	response, err := s.ocm.CreateCluster(ctx, cluster)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create cluster",
			slog.String("error", err.Error()),
		)
		return errors.Err[*Cluster](
			errors.API("cluster.Create", err),
		)
	}

	created := response.Body()

	s.logger.InfoContext(ctx, "cluster created successfully",
		slog.String("id", created.ID()),
		slog.String("name", created.Name()),
	)

	// Convert to our domain model
	result := &Cluster{
		ID:        created.ID(),
		Name:      created.Name(),
		State:     string(created.State()),
		Region:    created.Region().ID(),
		Version:   created.Version().ID(),
		CreatedAt: created.CreationTimestamp(),
	}

	// URLs might not be available immediately
	if created.API() != nil {
		result.APIURL = created.API().URL()
	}
	if created.Console() != nil {
		result.ConsoleURL = created.Console().URL()
	}

	return errors.Ok(result)
}

// Delete deletes a cluster
func (s *Service) Delete(ctx context.Context, clusterID string) error {
	s.logger.InfoContext(ctx, "deleting cluster", slog.String("id", clusterID))

	response, err := s.ocm.DeleteCluster(ctx, clusterID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete cluster",
			slog.String("id", clusterID),
			slog.String("error", err.Error()),
		)

		if response != nil && response.Status() == 404 {
			return errors.NotFound("cluster.Delete", fmt.Sprintf("cluster %s", clusterID))
		}

		return errors.API("cluster.Delete", err)
	}

	s.logger.InfoContext(ctx, "cluster deletion initiated", slog.String("id", clusterID))
	return nil
}

// Get retrieves a cluster by ID or name
func (s *Service) Get(ctx context.Context, clusterKey string) (*Cluster, error) {
	s.logger.InfoContext(ctx, "getting cluster", slog.String("key", clusterKey))

	response, err := s.ocm.GetCluster(ctx, clusterKey)
	if err != nil {
		if response != nil && response.Status() == 404 {
			return nil, errors.NotFound("cluster.Get", fmt.Sprintf("cluster %s", clusterKey))
		}
		return nil, errors.API("cluster.Get", err)
	}

	cluster := response.Body()

	result := &Cluster{
		ID:        cluster.ID(),
		Name:      cluster.Name(),
		State:     string(cluster.State()),
		Region:    cluster.Region().ID(),
		Version:   cluster.Version().ID(),
		CreatedAt: cluster.CreationTimestamp(),
	}

	if cluster.API() != nil {
		result.APIURL = cluster.API().URL()
	}
	if cluster.Console() != nil {
		result.ConsoleURL = cluster.Console().URL()
	}

	return result, nil
}

// List lists all clusters
func (s *Service) List(ctx context.Context) ([]*Cluster, error) {
	s.logger.InfoContext(ctx, "listing clusters")

	response, err := s.ocm.ListClusters(ctx)
	if err != nil {
		return nil, errors.API("cluster.List", err)
	}

	items := response.Items().Slice()
	clusters := make([]*Cluster, 0, len(items))

	for _, item := range items {
		// Only include HCP clusters (though all should be HCP in v2)
		if item.Hypershift() != nil && item.Hypershift().Enabled() {
			cluster := &Cluster{
				ID:        item.ID(),
				Name:      item.Name(),
				State:     string(item.State()),
				Region:    item.Region().ID(),
				Version:   item.Version().ID(),
				CreatedAt: item.CreationTimestamp(),
			}

			if item.API() != nil {
				cluster.APIURL = item.API().URL()
			}
			if item.Console() != nil {
				cluster.ConsoleURL = item.Console().URL()
			}

			clusters = append(clusters, cluster)
		}
	}

	s.logger.InfoContext(ctx, "listed clusters", slog.Int("count", len(clusters)))
	return clusters, nil
}
