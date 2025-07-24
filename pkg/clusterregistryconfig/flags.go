package clusterregistryconfig

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/alessio/shellescape"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/input"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
)

const (
	allowedRegistriesFlag          = "registry-config-allowed-registries"
	insecureRegistriesFlag         = "registry-config-insecure-registries"
	blockedRegistriesFlag          = "registry-config-blocked-registries"
	platformAllowlistFlag          = "registry-config-platform-allowlist"
	additionalTrustedCaPathFlag    = "registry-config-additional-trusted-ca"
	allowedRegistriesForImportFlag = "registry-config-allowed-registries-for-import"

	// NEW: IDMS/ITMS flags
	imageDigestMirrorSetsFlag = "registry-config-image-digest-mirror-sets"
	imageTagMirrorSetsFlag    = "registry-config-image-tag-mirror-sets"
)

type ClusterRegistryConfigArgs struct {
	allowedRegistries          []string
	blockedRegistries          []string
	insecureRegistries         []string
	allowedRegistriesForImport string
	platformAllowlist          string
	additionalTrustedCa        string

	// NEW: IDMS/ITMS fields
	imageDigestMirrorSets []string
	imageTagMirrorSets    []string
}

func AddClusterRegistryConfigFlags(cmd *cobra.Command) *ClusterRegistryConfigArgs {
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
	cmd.Flags().MarkHidden(platformAllowlistFlag)

	cmd.Flags().StringVar(
		&args.additionalTrustedCa,
		additionalTrustedCaPathFlag,
		"",
		"A json file containing the registry hostname as the key,"+
			" and the PEM-encoded certificate as the value, for each additional registry CA to trust.")

	// NEW: IDMS/ITMS flags
	cmd.Flags().StringSliceVar(
		&args.imageDigestMirrorSets,
		imageDigestMirrorSetsFlag,
		nil,
		"ImageDigestMirrorSet configurations for digest-based image mirroring. "+
			"Format: 'name:source=mirror1,mirror2'. Multiple sets can be specified. "+
			"Example: 'company-mirrors:registry.company.com/openshift=mirror1.company.com/openshift,mirror2.company.com/openshift'",
	)

	cmd.Flags().StringSliceVar(
		&args.imageTagMirrorSets,
		imageTagMirrorSetsFlag,
		nil,
		"ImageTagMirrorSet configurations for tag-based image mirroring. "+
			"Format: 'name:source=mirror1,mirror2'. Multiple sets can be specified. "+
			"Example: 'tag-mirrors:docker.io/library=mirror.company.com/docker-library'",
	)

	return args
}

// parseImageDigestMirrorSets parses IDMS string format into structured data
func parseImageDigestMirrorSets(input []string) ([]ocm.ImageDigestMirrorSet, error) {
	var result []ocm.ImageDigestMirrorSet

	for _, entry := range input {
		if entry == "" {
			continue
		}

		// Use the validation function to ensure format compliance
		if err := validateSingleMirrorSetFormat(entry); err != nil {
			return nil, err
		}

		// Parse format: "name:source=mirror1,mirror2"
		parts := strings.SplitN(entry, ":", 2)
		name := parts[0]
		sourceMirrors := strings.SplitN(parts[1], "=", 2)
		source := sourceMirrors[0]
		mirrors := strings.Split(sourceMirrors[1], ",")

		// Trim mirrors
		for i, mirror := range mirrors {
			mirrors[i] = strings.TrimSpace(mirror)
		}

		result = append(result, ocm.ImageDigestMirrorSet{
			Name: name,
			Mirrors: []ocm.ImageMirror{
				{
					Source:          source,
					MirrorsByDigest: mirrors,
				},
			},
		})
	}

	return result, nil
}

