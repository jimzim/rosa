package clusterregistryconfig

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/openshift/rosa/pkg/ocm"
)

const (
	maxMirrorSets       = 100
	maxMirrorsPerSet    = 50
	minOpenShiftVersion = "4.13"
)

var (
	// Kubernetes naming convention validation
	kubernetesNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?$`)

	// Platform registries that should be protected
	platformRegistries = []string{
		"quay.io",
		"registry.redhat.io",
		"registry.access.redhat.com",
		"registry.connect.redhat.com",
		"cloud.redhat.com",
		"console.redhat.com",
	}
)

func ValidateImageMirrorSetFormat(input interface{}) error {
	switch v := input.(type) {
	case string:
		return validateSingleMirrorSetFormat(v)
	case []string:
		for _, entry := range v {
			if err := validateSingleMirrorSetFormat(entry); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported input type for mirror set validation")
	}
	return nil
}

func validateSingleMirrorSetFormat(entry string) error {
	if entry == "" {
		return nil // Empty entries are allowed and will be filtered out
	}

	// Parse format: "name:source=mirror1,mirror2"
	parts := strings.SplitN(entry, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format: %s (expected 'name:source=mirror1,mirror2')", entry)
	}

	name := parts[0]
	if !kubernetesNameRegex.MatchString(name) {
		return fmt.Errorf("invalid name '%s': must follow Kubernetes naming conventions", name)
	}

	sourceMirrors := strings.SplitN(parts[1], "=", 2)
	if len(sourceMirrors) != 2 {
		return fmt.Errorf("invalid source=mirrors format: %s", parts[1])
	}

	source := sourceMirrors[0]
	if err := validateRegistryFormat(source); err != nil {
		return fmt.Errorf("invalid source registry '%s': %v", source, err)
	}

	mirrors := strings.Split(sourceMirrors[1], ",")
	if len(mirrors) > maxMirrorsPerSet {
		return fmt.Errorf("too many mirrors (%d) for mirror set '%s', maximum is %d",
			len(mirrors), name, maxMirrorsPerSet)
	}

	for _, mirror := range mirrors {
		mirror = strings.TrimSpace(mirror)
		if mirror == "" {
			return fmt.Errorf("empty mirror specified for source %s", source)
		}
		if err := validateRegistryFormat(mirror); err != nil {
			return fmt.Errorf("invalid mirror registry '%s': %v", mirror, err)
		}
	}

	return nil
}

func validateRegistryFormat(registry string) error {
	if registry == "" {
		return fmt.Errorf("registry cannot be empty")
	}

	// Basic URL validation (allow registry.com and registry.com/path formats)
	if strings.Contains(registry, "://") {
		return fmt.Errorf("registry should not include protocol (http:// or https://)")
	}

	// Try to parse as URL to validate format
	if _, err := url.Parse("https://" + registry); err != nil {
		return fmt.Errorf("invalid registry format: %v", err)
	}

	return nil
}

func ValidateImageMirrorSetLimits(idmsList []ocm.ImageDigestMirrorSet, itmsList []ocm.ImageTagMirrorSet) error {
	totalSets := len(idmsList) + len(itmsList)
	if totalSets > maxMirrorSets {
		return fmt.Errorf("too many mirror sets (%d), maximum allowed is %d", totalSets, maxMirrorSets)
	}

	// Check for duplicate names
	nameMap := make(map[string]bool)

	for _, idms := range idmsList {
		if nameMap[idms.Name] {
			return fmt.Errorf("duplicate mirror set name: %s", idms.Name)
		}
		nameMap[idms.Name] = true

		for _, mirror := range idms.Mirrors {
			if len(mirror.MirrorsByDigest) > maxMirrorsPerSet {
				return fmt.Errorf("too many mirrors (%d) for IDMS '%s', maximum is %d",
					len(mirror.MirrorsByDigest), idms.Name, maxMirrorsPerSet)
			}
		}
	}

	for _, itms := range itmsList {
		if nameMap[itms.Name] {
			return fmt.Errorf("duplicate mirror set name: %s", itms.Name)
		}
		nameMap[itms.Name] = true

		for _, mirror := range itms.Mirrors {
			if len(mirror.MirrorsByTag) > maxMirrorsPerSet {
				return fmt.Errorf("too many mirrors (%d) for ITMS '%s', maximum is %d",
					len(mirror.MirrorsByTag), itms.Name, maxMirrorsPerSet)
			}
		}
	}

	return nil
}

func ValidateMirrorSetsCompatibility(idmsList []ocm.ImageDigestMirrorSet, itmsList []ocm.ImageTagMirrorSet) error {
	// Check for conflicts between IDMS and ITMS source registries
	idmsSources := make(map[string]string)
	itmsSources := make(map[string]string)

	for _, idms := range idmsList {
		for _, mirror := range idms.Mirrors {
			idmsSources[mirror.Source] = idms.Name
		}
	}

	for _, itms := range itmsList {
		for _, mirror := range itms.Mirrors {
			if conflictingIDMS, exists := idmsSources[mirror.Source]; exists {
				return fmt.Errorf("source registry '%s' cannot be used in both IDMS '%s' and ITMS '%s'",
					mirror.Source, conflictingIDMS, itms.Name)
			}
			itmsSources[mirror.Source] = itms.Name
		}
	}

	return nil
}

func ValidateMirrorSetsAgainstPlatformRequirements(idmsList []ocm.ImageDigestMirrorSet, itmsList []ocm.ImageTagMirrorSet) error {
	// Ensure platform registries are not blocked or misconfigured
	for _, idms := range idmsList {
		for _, mirror := range idms.Mirrors {
			for _, platformReg := range platformRegistries {
				if strings.HasPrefix(mirror.Source, platformReg) {
					return fmt.Errorf("platform registry '%s' cannot be used as source in IDMS '%s'",
						mirror.Source, idms.Name)
				}
				for _, mirrorReg := range mirror.MirrorsByDigest {
					if strings.HasPrefix(mirrorReg, platformReg) {
						return fmt.Errorf("platform registry '%s' cannot be used as mirror in IDMS '%s'",
							mirrorReg, idms.Name)
					}
				}
			}
		}
	}

	for _, itms := range itmsList {
		for _, mirror := range itms.Mirrors {
			for _, platformReg := range platformRegistries {
				if strings.HasPrefix(mirror.Source, platformReg) {
					return fmt.Errorf("platform registry '%s' cannot be used as source in ITMS '%s'",
						mirror.Source, itms.Name)
				}
				for _, mirrorReg := range mirror.MirrorsByTag {
					if strings.HasPrefix(mirrorReg, platformReg) {
						return fmt.Errorf("platform registry '%s' cannot be used as mirror in ITMS '%s'",
							mirrorReg, itms.Name)
					}
				}
			}
		}
	}

	return nil
}

func ValidateOpenShiftVersionSupport(version string, hasIDMS bool, hasITMS bool) error {
	if !hasIDMS && !hasITMS {
		return nil // No validation needed if not using IDMS/ITMS
	}

	// Extract version number (handle formats like "4.13.1", "openshift-v4.13.1", etc.)
	versionNum := strings.TrimPrefix(version, "openshift-v")
	versionNum = strings.TrimPrefix(versionNum, "v")

	majorMinor := strings.SplitN(versionNum, ".", 3)
	if len(majorMinor) < 2 {
		return fmt.Errorf("invalid version format: %s", version)
	}

	if majorMinor[0] != "4" {
		return fmt.Errorf("IDMS/ITMS requires OpenShift 4.x, got version: %s", version)
	}

	// Simple string comparison for minor version (13, 14, 15, etc.)
	minorVersion := majorMinor[1]
	if len(minorVersion) == 1 && minorVersion < "13" {
		return fmt.Errorf("IDMS/ITMS requires OpenShift 4.13+, got version: %s", version)
	} else if len(minorVersion) == 2 && minorVersion < "13" {
		return fmt.Errorf("IDMS/ITMS requires OpenShift 4.13+, got version: %s", version)
	}

	return nil
}

// ValidateCompleteImageMirrorSetConfiguration performs comprehensive validation
func ValidateCompleteImageMirrorSetConfiguration(idmsList []ocm.ImageDigestMirrorSet, itmsList []ocm.ImageTagMirrorSet, version string) error {
	if err := ValidateImageMirrorSetLimits(idmsList, itmsList); err != nil {
		return err
	}

	if err := ValidateMirrorSetsCompatibility(idmsList, itmsList); err != nil {
		return err
	}

	if err := ValidateMirrorSetsAgainstPlatformRequirements(idmsList, itmsList); err != nil {
		return err
	}

	if err := ValidateOpenShiftVersionSupport(version, len(idmsList) > 0, len(itmsList) > 0); err != nil {
		return err
	}

	return nil
}
