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
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestImageMirror(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Image Mirror Suite")
}

var _ = Describe("Image Mirror Service", func() {
	var (
		service ImageMirrorService
		config  *ImageMirrorConfig
		ctx     context.Context
	)

	BeforeEach(func() {
		config = &ImageMirrorConfig{
			SourceRegistry: "source.io",
			TargetRegistry: "target.io",
			Authentication: AuthConfig{
				Username: "testuser",
				Password: "testpass",
			},
			Timeout:        30 * time.Second,
			MaxConcurrency: 5,
		}
		service = NewImageMirrorService(config)
		ctx = context.Background()
	})

	AfterEach(func() {
		if service != nil {
			service.Close()
		}
	})

	Describe("Service Configuration", func() {
		It("should create a service with default configuration", func() {
			defaultService := NewImageMirrorService(nil)
			Expect(defaultService).ToNot(BeNil())
			defaultService.Close()
		})

		It("should create a service with authentication", func() {
			authService := NewImageMirrorServiceWithAuth(
				"source.io",
				"target.io",
				AuthConfig{Username: "user", Password: "pass"},
			)
			Expect(authService).ToNot(BeNil())
			authService.Close()
		})

		It("should validate configuration", func() {
			err := service.ValidateConfiguration()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return configuration", func() {
			cfg := service.GetConfiguration()
			Expect(cfg).ToNot(BeNil())
			Expect(cfg.SourceRegistry).To(Equal("source.io"))
			Expect(cfg.TargetRegistry).To(Equal("target.io"))
		})
	})

	Describe("IDMS Functionality", func() {
		It("should list images in a registry", func() {
			images, err := service.ListImages(ctx, "test.registry.io")
			Expect(err).ToNot(HaveOccurred())
			Expect(images).ToNot(BeNil())
		})

		It("should get image information", func() {
			imageRef := "test.registry.io/repo:tag"
			image, err := service.GetImageInfo(ctx, imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(image).ToNot(BeNil())
			Expect(image.Repository).ToNot(BeEmpty())
			Expect(image.Tag).ToNot(BeEmpty())
		})

		It("should validate image access", func() {
			imageRef := "test.registry.io/repo:tag"
			err := service.ValidateImageAccess(ctx, imageRef)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail with empty registry", func() {
			_, err := service.ListImages(ctx, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("registry cannot be empty"))
		})

		It("should fail with empty image reference", func() {
			_, err := service.GetImageInfo(ctx, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("image reference cannot be empty"))
		})
	})

	Describe("ITMS Functionality", func() {
		It("should sync a single image", func() {
			sourceRef := "source.io/repo:tag"
			targetRef := "target.io/repo:tag"
			
			result, err := service.SyncImage(ctx, sourceRef, targetRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.Source).To(Equal(sourceRef))
			Expect(result.Target).To(Equal(targetRef))
			Expect(result.Success).To(BeTrue())
		})

		It("should sync multiple images", func() {
			images := map[string]string{
				"source.io/repo1:tag1": "target.io/repo1:tag1",
				"source.io/repo2:tag2": "target.io/repo2:tag2",
			}
			
			results, err := service.SyncImages(ctx, images)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(2))
			
			for _, result := range results {
				Expect(result.Success).To(BeTrue())
				Expect(result.Duration).To(BeNumerically(">", 0))
			}
		})

		It("should mirror an image set", func() {
			imageSet := []string{"repo1:tag1", "repo2:tag2"}
			
			results, err := service.MirrorImageSet(ctx, imageSet)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(2))
			
			for _, result := range results {
				Expect(result.Success).To(BeTrue())
				Expect(result.Source).To(ContainSubstring("source.io"))
				Expect(result.Target).To(ContainSubstring("target.io"))
			}
		})

		It("should fail with empty source reference", func() {
			_, err := service.SyncImage(ctx, "", "target.io/repo:tag")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("source image reference cannot be empty"))
		})

		It("should fail with empty target reference", func() {
			_, err := service.SyncImage(ctx, "source.io/repo:tag", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("target image reference cannot be empty"))
		})

		It("should fail with empty images map", func() {
			_, err := service.SyncImages(ctx, map[string]string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no images provided"))
		})
	})

	Describe("Error Handling", func() {
		It("should handle invalid configuration", func() {
			invalidConfig := &ImageMirrorConfig{
				MaxConcurrency: -1,
			}
			invalidService := NewImageMirrorService(invalidConfig)
			
			err := invalidService.ValidateConfiguration()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("max concurrency cannot be negative"))
			
			invalidService.Close()
		})

		It("should handle timeout configuration", func() {
			invalidConfig := &ImageMirrorConfig{
				Timeout: -1 * time.Second,
			}
			invalidService := NewImageMirrorService(invalidConfig)
			
			err := invalidService.ValidateConfiguration()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("timeout cannot be negative"))
			
			invalidService.Close()
		})

		It("should handle authentication validation", func() {
			invalidConfig := &ImageMirrorConfig{
				Authentication: AuthConfig{
					Username: "user",
					// Missing password and token
				},
			}
			invalidService := NewImageMirrorService(invalidConfig)
			
			err := invalidService.ValidateConfiguration()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("password or token required"))
			
			invalidService.Close()
		})
	})

	Describe("Context Handling", func() {
		It("should respect context cancellation", func() {
			cancelCtx, cancel := context.WithCancel(ctx)
			cancel() // Cancel immediately
			
			_, err := service.SyncImage(cancelCtx, "source.io/repo:tag", "target.io/repo:tag")
			// In the current simulated implementation, context cancellation is properly detected
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("context canceled"))
		})

		It("should respect timeout", func() {
			timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
			defer cancel()
			
			// Give some time for timeout to trigger
			time.Sleep(10 * time.Millisecond)
			
			_, err := service.SyncImage(timeoutCtx, "source.io/repo:tag", "target.io/repo:tag")
			// In the current implementation, timeout is properly detected
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("context deadline exceeded"))
		})
	})
})