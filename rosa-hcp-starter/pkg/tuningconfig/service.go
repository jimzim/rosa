package tuningconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"gopkg.in/yaml.v3"
)

// Service defines the interface for TuningConfig operations
type Service interface {
	Create(ctx context.Context, clusterID string, config Config) (*TuningConfig, error)
	List(ctx context.Context, clusterID string) ([]*TuningConfig, error)
	Delete(ctx context.Context, clusterID, configID string) error
	Get(ctx context.Context, clusterID, configID string) (*TuningConfig, error)
}

// service implements the Service interface
type service struct {
	logger *slog.Logger
	ocm    *sdk.Connection
}

// NewService creates a new TuningConfig service
func NewService(ctx context.Context, logger *slog.Logger, ocm *sdk.Connection) (Service, error) {
	return &service{
		logger: logger,
		ocm:    ocm,
	}, nil
}

// Config holds TuningConfig creation configuration
type Config struct {
	Name     string
	SpecPath string
	SpecData map[string]interface{} // For programmatic creation
}

// TuningConfig represents a TuningConfig
type TuningConfig struct {
	ID   string
	Name string
	Spec map[string]interface{}
}

// TuningSpec represents the structure of a tuning config spec
type TuningSpec struct {
	Profile []struct {
		Data string `json:"data" yaml:"data"`
		Name string `json:"name" yaml:"name"`
	} `json:"profile" yaml:"profile"`
	Recommend []struct {
		Priority int    `json:"priority" yaml:"priority"`
		Profile  string `json:"profile" yaml:"profile"`
		Match    []struct {
			Label string `json:"label,omitempty" yaml:"label,omitempty"`
			Value string `json:"value,omitempty" yaml:"value,omitempty"`
			Type  string `json:"type,omitempty" yaml:"type,omitempty"`
		} `json:"match,omitempty" yaml:"match,omitempty"`
	} `json:"recommend" yaml:"recommend"`
}

// Create creates a new TuningConfig
func (s *service) Create(ctx context.Context, clusterID string, config Config) (*TuningConfig, error) {
	s.logger.InfoContext(ctx, "creating tuning config",
		slog.String("cluster", clusterID),
		slog.String("name", config.Name))

	if config.Name == "" {
		return nil, fmt.Errorf("name is required for TuningConfig")
	}

	var spec map[string]interface{}

	// If SpecData is provided, use it directly
	if config.SpecData != nil {
		spec = config.SpecData
	} else if config.SpecPath != "" {
		// Read and parse spec file
		specData, err := os.ReadFile(config.SpecPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read spec file: %w", err)
		}

		// Try to parse as JSON first
		err = json.Unmarshal(specData, &spec)
		if err != nil {
			// If JSON fails, try YAML
			err = yaml.Unmarshal(specData, &spec)
			if err != nil {
				return nil, fmt.Errorf("spec file must be valid JSON or YAML: %w", err)
			}
		}

		// Validate the spec structure
		if err := validateTuningSpec(spec); err != nil {
			return nil, fmt.Errorf("invalid tuning spec: %w", err)
		}
	} else {
		return nil, fmt.Errorf("either SpecPath or SpecData must be provided")
	}

	builder := cmv1.NewTuningConfig().
		Name(config.Name).
		Spec(spec)

	tcfg, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build tuning config: %w", err)
	}

	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		TuningConfigs().
		Add().Body(tcfg).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create tuning config: %w", err)
	}

	result := convertTuningConfig(response.Body())
	s.logger.InfoContext(ctx, "tuning config created successfully",
		slog.String("id", result.ID),
		slog.String("name", result.Name))

	return result, nil
}

// List lists all TuningConfigs for a cluster
func (s *service) List(ctx context.Context, clusterID string) ([]*TuningConfig, error) {
	s.logger.InfoContext(ctx, "listing tuning configs", slog.String("cluster", clusterID))

	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		TuningConfigs().
		List().
		Page(1).
		Size(100).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tuning configs: %w", err)
	}

	configs := make([]*TuningConfig, 0, response.Total())
	response.Items().Each(func(tc *cmv1.TuningConfig) bool {
		configs = append(configs, convertTuningConfig(tc))
		return true
	})

	s.logger.InfoContext(ctx, "found tuning configs", slog.Int("count", len(configs)))
	return configs, nil
}

