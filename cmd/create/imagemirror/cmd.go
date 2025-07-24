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

package imagemirror

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/imagemirror"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	sourceRegistry   string
	targetRegistry   string
	images           []string
	username         string
	password         string
	token            string
	credentialsFile  string
	timeout          time.Duration
	maxConcurrency   int
	dryRun           bool
}

var Cmd = &cobra.Command{
	Use:     "image-mirror",
	Aliases: []string{"mirror"},
	Short:   "Create image mirror operations",
	Long:    "Create and manage image mirror operations between container registries using IDMS and ITMS",
	Example: `  # Mirror images from one registry to another
  rosa create image-mirror --source-registry source.io --target-registry target.io --images image1:tag1,image2:tag2

  # Mirror with authentication
  rosa create image-mirror --source-registry source.io --target-registry target.io --username user --password pass --images image1:latest

  # Mirror using token authentication
  rosa create image-mirror --source-registry source.io --target-registry target.io --token mytoken --images image1:latest`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.sourceRegistry,
		"source-registry",
		"",
		"Source container registry URL",
	)

	flags.StringVar(
		&args.targetRegistry,
		"target-registry",
		"",
		"Target container registry URL",
	)

	flags.StringSliceVar(
		&args.images,
		"images",
		[]string{},
		"Comma-separated list of images to mirror (format: repo:tag)",
	)

	flags.StringVar(
		&args.username,
		"username",
		"",
		"Username for registry authentication",
	)

	flags.StringVar(
		&args.password,
		"password",
		"",
		"Password for registry authentication",
	)

	flags.StringVar(
		&args.token,
		"token",
		"",
		"Token for registry authentication",
	)

	flags.StringVar(
		&args.credentialsFile,
		"credentials-file",
		"",
		"Path to credentials file for registry authentication",
	)

	flags.DurationVar(
		&args.timeout,
		"timeout",
		30*time.Second,
		"Timeout for operations",
	)

	flags.IntVar(
		&args.maxConcurrency,
		"max-concurrency",
		5,
		"Maximum number of concurrent operations",
	)

	flags.BoolVar(
		&args.dryRun,
		"dry-run",
		false,
		"Perform a dry run without actually mirroring images",
	)

	// Mark required flags
	cobra.MarkFlagRequired(flags, "source-registry")
	cobra.MarkFlagRequired(flags, "target-registry")
	cobra.MarkFlagRequired(flags, "images")

	confirm.AddFlag(flags)
	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Validate arguments
	if err := validateArgs(); err != nil {
		r.Reporter.Errorf("Validation failed: %v", err)
		os.Exit(1)
	}

	// Create authentication config
	authConfig := imagemirror.AuthConfig{
		Username:        args.username,
		Password:        args.password,
		Token:           args.token,
		CredentialsFile: args.credentialsFile,
	}

	// Create image mirror service
	service := imagemirror.NewImageMirrorServiceWithAuth(
		args.sourceRegistry,
		args.targetRegistry,
		authConfig,
	)
	defer service.Close()

	// Basic validation - since we don't have ValidateConfiguration in the interface,
	// we'll just check that the service was created successfully
	if service == nil {
		r.Reporter.Errorf("Failed to create image mirror service")
		os.Exit(1)
	}

	// Ask for confirmation unless auto-approved
	if !confirm.Prompt(true, "Are you sure you want to proceed with image mirroring?") {
		if !args.dryRun {
			r.Reporter.Infof("Image mirroring operation cancelled by user")
			os.Exit(0)
		}
	}

	if args.dryRun {
		r.Reporter.Infof("Dry run mode - no images will be actually mirrored")
		performDryRun(r, service)
		return
	}

	// Perform the actual mirroring
	performMirroring(r, service)
}

func validateArgs() error {
	if args.sourceRegistry == "" {
		return fmt.Errorf("source registry is required")
	}

	if args.targetRegistry == "" {
		return fmt.Errorf("target registry is required")
	}

	if len(args.images) == 0 {
		return fmt.Errorf("at least one image must be specified")
	}

	if args.maxConcurrency <= 0 {
		return fmt.Errorf("max concurrency must be greater than 0")
	}

	if args.timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}

	// Validate authentication
	if args.username != "" && args.password == "" && args.token == "" {
		return fmt.Errorf("password or token required when username is specified")
	}

	return nil
}

func performDryRun(r *rosa.Runtime, service imagemirror.ImageMirrorService) {
	r.Reporter.Infof("Dry run: Would mirror %d images from %s to %s",
		len(args.images), args.sourceRegistry, args.targetRegistry)

	for _, image := range args.images {
		sourceRef := fmt.Sprintf("%s/%s", args.sourceRegistry, image)
		targetRef := fmt.Sprintf("%s/%s", args.targetRegistry, image)
		r.Reporter.Infof("  %s -> %s", sourceRef, targetRef)
	}

	r.Reporter.Infof("Dry run completed successfully")
}

func performMirroring(r *rosa.Runtime, service imagemirror.ImageMirrorService) {
	ctx := context.Background()

	r.Reporter.Infof("Starting image mirroring operation...")
	r.Reporter.Infof("Source registry: %s", args.sourceRegistry)
	r.Reporter.Infof("Target registry: %s", args.targetRegistry)
	r.Reporter.Infof("Images to mirror: %d", len(args.images))

	// Build image mapping for mirroring
	images := make(map[string]string)
	for _, image := range args.images {
		sourceRef := fmt.Sprintf("%s/%s", args.sourceRegistry, image)
		targetRef := fmt.Sprintf("%s/%s", args.targetRegistry, image)
		images[sourceRef] = targetRef
	}

	r.Reporter.Infof("Mirroring %d images from %s to %s", 
		len(images), args.sourceRegistry, args.targetRegistry)

	// Mirror the images
	results, err := service.SyncImages(ctx, images)
	if err != nil {
		r.Reporter.Errorf("Failed to mirror images: %v", err)
		os.Exit(1)
	}

	// Display results
	displayResults(r, results)
}

func displayResults(r *rosa.Runtime, results []imagemirror.SyncResult) {
	successful := 0
	failed := 0

	r.Reporter.Infof("\nImage mirroring results:")
	r.Reporter.Infof("========================")

	for _, result := range results {
		if result.Success {
			successful++
			r.Reporter.Infof("✓ %s -> %s (%.2f seconds, %s transferred)",
				result.Source,
				result.Target,
				result.Duration.Seconds(),
				formatBytes(result.BytesTransferred))
		} else {
			failed++
			r.Reporter.Errorf("✗ %s -> %s: %s",
				result.Source,
				result.Target,
				result.Error)
		}
	}

	r.Reporter.Infof("\nSummary:")
	r.Reporter.Infof("========")
	r.Reporter.Infof("Total images: %d", len(results))
	r.Reporter.Infof("Successful: %d", successful)
	r.Reporter.Infof("Failed: %d", failed)

	if failed > 0 {
		os.Exit(1)
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}