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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Error Handling", func() {
	Describe("ImageMirrorError", func() {
		It("should create an error with operation context", func() {
			underlying := errors.New("underlying error")
			err := NewImageMirrorError("test-operation", underlying)
			
			Expect(err.Operation).To(Equal("test-operation"))
			Expect(err.Underlying).To(Equal(underlying))
			Expect(err.Error()).To(ContainSubstring("test-operation"))
			Expect(err.Error()).To(ContainSubstring("underlying error"))
		})

		It("should create an error with image context", func() {
			underlying := errors.New("image error")
			err := NewImageMirrorErrorWithImage("pull", "registry.io/repo:tag", underlying)
			
			Expect(err.Operation).To(Equal("pull"))
			Expect(err.ImageRef).To(Equal("registry.io/repo:tag"))
			Expect(err.Underlying).To(Equal(underlying))
			Expect(err.Error()).To(ContainSubstring("pull"))
			Expect(err.Error()).To(ContainSubstring("registry.io/repo:tag"))
		})

		It("should create an error with registry context", func() {
			underlying := errors.New("registry error")
			err := NewImageMirrorErrorWithRegistry("connect", "registry.io", underlying)
			
			Expect(err.Operation).To(Equal("connect"))
			Expect(err.Registry).To(Equal("registry.io"))
			Expect(err.Underlying).To(Equal(underlying))
			Expect(err.Error()).To(ContainSubstring("connect"))
			Expect(err.Error()).To(ContainSubstring("registry.io"))
		})

		It("should implement error unwrapping", func() {
			underlying := ErrAuthenticationFailed
			err := NewImageMirrorError("auth", underlying)
			
			Expect(errors.Unwrap(err)).To(Equal(underlying))
			Expect(errors.Is(err, ErrAuthenticationFailed)).To(BeTrue())
		})
	})

	Describe("Error Classification", func() {
		It("should identify retryable errors", func() {
			Expect(IsRetryableError(ErrRegistryUnavailable)).To(BeTrue())
			Expect(IsRetryableError(ErrOperationTimeout)).To(BeTrue())
			Expect(IsRetryableError(ErrAuthenticationFailed)).To(BeFalse())
			Expect(IsRetryableError(nil)).To(BeFalse())
		})

		It("should identify authentication errors", func() {
			Expect(IsAuthenticationError(ErrAuthenticationFailed)).To(BeTrue())
			Expect(IsAuthenticationError(ErrInsufficientPermissions)).To(BeTrue())
			Expect(IsAuthenticationError(ErrRegistryUnavailable)).To(BeFalse())
			Expect(IsAuthenticationError(nil)).To(BeFalse())
		})

		It("should identify configuration errors", func() {
			Expect(IsConfigurationError(ErrInvalidConfiguration)).To(BeTrue())
			Expect(IsConfigurationError(ErrInvalidImageReference)).To(BeTrue())
			Expect(IsConfigurationError(ErrAuthenticationFailed)).To(BeFalse())
			Expect(IsConfigurationError(nil)).To(BeFalse())
		})
	})

	Describe("Error Summary", func() {
		It("should create summary for empty error list", func() {
			summary := NewErrorSummary([]error{})
			
			Expect(summary.Total).To(Equal(0))
			Expect(summary.AuthErrors).To(Equal(0))
			Expect(summary.ConfigErrors).To(Equal(0))
			Expect(summary.RetryableErrors).To(Equal(0))
			Expect(summary.OtherErrors).To(Equal(0))
			Expect(summary.String()).To(Equal("No errors"))
		})

		It("should categorize different error types", func() {
			errorList := []error{
				ErrAuthenticationFailed,
				ErrInsufficientPermissions,
				ErrInvalidConfiguration,
				ErrInvalidImageReference,
				ErrRegistryUnavailable,
				ErrOperationTimeout,
				errors.New("other error"),
			}
			
			summary := NewErrorSummary(errorList)
			
			Expect(summary.Total).To(Equal(7))
			Expect(summary.AuthErrors).To(Equal(2))
			Expect(summary.ConfigErrors).To(Equal(2))
			Expect(summary.RetryableErrors).To(Equal(2))
			Expect(summary.OtherErrors).To(Equal(1))
			
			summaryStr := summary.String()
			Expect(summaryStr).To(ContainSubstring("Total errors: 7"))
			Expect(summaryStr).To(ContainSubstring("Auth: 2"))
			Expect(summaryStr).To(ContainSubstring("Config: 2"))
			Expect(summaryStr).To(ContainSubstring("Retryable: 2"))
			Expect(summaryStr).To(ContainSubstring("Other: 1"))
		})

		It("should handle wrapped errors correctly", func() {
			wrapped := NewImageMirrorError("auth", ErrAuthenticationFailed)
			errorList := []error{wrapped}
			
			summary := NewErrorSummary(errorList)
			
			Expect(summary.Total).To(Equal(1))
			Expect(summary.AuthErrors).To(Equal(1))
		})
	})

	Describe("Predefined Errors", func() {
		It("should have correct error messages", func() {
			Expect(ErrInvalidImageReference.Error()).To(Equal("invalid image reference"))
			Expect(ErrAuthenticationFailed.Error()).To(Equal("authentication failed"))
			Expect(ErrRegistryUnavailable.Error()).To(Equal("registry unavailable"))
			Expect(ErrImageNotFound.Error()).To(Equal("image not found"))
			Expect(ErrInsufficientPermissions.Error()).To(Equal("insufficient permissions"))
			Expect(ErrOperationTimeout.Error()).To(Equal("operation timeout"))
			Expect(ErrInvalidConfiguration.Error()).To(Equal("invalid configuration"))
			Expect(ErrOperationCancelled.Error()).To(Equal("operation cancelled"))
		})
	})
})