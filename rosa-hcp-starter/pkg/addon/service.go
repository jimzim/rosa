package addon

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	asv1 "github.com/openshift-online/ocm-sdk-go/addonsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	sdk "github.com/openshift-online/ocm-sdk-go"
)

// BillingModel represents the billing model for an add-on
type BillingModel string

const (
	BillingModelStandard       BillingModel = "standard"
	BillingModelMarketplace    BillingModel = "marketplace"
	BillingModelMarketplaceAWS BillingModel = "marketplace-aws"
)

// InstallationState represents the state of an add-on installation
type InstallationState string

const (
	StateInstalling InstallationState = "installing"
	StateReady      InstallationState = "ready"
	StateDeleting   InstallationState = "deleting"
	StateFailed     InstallationState = "failed"
	StateNotInstalled InstallationState = "not installed"
)

// Service defines the interface for Add-on operations
type Service interface {
	Install(ctx context.Context, clusterID string, config InstallConfig) (*Installation, error)
	Uninstall(ctx context.Context, clusterID, addonID string) error
	List(ctx context.Context, clusterID string) ([]*ClusterAddOn, error)
	ListAvailable(ctx context.Context) ([]*AddOn, error)
	Get(ctx context.Context, clusterID, addonID string) (*Installation, error)
	GetAddOn(ctx context.Context, addonID string) (*AddOn, error)
}

// service implements the Service interface
type service struct {
	logger *slog.Logger
	ocm    *sdk.Connection
}

// NewService creates a new Add-on service
func NewService(ctx context.Context, logger *slog.Logger, ocm *sdk.Connection) (Service, error) {
	return &service{
		logger: logger,
		ocm:    ocm,
	}, nil
}

// InstallConfig holds add-on installation configuration
type InstallConfig struct {
	AddonID              string
	BillingModel         BillingModel
	BillingAccountID     string
	Parameters           map[string]string
	OperatorRolePrefix   string // For STS-enabled add-ons
}

// Installation represents an installed add-on
type Installation struct {
	ID               string
	AddonID          string
	AddonName        string
	State            InstallationState
	StateDescription string
	Parameters       map[string]string
	Version          string
}

// AddOn represents an available add-on
type AddOn struct {
	ID                  string
	Name                string
	Description         string
	DocsLink            string
	Version             string
	RequiresSTS         bool
	CredentialsRequests []string
	Available           bool
	HasParameters       bool
}

// ClusterAddOn represents an add-on's status on a cluster
type ClusterAddOn struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	State     InstallationState `json:"state"`
	Available bool              `json:"available"`
}

// Install installs an add-on to a cluster
func (s *service) Install(ctx context.Context, clusterID string, config InstallConfig) (*Installation, error) {
	s.logger.InfoContext(ctx, "installing add-on",
		slog.String("cluster", clusterID),
		slog.String("addon", config.AddonID))

	// Build the installation
	builder := asv1.NewAddonInstallation().
		Addon(asv1.NewAddon().ID(config.AddonID))

	// Add parameters if provided
	if len(config.Parameters) > 0 {
		paramList := make([]*asv1.AddonInstallationParameterBuilder, 0, len(config.Parameters))
		for key, value := range config.Parameters {
			paramList = append(paramList, 
				asv1.NewAddonInstallationParameter().Id(key).Value(value))
		}
		builder.Parameters(asv1.NewAddonInstallationParameterList().Items(paramList...))
	}

	// Set billing model
	billingModel := getBillingModel(config.BillingModel)
	billingBuilder := asv1.NewAddonInstallationBilling().
		BillingModel(billingModel)
	
	if config.BillingAccountID != "" {
		billingBuilder.BillingMarketplaceAccount(config.BillingAccountID)
	}
	
	builder.Billing(billingBuilder)

	installation, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build add-on installation: %w", err)
	}

	// Install via API
	response, err := s.ocm.AddonsMgmt().V1().
		Clusters().Cluster(clusterID).
		Addons().
		Add().Body(installation).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to install add-on: %w", err)
	}

	result := convertInstallation(response.Body())
	s.logger.InfoContext(ctx, "add-on installation initiated",
		slog.String("addon", result.AddonID),
		slog.String("state", string(result.State)))

	return result, nil
}

