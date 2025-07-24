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
	"time"
)

// ImageMirrorConfig represents the configuration for image mirroring operations
type ImageMirrorConfig struct {
	// SourceRegistry is the source container registry
	SourceRegistry string `json:"sourceRegistry" yaml:"sourceRegistry"`
	// TargetRegistry is the target container registry
	TargetRegistry string `json:"targetRegistry" yaml:"targetRegistry"`
	// Authentication credentials
	Authentication AuthConfig `json:"authentication" yaml:"authentication"`
	// Timeout for operations
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
	// MaxConcurrency limits concurrent operations
	MaxConcurrency int `json:"maxConcurrency" yaml:"maxConcurrency"`
}

// AuthConfig contains authentication credentials for registries
type AuthConfig struct {
	// Username for registry authentication
	Username string `json:"username" yaml:"username"`
	// Password for registry authentication
	Password string `json:"password" yaml:"password"`
	// Token for token-based authentication
	Token string `json:"token" yaml:"token"`
	// CredentialsFile path to credentials file
	CredentialsFile string `json:"credentialsFile" yaml:"credentialsFile"`
}

// ImageManifest represents an image manifest with metadata
type ImageManifest struct {
	// Repository is the repository name
	Repository string `json:"repository" yaml:"repository"`
	// Tag is the image tag
	Tag string `json:"tag" yaml:"tag"`
	// Digest is the image digest
	Digest string `json:"digest" yaml:"digest"`
	// Size is the image size in bytes
	Size int64 `json:"size" yaml:"size"`
	// LastModified is the last modification time
	LastModified time.Time `json:"lastModified" yaml:"lastModified"`
}

// SyncResult represents the result of a synchronization operation
type SyncResult struct {
	// Source is the source image reference
	Source string `json:"source" yaml:"source"`
	// Target is the target image reference
	Target string `json:"target" yaml:"target"`
	// Success indicates if the operation was successful
	Success bool `json:"success" yaml:"success"`
	// Error contains any error that occurred
	Error string `json:"error,omitempty" yaml:"error,omitempty"`
	// Duration is the time taken for the operation
	Duration time.Duration `json:"duration" yaml:"duration"`
	// BytesTransferred is the number of bytes transferred
	BytesTransferred int64 `json:"bytesTransferred" yaml:"bytesTransferred"`
}

// IDMS (Image Distribution Management System) interface
type IDMS interface {
	// ListImages lists all images in a registry
	ListImages(ctx context.Context, registry string) ([]ImageManifest, error)
	// GetImageInfo retrieves detailed information about a specific image
	GetImageInfo(ctx context.Context, imageRef string) (*ImageManifest, error)
	// ValidateImageAccess validates access to an image
	ValidateImageAccess(ctx context.Context, imageRef string) error
}

// ITMS (Image Transfer Management System) interface
type ITMS interface {
	// SyncImage synchronizes a single image from source to target
	SyncImage(ctx context.Context, sourceRef, targetRef string) (*SyncResult, error)
	// SyncImages synchronizes multiple images from source to target
	SyncImages(ctx context.Context, images map[string]string) ([]SyncResult, error)
	// GetSyncStatus retrieves the status of an ongoing sync operation
	GetSyncStatus(ctx context.Context, operationID string) (*SyncStatus, error)
}

// SyncStatus represents the status of a sync operation
type SyncStatus struct {
	// OperationID is the unique identifier for the operation
	OperationID string `json:"operationId" yaml:"operationId"`
	// Status is the current status of the operation
	Status string `json:"status" yaml:"status"`
	// Progress indicates the completion percentage (0-100)
	Progress int `json:"progress" yaml:"progress"`
	// StartTime is when the operation started
	StartTime time.Time `json:"startTime" yaml:"startTime"`
	// EndTime is when the operation completed (if applicable)
	EndTime *time.Time `json:"endTime,omitempty" yaml:"endTime,omitempty"`
	// Results contains the results of completed syncs
	Results []SyncResult `json:"results" yaml:"results"`
}

// ImageMirrorService combines IDMS and ITMS functionality
type ImageMirrorService interface {
	IDMS
	ITMS
	// ValidateConfiguration validates the service configuration
	ValidateConfiguration() error
	// GetConfiguration returns the current configuration
	GetConfiguration() *ImageMirrorConfig
	// MirrorImageSet mirrors a predefined set of images from source to target registry
	MirrorImageSet(ctx context.Context, imageSet []string) ([]SyncResult, error)
	// Close cleans up resources
	Close() error
}