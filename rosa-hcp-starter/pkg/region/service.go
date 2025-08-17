package region

import (
	"context"
	"fmt"
	"log/slog"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	sdk "github.com/openshift-online/ocm-sdk-go"
)

// Service defines the interface for region operations
type Service interface {
	ListRegions(ctx context.Context, hcpOnly bool) ([]*Region, error)
}

// service implements the Service interface
type service struct {
	logger *slog.Logger
	ocm    *sdk.Connection
}

// NewService creates a new region service
func NewService(ctx context.Context, logger *slog.Logger, ocm *sdk.Connection) (Service, error) {
	return &service{
		logger: logger,
		ocm:    ocm,
	}, nil
}

// Region represents an AWS region
type Region struct {
	ID               string
	DisplayName      string
	Enabled          bool
	SupportsHCP      bool
	SupportsMultiAZ  bool
	CCSOnly          bool
	GovCloud         bool
	KMSLocationID    string
	KMSLocationName  string
}

// ListRegions lists all available AWS regions
func (s *service) ListRegions(ctx context.Context, hcpOnly bool) ([]*Region, error) {
	s.logger.InfoContext(ctx, "listing regions", slog.Bool("hcp_only", hcpOnly))

	collection := s.ocm.ClustersMgmt().V1().CloudProviders().CloudProvider("aws").Regions()
	
	// Build the request
	request := collection.List().
		Page(1).
		Size(100) // Get all regions in one page

	// List regions
	response, err := request.SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	regions := make([]*Region, 0, response.Total())
	response.Items().Each(func(r *cmv1.CloudRegion) bool {
		// Skip if HCP-only filter is enabled and region doesn't support HCP
		if hcpOnly && !r.SupportsHypershift() {
			return true
		}

		// Skip disabled regions unless explicitly requested
		if !r.Enabled() {
			return true
		}

		region := &Region{
			ID:               r.ID(),
			DisplayName:      r.DisplayName(),
			Enabled:          r.Enabled(),
			SupportsHCP:      r.SupportsHypershift(),
			SupportsMultiAZ:  r.SupportsMultiAZ(),
			CCSOnly:          r.CCSOnly(),
			GovCloud:         r.GovCloud(),
		}

		// Add KMS information if available
		if r.KMSLocationID() != "" {
			region.KMSLocationID = r.KMSLocationID()
		}
		if r.KMSLocationName() != "" {
			region.KMSLocationName = r.KMSLocationName()
		}

		regions = append(regions, region)
		return true
	})

	s.logger.InfoContext(ctx, "found regions", slog.Int("count", len(regions)))
	return regions, nil
}
