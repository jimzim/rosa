package instance

import (
	"context"
	"fmt"
	"log/slog"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	sdk "github.com/openshift-online/ocm-sdk-go"
)

// Service defines the interface for instance type operations
type Service interface {
	ListInstanceTypes(ctx context.Context) ([]*InstanceType, error)
	ListInstanceTypesForRegion(ctx context.Context, region string) ([]*InstanceType, error)
}

// service implements the Service interface
type service struct {
	logger *slog.Logger
	ocm    *sdk.Connection
}

// NewService creates a new instance type service
func NewService(ctx context.Context, logger *slog.Logger, ocm *sdk.Connection) (Service, error) {
	return &service{
		logger: logger,
		ocm:    ocm,
	}, nil
}

// InstanceType represents an available instance type
type InstanceType struct {
	ID            string
	Name          string
	CPU           int
	Memory        float64 // in GiB
	Category      string
	CloudProvider string
	Size          string
}

// ListInstanceTypes lists all available instance types
func (s *service) ListInstanceTypes(ctx context.Context) ([]*InstanceType, error) {
	s.logger.InfoContext(ctx, "listing all instance types")

	collection := s.ocm.ClustersMgmt().V1().MachineTypes()
	
	// List all machine types
	response, err := collection.List().
		Page(1).
		Size(1000). // Get all in one page
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list instance types: %w", err)
	}

	instances := make([]*InstanceType, 0, response.Total())
	response.Items().Each(func(mt *cmv1.MachineType) bool {
		instance := &InstanceType{
			ID:            mt.ID(),
			Name:          mt.Name(),
			CPU:           int(mt.CPU().Value()),
			Memory:        float64(mt.Memory().Value()) / (1024 * 1024 * 1024), // Convert bytes to GiB
			Category:      string(mt.Category()),
			CloudProvider: "aws",
			Size:          string(mt.Size()),
		}
		instances = append(instances, instance)
		return true
	})

	s.logger.InfoContext(ctx, "found instance types", slog.Int("count", len(instances)))
	return instances, nil
}

// ListInstanceTypesForRegion lists instance types available in a specific region
func (s *service) ListInstanceTypesForRegion(ctx context.Context, region string) ([]*InstanceType, error) {
	s.logger.InfoContext(ctx, "listing instance types for region", slog.String("region", region))

	// First, verify the region exists and is enabled
	regionResp, err := s.ocm.ClustersMgmt().V1().
		CloudProviders().CloudProvider("aws").
		Regions().Region(region).
		Get().SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get region %s: %w", region, err)
	}

	if !regionResp.Body().Enabled() {
		return nil, fmt.Errorf("region %s is not enabled", region)
	}

	// List machine types with region filter
	// Note: Region-specific filtering might not be fully supported by the API,
	// so we attempt it but fall back to all types if needed
	collection := s.ocm.ClustersMgmt().V1().MachineTypes()
	
	response, err := collection.List().
		Parameter("cloud_provider.id", "aws").
		Parameter("cloud_provider.region.id", region).
		Page(1).
		Size(1000).
		SendContext(ctx)
	if err != nil {
		// Fall back to all types if region filtering fails
		s.logger.WarnContext(ctx, "failed to filter by region, listing all types", 
			slog.String("region", region), slog.String("error", err.Error()))
		return s.ListInstanceTypes(ctx)
	}

	instances := make([]*InstanceType, 0, response.Total())
	response.Items().Each(func(mt *cmv1.MachineType) bool {
		instance := &InstanceType{
			ID:            mt.ID(),
			Name:          mt.Name(),
			CPU:           int(mt.CPU().Value()),
			Memory:        float64(mt.Memory().Value()) / (1024 * 1024 * 1024), // Convert bytes to GiB
			Category:      string(mt.Category()),
			CloudProvider: "aws",
			Size:          string(mt.Size()),
		}
		instances = append(instances, instance)
		return true
	})

	s.logger.InfoContext(ctx, "found instance types for region", 
		slog.String("region", region), slog.Int("count", len(instances)))
	return instances, nil
}
