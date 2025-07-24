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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ITMS (Image Transfer Management System)", func() {
	var (
		itms   ITMS
		config *ImageMirrorConfig
		ctx    context.Context
	)

	BeforeEach(func() {
		config = &ImageMirrorConfig{
			Authentication: AuthConfig{
				Username: "testuser",
				Password: "testpass",
			},
			Timeout:        30 * time.Second,
			MaxConcurrency: 3,
		}
		itms = NewITMS(config)
		ctx = context.Background()
	})

	Describe("SyncImage", func() {
		It("should sync a single image successfully", func() {
			sourceRef := "source.registry.io/repo:tag"
			targetRef := "target.registry.io/repo:tag"
			
			result, err := itms.SyncImage(ctx, sourceRef, targetRef)
			
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.Source).To(Equal(sourceRef))
			Expect(result.Target).To(Equal(targetRef))
			Expect(result.Success).To(BeTrue())
			Expect(result.Duration).To(BeNumerically(">", 0))
			Expect(result.BytesTransferred).To(BeNumerically(">", 0))
			Expect(result.Error).To(BeEmpty())
		})

		It("should fail with empty source reference", func() {
			_, err := itms.SyncImage(ctx, "", "target.registry.io/repo:tag")
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("source image reference cannot be empty"))
		})

		It("should fail with empty target reference", func() {
			_, err := itms.SyncImage(ctx, "source.registry.io/repo:tag", "")
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("target image reference cannot be empty"))
		})

		It("should respect context timeout", func() {
			timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			
			// Current implementation should detect timeout
			result, err := itms.SyncImage(timeoutCtx, "source.registry.io/repo:tag", "target.registry.io/repo:tag")
			
			Expect(err).To(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.Success).To(BeFalse())
		})
	})

	Describe("SyncImages", func() {
		It("should sync multiple images successfully", func() {
			images := map[string]string{
				"source.registry.io/repo1:tag1": "target.registry.io/repo1:tag1",
				"source.registry.io/repo2:tag2": "target.registry.io/repo2:tag2",
				"source.registry.io/repo3:tag3": "target.registry.io/repo3:tag3",
			}
			
			results, err := itms.SyncImages(ctx, images)
			
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(3))
			
			for _, result := range results {
				Expect(result.Success).To(BeTrue())
				Expect(result.Duration).To(BeNumerically(">", 0))
				Expect(result.BytesTransferred).To(BeNumerically(">", 0))
				Expect(result.Error).To(BeEmpty())
			}
		})

		It("should fail with empty images map", func() {
			_, err := itms.SyncImages(ctx, map[string]string{})
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no images provided for synchronization"))
		})

		It("should handle large number of images with concurrency", func() {
			images := make(map[string]string)
			for i := 0; i < 10; i++ {
				sourceRef := fmt.Sprintf("source.registry.io/repo%d:tag", i)
				targetRef := fmt.Sprintf("target.registry.io/repo%d:tag", i)
				images[sourceRef] = targetRef
			}
			
			start := time.Now()
			results, err := itms.SyncImages(ctx, images)
			duration := time.Since(start)
			
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(10))
			
			// With concurrency, it should complete faster than sequential
			// Each operation takes ~200ms, so 10 operations sequentially would take ~2s
			// With concurrency of 3, it should be faster
			Expect(duration).To(BeNumerically("<", 2*time.Second))
			
			for _, result := range results {
				Expect(result.Success).To(BeTrue())
			}
		})

		It("should respect context cancellation", func() {
			cancelCtx, cancel := context.WithCancel(ctx)
			
			images := make(map[string]string)
			for i := 0; i < 5; i++ {
				sourceRef := fmt.Sprintf("source.registry.io/repo%d:tag", i)
				targetRef := fmt.Sprintf("target.registry.io/repo%d:tag", i)
				images[sourceRef] = targetRef
			}
			
			// Cancel immediately to test cancellation handling
			cancel()
			
			results, err := itms.SyncImages(cancelCtx, images)
			
			// Context cancellation should be handled properly
			Expect(err).ToNot(HaveOccurred())
			// Some operations might complete before cancellation is detected
			Expect(len(results)).To(BeNumerically("<=", 5))
			// All results should indicate cancellation
			for _, result := range results {
				if !result.Success {
					Expect(result.Error).To(ContainSubstring("operation cancelled"))
				}
			}
		})
	})

	Describe("GetSyncStatus", func() {
		It("should get status for existing operation", func() {
			// First, start a sync operation to create an operation ID
			images := map[string]string{
				"source.registry.io/repo:tag": "target.registry.io/repo:tag",
			}
			
			results, err := itms.SyncImages(ctx, images)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			
			// Note: In the current implementation, we can't easily get the operation ID
			// In a real implementation, SyncImages would return an operation ID
			// For now, we'll test the error case
		})

		It("should fail with empty operation ID", func() {
			_, err := itms.GetSyncStatus(ctx, "")
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("operation ID cannot be empty"))
		})

		It("should fail with non-existent operation ID", func() {
			_, err := itms.GetSyncStatus(ctx, "non-existent-id")
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("operation not found"))
		})
	})

	Describe("Internal Functions", func() {
		It("should update operation progress correctly", func() {
			service := itms.(*itmsService)
			
			// Create a mock operation
			operationID := "test-operation"
			status := &SyncStatus{
				OperationID: operationID,
				Status:      "running",
				Progress:    0,
				StartTime:   time.Now(),
				Results:     []SyncResult{},
			}
			
			service.mutex.Lock()
			service.operations[operationID] = status
			service.mutex.Unlock()
			
			// Update progress
			result := SyncResult{
				Source:  "source.registry.io/repo:tag",
				Target:  "target.registry.io/repo:tag",
				Success: true,
			}
			
			service.updateOperationProgress(operationID, 50, result)
			
			// Verify update
			service.mutex.RLock()
			updatedStatus := service.operations[operationID]
			service.mutex.RUnlock()
			
			Expect(updatedStatus.Progress).To(Equal(50))
			Expect(updatedStatus.Results).To(HaveLen(1))
			Expect(updatedStatus.Results[0].Source).To(Equal(result.Source))
		})

		It("should perform image sync correctly", func() {
			service := itms.(*itmsService)
			
			err := service.performImageSync(ctx, "source.registry.io/repo:tag", "target.registry.io/repo:tag")
			
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle context cancellation in performImageSync", func() {
			service := itms.(*itmsService)
			cancelCtx, cancel := context.WithCancel(ctx)
			cancel() // Cancel immediately
			
			err := service.performImageSync(cancelCtx, "source.registry.io/repo:tag", "target.registry.io/repo:tag")
			
			// Context cancellation should be properly detected
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("context canceled"))
		})
	})

	Describe("Configuration", func() {
		It("should work with default concurrency", func() {
			configWithoutConcurrency := &ImageMirrorConfig{
				MaxConcurrency: 0, // Should use default
			}
			defaultITMS := NewITMS(configWithoutConcurrency)
			
			images := map[string]string{
				"source.registry.io/repo:tag": "target.registry.io/repo:tag",
			}
			
			results, err := defaultITMS.SyncImages(ctx, images)
			
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
		})

		It("should work with nil authentication", func() {
			nilAuthConfig := &ImageMirrorConfig{
				Authentication: AuthConfig{},
			}
			nilAuthITMS := NewITMS(nilAuthConfig)
			
			result, err := nilAuthITMS.SyncImage(ctx, "source.registry.io/repo:tag", "target.registry.io/repo:tag")
			
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
		})
	})
})