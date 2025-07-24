package imagemirrorconfig

import (
	"fmt"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
)

const (
	imageContentSourcesFlag  = "image-content-sources"
	imageDigestMirrorSetFlag = "image-digest-mirror-set"
	imageTagMirrorSetFlag    = "image-tag-mirror-set"
)

type ImageMirrorConfigArgs struct {
	imageContentSources  []string
	imageDigestMirrorSet []string
	imageTagMirrorSet    []string
}

func AddImageMirrorConfigFlags(cmd *cobra.Command) *ImageMirrorConfigArgs {
	args := &ImageMirrorConfigArgs{}

	cmd.Flags().StringSliceVar(
		&args.imageContentSources,
		imageContentSourcesFlag,
		nil,
		"Image content source policy for image mirroring. "+
			"Use the format 'source=mirror1,mirror2'. Can be used multiple times.",
	)

	cmd.Flags().StringSliceVar(
		&args.imageDigestMirrorSet,
		imageDigestMirrorSetFlag,
		nil,
		"Image digest mirror set for image mirroring. "+
			"Use the format 'source=mirror1,mirror2'. Can be used multiple times.",
	)

	cmd.Flags().StringSliceVar(
		&args.imageTagMirrorSet,
		imageTagMirrorSetFlag,
		nil,
		"Image tag mirror set for image mirroring. "+
			"Use the format 'source=mirror1,mirror2'. Can be used multiple times.",
	)

	return args
}

func GetImageMirrorConfigArgs(args *ImageMirrorConfigArgs) ([]string, []string, []string) {
	return args.imageContentSources, args.imageDigestMirrorSet, args.imageTagMirrorSet
}

func GetImageMirrorConfigOptions(cmd *pflag.FlagSet,
	args *ImageMirrorConfigArgs, isHostedCP bool) (*ImageMirrorConfigArgs, error) {

	if !isHostedCP {
		if IsImageMirrorConfigSetViaCLI(cmd) {
			return nil, fmt.Errorf("Setting image mirror configuration is only supported for hosted clusters")
		}
		return nil, nil
	}

	result := &ImageMirrorConfigArgs{}
	result.imageContentSources = args.imageContentSources
	result.imageDigestMirrorSet = args.imageDigestMirrorSet
	result.imageTagMirrorSet = args.imageTagMirrorSet

	if !IsImageMirrorConfigSetViaCLI(cmd) && !interactive.Enabled() {
		return nil, nil
	}

	enableImageMirrorConfig := IsImageMirrorConfigSetViaCLI(cmd)

	if interactive.Enabled() {
		updateImageMirrorConfigValue, err := interactive.GetBool(interactive.Input{
			Question: "Enable image mirror configuration for disconnected environments",
			Default:  enableImageMirrorConfig,
		})
		if err != nil {
			return nil, fmt.Errorf("Expected a valid image mirror config value: %s", err)
		}
		enableImageMirrorConfig = updateImageMirrorConfigValue

		if enableImageMirrorConfig {
			// Interactive input for image content sources
			icsInputs, err := interactive.GetString(interactive.Input{
				Question: "Image Content Sources (format: source=mirror1,mirror2)",
				Help:     cmd.Lookup(imageContentSourcesFlag).Usage,
				Default:  strings.Join(args.imageContentSources, ";"),
			})
			if err != nil {
				return nil, fmt.Errorf("Expected valid image content sources: %s", err)
			}
			if icsInputs != "" {
				result.imageContentSources = helper.HandleEmptyStringOnSlice(strings.Split(icsInputs, ";"))
			}

			// Interactive input for image digest mirror set
			idmsInputs, err := interactive.GetString(interactive.Input{
				Question: "Image Digest Mirror Set (format: source=mirror1,mirror2)",
				Help:     cmd.Lookup(imageDigestMirrorSetFlag).Usage,
				Default:  strings.Join(args.imageDigestMirrorSet, ";"),
			})
			if err != nil {
				return nil, fmt.Errorf("Expected valid image digest mirror set: %s", err)
			}
			if idmsInputs != "" {
				result.imageDigestMirrorSet = helper.HandleEmptyStringOnSlice(strings.Split(idmsInputs, ";"))
			}

			// Interactive input for image tag mirror set
			itmsInputs, err := interactive.GetString(interactive.Input{
				Question: "Image Tag Mirror Set (format: source=mirror1,mirror2)",
				Help:     cmd.Lookup(imageTagMirrorSetFlag).Usage,
				Default:  strings.Join(args.imageTagMirrorSet, ";"),
			})
			if err != nil {
				return nil, fmt.Errorf("Expected valid image tag mirror set: %s", err)
			}
			if itmsInputs != "" {
				result.imageTagMirrorSet = helper.HandleEmptyStringOnSlice(strings.Split(itmsInputs, ";"))
			}
		}
	}

	return result, nil
}

func IsImageMirrorConfigSetViaCLI(cmd *pflag.FlagSet) bool {
	for _, parameter := range []string{imageContentSourcesFlag,
		imageDigestMirrorSetFlag, imageTagMirrorSetFlag} {

		if cmd.Changed(parameter) {
			return true
		}
	}

	return false
}

func BuildImageMirrorConfigOptions(spec ocm.Spec) string {
	command := ""

	if len(spec.ImageContentSources) > 0 {
		sources := make([]string, len(spec.ImageContentSources))
		for i, ics := range spec.ImageContentSources {
			sources[i] = fmt.Sprintf("%s=%s", ics.Source, strings.Join(ics.Mirrors, ","))
		}
		command += fmt.Sprintf(" --%s %s",
			imageContentSourcesFlag,
			shellescape.Quote(strings.Join(sources, ";")))
	}

	if len(spec.ImageDigestMirrorSet) > 0 {
		sets := make([]string, len(spec.ImageDigestMirrorSet))
		for i, idms := range spec.ImageDigestMirrorSet {
			sets[i] = fmt.Sprintf("%s=%s", idms.Source, strings.Join(idms.Mirrors, ","))
		}
		command += fmt.Sprintf(" --%s %s",
			imageDigestMirrorSetFlag,
			shellescape.Quote(strings.Join(sets, ";")))
	}

	if len(spec.ImageTagMirrorSet) > 0 {
		sets := make([]string, len(spec.ImageTagMirrorSet))
		for i, itms := range spec.ImageTagMirrorSet {
			sets[i] = fmt.Sprintf("%s=%s", itms.Source, strings.Join(itms.Mirrors, ","))
		}
		command += fmt.Sprintf(" --%s %s",
			imageTagMirrorSetFlag,
			shellescape.Quote(strings.Join(sets, ";")))
	}

	return command
}

func ParseImageMirrorSpecs(specs []string) ([]ocm.ImageMirrorSet, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	result := make([]ocm.ImageMirrorSet, 0, len(specs))
	for _, spec := range specs {
		mirrorSet, err := ocm.ParseImageMirrorSpec(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to parse mirror spec '%s': %w", spec, err)
		}
		result = append(result, mirrorSet)
	}

	return result, nil
}

func ParseImageContentSources(specs []string) ([]ocm.ImageContentSource, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	result := make([]ocm.ImageContentSource, 0, len(specs))
	for _, spec := range specs {
		mirrorSet, err := ocm.ParseImageMirrorSpec(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to parse content source spec '%s': %w", spec, err)
		}
		result = append(result, ocm.ImageContentSource{
			Source:  mirrorSet.Source,
			Mirrors: mirrorSet.Mirrors,
		})
	}

	return result, nil
}