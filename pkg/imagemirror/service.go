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

package imagemirror

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/logging"
)

// service implements the ImageMirrorService interface combining IDMS and ITMS
type service struct {
	idms   IDMS
	itms   ITMS
	config *ImageMirrorConfig
	logger *logrus.Logger
}

// NewImageMirrorService creates a new image mirror service combining IDMS and ITMS functionality
func NewImageMirrorService(config *ImageMirrorConfig) ImageMirrorService {
	if config == nil {
		config = &ImageMirrorConfig{
			Timeout:        30 * time.Second,
			MaxConcurrency: 5,
		}
	}

	logger := logging.NewLogger()
	
	return &service{
		idms:   NewIDMS(config),
		itms:   NewITMS(config),
		config: config,
		logger: logger,
	}
}

// NewImageMirrorServiceWithAuth creates a new image mirror service with authentication
func NewImageMirrorServiceWithAuth(sourceRegistry, targetRegistry string, auth AuthConfig) ImageMirrorService {
	config := &ImageMirrorConfig{
		SourceRegistry: sourceRegistry,
		TargetRegistry: targetRegistry,
		Authentication: auth,
		Timeout:        30 * time.Second,
		MaxConcurrency: 5,
	}

	return NewImageMirrorService(config)
}

// IDMS methods

// ListImages lists all images in a registry
func (s *service) ListImages(ctx context.Context, registry string) ([]ImageManifest, error) {
	s.logger.Infof("Listing images in registry: %s", registry)
	return s.idms.ListImages(ctx, registry)
}

// GetImageInfo retrieves detailed information about a specific image
func (s *service) GetImageInfo(ctx context.Context, imageRef string) (*ImageManifest, error) {
	s.logger.Infof("Getting image info for: %s", imageRef)
	return s.idms.GetImageInfo(ctx, imageRef)
}

// ValidateImageAccess validates access to an image
func (s *service) ValidateImageAccess(ctx context.Context, imageRef string) error {
	s.logger.Infof("Validating access to image: %s", imageRef)
	return s.idms.ValidateImageAccess(ctx, imageRef)
}

// ITMS methods

// SyncImage synchronizes a single image from source to target
func (s *service) SyncImage(ctx context.Context, sourceRef, targetRef string) (*SyncResult, error) {
	s.logger.Infof("Syncing image from %s to %s", sourceRef, targetRef)
	return s.itms.SyncImage(ctx, sourceRef, targetRef)
}

// SyncImages synchronizes multiple images from source to target
func (s *service) SyncImages(ctx context.Context, images map[string]string) ([]SyncResult, error) {
	s.logger.Infof("Starting batch sync of %d images", len(images))
	return s.itms.SyncImages(ctx, images)
}

// GetSyncStatus retrieves the status of an ongoing sync operation
func (s *service) GetSyncStatus(ctx context.Context, operationID string) (*SyncStatus, error) {
	return s.itms.GetSyncStatus(ctx, operationID)
}

// Additional utility methods

// ValidateConfiguration validates the service configuration
func (s *service) ValidateConfiguration() error {
	if s.config == nil {
		return fmt.Errorf("configuration is required")
	}

	if s.config.MaxConcurrency < 0 {
		return fmt.Errorf("max concurrency cannot be negative")
	}

	if s.config.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	// Validate authentication if provided
	auth := s.config.Authentication
	if auth.Username != "" && auth.Password == "" && auth.Token == "" {
		return fmt.Errorf("password or token required when username is specified")
	}

	s.logger.Debug("Configuration validation successful")
	return nil
}

// MirrorImageSet mirrors a predefined set of images from source to target registry
func (s *service) MirrorImageSet(ctx context.Context, imageSet []string) ([]SyncResult, error) {
	if len(imageSet) == 0 {
		return nil, fmt.Errorf("image set cannot be empty")
	}

	if s.config.SourceRegistry == "" || s.config.TargetRegistry == "" {
		return nil, fmt.Errorf("source and target registries must be configured")
	}

	// Build image mapping
	images := make(map[string]string)
	for _, image := range imageSet {
		sourceRef := fmt.Sprintf("%s/%s", s.config.SourceRegistry, image)
		targetRef := fmt.Sprintf("%s/%s", s.config.TargetRegistry, image)
		images[sourceRef] = targetRef
	}

	s.logger.Infof("Mirroring %d images from %s to %s", 
		len(images), s.config.SourceRegistry, s.config.TargetRegistry)

	return s.SyncImages(ctx, images)
}

// GetConfiguration returns the current configuration
func (s *service) GetConfiguration() *ImageMirrorConfig {
	// Return a copy to prevent external modification
	config := *s.config
	return &config
}

// Close cleans up resources
func (s *service) Close() error {
	s.logger.Info("Shutting down image mirror service")
	// In a real implementation, this would clean up any active connections,
	// cancel ongoing operations, etc.
	return nil
}