// Get retrieves a specific TuningConfig
func (s *service) Get(ctx context.Context, clusterID, configID string) (*TuningConfig, error) {
	s.logger.InfoContext(ctx, "getting tuning config",
		slog.String("cluster", clusterID),
		slog.String("config", configID))

	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		TuningConfigs().TuningConfig(configID).
		Get().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tuning config: %w", err)
	}

	return convertTuningConfig(response.Body()), nil
}

// Delete deletes a TuningConfig
func (s *service) Delete(ctx context.Context, clusterID, configID string) error {
	s.logger.InfoContext(ctx, "deleting tuning config",
		slog.String("cluster", clusterID),
		slog.String("config", configID))

	// First check if any node pools are using this config
	nodePools, err := s.getNodePoolsUsingConfig(ctx, clusterID, configID)
	if err != nil {
		return fmt.Errorf("failed to check nodepool usage: %w", err)
	}

	if len(nodePools) > 0 {
		return fmt.Errorf("cannot delete tuning config '%s': used by nodepools: %v", configID, nodePools)
	}

	_, err = s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		TuningConfigs().TuningConfig(configID).
		Delete().
		SendContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete tuning config: %w", err)
	}

	s.logger.InfoContext(ctx, "tuning config deleted successfully", slog.String("config", configID))
	return nil
}

// getNodePoolsUsingConfig returns a list of node pool IDs using the given tuning config
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
		if np.TuningConfigs() != nil {
			for _, tc := range np.TuningConfigs() {
				if tc == configID {
					nodePools = append(nodePools, np.ID())
					break
				}
			}
		}
		return true
	})

	return nodePools, nil
}

// convertTuningConfig converts from OCM SDK type to our domain type
func convertTuningConfig(tc *cmv1.TuningConfig) *TuningConfig {
	if tc == nil {
		return nil
	}

	spec, _ := tc.GetSpec()
	// Type assert to map[string]interface{}
	specMap, ok := spec.(map[string]interface{})
	if !ok {
		// If type assertion fails, create empty map
		specMap = make(map[string]interface{})
	}
	
	return &TuningConfig{
		ID:   tc.ID(),
		Name: tc.Name(),
		Spec: specMap,
	}
}

// validateTuningSpec validates the structure of a tuning spec
func validateTuningSpec(spec map[string]interface{}) error {
	// Check for required fields
	if _, ok := spec["profile"]; !ok {
		return fmt.Errorf("spec must contain 'profile' field")
	}

	if _, ok := spec["recommend"]; !ok {
		return fmt.Errorf("spec must contain 'recommend' field")
	}

	// Validate profile is an array
	profiles, ok := spec["profile"].([]interface{})
	if !ok {
		return fmt.Errorf("'profile' must be an array")
	}

	if len(profiles) == 0 {
		return fmt.Errorf("'profile' array cannot be empty")
	}

	// Validate each profile has required fields
	for i, p := range profiles {
		profile, ok := p.(map[string]interface{})
		if !ok {
			return fmt.Errorf("profile[%d] must be an object", i)
		}

		if _, ok := profile["name"]; !ok {
			return fmt.Errorf("profile[%d] must have a 'name' field", i)
		}

		if _, ok := profile["data"]; !ok {
			return fmt.Errorf("profile[%d] must have a 'data' field", i)
		}
	}

	// Validate recommend is an array
	recommends, ok := spec["recommend"].([]interface{})
	if !ok {
		return fmt.Errorf("'recommend' must be an array")
	}

	if len(recommends) == 0 {
		return fmt.Errorf("'recommend' array cannot be empty")
	}

	// Validate each recommend has required fields
	for i, r := range recommends {
		recommend, ok := r.(map[string]interface{})
		if !ok {
			return fmt.Errorf("recommend[%d] must be an object", i)
		}

		if _, ok := recommend["profile"]; !ok {
			return fmt.Errorf("recommend[%d] must have a 'profile' field", i)
		}

		if _, ok := recommend["priority"]; !ok {
			return fmt.Errorf("recommend[%d] must have a 'priority' field", i)
		}
	}

	return nil
}

// CreateExampleSpec creates an example tuning spec
func CreateExampleSpec() map[string]interface{} {
	return map[string]interface{}{
		"profile": []map[string]interface{}{
			{
				"name": "custom-profile",
				"data": "[main]\nsummary=Custom OpenShift profile\ninclude=openshift-node\n\n[sysctl]\nvm.dirty_ratio=\"20\"\nvm.swappiness=\"10\"\n",
			},
		},
		"recommend": []map[string]interface{}{
			{
				"priority": 20,
				"profile":  "custom-profile",
			},
		},
	}
}
