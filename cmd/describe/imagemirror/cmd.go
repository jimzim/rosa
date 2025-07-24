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
	imageRef        string
	operationID     string
	username        string
	password        string
	token           string
	credentialsFile string
	timeout         time.Duration
}

var Cmd = &cobra.Command{
	Use:     "image-mirror",
	Aliases: []string{"mirror"},
	Short:   "Describe image or operation details",
	Long:    "Describe image details or mirror operation status using IDMS and ITMS",
	Example: `  # Describe an image
  rosa describe image-mirror --image registry.io/repo:tag

  # Describe a sync operation
  rosa describe image-mirror --operation-id abc123

  # Describe image with authentication
  rosa describe image-mirror --image registry.io/repo:tag --username user --password pass`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.imageRef,
		"image",
		"",
		"Image reference to describe (format: registry.io/repo:tag)",
	)

	flags.StringVar(
		&args.operationID,
		"operation-id",
		"",
		"Sync operation ID to describe",
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

	// Create image mirror service
	service := imagemirror.NewImageMirrorServiceWithAuth("", "", authConfig)
	defer service.Close()

	ctx := context.Background()

	if args.imageRef != "" {
		describeImage(r, service, ctx)
	} else if args.operationID != "" {
		describeOperation(r, service, ctx)
	}
}

func validateArgs() error {
	if args.imageRef == "" && args.operationID == "" {
		return fmt.Errorf("either --image or --operation-id must be specified")
	}

	if args.imageRef != "" && args.operationID != "" {
		return fmt.Errorf("cannot specify both --image and --operation-id")
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

func describeImage(r *rosa.Runtime, service imagemirror.ImageMirrorService, ctx context.Context) {
	r.Reporter.Infof("Retrieving image information for: %s", args.imageRef)

	// Get image info
	image, err := service.GetImageInfo(ctx, args.imageRef)
	if err != nil {
		r.Reporter.Errorf("Failed to get image info: %v", err)
		os.Exit(1)
	}

	// Validate image access
	if err := service.ValidateImageAccess(ctx, args.imageRef); err != nil {
		r.Reporter.Warnf("Image access validation failed: %v", err)
	}

	// Display image information
	if output.HasFlag() {
		err := output.Print(image)
		if err != nil {
			r.Reporter.Errorf("Failed to output image info: %v", err)
			os.Exit(1)
		}
		return
	}

	displayImageInfo(r, image)
}

func describeOperation(r *rosa.Runtime, service imagemirror.ImageMirrorService, ctx context.Context) {
	r.Reporter.Infof("Retrieving operation status for: %s", args.operationID)

	// Get operation status
	status, err := service.GetSyncStatus(ctx, args.operationID)
	if err != nil {
		r.Reporter.Errorf("Failed to get operation status: %v", err)
		os.Exit(1)
	}

	// Display operation status
	if output.HasFlag() {
		err := output.Print(status)
		if err != nil {
			r.Reporter.Errorf("Failed to output operation status: %v", err)
			os.Exit(1)
		}
		return
	}

	displayOperationStatus(r, status)
}

func displayImageInfo(r *rosa.Runtime, image *imagemirror.ImageManifest) {
	r.Reporter.Infof("Image Information:")
	r.Reporter.Infof("==================")
	r.Reporter.Infof("Repository: %s", image.Repository)
	r.Reporter.Infof("Tag: %s", image.Tag)
	r.Reporter.Infof("Digest: %s", image.Digest)
	r.Reporter.Infof("Size: %s", formatBytes(image.Size))
	r.Reporter.Infof("Last Modified: %s", image.LastModified.Format(time.RFC3339))
}

func displayOperationStatus(r *rosa.Runtime, status *imagemirror.SyncStatus) {
	r.Reporter.Infof("Operation Status:")
	r.Reporter.Infof("=================")
	r.Reporter.Infof("Operation ID: %s", status.OperationID)
	r.Reporter.Infof("Status: %s", status.Status)
	r.Reporter.Infof("Progress: %d%%", status.Progress)
	r.Reporter.Infof("Start Time: %s", status.StartTime.Format(time.RFC3339))
	
	if status.EndTime != nil {
		r.Reporter.Infof("End Time: %s", status.EndTime.Format(time.RFC3339))
		duration := status.EndTime.Sub(status.StartTime)
		r.Reporter.Infof("Duration: %s", duration.String())
	}

	if len(status.Results) > 0 {
		r.Reporter.Infof("\nResults:")
		r.Reporter.Infof("========")
		
		successful := 0
		failed := 0
		
		for _, result := range status.Results {
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
		
		r.Reporter.Infof("\nSummary: %d successful, %d failed", successful, failed)
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