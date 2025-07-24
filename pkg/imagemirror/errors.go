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
	"errors"
	"fmt"
)

// Error types for image mirroring operations
var (
	// ErrInvalidImageReference indicates an invalid image reference format
	ErrInvalidImageReference = errors.New("invalid image reference")
	
	// ErrAuthenticationFailed indicates authentication failure
	ErrAuthenticationFailed = errors.New("authentication failed")
	
	// ErrRegistryUnavailable indicates the registry is not accessible
	ErrRegistryUnavailable = errors.New("registry unavailable")
	
	// ErrImageNotFound indicates the specified image was not found
	ErrImageNotFound = errors.New("image not found")
	
	// ErrInsufficientPermissions indicates insufficient permissions for the operation
	ErrInsufficientPermissions = errors.New("insufficient permissions")
	
	// ErrOperationTimeout indicates the operation timed out
	ErrOperationTimeout = errors.New("operation timeout")
	
	// ErrInvalidConfiguration indicates invalid configuration
	ErrInvalidConfiguration = errors.New("invalid configuration")
	
	// ErrOperationCancelled indicates the operation was cancelled
	ErrOperationCancelled = errors.New("operation cancelled")
)

// ImageMirrorError wraps errors with additional context for image mirroring operations
type ImageMirrorError struct {
	// Operation describes what operation was being performed
	Operation string
	// ImageRef is the image reference that caused the error (if applicable)
	ImageRef string
	// Registry is the registry that caused the error (if applicable)
	Registry string
	// Underlying is the underlying error
	Underlying error
}

// Error implements the error interface
func (e *ImageMirrorError) Error() string {
	if e.ImageRef != "" {
		return fmt.Sprintf("operation '%s' failed for image '%s': %v", e.Operation, e.ImageRef, e.Underlying)
	}
	if e.Registry != "" {
		return fmt.Sprintf("operation '%s' failed for registry '%s': %v", e.Operation, e.Registry, e.Underlying)
	}
	return fmt.Sprintf("operation '%s' failed: %v", e.Operation, e.Underlying)
}

// Unwrap returns the underlying error
func (e *ImageMirrorError) Unwrap() error {
	return e.Underlying
}

// Is implements error matching
func (e *ImageMirrorError) Is(target error) bool {
	return errors.Is(e.Underlying, target)
}

// NewImageMirrorError creates a new ImageMirrorError
func NewImageMirrorError(operation string, err error) *ImageMirrorError {
	return &ImageMirrorError{
		Operation:  operation,
		Underlying: err,
	}
}

// NewImageMirrorErrorWithImage creates a new ImageMirrorError with image context
func NewImageMirrorErrorWithImage(operation, imageRef string, err error) *ImageMirrorError {
	return &ImageMirrorError{
		Operation:  operation,
		ImageRef:   imageRef,
		Underlying: err,
	}
}

// NewImageMirrorErrorWithRegistry creates a new ImageMirrorError with registry context
func NewImageMirrorErrorWithRegistry(operation, registry string, err error) *ImageMirrorError {
	return &ImageMirrorError{
		Operation:  operation,
		Registry:   registry,
		Underlying: err,
	}
}

// IsRetryableError determines if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific retryable errors
	if errors.Is(err, ErrRegistryUnavailable) ||
		errors.Is(err, ErrOperationTimeout) {
		return true
	}

	return false
}

// IsAuthenticationError determines if an error is authentication-related
func IsAuthenticationError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, ErrAuthenticationFailed) ||
		errors.Is(err, ErrInsufficientPermissions)
}

// IsConfigurationError determines if an error is configuration-related
func IsConfigurationError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, ErrInvalidConfiguration) ||
		errors.Is(err, ErrInvalidImageReference)
}

// ErrorSummary provides a summary of multiple errors
type ErrorSummary struct {
	Total          int
	AuthErrors     int
	ConfigErrors   int
	RetryableErrors int
	OtherErrors    int
	Errors         []error
}

// NewErrorSummary creates a new error summary from a list of errors
func NewErrorSummary(errors []error) *ErrorSummary {
	summary := &ErrorSummary{
		Total:  len(errors),
		Errors: errors,
	}

	for _, err := range errors {
		if IsAuthenticationError(err) {
			summary.AuthErrors++
		} else if IsConfigurationError(err) {
			summary.ConfigErrors++
		} else if IsRetryableError(err) {
			summary.RetryableErrors++
		} else {
			summary.OtherErrors++
		}
	}

	return summary
}

// String returns a string representation of the error summary
func (s *ErrorSummary) String() string {
	if s.Total == 0 {
		return "No errors"
	}

	return fmt.Sprintf("Total errors: %d (Auth: %d, Config: %d, Retryable: %d, Other: %d)",
		s.Total, s.AuthErrors, s.ConfigErrors, s.RetryableErrors, s.OtherErrors)
}