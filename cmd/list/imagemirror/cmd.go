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
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	registry        string
	username        string
	password        string
	token           string
	credentialsFile string
	timeout         time.Duration
	outputFormat    string
}

var Cmd = &cobra.Command{
	Use:     "image-mirror",
	Aliases: []string{"mirror", "images"},
	Short:   "List images in a registry",
	Long:    "List images available in a container registry using IDMS",
	Example: `  # List images in a registry
  rosa list image-mirror --registry registry.io

  # List images with authentication
  rosa list image-mirror --registry registry.io --username user --password pass

  # List images using token authentication
  rosa list image-mirror --registry registry.io --token mytoken

  # List images with JSON output
  rosa list image-mirror --registry registry.io --output json`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.registry,
		"registry",
		"",
		"Container registry URL to list images from",
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

	// Mark required flags
	cobra.MarkFlagRequired(flags, "registry")

	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
	output.AddFlag(Cmd)
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

	// Create image mirror configuration
	config := &imagemirror.ImageMirrorConfig{
		Authentication: authConfig,
		Timeout:        args.timeout,
	}

	// Create IDMS service
	idms := imagemirror.NewIDMS(config)

	// List images
	ctx := context.Background()
	images, err := idms.ListImages(ctx, args.registry)
	if err != nil {
		r.Reporter.Errorf("Failed to list images: %v", err)
		os.Exit(1)
	}

	// Display results
	displayImages(r, images)
}

func validateArgs() error {
	if args.registry == "" {
		return fmt.Errorf("registry is required")
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

func displayImages(r *rosa.Runtime, images []imagemirror.ImageManifest) {
	if output.HasFlag() {
		err := output.Print(images)
		if err != nil {
			r.Reporter.Errorf("Failed to output images: %v", err)
			os.Exit(1)
		}
		return
	}

	// Display in table format
	if len(images) == 0 {
		r.Reporter.Infof("No images found in registry")
		return
	}

	r.Reporter.Infof("Images in registry:")
	r.Reporter.Infof("===================")

	for _, image := range images {
		r.Reporter.Infof("Repository: %s", image.Repository)
		r.Reporter.Infof("Tag: %s", image.Tag)
		r.Reporter.Infof("Digest: %s", image.Digest)
		r.Reporter.Infof("Size: %s", formatBytes(image.Size))
		r.Reporter.Infof("Last Modified: %s", image.LastModified.Format(time.RFC3339))
		r.Reporter.Infof("---")
	}

	r.Reporter.Infof("Total images: %d", len(images))
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