package ocm_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/ocm"
)

var _ = Describe("Image Mirror Config", func() {
	Describe("ParseImageMirrorSpec", func() {
		It("should parse valid mirror specification", func() {
			spec := "registry.redhat.io=mirror1.example.com,mirror2.example.com"
			result, err := ocm.ParseImageMirrorSpec(spec)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Source).To(Equal("registry.redhat.io"))
			Expect(result.Mirrors).To(Equal([]string{"mirror1.example.com", "mirror2.example.com"}))
		})

		It("should handle single mirror", func() {
			spec := "quay.io/openshift-release-dev=mirror.example.com"
			result, err := ocm.ParseImageMirrorSpec(spec)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Source).To(Equal("quay.io/openshift-release-dev"))
			Expect(result.Mirrors).To(Equal([]string{"mirror.example.com"}))
		})

		It("should trim whitespace", func() {
			spec := " registry.redhat.io = mirror1.example.com , mirror2.example.com "
			result, err := ocm.ParseImageMirrorSpec(spec)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Source).To(Equal("registry.redhat.io"))
			Expect(result.Mirrors).To(Equal([]string{"mirror1.example.com", "mirror2.example.com"}))
		})

		It("should reject empty specification", func() {
			_, err := ocm.ParseImageMirrorSpec("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be empty"))
		})

		It("should reject invalid format", func() {
			_, err := ocm.ParseImageMirrorSpec("invalid-format")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid format"))
		})

		It("should reject empty source", func() {
			_, err := ocm.ParseImageMirrorSpec("=mirror.example.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("source cannot be empty"))
		})

		It("should reject empty mirrors", func() {
			_, err := ocm.ParseImageMirrorSpec("registry.redhat.io=")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one mirror must be specified"))
		})
	})

	Describe("ValidateImageMirrorConfig", func() {
		It("should validate valid configuration", func() {
			spec := ocm.Spec{
				ImageContentSources: []ocm.ImageContentSource{
					{
						Source:  "registry.redhat.io",
						Mirrors: []string{"mirror.example.com"},
					},
				},
				ImageDigestMirrorSet: []ocm.ImageMirrorSet{
					{
						Source:  "quay.io/openshift-release-dev",
						Mirrors: []string{"mirror.example.com", "backup.example.com"},
					},
				},
			}

			err := ocm.ValidateImageMirrorConfig(spec)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject invalid source in image content sources", func() {
			spec := ocm.Spec{
				ImageContentSources: []ocm.ImageContentSource{
					{
						Source:  "",
						Mirrors: []string{"mirror.example.com"},
					},
				},
			}

			err := ocm.ValidateImageMirrorConfig(spec)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid image content source"))
		})

		It("should reject invalid mirrors in digest mirror set", func() {
			spec := ocm.Spec{
				ImageDigestMirrorSet: []ocm.ImageMirrorSet{
					{
						Source:  "registry.redhat.io",
						Mirrors: []string{},
					},
				},
			}

			err := ocm.ValidateImageMirrorConfig(spec)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid image digest mirror set"))
		})
	})

	Describe("BuildImageMirrorProperties", func() {
		It("should build properties for HCP cluster", func() {
			spec := ocm.Spec{
				ImageContentSources: []ocm.ImageContentSource{
					{
						Source:  "registry.redhat.io",
						Mirrors: []string{"mirror.example.com"},
					},
				},
				ImageDigestMirrorSet: []ocm.ImageMirrorSet{
					{
						Source:  "quay.io/openshift-release-dev",
						Mirrors: []string{"mirror.example.com"},
					},
				},
				ImageTagMirrorSet: []ocm.ImageMirrorSet{
					{
						Source:  "docker.io",
						Mirrors: []string{"mirror.example.com"},
					},
				},
			}

			props, err := ocm.BuildImageMirrorProperties(spec)
			Expect(err).NotTo(HaveOccurred())
			Expect(props).To(HaveLen(3))

			// Check that properties contain expected keys
			Expect(props).To(HaveKey("hypershift.openshift.io/image-content-sources"))
			Expect(props).To(HaveKey("hypershift.openshift.io/image-digest-mirror-set"))
			Expect(props).To(HaveKey("hypershift.openshift.io/image-tag-mirror-set"))

			// Verify content sources JSON
			var icsData map[string]interface{}
			err = json.Unmarshal([]byte(props["hypershift.openshift.io/image-content-sources"]), &icsData)
			Expect(err).NotTo(HaveOccurred())
			Expect(icsData["type"]).To(Equal("imageContentSources"))
			Expect(icsData["mirrors"]).NotTo(BeNil())

			// Verify digest mirror set JSON
			var idmsData map[string]interface{}
			err = json.Unmarshal([]byte(props["hypershift.openshift.io/image-digest-mirror-set"]), &idmsData)
			Expect(err).NotTo(HaveOccurred())
			Expect(idmsData["type"]).To(Equal("imageDigestMirrorSet"))
			Expect(idmsData["mirrors"]).NotTo(BeNil())

			// Verify tag mirror set JSON
			var itmsData map[string]interface{}
			err = json.Unmarshal([]byte(props["hypershift.openshift.io/image-tag-mirror-set"]), &itmsData)
			Expect(err).NotTo(HaveOccurred())
			Expect(itmsData["type"]).To(Equal("imageTagMirrorSet"))
			Expect(itmsData["mirrors"]).NotTo(BeNil())
		})

		It("should return empty properties when no image mirrors configured", func() {
			spec := ocm.Spec{}
			props, err := ocm.BuildImageMirrorProperties(spec)
			Expect(err).NotTo(HaveOccurred())
			Expect(props).To(BeEmpty())
		})

		It("should fail validation for invalid configuration", func() {
			spec := ocm.Spec{
				ImageContentSources: []ocm.ImageContentSource{
					{
						Source:  "",
						Mirrors: []string{"mirror.example.com"},
					},
				},
			}

			_, err := ocm.BuildImageMirrorProperties(spec)
			Expect(err).To(HaveOccurred())
		})
	})
})