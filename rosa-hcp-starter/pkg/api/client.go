package api

import (
	"context"
	"fmt"
	"net/http"

	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

// Client is a wrapper around the OCM SDK client
type Client interface {
	CreateCluster(ctx context.Context, cluster *cmv1.Cluster) (*cmv1.ClustersAddResponse, error)
	GetCluster(ctx context.Context, clusterKey string) (*cmv1.ClusterGetResponse, error)
	DeleteCluster(ctx context.Context, clusterKey string) (*cmv1.ClusterDeleteResponse, error)
	ListClusters(ctx context.Context) (*cmv1.ClustersListResponse, error)
	CreateNodePool(ctx context.Context, clusterID string, nodePool *cmv1.NodePool) (*cmv1.NodePoolsAddResponse, error)
	GetConnection() *sdk.Connection
}

// Config holds the configuration for the API client
type Config struct {
	URL          string
	Token        string
	RefreshToken string
	ClientID     string
	ClientSecret string
	Scopes       []string
}

type client struct {
	connection *sdk.Connection
	clusters   *cmv1.ClustersClient
}

// NewClient creates a new OCM API client
func NewClient(ctx context.Context, config Config) (Client, error) {
	// Create logger
	logger, err := logging.NewGoLoggerBuilder().
		Debug(false).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Start building connection
	builder := sdk.NewConnectionBuilder().
		Logger(logger)

	// Set URL if provided
	if config.URL != "" {
		builder.URL(config.URL)
	}

	// Configure authentication
	if config.Token != "" {
		builder.Tokens(config.Token, config.RefreshToken)
	} else if config.ClientID != "" && config.ClientSecret != "" {
		builder.Client(config.ClientID, config.ClientSecret)
	} else {
		// Try to load from config file
		builder.Load(nil)
	}

	// Set scopes if provided
	if len(config.Scopes) > 0 {
		builder.Scopes(config.Scopes...)
	}

	// Build connection
	connection, err := builder.BuildContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCM connection: %w", err)
	}

	return &client{
		connection: connection,
		clusters:   connection.ClustersMgmt().V1().Clusters(),
	}, nil
}

// CreateCluster creates a new cluster
func (c *client) CreateCluster(ctx context.Context, cluster *cmv1.Cluster) (*cmv1.ClustersAddResponse, error) {
	response, err := c.clusters.Add().
		Body(cluster).
		SendContext(ctx)

	if err != nil {
		return response, fmt.Errorf("failed to create cluster: %w", err)
	}

	if response.Status() != http.StatusCreated && response.Status() != http.StatusAccepted {
		return response, fmt.Errorf("unexpected status code %d", response.Status())
	}

	return response, nil
}

// GetCluster retrieves a cluster by ID or name
func (c *client) GetCluster(ctx context.Context, clusterKey string) (*cmv1.ClusterGetResponse, error) {
	response, err := c.clusters.Cluster(clusterKey).
		Get().
		SendContext(ctx)

	if err != nil {
		return response, fmt.Errorf("failed to get cluster: %w", err)
	}

	return response, nil
}

// DeleteCluster deletes a cluster
func (c *client) DeleteCluster(ctx context.Context, clusterKey string) (*cmv1.ClusterDeleteResponse, error) {
	response, err := c.clusters.Cluster(clusterKey).
		Delete().
		SendContext(ctx)

	if err != nil {
		return response, fmt.Errorf("failed to delete cluster: %w", err)
	}

	return response, nil
}

// ListClusters lists all clusters
func (c *client) ListClusters(ctx context.Context) (*cmv1.ClustersListResponse, error) {
	// Filter for HCP clusters only
	response, err := c.clusters.List().
		Search("hypershift.enabled = 'true'").
		Size(100). // Reasonable default, pagination can be added later
		SendContext(ctx)

	if err != nil {
		return response, fmt.Errorf("failed to list clusters: %w", err)
	}

	return response, nil
}

// CreateNodePool creates a new node pool for a cluster
func (c *client) CreateNodePool(ctx context.Context, clusterID string, nodePool *cmv1.NodePool) (*cmv1.NodePoolsAddResponse, error) {
	response, err := c.clusters.Cluster(clusterID).
		NodePools().
		Add().
		Body(nodePool).
		SendContext(ctx)

	if err != nil {
		return response, fmt.Errorf("failed to create node pool: %w", err)
	}

	return response, nil
}

// GetConnection returns the underlying SDK connection
func (c *client) GetConnection() *sdk.Connection {
	return c.connection
}
