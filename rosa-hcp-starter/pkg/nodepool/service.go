package nodepool

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/aws"
)

// Service provides node pool operations
type Service struct {
	ocm    api.Client
	aws    aws.Client
	logger *slog.Logger
}

// NewService creates a new node pool service
func NewService(ocm api.Client, aws aws.Client, logger *slog.Logger) *Service {
	return &Service{
		ocm:    ocm,
		aws:    aws,
		logger: logger.With("service", "nodepool"),
	}
}

// NodePool represents a node pool
type NodePool struct {
	ID             string
	Name           string
	State          string
	Replicas       int
	MinReplicas    int
	MaxReplicas    int
	InstanceType   string
	DiskSize       int
	Labels         map[string]string
	Taints         []Taint
	Version        string
	Subnet         string
	AutoRepair     bool
	Autoscaling    bool
	KubeletConfigs []string
	TuningConfigs  []string
	Message        string
	CreatedAt      time.Time
}

// Taint represents a node taint
type Taint struct {
	Key    string
	Value  string
	Effect string
}

// CreateConfig holds node pool creation configuration
type CreateConfig struct {
	ClusterID      string
	Name           string
	Replicas       int
	MinReplicas    int
	MaxReplicas    int
	InstanceType   string
	DiskSize       int
	Labels         map[string]string
	Taints         []Taint
	Version        string
	Subnet         string
	AutoRepair     bool
	Autoscaling    bool
	KubeletConfigs []string
	TuningConfigs  []string
}

// UpdateConfig holds node pool update configuration
type UpdateConfig struct {
	Replicas    int
	MinReplicas int
	MaxReplicas int
	AutoRepair  *bool
}

// Create creates a new node pool
func (s *Service) Create(ctx context.Context, config CreateConfig) (*NodePool, error) {
	s.logger.InfoContext(ctx, "creating node pool",
		slog.String("cluster", config.ClusterID),
		slog.String("name", config.Name),
	)

	// Build node pool object
	builder := cmv1.NewNodePool().
		ID(config.Name).
		Replicas(config.Replicas)

	// Set autoscaling if configured
	if config.Autoscaling {
		builder.Autoscaling(
			cmv1.NewNodePoolAutoscaling().
				MinReplica(config.MinReplicas).
				MaxReplica(config.MaxReplicas),
		)
	}

	// Set AWS configuration
	awsBuilder := cmv1.NewAWSNodePool().
		InstanceType(config.InstanceType)

	builder.AWSNodePool(awsBuilder)

	// Set labels
	if len(config.Labels) > 0 {
		builder.Labels(config.Labels)
	}

	// Set taints
	if len(config.Taints) > 0 {
		var taintBuilders []*cmv1.TaintBuilder
		for _, taint := range config.Taints {
			taintBuilder := cmv1.NewTaint().
				Key(taint.Key).
				Value(taint.Value).
				Effect(taint.Effect)
			taintBuilders = append(taintBuilders, taintBuilder)
		}
		builder.Taints(taintBuilders...)
	}

	// Set version if specified
	if config.Version != "" {
		builder.Version(cmv1.NewVersion().ID(config.Version))
	}

	// Set subnet if specified
	if config.Subnet != "" {
		builder.Subnet(config.Subnet)
	}

	// Set auto-repair
	builder.AutoRepair(config.AutoRepair)

	// Set kubelet configs
	if len(config.KubeletConfigs) > 0 {
		builder.KubeletConfigs(config.KubeletConfigs...)
	}

	// Set tuning configs
	if len(config.TuningConfigs) > 0 {
		builder.TuningConfigs(config.TuningConfigs...)
	}

	// Build the node pool
	nodePool, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build node pool: %w", err)
	}

	// Create via API
	response, err := s.ocm.CreateNodePool(ctx, config.ClusterID, nodePool)
	if err != nil {
		return nil, fmt.Errorf("failed to create node pool: %w", err)
	}

	created := response.Body()

	s.logger.InfoContext(ctx, "node pool created",
		slog.String("id", created.ID()),
		slog.String("name", config.Name),
	)

	// Convert to our model
	return convertNodePool(created), nil
}

// Delete deletes a node pool
func (s *Service) Delete(ctx context.Context, clusterID string, nodePoolID string) error {
	s.logger.InfoContext(ctx, "deleting node pool",
		slog.String("cluster", clusterID),
		slog.String("nodepool", nodePoolID),
	)

	conn := s.ocm.GetConnection()
	if conn == nil {
		return fmt.Errorf("no connection available")
	}

	_, err := conn.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		NodePools().NodePool(nodePoolID).
		Delete().
		SendContext(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete node pool: %w", err)
	}

	s.logger.InfoContext(ctx, "node pool deleted",
		slog.String("nodepool", nodePoolID),
	)

	return nil
}