// Uninstall uninstalls an add-on from a cluster
func (s *service) Uninstall(ctx context.Context, clusterID, addonID string) error {
	s.logger.InfoContext(ctx, "uninstalling add-on",
		slog.String("cluster", clusterID),
		slog.String("addon", addonID))

	_, err := s.ocm.AddonsMgmt().V1().
		Clusters().Cluster(clusterID).
		Addons().Addon(addonID).
		Delete().
		SendContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to uninstall add-on: %w", err)
	}

	s.logger.InfoContext(ctx, "add-on uninstall initiated", slog.String("addon", addonID))
	return nil
}

// List lists all add-ons for a cluster with their installation status
func (s *service) List(ctx context.Context, clusterID string) ([]*ClusterAddOn, error) {
	s.logger.InfoContext(ctx, "listing cluster add-ons", slog.String("cluster", clusterID))

	// Get available add-ons
	available, err := s.ListAvailable(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list available add-ons: %w", err)
	}

	// Get installed add-ons
	installResponse, err := s.ocm.AddonsMgmt().V1().
		Clusters().Cluster(clusterID).
		Addons().
		List().
		Page(1).
		Size(100).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list installed add-ons: %w", err)
	}

	// Build a map of installed add-ons
	installed := make(map[string]*asv1.AddonInstallation)
	installResponse.Items().Each(func(i *asv1.AddonInstallation) bool {
		if i.Addon() != nil {
			installed[i.Addon().ID()] = i
		}
		return true
	})

	// Combine available and installed information
	var clusterAddOns []*ClusterAddOn
	for _, addon := range available {
		ca := &ClusterAddOn{
			ID:        addon.ID,
			Name:      addon.Name,
			State:     StateNotInstalled,
			Available: addon.Available,
		}

		// Check if installed
		if installation, ok := installed[addon.ID]; ok {
			state := installation.State()
			if state != "" {
				ca.State = InstallationState(state)
			} else {
				ca.State = StateInstalling
			}
		}

		clusterAddOns = append(clusterAddOns, ca)
	}

	s.logger.InfoContext(ctx, "found cluster add-ons", slog.Int("count", len(clusterAddOns)))
	return clusterAddOns, nil
}

// ListAvailable lists all available add-ons
func (s *service) ListAvailable(ctx context.Context) ([]*AddOn, error) {
	s.logger.InfoContext(ctx, "listing available add-ons")

	response, err := s.ocm.AddonsMgmt().V1().
		Addons().
		List().
		Page(1).
		Size(100).
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list available add-ons: %w", err)
	}

	var addons []*AddOn
	response.Items().Each(func(a *asv1.Addon) bool {
		addon := convertAddOn(a)
		addons = append(addons, addon)
		return true
	})

	s.logger.InfoContext(ctx, "found available add-ons", slog.Int("count", len(addons)))
	return addons, nil
}

// Get retrieves a specific add-on installation
func (s *service) Get(ctx context.Context, clusterID, addonID string) (*Installation, error) {
	s.logger.InfoContext(ctx, "getting add-on installation",
		slog.String("cluster", clusterID),
		slog.String("addon", addonID))

	response, err := s.ocm.AddonsMgmt().V1().
		Clusters().Cluster(clusterID).
		Addons().Addon(addonID).
		Get().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get add-on installation: %w", err)
	}

	return convertInstallation(response.Body()), nil
}

