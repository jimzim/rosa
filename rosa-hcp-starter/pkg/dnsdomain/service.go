package dnsdomain

import (
	"context"
	"fmt"
	"log/slog"

	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// Service provides DNS Domain operations
type Service interface {
	Create(ctx context.Context, isHCP bool) (*DNSDomain, error)
	List(ctx context.Context, hcpOnly bool) ([]*DNSDomain, error)
	Delete(ctx context.Context, domainID string) error
}

type service struct {
	logger *slog.Logger
	ocm    *sdk.Connection
}

// NewService creates a new DNS Domain service
func NewService(ctx context.Context, logger *slog.Logger, ocm *sdk.Connection) (Service, error) {
	return &service{
		logger: logger,
		ocm:    ocm,
	}, nil
}

// DNSDomain represents a DNS domain reservation
type DNSDomain struct {
	ID           string
	ClusterID    string
	ReservedTime string
	UserDefined  bool
	Architecture string
}

// Create creates a new DNS domain
func (s *service) Create(ctx context.Context, isHCP bool) (*DNSDomain, error) {
	s.logger.InfoContext(ctx, "creating DNS domain", slog.Bool("hcp", isHCP))

	// Build the DNS domain
	builder := cmv1.NewDNSDomain()
	
	if isHCP {
		builder.ClusterArch(cmv1.ClusterArchitectureHcp)
	} else {
		builder.ClusterArch(cmv1.ClusterArchitectureClassic)
	}

	dnsDomain, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build DNS domain: %w", err)
	}

	// Create via API
	response, err := s.ocm.ClustersMgmt().V1().
		DNSDomains().
		Add().Body(dnsDomain).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS domain: %w", err)
	}

	result := convertDNSDomain(response.Body())
	s.logger.InfoContext(ctx, "DNS domain created successfully",
		slog.String("id", result.ID))

	return result, nil
}

// List lists all DNS domains
func (s *service) List(ctx context.Context, hcpOnly bool) ([]*DNSDomain, error) {
	s.logger.InfoContext(ctx, "listing DNS domains", slog.Bool("hcp_only", hcpOnly))

	// Build search query
	search := "user_defined='true'"
	if hcpOnly {
		if search != "" {
			search += " AND "
		}
		search += "cluster_arch='Hcp'"
	}

	var listQuery *cmv1.DNSDomainsListRequest
	if search != "" {
		listQuery = s.ocm.ClustersMgmt().V1().
			DNSDomains().
			List().
			Search(search)
	} else {
		listQuery = s.ocm.ClustersMgmt().V1().
			DNSDomains().
			List()
	}

	response, err := listQuery.
		Page(1).
		Size(100).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS domains: %w", err)
	}

	domains := make([]*DNSDomain, 0, response.Total())
	response.Items().Each(func(domain *cmv1.DNSDomain) bool {
		if !hcpOnly || domain.ClusterArch() == cmv1.ClusterArchitectureHcp {
			domains = append(domains, convertDNSDomain(domain))
		}
		return true
	})

	s.logger.InfoContext(ctx, "found DNS domains", slog.Int("count", len(domains)))
	return domains, nil
}

// Delete deletes a DNS domain
func (s *service) Delete(ctx context.Context, domainID string) error {
	s.logger.InfoContext(ctx, "deleting DNS domain", slog.String("domain_id", domainID))

	_, err := s.ocm.ClustersMgmt().V1().
		DNSDomains().
		DNSDomain(domainID).
		Delete().
		SendContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete DNS domain: %w", err)
	}

	s.logger.InfoContext(ctx, "DNS domain deleted successfully",
		slog.String("domain_id", domainID))
	return nil
}

// Helper functions

func convertDNSDomain(domain *cmv1.DNSDomain) *DNSDomain {
	if domain == nil {
		return nil
	}

	d := &DNSDomain{
		ID:          domain.ID(),
		UserDefined: domain.UserDefined(),
	}

	// Set cluster ID if available
	if domain.Cluster() != nil {
		d.ClusterID = domain.Cluster().ID()
	}

	// Set reserved time
	if domain.ReservedAtTimestamp() != nil && !domain.ReservedAtTimestamp().IsZero() {
		d.ReservedTime = domain.ReservedAtTimestamp().Format("2006-01-02 15:04:05")
	}

	// Set architecture
	switch domain.ClusterArch() {
	case cmv1.ClusterArchitectureHcp:
		d.Architecture = "HCP"
	case cmv1.ClusterArchitectureClassic:
		d.Architecture = "Classic"
	default:
		d.Architecture = "Unknown"
	}

	return d
}