// List lists all node pools for a cluster
func (s *Service) List(ctx context.Context, clusterID string) ([]*NodePool, error) {
	s.logger.InfoContext(ctx, "listing node pools",
		slog.String("cluster", clusterID),
	)

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	response, err := conn.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		NodePools().
		List().
		Size(100).
		SendContext(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to list node pools: %w", err)
	}

	items := response.Items().Slice()
	nodePools := make([]*NodePool, 0, len(items))

	for _, item := range items {
		nodePools = append(nodePools, convertNodePool(item))
	}

	s.logger.InfoContext(ctx, "listed node pools",
		slog.Int("count", len(nodePools)),
	)

	return nodePools, nil
}

// Get retrieves a specific node pool
func (s *Service) Get(ctx context.Context, clusterID string, nodePoolID string) (*NodePool, error) {
	s.logger.InfoContext(ctx, "getting node pool",
		slog.String("cluster", clusterID),
		slog.String("nodepool", nodePoolID),
	)

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	response, err := conn.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		NodePools().NodePool(nodePoolID).
		Get().
		SendContext(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get node pool: %w", err)
	}

	return convertNodePool(response.Body()), nil
}

// Update updates a node pool
func (s *Service) Update(ctx context.Context, clusterID string, nodePoolID string, config UpdateConfig) (*NodePool, error) {
	s.logger.InfoContext(ctx, "updating node pool",
		slog.String("cluster", clusterID),
		slog.String("nodepool", nodePoolID),
	)

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	// Build update
	builder := cmv1.NewNodePool()

	// Update replicas or autoscaling
	if config.MinReplicas > 0 && config.MaxReplicas > 0 {
		// Enable autoscaling
		builder.Autoscaling(
			cmv1.NewNodePoolAutoscaling().
				MinReplica(config.MinReplicas).
				MaxReplica(config.MaxReplicas),
		)
	} else if config.Replicas > 0 {
		// Set fixed replicas (disables autoscaling)
		builder.Replicas(config.Replicas)
	}

	// Update auto-repair if specified
	if config.AutoRepair != nil {
		builder.AutoRepair(*config.AutoRepair)
	}

	// Send update
	nodePool, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build update: %w", err)
	}

	response, err := conn.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		NodePools().NodePool(nodePoolID).
		Update().
		Body(nodePool).
		SendContext(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to update node pool: %w", err)
	}

	s.logger.InfoContext(ctx, "node pool updated",
		slog.String("nodepool", nodePoolID),
	)

	return convertNodePool(response.Body()), nil
}

// convertNodePool converts OCM node pool to our model
func convertNodePool(np *cmv1.NodePool) *NodePool {
	result := &NodePool{
		ID:         np.ID(),
		State:      "ready", // Default state, as Status() returns a complex type
		Replicas:   np.Replicas(),
		AutoRepair: np.AutoRepair(),
	}

	// Get status if available
	if np.Status() != nil {
		// The status is a complex object, extract the string representation
		if np.Status().CurrentReplicas() > 0 {
			result.State = "ready"
		} else {
			result.State = "pending"
		}
	}

	// Basic properties
	if np.ID() != "" {
		result.Name = np.ID()
	}

	// AWS properties
	if np.AWSNodePool() != nil {
		result.InstanceType = np.AWSNodePool().InstanceType()
	}

	// Autoscaling
	if np.Autoscaling() != nil {
		result.Autoscaling = true
		result.MinReplicas = np.Autoscaling().MinReplica()
		result.MaxReplicas = np.Autoscaling().MaxReplica()
	}

	// Version
	if np.Version() != nil {
		result.Version = np.Version().ID()
	}

	// Subnet
	result.Subnet = np.Subnet()

	// Labels
	result.Labels = np.Labels()

	// Taints
	if np.Taints() != nil {
		for _, taint := range np.Taints() {
			result.Taints = append(result.Taints, Taint{
				Key:    taint.Key(),
				Value:  taint.Value(),
				Effect: taint.Effect(),
			})
		}
	}

	// Kubelet configs
	result.KubeletConfigs = np.KubeletConfigs()

	// Tuning configs
	result.TuningConfigs = np.TuningConfigs()

	// Status message - this field might not be available in the current SDK version
	// result.Message would be set here if available

	return result
}
