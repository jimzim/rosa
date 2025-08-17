package ingress

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	sdk "github.com/openshift-online/ocm-sdk-go"
)

var (
	// IngressKeyRE is a regular expression for ingress identifiers
	IngressKeyRE = regexp.MustCompile(`^[a-z0-9]{3,5}$`)
)

// LoadBalancerType represents the type of load balancer
type LoadBalancerType string

const (
	LoadBalancerClassic LoadBalancerType = "classic"
	LoadBalancerNLB     LoadBalancerType = "nlb"
)

// ListeningMethod represents how the ingress listens
type ListeningMethod string

const (
	ListeningExternal ListeningMethod = "external"
	ListeningInternal ListeningMethod = "internal"
)

// Service defines the interface for Ingress operations
type Service interface {
	Create(ctx context.Context, clusterID string, config CreateConfig) (*Ingress, error)
	List(ctx context.Context, clusterID string) ([]*Ingress, error)
	Get(ctx context.Context, clusterID string, ingressID string) (*Ingress, error)
	Update(ctx context.Context, clusterID string, ingressID string, config UpdateConfig) (*Ingress, error)
	Delete(ctx context.Context, clusterID string, ingressID string) error
}

// service implements the Service interface
type service struct {
	logger *slog.Logger
	ocm    *sdk.Connection
}

// NewService creates a new Ingress service
func NewService(ctx context.Context, logger *slog.Logger, ocm *sdk.Connection) (Service, error) {
	return &service{
		logger: logger,
		ocm:    ocm,
	}, nil
}

// CreateConfig holds ingress creation configuration
type CreateConfig struct {
	ID               string // Optional, will be generated if not provided
	Private          bool
	LoadBalancerType LoadBalancerType
	RouteSelectors   map[string]string // Label selectors for routes
}

// UpdateConfig holds ingress update configuration
type UpdateConfig struct {
	Private          *bool
	LoadBalancerType *LoadBalancerType
	RouteSelectors   map[string]string // nil means no change, empty map means clear
}

// Ingress represents an ingress (load balancer)
type Ingress struct {
	ID               string
	Default          bool
	Listening        ListeningMethod
	LoadBalancerType LoadBalancerType
	RouteSelectors   map[string]string
	DNSName          string
}

// Create creates a new ingress
func (s *service) Create(ctx context.Context, clusterID string, config CreateConfig) (*Ingress, error) {
	s.logger.InfoContext(ctx, "creating ingress",
		slog.String("cluster", clusterID),
		slog.String("id", config.ID))

	// Validate ingress ID if provided
	if config.ID != "" && !IngressKeyRE.MatchString(config.ID) {
		return nil, fmt.Errorf("ingress identifier '%s' must contain between 3 and 5 lowercase letters or digits", config.ID)
	}

	// Build the ingress
	builder := cmv1.NewIngress()

	// Set ID if provided
	if config.ID != "" {
		builder.ID(config.ID)
	}

	// Set listening method
	if config.Private {
		builder.Listening(cmv1.ListeningMethodInternal)
	} else {
		builder.Listening(cmv1.ListeningMethodExternal)
	}

	// Set load balancer type
	if config.LoadBalancerType != "" {
		builder.LoadBalancerType(cmv1.LoadBalancerFlavor(config.LoadBalancerType))
	} else {
		// Default to classic
		builder.LoadBalancerType(cmv1.LoadBalancerFlavorClassic)
	}

	// Set route selectors
	if len(config.RouteSelectors) > 0 {
		builder.RouteSelectors(config.RouteSelectors)
	}

	ingress, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build ingress: %w", err)
	}

	// Create via API
	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Ingresses().
		Add().Body(ingress).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create ingress: %w", err)
	}

	result := convertIngress(response.Body())
	s.logger.InfoContext(ctx, "ingress created successfully",
		slog.String("id", result.ID))

	return result, nil
}

// List lists all ingresses for a cluster
func (s *service) List(ctx context.Context, clusterID string) ([]*Ingress, error) {
	s.logger.InfoContext(ctx, "listing ingresses", slog.String("cluster", clusterID))

	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Ingresses().
		List().
		Page(1).
		Size(100).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	ingresses := make([]*Ingress, 0, response.Total())
	response.Items().Each(func(i *cmv1.Ingress) bool {
		ingresses = append(ingresses, convertIngress(i))
		return true
	})

	s.logger.InfoContext(ctx, "found ingresses", slog.Int("count", len(ingresses)))
	return ingresses, nil
}

// Get retrieves a specific ingress
func (s *service) Get(ctx context.Context, clusterID string, ingressID string) (*Ingress, error) {
	s.logger.InfoContext(ctx, "getting ingress",
		slog.String("cluster", clusterID),
		slog.String("ingress", ingressID))

	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Ingresses().Ingress(ingressID).
		Get().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ingress: %w", err)
	}

	return convertIngress(response.Body()), nil
}