// parseImageTagMirrorSets parses ITMS string format into structured data
func parseImageTagMirrorSets(input []string) ([]ocm.ImageTagMirrorSet, error) {
	var result []ocm.ImageTagMirrorSet

	for _, entry := range input {
		if entry == "" {
			continue
		}

		// Use the validation function to ensure format compliance
		if err := validateSingleMirrorSetFormat(entry); err != nil {
			return nil, err
		}

		// Parse format: "name:source=mirror1,mirror2"
		parts := strings.SplitN(entry, ":", 2)
		name := parts[0]
		sourceMirrors := strings.SplitN(parts[1], "=", 2)
		source := sourceMirrors[0]
		mirrors := strings.Split(sourceMirrors[1], ",")

		// Trim mirrors
		for i, mirror := range mirrors {
			mirrors[i] = strings.TrimSpace(mirror)
		}

		result = append(result, ocm.ImageTagMirrorSet{
			Name: name,
			Mirrors: []ocm.ImageMirror{
				{
					Source:       source,
					MirrorsByTag: mirrors,
				},
			},
		})
	}

	return result, nil
}

// GetImageMirrorSetArgs parses and returns IDMS/ITMS configurations
func GetImageMirrorSetArgs(args *ClusterRegistryConfigArgs) ([]ocm.ImageDigestMirrorSet, []ocm.ImageTagMirrorSet, error) {
	idmsList, err := parseImageDigestMirrorSets(args.imageDigestMirrorSets)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse IDMS: %v", err)
	}

	itmsList, err := parseImageTagMirrorSets(args.imageTagMirrorSets)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse ITMS: %v", err)
	}

	return idmsList, itmsList, nil
}

func GetClusterRegistryConfigArgs(args *ClusterRegistryConfigArgs) (
	[]string, []string, []string, string, string, string) {
	return args.allowedRegistries, args.blockedRegistries,
		args.insecureRegistries, args.additionalTrustedCa, args.allowedRegistriesForImport,
		args.platformAllowlist
}

