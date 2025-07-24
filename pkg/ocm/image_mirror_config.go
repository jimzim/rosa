/**
Copyright (c) 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocm

import (
	"encoding/json"
	"fmt"
	"strings"
)

// validateImageMirrorSource validates the source and mirrors for image mirror configuration
func validateImageMirrorSource(source string, mirrors []string) error {
	if source == "" {
		return fmt.Errorf("source cannot be empty")
	}

	if len(mirrors) == 0 {
		return fmt.Errorf("at least one mirror must be specified")
	}

	// Basic validation for source format
	if !isValidRegistryAddress(source) {
		return fmt.Errorf("invalid source registry address: %s", source)
	}

	// Validate each mirror
	for _, mirror := range mirrors {
		if mirror == "" {
			return fmt.Errorf("mirror cannot be empty")
		}
		if !isValidRegistryAddress(mirror) {
			return fmt.Errorf("invalid mirror registry address: %s", mirror)
		}
	}

	return nil
}

// isValidRegistryAddress performs basic validation on registry addresses
func isValidRegistryAddress(address string) bool {
	if address == "" {
		return false
	}

	// Remove any protocol prefix if present
	address = strings.TrimPrefix(address, "https://")
	address = strings.TrimPrefix(address, "http://")

	// Basic checks for valid registry format
	// Should contain at least a hostname, optionally with port and path
	parts := strings.Split(address, "/")
	if len(parts) == 0 {
		return false
	}

	hostname := parts[0]
	if hostname == "" {
		return false
	}

	// Should contain at least one dot or be localhost
	if !strings.Contains(hostname, ".") && hostname != "localhost" && !strings.Contains(hostname, ":") {
		return false
	}

	return true
}

// ParseImageMirrorSpec parses a string specification for image mirrors
// Format: "source=mirror1,mirror2,mirror3"
func ParseImageMirrorSpec(spec string) (ImageMirrorSet, error) {
	if spec == "" {
		return ImageMirrorSet{}, fmt.Errorf("image mirror specification cannot be empty")
	}

	parts := strings.SplitN(spec, "=", 2)
	if len(parts) != 2 {
		return ImageMirrorSet{}, fmt.Errorf("invalid format, expected 'source=mirror1,mirror2'")
	}

	source := strings.TrimSpace(parts[0])
	mirrorsStr := strings.TrimSpace(parts[1])

	if source == "" {
		return ImageMirrorSet{}, fmt.Errorf("source cannot be empty")
	}

	mirrors := strings.Split(mirrorsStr, ",")
	for i, mirror := range mirrors {
		mirrors[i] = strings.TrimSpace(mirror)
	}

	// Remove empty mirrors
	filteredMirrors := make([]string, 0, len(mirrors))
	for _, mirror := range mirrors {
		if mirror != "" {
			filteredMirrors = append(filteredMirrors, mirror)
		}
	}

	if len(filteredMirrors) == 0 {
		return ImageMirrorSet{}, fmt.Errorf("at least one mirror must be specified")
	}

	return ImageMirrorSet{
		Source:  source,
		Mirrors: filteredMirrors,
	}, nil
}

// ValidateImageMirrorConfig validates all image mirror configurations in the spec
func ValidateImageMirrorConfig(spec Spec) error {
	// Validate Image Content Sources
	for i, ics := range spec.ImageContentSources {
		if err := validateImageMirrorSource(ics.Source, ics.Mirrors); err != nil {
			return fmt.Errorf("invalid image content source %d: %w", i, err)
		}
	}

	// Validate Image Digest Mirror Set
	for i, idms := range spec.ImageDigestMirrorSet {
		if err := validateImageMirrorSource(idms.Source, idms.Mirrors); err != nil {
			return fmt.Errorf("invalid image digest mirror set %d: %w", i, err)
		}
	}

	// Validate Image Tag Mirror Set
	for i, itms := range spec.ImageTagMirrorSet {
		if err := validateImageMirrorSource(itms.Source, itms.Mirrors); err != nil {
			return fmt.Errorf("invalid image tag mirror set %d: %w", i, err)
		}
	}

	return nil
}

// BuildImageMirrorProperties builds custom properties for image mirror configuration in HCP clusters
func BuildImageMirrorProperties(spec Spec) (map[string]string, error) {
	props := make(map[string]string)

	// Validate all image mirror configurations first
	if err := ValidateImageMirrorConfig(spec); err != nil {
		return nil, err
	}

	// Add Image Content Source Policies (ICSP)
	if len(spec.ImageContentSources) > 0 {
		icspData, err := serializeImageMirrorConfig("imageContentSources", spec.ImageContentSources)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize image content sources: %w", err)
		}
		props["hypershift.openshift.io/image-content-sources"] = icspData
	}

	// Add Image Digest Mirror Set (IDMS)
	if len(spec.ImageDigestMirrorSet) > 0 {
		idmsData, err := serializeImageMirrorConfig("imageDigestMirrorSet", spec.ImageDigestMirrorSet)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize image digest mirror set: %w", err)
		}
		props["hypershift.openshift.io/image-digest-mirror-set"] = idmsData
	}

	// Add Image Tag Mirror Set (ITMS)
	if len(spec.ImageTagMirrorSet) > 0 {
		itmsData, err := serializeImageMirrorConfig("imageTagMirrorSet", spec.ImageTagMirrorSet)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize image tag mirror set: %w", err)
		}
		props["hypershift.openshift.io/image-tag-mirror-set"] = itmsData
	}

	return props, nil
}

// serializeImageMirrorConfig serializes image mirror configuration to JSON string
func serializeImageMirrorConfig(configType string, mirrors interface{}) (string, error) {
	data := map[string]interface{}{
		"type":    configType,
		"mirrors": mirrors,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal %s configuration: %w", configType, err)
	}

	return string(jsonData), nil
}