// GetAddOn retrieves details about a specific add-on
func (s *service) GetAddOn(ctx context.Context, addonID string) (*AddOn, error) {
	s.logger.InfoContext(ctx, "getting add-on details", slog.String("addon", addonID))

	response, err := s.ocm.AddonsMgmt().V1().
		Addons().Addon(addonID).
		Get().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get add-on: %w", err)
	}

	return convertAddOn(response.Body()), nil
}

// Helper functions

func getBillingModel(model BillingModel) asv1.BillingModel {
	switch model {
	case BillingModelMarketplace:
		return asv1.BillingModelMarketplace
	case BillingModelMarketplaceAWS:
		return asv1.BillingModelMarketplaceAws
	default:
		return asv1.BillingModelStandard
	}
}

func convertInstallation(i *asv1.AddonInstallation) *Installation {
	if i == nil {
		return nil
	}

	installation := &Installation{
		ID:    i.ID(),
		State: StateInstalling,
	}

	// Get add-on details
	if i.Addon() != nil {
		installation.AddonID = i.Addon().ID()
		installation.AddonName = i.Addon().Name()
		if i.Addon().Version() != nil {
			installation.Version = i.Addon().Version().ID()
		}
	}

	// Get state
	if i.State() != "" {
		installation.State = InstallationState(i.State())
	}
	
	if i.StateDescription() != "" {
		installation.StateDescription = i.StateDescription()
	}

	// Get parameters
	if i.Parameters() != nil && i.Parameters().Items() != nil {
		installation.Parameters = make(map[string]string)
		for _, p := range i.Parameters().Items() {
			installation.Parameters[p.Id()] = p.Value()
		}
	}

	return installation
}

func convertAddOn(a *asv1.Addon) *AddOn {
	if a == nil {
		return nil
	}

	addon := &AddOn{
		ID:          a.ID(),
		Name:        a.Name(),
		Description: a.Description(),
		DocsLink:    a.DocsLink(),
		Available:   a.Enabled(),
	}

	// Check if add-on requires STS
	if a.CredentialsRequests() != nil && len(a.CredentialsRequests()) > 0 {
		addon.RequiresSTS = true
		for _, cr := range a.CredentialsRequests() {
			addon.CredentialsRequests = append(addon.CredentialsRequests, cr.Name())
		}
	}

	// Check if add-on has parameters
	if a.Parameters() != nil && a.Parameters().Items() != nil {
		addon.HasParameters = len(a.Parameters().Items()) > 0
	}

	// Get version
	if a.Version() != nil {
		addon.Version = a.Version().ID()
	}

	return addon
}

// GetClusterInfo retrieves basic cluster information needed for add-on operations
func GetClusterInfo(cluster *cmv1.Cluster) (isSTS bool, operatorRolePrefix string, accountID string) {
	if cluster.AWS() != nil && cluster.AWS().STS() != nil {
		sts := cluster.AWS().STS()
		if sts.RoleARN() != "" {
			isSTS = true
			// Extract operator role prefix from the role ARN
			// Format: arn:aws:iam::ACCOUNT:role/PREFIX-Installer-Role
			roleARN := sts.RoleARN()
			parts := strings.Split(roleARN, "/")
			if len(parts) >= 2 {
				roleName := parts[len(parts)-1]
				if strings.HasSuffix(roleName, "-Installer-Role") {
					operatorRolePrefix = strings.TrimSuffix(roleName, "-Installer-Role")
				}
			}
			// Extract account ID
			arnParts := strings.Split(roleARN, ":")
			if len(arnParts) >= 5 {
				accountID = arnParts[4]
			}
		}
	}
	return
}

// FormatParameters formats add-on parameters for display
func FormatParameters(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	var parts []string
	for k, v := range params {
		// Mask sensitive values
		displayValue := v
		if strings.Contains(strings.ToLower(k), "password") || 
		   strings.Contains(strings.ToLower(k), "secret") ||
		   strings.Contains(strings.ToLower(k), "token") {
			displayValue = "****"
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, displayValue))
	}
	return strings.Join(parts, ", ")
}
