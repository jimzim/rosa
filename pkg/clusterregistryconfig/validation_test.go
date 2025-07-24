package clusterregistryconfig_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/clusterregistryconfig"
	"github.com/openshift/rosa/pkg/ocm"
)

var _ = Describe("IDMS/ITMS Validation", func() {
	Context("validateImageMirrorSetFormat", func() {
		It("should accept valid IDMS format", func() {
			input := "company-mirrors:registry.company.com/openshift=mirror1.company.com/openshift,mirror2.company.com/openshift"
			err := clusterregistryconfig.ValidateImageMirrorSetFormat(input)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should accept valid ITMS format", func() {
			input := "tag-mirrors:docker.io/library=mirror.company.com/docker-library"
			err := clusterregistryconfig.ValidateImageMirrorSetFormat(input)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject invalid format missing colon", func() {
			input := "invalid-format-missing-colon"
			err := clusterregistryconfig.ValidateImageMirrorSetFormat(input)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid format"))
		})

		It("should reject invalid format missing equals", func() {
			input := "valid-name:invalid-source-without-equals"
			err := clusterregistryconfig.ValidateImageMirrorSetFormat(input)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid source=mirrors format"))
		})

		It("should reject invalid Kubernetes names", func() {
			input := "Invalid_Name:registry.com=mirror.com"
			err := clusterregistryconfig.ValidateImageMirrorSetFormat(input)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Kubernetes naming conventions"))
		})

		It("should reject names that are too long", func() {
			longName := "this-is-a-very-long-name-that-exceeds-the-kubernetes-naming-limit-of-sixty-three-characters"
			input := longName + ":registry.com=mirror.com"
			err := clusterregistryconfig.ValidateImageMirrorSetFormat(input)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Kubernetes naming conventions"))
		})

		It("should reject invalid registry formats with protocol", func() {
			input := "valid-name:https://registry.com=mirror.com"
			err := clusterregistryconfig.ValidateImageMirrorSetFormat(input)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("should not include protocol"))
		})

		It("should reject too many mirrors per set", func() {
			mirrors := make([]string, 51) // maxMirrorsPerSet is 50
			for i := range mirrors {
				mirrors[i] = "mirror" + string(rune(i+'0')) + ".com"
			}
			input := "valid-name:registry.com=" + strings.Join(mirrors, ",")
			err := clusterregistryconfig.ValidateImageMirrorSetFormat(input)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("too many mirrors"))
		})

		It("should reject empty mirrors", func() {
			input := "valid-name:registry.com=mirror1.com,,mirror2.com"
			err := clusterregistryconfig.ValidateImageMirrorSetFormat(input)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty mirror specified"))
		})

		It("should handle string slice input", func() {
			input := []string{
				"company-mirrors:registry.company.com/openshift=mirror1.company.com/openshift",
				"tag-mirrors:docker.io/library=mirror.company.com/docker-library",
			}
			err := clusterregistryconfig.ValidateImageMirrorSetFormat(input)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should allow empty entries in slice", func() {
			input := []string{
				"company-mirrors:registry.company.com/openshift=mirror1.company.com/openshift",
				"", // Empty entry should be ignored
				"tag-mirrors:docker.io/library=mirror.company.com/docker-library",
			}
			err := clusterregistryconfig.ValidateImageMirrorSetFormat(input)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("validateImageMirrorSetLimits", func() {
		It("should accept valid limits", func() {
			idmsList := []ocm.ImageDigestMirrorSet{
				{
					Name: "test-idms",
					Mirrors: []ocm.ImageMirror{
						{
							Source:          "registry.com",
							MirrorsByDigest: []string{"mirror1.com", "mirror2.com"},
						},
					},
				},
			}
			itmsList := []ocm.ImageTagMirrorSet{}

			err := clusterregistryconfig.ValidateImageMirrorSetLimits(idmsList, itmsList)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject duplicate names between IDMS and ITMS", func() {
			idmsList := []ocm.ImageDigestMirrorSet{
				{Name: "duplicate-name"},
			}
			itmsList := []ocm.ImageTagMirrorSet{
				{Name: "duplicate-name"},
			}

			err := clusterregistryconfig.ValidateImageMirrorSetLimits(idmsList, itmsList)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate mirror set name"))
		})

		It("should reject too many total mirror sets", func() {
			idmsList := make([]ocm.ImageDigestMirrorSet, 51) // maxMirrorSets is 100
			itmsList := make([]ocm.ImageTagMirrorSet, 50)    // Total will be 101

			for i := range idmsList {
				idmsList[i] = ocm.ImageDigestMirrorSet{Name: "idms-" + string(rune(i+'0'))}
			}
			for i := range itmsList {
				itmsList[i] = ocm.ImageTagMirrorSet{Name: "itms-" + string(rune(i+'0'))}
			}

			err := clusterregistryconfig.ValidateImageMirrorSetLimits(idmsList, itmsList)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("too many mirror sets"))
		})

		It("should reject too many mirrors in a single IDMS", func() {
			idmsList := []ocm.ImageDigestMirrorSet{
				{
					Name: "test-idms",
					Mirrors: []ocm.ImageMirror{
						{
							Source:          "registry.com",
							MirrorsByDigest: make([]string, 51), // maxMirrorsPerSet is 50
						},
					},
				},
			}

			err := clusterregistryconfig.ValidateImageMirrorSetLimits(idmsList, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("too many mirrors"))
		})
	})

	Context("validateMirrorSetsCompatibility", func() {
		It("should accept non-conflicting source registries", func() {
			idmsList := []ocm.ImageDigestMirrorSet{
				{
					Name: "digest-mirrors",
					Mirrors: []ocm.ImageMirror{
						{Source: "registry1.com"},
					},
				},
			}
			itmsList := []ocm.ImageTagMirrorSet{
				{
					Name: "tag-mirrors",
					Mirrors: []ocm.ImageMirror{
						{Source: "registry2.com"},
					},
				},
			}

			err := clusterregistryconfig.ValidateMirrorSetsCompatibility(idmsList, itmsList)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject conflicting source registries", func() {
			idmsList := []ocm.ImageDigestMirrorSet{
				{
					Name: "digest-mirrors",
					Mirrors: []ocm.ImageMirror{
						{Source: "registry.com"},
					},
				},
			}
			itmsList := []ocm.ImageTagMirrorSet{
				{
					Name: "tag-mirrors",
					Mirrors: []ocm.ImageMirror{
						{Source: "registry.com"}, // Same source as IDMS
					},
				},
			}

			err := clusterregistryconfig.ValidateMirrorSetsCompatibility(idmsList, itmsList)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be used in both IDMS"))
		})
	})

	Context("validateMirrorSetsAgainstPlatformRequirements", func() {
		It("should accept non-platform registries", func() {
			idmsList := []ocm.ImageDigestMirrorSet{
				{
					Name: "company-mirrors",
					Mirrors: []ocm.ImageMirror{
						{
							Source:          "registry.company.com",
							MirrorsByDigest: []string{"mirror.company.com"},
						},
					},
				},
			}

			err := clusterregistryconfig.ValidateMirrorSetsAgainstPlatformRequirements(idmsList, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject platform registries as source", func() {
			idmsList := []ocm.ImageDigestMirrorSet{
				{
					Name: "bad-mirrors",
					Mirrors: []ocm.ImageMirror{
						{
							Source:          "quay.io/openshift",
							MirrorsByDigest: []string{"mirror.company.com"},
						},
					},
				},
			}

			err := clusterregistryconfig.ValidateMirrorSetsAgainstPlatformRequirements(idmsList, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("platform registry"))
			Expect(err.Error()).To(ContainSubstring("cannot be used as source"))
		})

		It("should reject platform registries as mirror", func() {
			idmsList := []ocm.ImageDigestMirrorSet{
				{
					Name: "bad-mirrors",
					Mirrors: []ocm.ImageMirror{
						{
							Source:          "registry.company.com",
							MirrorsByDigest: []string{"registry.redhat.io"},
						},
					},
				},
			}

			err := clusterregistryconfig.ValidateMirrorSetsAgainstPlatformRequirements(idmsList, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("platform registry"))
			Expect(err.Error()).To(ContainSubstring("cannot be used as mirror"))
		})
	})

	Context("validateOpenShiftVersionSupport", func() {
		It("should accept OpenShift 4.13+", func() {
			err := clusterregistryconfig.ValidateOpenShiftVersionSupport("4.13.1", true, false)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should accept OpenShift 4.14+", func() {
			err := clusterregistryconfig.ValidateOpenShiftVersionSupport("4.14.0", false, true)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should accept prefixed versions", func() {
			err := clusterregistryconfig.ValidateOpenShiftVersionSupport("openshift-v4.13.1", true, false)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject OpenShift 4.12", func() {
			err := clusterregistryconfig.ValidateOpenShiftVersionSupport("4.12.1", true, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("requires OpenShift 4.13+"))
		})

		It("should reject non-OpenShift 4.x versions", func() {
			err := clusterregistryconfig.ValidateOpenShiftVersionSupport("3.11.0", true, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("requires OpenShift 4.x"))
		})

		It("should skip validation when no IDMS/ITMS", func() {
			err := clusterregistryconfig.ValidateOpenShiftVersionSupport("4.12.1", false, false)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject invalid version format", func() {
			err := clusterregistryconfig.ValidateOpenShiftVersionSupport("invalid", true, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid version format"))
		})
	})

	Context("ValidateCompleteImageMirrorSetConfiguration", func() {
		It("should pass comprehensive validation for valid configuration", func() {
			idmsList := []ocm.ImageDigestMirrorSet{
				{
					Name: "company-digest-mirrors",
					Mirrors: []ocm.ImageMirror{
						{
							Source:          "registry.company.com/app",
							MirrorsByDigest: []string{"mirror1.company.com/app", "mirror2.company.com/app"},
						},
					},
				},
			}
			itmsList := []ocm.ImageTagMirrorSet{
				{
					Name: "company-tag-mirrors",
					Mirrors: []ocm.ImageMirror{
						{
							Source:       "docker.io/library",
							MirrorsByTag: []string{"mirror.company.com/docker-library"},
						},
					},
				},
			}

			err := clusterregistryconfig.ValidateCompleteImageMirrorSetConfiguration(idmsList, itmsList, "4.13.1")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail comprehensive validation for invalid configuration", func() {
			idmsList := []ocm.ImageDigestMirrorSet{
				{
					Name: "duplicate-name",
					Mirrors: []ocm.ImageMirror{
						{Source: "registry.company.com"},
					},
				},
			}
			itmsList := []ocm.ImageTagMirrorSet{
				{
					Name: "duplicate-name", // Duplicate name
					Mirrors: []ocm.ImageMirror{
						{Source: "registry.company.com"}, // Conflicting source
					},
				},
			}

			err := clusterregistryconfig.ValidateCompleteImageMirrorSetConfiguration(idmsList, itmsList, "4.12.1")
			Expect(err).To(HaveOccurred())
			// Should fail on the first validation error (duplicate names)
		})
	})
})