// Update updates an existing ingress
func (s *service) Update(ctx context.Context, clusterID string, ingressID string, config UpdateConfig) (*Ingress, error) {
	s.logger.InfoContext(ctx, "updating ingress",
		slog.String("cluster", clusterID),
		slog.String("ingress", ingressID))

	// Get existing ingress first
	existing, err := s.Get(ctx, clusterID, ingressID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing ingress: %w", err)
	}

	// Build the update
	builder := cmv1.NewIngress().ID(ingressID)

	// Update listening method if specified
	if config.Private != nil {
		if *config.Private {
			builder.Listening(cmv1.ListeningMethodInternal)
		} else {
			builder.Listening(cmv1.ListeningMethodExternal)
		}
	} else {
		// Keep existing
		builder.Listening(cmv1.ListeningMethod(existing.Listening))
	}

	// Update load balancer type if specified
	if config.LoadBalancerType != nil {
		builder.LoadBalancerType(cmv1.LoadBalancerFlavor(*config.LoadBalancerType))
	} else {
		// Keep existing
		builder.LoadBalancerType(cmv1.LoadBalancerFlavor(existing.LoadBalancerType))
	}

	// Update route selectors if specified
	if config.RouteSelectors != nil {
		builder.RouteSelectors(config.RouteSelectors)
	} else {
		// Keep existing
		if len(existing.RouteSelectors) > 0 {
			builder.RouteSelectors(existing.RouteSelectors)
		}
	}

	ingress, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build ingress update: %w", err)
	}

	// Update via API
	response, err := s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Ingresses().Ingress(ingressID).
		Update().Body(ingress).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update ingress: %w", err)
	}

	result := convertIngress(response.Body())
	s.logger.InfoContext(ctx, "ingress updated successfully",
		slog.String("id", result.ID))

	return result, nil
}

// Delete deletes an ingress
func (s *service) Delete(ctx context.Context, clusterID string, ingressID string) error {
	s.logger.InfoContext(ctx, "deleting ingress",
		slog.String("cluster", clusterID),
		slog.String("ingress", ingressID))

	// Check if this is the default ingress
	ingress, err := s.Get(ctx, clusterID, ingressID)
	if err != nil {
		return fmt.Errorf("failed to get ingress: %w", err)
	}

	if ingress.Default {
		return fmt.Errorf("cannot delete default ingress")
	}

	_, err = s.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		Ingresses().Ingress(ingressID).
		Delete().
		SendContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete ingress: %w", err)
	}

	s.logger.InfoContext(ctx, "ingress deleted successfully", slog.String("ingress", ingressID))
	return nil
}

// convertIngress converts from OCM SDK type to our domain type
func convertIngress(i *cmv1.Ingress) *Ingress {
	if i == nil {
		return nil
	}

	ingress := &Ingress{
		ID:               i.ID(),
		Default:          i.Default(),
		RouteSelectors:   i.RouteSelectors(),
	}

	// Set listening method
	switch i.Listening() {
	case cmv1.ListeningMethodInternal:
		ingress.Listening = ListeningInternal
	case cmv1.ListeningMethodExternal:
		ingress.Listening = ListeningExternal
	default:
		ingress.Listening = ListeningExternal
	}

	// Set load balancer type
	switch i.LoadBalancerType() {
	case cmv1.LoadBalancerFlavorNlb:
		ingress.LoadBalancerType = LoadBalancerNLB
	case cmv1.LoadBalancerFlavorClassic:
		ingress.LoadBalancerType = LoadBalancerClassic
	default:
		ingress.LoadBalancerType = LoadBalancerClassic
	}

	// Set DNS name if available
	if i.DNSName() != "" {
		ingress.DNSName = i.DNSName()
	}

	return ingress
}

// ParseRouteSelectors parses route selector string into map
func ParseRouteSelectors(labelMatches string) (map[string]string, error) {
	routeSelectors := make(map[string]string)
	
	if labelMatches == "" {
		return routeSelectors, nil
	}

	for _, labelMatch := range strings.Split(labelMatches, ",") {
		if !strings.Contains(labelMatch, "=") {
			return nil, fmt.Errorf("invalid label match: %s (must be in key=value format)", labelMatch)
		}
		
		tokens := strings.Split(labelMatch, "=")
		if len(tokens) != 2 || tokens[0] == "" || tokens[1] == "" {
			return nil, fmt.Errorf("invalid label match: %s", labelMatch)
		}
		
		routeSelectors[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
	}

	return routeSelectors, nil
}

// FormatRouteSelectors formats route selectors map for display
func FormatRouteSelectors(selectors map[string]string) string {
	if len(selectors) == 0 {
		return ""
	}

	var parts []string
	for k, v := range selectors {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ", ")
}
