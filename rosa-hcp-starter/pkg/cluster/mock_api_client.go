package cluster

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa-hcp/pkg/api"
)

// MockAPIClient is a mock implementation of the API client for testing
type MockAPIClient struct {
	clusters map[string]*cmv1.Cluster
}

// NewMockAPIClient creates a new mock API client
func NewMockAPIClient() api.Client {
	return &MockAPIClient{
		clusters: make(map[string]*cmv1.Cluster),
	}
}

// CreateCluster creates a mock cluster
func (m *MockAPIClient) CreateCluster(ctx context.Context, cluster *cmv1.Cluster) (*cmv1.ClustersAddResponse, error) {
	// Simulate cluster creation
	id := fmt.Sprintf("fake-%d", time.Now().Unix())

	// Build a fake response
	regionID := "us-west-2"
	if cluster.Region() != nil {
		regionID = cluster.Region().ID()
	}

	createdCluster, _ := cmv1.NewCluster().
		ID(id).
		Name(cluster.Name()).
		State(cmv1.ClusterStatePending).
		Region(cmv1.NewCloudRegion().ID(regionID)).
		Version(cmv1.NewVersion().ID("4.14.0")).
		CreationTimestamp(time.Now()).
		Build()

	m.clusters[id] = createdCluster
	m.clusters[cluster.Name()] = createdCluster

	response := &cmv1.ClustersAddResponse{}
	// This is a simplification - in reality we'd need to properly construct the response

	return response, nil
}

// GetCluster retrieves a mock cluster
func (m *MockAPIClient) GetCluster(ctx context.Context, clusterKey string) (*cmv1.ClusterGetResponse, error) {
	_, exists := m.clusters[clusterKey]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterKey)
	}

	response := &cmv1.ClusterGetResponse{}
	// This is a simplification
	return response, nil
}

// DeleteCluster deletes a mock cluster
func (m *MockAPIClient) DeleteCluster(ctx context.Context, clusterKey string) (*cmv1.ClusterDeleteResponse, error) {
	if _, exists := m.clusters[clusterKey]; !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterKey)
	}

	delete(m.clusters, clusterKey)

	response := &cmv1.ClusterDeleteResponse{}
	return response, nil
}

// ListClusters lists mock clusters
func (m *MockAPIClient) ListClusters(ctx context.Context) (*cmv1.ClustersListResponse, error) {
	var items []*cmv1.Cluster
	for _, cluster := range m.clusters {
		items = append(items, cluster)
	}

	// Build response with items
	response := &cmv1.ClustersListResponse{}
	// This is a simplification

	return response, nil
}

// CreateNodePool creates a mock node pool
func (m *MockAPIClient) CreateNodePool(ctx context.Context, clusterID string, nodePool *cmv1.NodePool) (*cmv1.NodePoolsAddResponse, error) {
	response := &cmv1.NodePoolsAddResponse{}
	return response, nil
}

// GetConnection returns nil for mock
func (m *MockAPIClient) GetConnection() *sdk.Connection {
	return nil
}
