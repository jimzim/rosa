package imagemirrorconfig_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/imagemirrorconfig"
	"github.com/openshift/rosa/pkg/ocm"
)

var _ = Describe("Image Mirror Config Flags", func() {
	Describe("ParseImageMirrorSpecs", func() {
		It("should parse multiple mirror specifications", func() {
			specs := []string{
				"registry.redhat.io=mirror1.example.com,mirror2.example.com",
				"quay.io/openshift=mirror.example.com",
			}

			result, err := imagemirrorconfig.ParseImageMirrorSpecs(specs)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))

			Expect(result[0].Source).To(Equal("registry.redhat.io"))
			Expect(result[0].Mirrors).To(Equal([]string{"mirror1.example.com", "mirror2.example.com"}))

			Expect(result[1].Source).To(Equal("quay.io/openshift"))
			Expect(result[1].Mirrors).To(Equal([]string{"mirror.example.com"}))
		})

		It("should handle empty specs", func() {
			result, err := imagemirrorconfig.ParseImageMirrorSpecs([]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should reject invalid spec format", func() {
			specs := []string{"invalid-format"}
			_, err := imagemirrorconfig.ParseImageMirrorSpecs(specs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse mirror spec"))
		})
	})

	Describe("ParseImageContentSources", func() {
		It("should parse content source specifications", func() {
			specs := []string{
				"registry.redhat.io=mirror1.example.com",
				"quay.io=mirror2.example.com,mirror3.example.com",
			}

			result, err := imagemirrorconfig.ParseImageContentSources(specs)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))

			Expect(result[0].Source).To(Equal("registry.redhat.io"))
			Expect(result[0].Mirrors).To(Equal([]string{"mirror1.example.com"}))

			Expect(result[1].Source).To(Equal("quay.io"))
			Expect(result[1].Mirrors).To(Equal([]string{"mirror2.example.com", "mirror3.example.com"}))
		})

		It("should handle empty specs", func() {
			result, err := imagemirrorconfig.ParseImageContentSources([]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("BuildImageMirrorConfigOptions", func() {
		It("should build command options for image mirror config", func() {
			spec := ocm.Spec{
				ImageContentSources: []ocm.ImageContentSource{
					{
						Source:  "registry.redhat.io",
						Mirrors: []string{"mirror1.example.com", "mirror2.example.com"},
					},
				},
				ImageDigestMirrorSet: []ocm.ImageMirrorSet{
					{
						Source:  "quay.io/openshift",
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

			command := imagemirrorconfig.BuildImageMirrorConfigOptions(spec)
			Expect(command).To(ContainSubstring("--image-content-sources"))
			Expect(command).To(ContainSubstring("registry.redhat.io=mirror1.example.com,mirror2.example.com"))
			Expect(command).To(ContainSubstring("--image-digest-mirror-set"))
			Expect(command).To(ContainSubstring("quay.io/openshift=mirror.example.com"))
			Expect(command).To(ContainSubstring("--image-tag-mirror-set"))
			Expect(command).To(ContainSubstring("docker.io=mirror.example.com"))
		})

		It("should return empty string when no image mirror config", func() {
			spec := ocm.Spec{}
			command := imagemirrorconfig.BuildImageMirrorConfigOptions(spec)
			Expect(command).To(BeEmpty())
		})
	})
})