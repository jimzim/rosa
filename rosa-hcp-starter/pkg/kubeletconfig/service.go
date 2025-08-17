package kubeletconfig

import (
	"context"
	"fmt"
	"log/slog"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	sdk "github.com/openshift-online/ocm-sdk-go"
)

// Service defines the interface for KubeletConfig operations
type Service interface {
	Create(ctx context.Context, clusterID string, config Config) (*KubeletConfig, error)
	List(ctx context.Context, clusterID string) ([]*KubeletConfig, error)
	Delete(ctx context.Context, clusterID, configID string) error
	Get(ctx context.Context, clusterID, configID string) (*KubeletConfig, error)
}

// service implements the Service interface
type service struct {
	logger *slog.Logger
	ocm    *sdk.Connection
}

// NewService creates a new KubeletConfig service
func NewService(ctx context.Context, logger *slog.Logger, ocm *sdk.Connection) (Service, error) {
	return &service{
		logger: logger,
		ocm:    ocm,
	}, nil
}

// Config holds KubeletConfig creation configuration
type Config struct {
	Name         string
	PodPidsLimit int
}

// KubeletConfig represents a KubeletConfig
type KubeletConfig struct {
	ID           string
	Name         string
	PodPidsLimit int
}

// Create creates a new KubeletConfig
func (s *service) Create(ctx context.Context, clusterID string, config Config) (*KubeletConfig, error) {
	s.logger.InfoContext(ctx, "creating kubeletconfig",
		slog.String("cluster", clusterID),
		slog.String("name", config.Name))

	// Name is required for HCP clusters
	if config.Name == "" {
		return nil, fmt.Errorf("name is required for KubeletConfig in HCP clusters")
	}

	// Validate pod pids limit
	if config.PodPidsLimit < 4096 {
		return nil, fmt.Errorf("pod-pids-limit must be at least 4096, got %d", config.PodPidsLimit)
	}

	builder := cmv1.NewKubeletConfig().
		Name(config.Name).
		PodPidsLimit(config.PodPidsLimit)

	kcfg, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeletconfig: %w", err)
	}

	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		KubeletConfigs().
		Add().Body(kcfg).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubeletconfig: %w", err)
	}

	result := convertKubeletConfig(response.Body())
	s.logger.InfoContext(ctx, "kubeletconfig created successfully",
		slog.String("id", result.ID),
		slog.String("name", result.Name))

	return result, nil
}

// List lists all KubeletConfigs for a cluster
func (s *service) List(ctx context.Context, clusterID string) ([]*KubeletConfig, error) {
	s.logger.InfoContext(ctx, "listing kubeletconfigs", slog.String("cluster", clusterID))

	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		KubeletConfigs().
		List().
		Page(1).
		Size(100).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list kubeletconfigs: %w", err)
	}

	configs := make([]*KubeletConfig, 0, response.Total())
	response.Items().Each(func(kc *cmv1.KubeletConfig) bool {
		configs = append(configs, convertKubeletConfig(kc))
		return true
	})

	s.logger.InfoContext(ctx, "found kubeletconfigs", slog.Int("count", len(configs)))
	return configs, nil
}

// Get retrieves a specific KubeletConfig
func (s *service) Get(ctx context.Context, clusterID, configID string) (*KubeletConfig, error) {
	s.logger.InfoContext(ctx, "getting kubeletconfig",
		slog.String("cluster", clusterID),
		slog.String("config", configID))

	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		KubeletConfigs().KubeletConfig(configID).
		Get().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeletconfig: %w", err)
	}

	return convertKubeletConfig(response.Body()), nil
}

// Delete deletes a KubeletConfig
func (s *service) Delete(ctx context.Context, clusterID, configID string) error {
	s.logger.InfoContext(ctx, "deleting kubeletconfig",
		slog.String("cluster", clusterID),
		slog.String("config", configID))

	// First check if any node pools are using this config
	nodePools, err := s.getNodePoolsUsingConfig(ctx, clusterID, configID)
	if err != nil {
		return fmt.Errorf("failed to check nodepool usage: %w", err)
	}

	if len(nodePools) > 0 {
		return fmt.Errorf("cannot delete kubeletconfig '%s': used by nodepools: %v", configID, nodePools)
	}

	_, err = s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		KubeletConfigs().KubeletConfig(configID).
		Delete().
		SendContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete kubeletconfig: %w", err)
	}

	s.logger.InfoContext(ctx, "kubeletconfig deleted successfully", slog.String("config", configID))
	return nil
}

// getNodePoolsUsingConfig returns a list of node pool IDs using the given kubeletconfig
func (s *service) getNodePoolsUsingConfig(ctx context.Context, clusterID, configID string) ([]string, error) {
	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		NodePools().
		List().
		Page(1).
		Size(100).
		SendContext(ctx)
	if err != nil {
		return nil, err
	}

	var nodePools []string
	response.Items().Each(func(np *cmv1.NodePool) bool {
		if np.KubeletConfigs() != nil {
			for _, kc := range np.KubeletConfigs() {
				if kc == configID {
					nodePools = append(nodePools, np.ID())
					break
				}
			}
		}
		return true
	})

	return nodePools, nil
}

// convertKubeletConfig converts from OCM SDK type to our domain type
func convertKubeletConfig(kc *cmv1.KubeletConfig) *KubeletConfig {
	if kc == nil {
		return nil
	}

	return &KubeletConfig{
		ID:           kc.ID(),
		Name:         kc.Name(),
		PodPidsLimit: kc.PodPidsLimit(),
	}
}
