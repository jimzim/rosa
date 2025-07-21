/*
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

package clusterregistryconfig

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift/rosa/pkg/ocm"
)

var _ = Describe("Cluster Registry Config tests", func() {
	Context("BuildAdditionalTrustedCAFromInputFile", func() {
		It("OK: shoud work with proper json format", func() {
			caPath := "specRegistryAdditionalTrustedCa.json"
			ca, err := BuildAdditionalTrustedCAFromInputFile(caPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(ca["userRegistry.io"]).To(
				Equal("-----BEGIN CERTIFICATE-----\n/abc\n-----END CERTIFICATE-----"))

		})
		It("KO: fail if the spec file is invalid", func() {
			_, err := BuildAdditionalTrustedCAFromInputFile("not-exist")
			Expect(err).To(MatchError(
				"expected a valid additional trusted certificate spec file: open not-exist: no such file or directory"))
		})
		It("KO: fail if the content type is incorrect", func() {
			caPath := "specRegistryAdditionalTrustedCaInvalid.json"
			_, err := BuildAdditionalTrustedCAFromInputFile(caPath)
			Expect(
				err,
			).To(MatchError("expected a valid value for 'additional_trusted_ca'. Should be in a <registry>:<boolean> format."))

		})
	})

	Context("BuildImageTagMirrorSetsFromInputFile", func() {
		It("OK: should work with proper json format", func() {
			itmsPath := "specImageTagMirrorSets.json"
			itms, err := BuildImageTagMirrorSetsFromInputFile(itmsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(itms["imageTagMirrorSets"]).ToNot(BeNil())
		})
		It("KO: fail if the spec file is invalid", func() {
			_, err := BuildImageTagMirrorSetsFromInputFile("not-exist")
			Expect(err).To(MatchError(
				"expected a valid ImageTagMirrorSets spec file: open not-exist: no such file or directory"))
		})
		It("OK: return nil if empty path", func() {
			itms, err := BuildImageTagMirrorSetsFromInputFile("")
			Expect(err).NotTo(HaveOccurred())
			Expect(itms).To(BeNil())
		})
	})

	Context("BuildImageDigestMirrorSourcesFromInputFile", func() {
		It("OK: should work with proper json format", func() {
			idmsPath := "specImageDigestMirrorSources.json"
			idms, err := BuildImageDigestMirrorSourcesFromInputFile(idmsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(idms["imageDigestMirrorSources"]).ToNot(BeNil())
		})
		It("KO: fail if the spec file is invalid", func() {
			_, err := BuildImageDigestMirrorSourcesFromInputFile("not-exist")
			Expect(err).To(MatchError(
				"expected a valid ImageDigestMirrorSources spec file: open not-exist: no such file or directory"))
		})
		It("OK: return nil if empty path", func() {
			idms, err := BuildImageDigestMirrorSourcesFromInputFile("")
			Expect(err).NotTo(HaveOccurred())
			Expect(idms).To(BeNil())
		})
	})

	Context("BuildRegistryConfigOptions", func() {
		spec := ocm.Spec{}

		It("Returns empty string if nothing is set", func() {
			output := BuildRegistryConfigOptions(spec)
			Expect(output).To(Equal(""))
		})

		It("Returns the expected string if set", func() {
			spec.AllowedRegistries = []string{"abc.com", "efg.com"}
			spec.InsecureRegistries = []string{"insecure.com", "*.insecure.com"}
			spec.BlockedRegistries = []string{"blocked.com"}
			spec.AdditionalTrustedCaFile = "ca.json"
			spec.PlatformAllowlist = "allowlist-id"
			spec.AllowedRegistriesForImport = "lala.com:true,*.io:false"
			spec.ImageTagMirrorSets = "itms.json"
			spec.ImageDigestMirrorSources = "idms.json"
			output := BuildRegistryConfigOptions(spec)
			expectedOutput := " --registry-config-allowed-registries abc.com,efg.com" +
				" --registry-config-blocked-registries blocked.com" +
				" --registry-config-insecure-registries 'insecure.com,*.insecure.com'" +
				" --registry-config-additional-trusted-ca ca.json" +
				" --registry-config-platform-allowlist allowlist-id" +
				" --registry-config-allowed-registries-for-import 'lala.com:true,*.io:false'" +
				" --registry-config-image-tag-mirror-sets itms.json" +
				" --registry-config-image-digest-mirror-sources idms.json"
			Expect(output).To(Equal(expectedOutput))
		})
	})

	Context("GetClusterRegistryConfigOptions", func() {
		args := &ClusterRegistryConfigArgs{}
		cmd := &cobra.Command{}
		cmd.Flags().StringSliceVar(
			&args.allowedRegistries,
			allowedRegistriesFlag,
			nil,
			"A comma-separated list of registries for which image pull and push actions are allowed.",
		)
		flags := cmd.Flags()
		BeforeEach(func() {
			args.allowedRegistries = []string{}
			flags.VisitAll(func(f *pflag.Flag) {
				if f.Changed {
					f.Changed = false
					cmd.SetArgs([]string{fmt.Sprintf("--%s=%s", allowedRegistriesFlag, f.DefValue)})
				}
			})
		})
		It("OK: classic clusters when no changes to flags", func() {
			args, err := GetClusterRegistryConfigOptions(flags, args, false, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(args).To(BeNil())
		})
		It("KO: throw error for classic clusters when flag set via cli", func() {
			flags.Set(allowedRegistriesFlag, "test.com")
			_, err := GetClusterRegistryConfigOptions(flags, args, false, nil)
			Expect(err).To(MatchError("Setting the registry config is only supported for hosted clusters"))
		})
		It("OK: returns the correct output", func() {
			flags.Set(allowedRegistriesFlag, "test.com")
			args, err := GetClusterRegistryConfigOptions(flags, args, true, nil)
			Expect(err).To(Not(HaveOccurred()))
			Expect(args.allowedRegistries).To(Equal([]string{"test.com"}))
		})
	})

	Context("IsClusterRegistryConfigSetViaCLI", func() {
		cmd := &cobra.Command{}
		flags := cmd.Flags()
		args := &ClusterRegistryConfigArgs{}
		cmd.Flags().StringSliceVar(
			&args.allowedRegistries,
			allowedRegistriesFlag,
			nil,
			"A comma-separated list of registries for which image pull and push actions are allowed.",
		)
		cmd.Flags().StringSliceVar(
			&args.insecureRegistries,
			insecureRegistriesFlag,
			nil,
			"A comma-separated list of registries which do not have a valid TLS certificate or only support HTTP connections.",
		)
		cmd.Flags().StringSliceVar(
			&args.blockedRegistries,
			blockedRegistriesFlag,
			nil,
			"A comma-separated list of registries for which image pull and push actions are denied.",
		)
		cmd.Flags().StringVar(
			&args.allowedRegistriesForImport,
			allowedRegistriesForImportFlag,
			"",
			"Limits the container image registries from which normal users can import images. "+
				"The format should be a comma-separated list of 'domainName:insecure'. "+
				"'domainName' specifies a domain name for the registry. "+
				"'insecure' indicates whether the registry is secure or insecure.",
		)
		cmd.Flags().StringVar(
			&args.platformAllowlist,
			platformAllowlistFlag,
			"",
			"A reference to the id of the list of registries that needs to be whitelisted for the platform to work. "+
				"It can be omitted at creation and updating and its lifecycle can be managed separately if needed.",
		)
		cmd.Flags().StringVar(
			&args.additionalTrustedCa,
			additionalTrustedCaPathFlag,
			"",
			"A json file containing the registry hostname as the key,"+
				" and the PEM-encoded certificate as the value, for each additional registry CA to trust.")

		cmd.Flags().StringVar(
			&args.imageTagMirrorSets,
			imageTagMirrorSetsFlag,
			"",
			"A json file containing ImageTagMirrorSets configuration for mirroring images by tag.")

		cmd.Flags().StringVar(
			&args.imageDigestMirrorSources,
			imageDigestMirrorSourcesFlag,
			"",
			"A json file containing ImageDigestMirrorSources configuration for mirroring images by digest.")

		It("KO: return false if nothing is set", func() {
			isClusterRegistryConfigSetViaCLI := IsClusterRegistryConfigSetViaCLI(flags)
			Expect(isClusterRegistryConfigSetViaCLI).To(Equal(false))
		})
		It("OK: return true if sets allowed registries", func() {
			flags.Set(allowedRegistriesFlag, "allow.com")
			isClusterRegistryConfigSetViaCLI := IsClusterRegistryConfigSetViaCLI(flags)
			Expect(isClusterRegistryConfigSetViaCLI).To(Equal(true))
		})
		It("OK: return true if sets blocked registries", func() {
			flags.Set(blockedRegistriesFlag, "block.com")
			isClusterRegistryConfigSetViaCLI := IsClusterRegistryConfigSetViaCLI(flags)
			Expect(isClusterRegistryConfigSetViaCLI).To(Equal(true))
		})
		It("OK: return true if sets insecure registries", func() {
			flags.Set(insecureRegistriesFlag, "insecure.com")
			isClusterRegistryConfigSetViaCLI := IsClusterRegistryConfigSetViaCLI(flags)
			Expect(isClusterRegistryConfigSetViaCLI).To(Equal(true))
		})
		It("OK: return true if sets platform allowlist", func() {
			flags.Set(platformAllowlistFlag, "allowlist-id")
			isClusterRegistryConfigSetViaCLI := IsClusterRegistryConfigSetViaCLI(flags)
			Expect(isClusterRegistryConfigSetViaCLI).To(Equal(true))
		})
		It("OK: return true if sets allowed registries for import", func() {
			flags.Set(allowedRegistriesForImportFlag, "test.com:true")
			isClusterRegistryConfigSetViaCLI := IsClusterRegistryConfigSetViaCLI(flags)
			Expect(isClusterRegistryConfigSetViaCLI).To(Equal(true))
		})
		It("OK: return true if sets additional trusted ca", func() {
			flags.Set(additionalTrustedCaPathFlag, "ca.json")
			isClusterRegistryConfigSetViaCLI := IsClusterRegistryConfigSetViaCLI(flags)
			Expect(isClusterRegistryConfigSetViaCLI).To(Equal(true))
		})
		It("OK: return true if sets image tag mirror sets", func() {
			flags.Set(imageTagMirrorSetsFlag, "itms.json")
			isClusterRegistryConfigSetViaCLI := IsClusterRegistryConfigSetViaCLI(flags)
			Expect(isClusterRegistryConfigSetViaCLI).To(Equal(true))
		})
		It("OK: return true if sets image digest mirror sources", func() {
			flags.Set(imageDigestMirrorSourcesFlag, "idms.json")
			isClusterRegistryConfigSetViaCLI := IsClusterRegistryConfigSetViaCLI(flags)
			Expect(isClusterRegistryConfigSetViaCLI).To(Equal(true))
		})
	})

	Context("GetClusterRegistryConfigQuestion", func() {
		It("Asks to enable config option for new clusters", func() {
			question := GetClusterRegistryConfigQuestion(nil)
			Expect(question).To(ContainSubstring("Enable"))
		})
		It("Asks to update cluster registry config option for existing clusters", func() {
			question := GetClusterRegistryConfigQuestion(&cmv1.Cluster{})
			Expect(question).To(ContainSubstring("Update"))
		})
	})

})