func GetClusterRegistryConfigOptions(cmd *pflag.FlagSet,
	args *ClusterRegistryConfigArgs, isHostedCP bool, cluster *cmv1.Cluster) (
	*ClusterRegistryConfigArgs, error) {

	var allowedRegistries []string

	if !isHostedCP {
		if IsClusterRegistryConfigSetViaCLI(cmd) {
			return nil, fmt.Errorf("Setting the registry config is only supported for hosted clusters")
		}
		return nil, nil
	}

	result := &ClusterRegistryConfigArgs{}

	result.allowedRegistries = args.allowedRegistries
	result.insecureRegistries = args.insecureRegistries
	result.blockedRegistries = args.blockedRegistries
	result.additionalTrustedCa = args.additionalTrustedCa
	result.allowedRegistriesForImport = args.allowedRegistriesForImport
	result.platformAllowlist = args.platformAllowlist

	if !IsClusterRegistryConfigSetViaCLI(cmd) && !interactive.Enabled() {
		return nil, nil
	}

	defaultAllowedRegistries := args.allowedRegistries
	defaultBlockedRegistries := args.blockedRegistries
	defaultInsecureRegistries := args.insecureRegistries
	defaultAllowedRegistriesForImport := args.allowedRegistriesForImport

	if defaultAllowedRegistries == nil {
		defaultAllowedRegistries = cluster.RegistryConfig().RegistrySources().AllowedRegistries()
	}
	if defaultBlockedRegistries == nil {
		defaultBlockedRegistries = cluster.RegistryConfig().RegistrySources().BlockedRegistries()
	}
	if defaultInsecureRegistries == nil {
		defaultInsecureRegistries = cluster.RegistryConfig().RegistrySources().InsecureRegistries()
	}
	if defaultAllowedRegistriesForImport == "" {
		var list []string
		for _, location := range cluster.RegistryConfig().AllowedRegistriesForImport() {
			list = append(list, fmt.Sprintf("%s:%s", location.DomainName(), strconv.FormatBool(location.Insecure())))
		}
		defaultAllowedRegistriesForImport = strings.Join(list, ",")
	}

	enableRegistriesConfig := IsClusterRegistryConfigSetViaCLI(cmd)
	if cluster.RegistryConfig() != nil {
		enableRegistriesConfig = true
	}

	if interactive.Enabled() {
		regConfigQuestion := GetClusterRegistryConfigQuestion(cluster)

		updateRegistriesConfigValue, err := interactive.GetBool(interactive.Input{
			Question: regConfigQuestion,
			Default:  enableRegistriesConfig,
		})
		if err != nil {
			return nil, fmt.Errorf("Expected a valid registries config value: %s", err)
		}
		enableRegistriesConfig = updateRegistriesConfigValue
	}

	isBlockedRegistryNotSet := result.blockedRegistries == nil || strings.Join(result.blockedRegistries, ",") == ""
	isAllowedRegistryNotSet := result.allowedRegistries == nil || strings.Join(result.allowedRegistries, ",") == ""

	if enableRegistriesConfig && interactive.Enabled() {
		// Allowed registries and blocked registries are mutually exclusive
		if isBlockedRegistryNotSet {
			allowedRegistriesInputs, err := interactive.GetString(interactive.Input{
				Question: "Allowed Registries",
				Help:     cmd.Lookup(allowedRegistriesFlag).Usage,
				Default:  strings.Join(defaultAllowedRegistries, ","),
			})
			if err != nil {
				return nil, fmt.Errorf("Expected a comma-separated list of allowed registries: %s", err)
			}
			allowedRegistries = helper.HandleEmptyStringOnSlice(strings.Split(allowedRegistriesInputs, ","))

			if len(allowedRegistries) > 0 {
				// received double quotes from the user. need to remove the existing value
				if len(allowedRegistries) == 1 && allowedRegistries[0] == input.DoubleQuotesToRemove {
					allowedRegistries[0] = ""
				}
			}
			result.allowedRegistries = allowedRegistries
			isAllowedRegistryNotSet = result.allowedRegistries == nil || strings.Join(result.allowedRegistries, ",") == ""
		} else {
			// if blocked registries is set, remove allowed registries
			result.allowedRegistries = []string{}
		}

		if isAllowedRegistryNotSet {
			blockedRegistriesInputs, err := interactive.GetString(interactive.Input{
				Question: "Blocked Registries",
				Help:     cmd.Lookup(blockedRegistriesFlag).Usage,
				Default:  strings.Join(defaultBlockedRegistries, ","),
			})
			if err != nil {
				return nil, fmt.Errorf("Expected a comma-separated list of blocked registries: %s", err)
			}
			result.blockedRegistries = helper.HandleEmptyStringOnSlice(strings.Split(blockedRegistriesInputs, ","))
			isBlockedRegistryNotSet = result.blockedRegistries == nil || strings.Join(result.blockedRegistries, ",") == ""
		} else {
			// if allowed registries is set, remove blocked registries
			result.blockedRegistries = []string{}
		}

		insecureRegistriesInputs, err := interactive.GetString(interactive.Input{
			Question: "Insecure Registries",
			Help:     cmd.Lookup(insecureRegistriesFlag).Usage,
			Default:  strings.Join(defaultInsecureRegistries, ","),
		})
		if err != nil {
			return nil, fmt.Errorf("Expected a comma-separated list of insecure registries: %s", err)
		}
		result.insecureRegistries = helper.HandleEmptyStringOnSlice(strings.Split(insecureRegistriesInputs, ","))

		result.allowedRegistriesForImport, err = interactive.GetString(interactive.Input{
			Question: "Allowed Registries For Import",
			Help:     cmd.Lookup(allowedRegistriesForImportFlag).Usage,
			Default:  defaultAllowedRegistriesForImport,
			Validators: []interactive.Validator{
				ocm.ValidateAllowedRegistriesForImport,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("Expected a comma-separated list of allowed registries for import: %s", err)
		}

		result.additionalTrustedCa, err = interactive.GetString(interactive.Input{
			Question: "Registry Additional Trusted CA",
			Help:     cmd.Lookup(additionalTrustedCaPathFlag).Usage,
			Default:  args.additionalTrustedCa,
		})
		if err != nil {
			return nil, fmt.Errorf("Expected a valid certificate: %s", err)
		}
	}
	if err := ocm.ValidateAllowedRegistriesForImport(result.allowedRegistriesForImport); err != nil {
		return nil, fmt.Errorf("Expected valid allowed registries for import values: %v", err)
	}

	if !isBlockedRegistryNotSet && !isAllowedRegistryNotSet {
		return nil, fmt.Errorf("Allowed registries and blocked registries are mutually exclusive fields")
	}

	return result, nil
}

func GetClusterRegistryConfigQuestion(cluster *cmv1.Cluster) string {
	if cluster != nil {
		return "Update registries config"
	}
	return "Enable registries config"
}

func IsClusterRegistryConfigSetViaCLI(cmd *pflag.FlagSet) bool {
	for _, parameter := range []string{
		allowedRegistriesFlag,
		insecureRegistriesFlag,
		blockedRegistriesFlag,
		platformAllowlistFlag,
		allowedRegistriesForImportFlag,
		additionalTrustedCaPathFlag,
		imageDigestMirrorSetsFlag,
		imageTagMirrorSetsFlag,
	} {
		if cmd.Changed(parameter) {
			return true
		}
	}

	return false
}

func BuildRegistryConfigOptions(spec ocm.Spec) string {
	command := ""

	if len(spec.AllowedRegistries) > 0 {
		command += fmt.Sprintf(" --%s %s",
			allowedRegistriesFlag,
			shellescape.Quote(strings.Join(spec.AllowedRegistries, ",")))
	}

	if len(spec.BlockedRegistries) > 0 {
		command += fmt.Sprintf(" --%s %s",
			blockedRegistriesFlag,
			shellescape.Quote(strings.Join(spec.BlockedRegistries, ",")))
	}

	if len(spec.InsecureRegistries) > 0 {
		command += fmt.Sprintf(" --%s %s",
			insecureRegistriesFlag,
			shellescape.Quote(strings.Join(spec.InsecureRegistries, ",")))
	}

	if spec.AdditionalTrustedCaFile != "" {
		command += fmt.Sprintf(" --%s %s",
			additionalTrustedCaPathFlag,
			shellescape.Quote(spec.AdditionalTrustedCaFile))
	}

	if spec.PlatformAllowlist != "" {
		command += fmt.Sprintf(" --%s %s",
			platformAllowlistFlag,
			shellescape.Quote(spec.PlatformAllowlist))
	}

	if spec.AllowedRegistriesForImport != "" {
		command += fmt.Sprintf(" --%s %s",
			allowedRegistriesForImportFlag,
			shellescape.Quote(spec.AllowedRegistriesForImport))
	}

	return command
}

// BuildImageMirrorSetOptions builds command line options for IDMS/ITMS
func BuildImageMirrorSetOptions(spec ocm.Spec) string {
	command := ""

	// Build IDMS options
	if len(spec.ImageDigestMirrorSets) > 0 {
		var idmsStrings []string
		for _, idms := range spec.ImageDigestMirrorSets {
			for _, mirror := range idms.Mirrors {
				idmsString := fmt.Sprintf("%s:%s=%s",
					idms.Name,
					mirror.Source,
					strings.Join(mirror.MirrorsByDigest, ","))
				idmsStrings = append(idmsStrings, idmsString)
			}
		}
		if len(idmsStrings) > 0 {
			command += fmt.Sprintf(" --%s %s",
				imageDigestMirrorSetsFlag,
				shellescape.Quote(strings.Join(idmsStrings, " ")))
		}
	}

	// Build ITMS options
	if len(spec.ImageTagMirrorSets) > 0 {
		var itmsStrings []string
		for _, itms := range spec.ImageTagMirrorSets {
			for _, mirror := range itms.Mirrors {
				itmsString := fmt.Sprintf("%s:%s=%s",
					itms.Name,
					mirror.Source,
					strings.Join(mirror.MirrorsByTag, ","))
				itmsStrings = append(itmsStrings, itmsString)
			}
		}
		if len(itmsStrings) > 0 {
			command += fmt.Sprintf(" --%s %s",
				imageTagMirrorSetsFlag,
				shellescape.Quote(strings.Join(itmsStrings, " ")))
		}
	}

	return command
}

func BuildAdditionalTrustedCAFromInputFile(specPath string) (map[string]string, error) {
	specJson, err := input.UnmarshalInputFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("expected a valid additional trusted certificate spec file: %v", err)
	}
	form := make(map[string]string)
	var ok bool
	for k, v := range specJson {
		form[k], ok = v.(string)
		if !ok {
			return nil, fmt.Errorf("expected a valid value for 'additional_trusted_ca'. " +
				"Should be in a <registry>:<boolean> format.")
		}
	}

	caBuilder := cmv1.NewClusterRegistryConfig().AdditionalTrustedCa(form)

	ca, err := caBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build additional trusted certificate: %v", err)
	}
	return ca.AdditionalTrustedCa(), nil
}
