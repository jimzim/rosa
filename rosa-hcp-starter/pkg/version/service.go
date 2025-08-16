package version

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa-hcp/pkg/api"
)

// Service provides version operations
type Service struct {
	ocm    api.Client
	logger *slog.Logger
}

// NewService creates a new version service
func NewService(ocm api.Client, logger *slog.Logger) *Service {
	return &Service{
		ocm:    ocm,
		logger: logger.With("service", "version"),
	}
}

// Version represents an OpenShift version
type Version struct {
	ID           string
	RawID        string
	ChannelGroup string
	Version      string
	Default      bool
	Available    bool
	STSOnly      bool
	HCPDefault   bool
	HCPAvailable bool
	ReleaseImage string
	UpgradesFrom []string
	UpgradesTo   []string
}

// ListOptions contains options for listing versions
type ListOptions struct {
	ChannelGroup string
	HCPOnly      bool
	Available    bool
}

// List lists available OpenShift versions
func (s *Service) List(ctx context.Context, opts ListOptions) ([]*Version, error) {
	s.logger.InfoContext(ctx, "listing OpenShift versions",
		slog.String("channel_group", opts.ChannelGroup))

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	// Build the query
	query := conn.ClustersMgmt().V1().Versions().List()

	// Apply filters
	var filters []string
	if opts.ChannelGroup != "" {
		filters = append(filters, fmt.Sprintf("channel_group = '%s'", opts.ChannelGroup))
	}
	if opts.Available {
		filters = append(filters, "enabled = true")
	}
	if opts.HCPOnly {
		filters = append(filters, "hosted_control_plane_enabled = true")
	}

	if len(filters) > 0 {
		query = query.Search(strings.Join(filters, " and "))
	}

	// Set page size
	query = query.Size(100)

	// Execute query
	response, err := query.SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	var versions []*Version
	response.Items().Each(func(v *cmv1.Version) bool {
		version := &Version{
			ID:           v.ID(),
			RawID:        v.RawID(),
			ChannelGroup: v.ChannelGroup(),
			Version:      v.RawID(),
			Default:      v.Default(),
			Available:    v.Enabled(),
			STSOnly:      true, // HCP is always STS
		}

		// Check HCP availability
		if v.HostedControlPlaneEnabled() {
			version.HCPAvailable = true
		}
		if v.HostedControlPlaneDefault() {
			version.HCPDefault = true
		}

		// Get release image if available
		if v.ReleaseImage() != "" {
			version.ReleaseImage = v.ReleaseImage()
		}

		// Get available upgrades
		if v.AvailableUpgrades() != nil {
			version.UpgradesTo = v.AvailableUpgrades()
		}

		versions = append(versions, version)
		return true
	})

	// Sort versions by semantic versioning
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i].Version, versions[j].Version) > 0
	})

	s.logger.InfoContext(ctx, "found versions",
		slog.Int("count", len(versions)))

	return versions, nil
}

// GetLatest gets the latest available version
func (s *Service) GetLatest(ctx context.Context, channelGroup string) (*Version, error) {
	opts := ListOptions{
		ChannelGroup: channelGroup,
		HCPOnly:      true,
		Available:    true,
	}

	versions, err := s.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions available for channel group %s", channelGroup)
	}

	// Find the default version
	for _, v := range versions {
		if v.HCPDefault {
			return v, nil
		}
	}

	// If no default, return the first (newest) version
	return versions[0], nil
}

// GetVersion gets a specific version
func (s *Service) GetVersion(ctx context.Context, versionID string) (*Version, error) {
	s.logger.InfoContext(ctx, "getting version",
		slog.String("version", versionID))

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	response, err := conn.ClustersMgmt().V1().
		Versions().
		Version(versionID).
		Get().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	v := response.Body()

	version := &Version{
		ID:           v.ID(),
		RawID:        v.RawID(),
		ChannelGroup: v.ChannelGroup(),
		Version:      v.RawID(),
		Default:      v.Default(),
		Available:    v.Enabled(),
		STSOnly:      true,
	}

	if v.HostedControlPlaneEnabled() {
		version.HCPAvailable = true
	}
	if v.HostedControlPlaneDefault() {
		version.HCPDefault = true
	}

	if v.AvailableUpgrades() != nil {
		version.UpgradesTo = v.AvailableUpgrades()
	}

	return version, nil
}

// GetUpgradePaths gets available upgrade paths for a cluster
func (s *Service) GetUpgradePaths(ctx context.Context, clusterID string) ([]*Version, error) {
	s.logger.InfoContext(ctx, "getting upgrade paths",
		slog.String("cluster", clusterID))

	conn := s.ocm.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection available")
	}

	// Get the cluster's current version
	cluster, err := conn.ClustersMgmt().V1().
		Clusters().
		Cluster(clusterID).
		Get().
		SendContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	currentVersion := cluster.Body().Version().ID()

	// Get the version details
	version, err := s.GetVersion(ctx, currentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	// Get available upgrades
	var upgrades []*Version
	for _, upgradeID := range version.UpgradesTo {
		upgrade, err := s.GetVersion(ctx, upgradeID)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to get upgrade version",
				slog.String("version", upgradeID),
				slog.String("error", err.Error()))
			continue
		}

		// Only include HCP-compatible versions
		if upgrade.HCPAvailable {
			upgrades = append(upgrades, upgrade)
		}
	}

	return upgrades, nil
}

// compareVersions compares two semantic versions
// Returns 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func compareVersions(v1, v2 string) int {
	// Remove 'openshift-v' prefix if present
	v1 = strings.TrimPrefix(v1, "openshift-v")
	v2 = strings.TrimPrefix(v2, "openshift-v")

	// Split versions
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare major, minor, patch
	for i := 0; i < 3; i++ {
		if i >= len(parts1) && i >= len(parts2) {
			return 0
		}
		if i >= len(parts1) {
			return -1
		}
		if i >= len(parts2) {
			return 1
		}

		// Parse version part (handle pre-release versions like 4.14.0-rc.1)
		var num1, num2 int
		fmt.Sscanf(parts1[i], "%d", &num1)
		fmt.Sscanf(parts2[i], "%d", &num2)

		if num1 > num2 {
			return 1
		}
		if num1 < num2 {
			return -1
		}
	}

	return 0
}
