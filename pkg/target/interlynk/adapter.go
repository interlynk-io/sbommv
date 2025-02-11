// Copyright 2025 Interlynk.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package interlynk

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// InterlynkAdapter manages SBOM uploads to the Interlynk service.
type InterlynkAdapter struct {
	// Config fields
	ProjectName string
	ProjectEnv  string

	BaseURL string
	ApiKey  string
	Role    types.AdapterRole

	// HTTP client for API requests
	client   *http.Client
	settings UploadSettings
}

type UploadMode string

const (
	UploadParallel   UploadMode = "parallel"
	UploadBatching   UploadMode = "batch"
	UploadSequential UploadMode = "sequential"
)

// UploadSettings contains configuration for SBOM uploads
type UploadSettings struct {
	ProcessingMode UploadMode // "sequential", "parallel", or "batch"
}

// AddCommandParams adds GitHub-specific CLI flags
func (i *InterlynkAdapter) AddCommandParams(cmd *cobra.Command) {
	cmd.Flags().String("out-interlynk-url", "https://api.interlynk.io/lynkapi", "Interlynk API URL")
	cmd.Flags().String("out-interlynk-project-name", "", "Interlynk Project Name")
	cmd.Flags().String("out-interlynk-project-env", "default", "Interlynk Project Environment")
}

// ParseAndValidateParams validates the GitHub adapter params
func (i *InterlynkAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var urlFlag, projectNameFlag, projectEnvFlag string

	if i.Role == types.InputAdapter {
		urlFlag = "in-interlynk-url"
		projectNameFlag = "in-interlynk-project-name"
		projectEnvFlag = "in-interlynk-project-env"
	} else {
		urlFlag = "out-interlynk-url"
		projectNameFlag = "out-interlynk-project-name"
		projectEnvFlag = "out-interlynk-project-env"
	}

	url, _ := cmd.Flags().GetString(urlFlag)
	projectName, _ := cmd.Flags().GetString(projectNameFlag)
	projectEnv, _ := cmd.Flags().GetString(projectEnvFlag)

	token := viper.GetString("INTERLYNK_SECURITY_TOKEN")
	if token == "" {
		return fmt.Errorf("missing INTERLYNK_SECURITY_TOKEN: authentication required")
	}

	if url == "" {
		i.BaseURL = "https://api.interlynk.io/lynkapi"
	} else {
		i.BaseURL = url
	}

	i.ProjectName = projectName
	i.ProjectEnv = projectEnv
	i.ApiKey = token
	i.settings = UploadSettings{
		ProcessingMode: UploadSequential,
	}

	// 🔹 Validate Interlynk connection before proceeding
	if err := ValidateInterlynkConnection(i.BaseURL, i.ApiKey); err != nil {
		return fmt.Errorf("Interlynk validation failed: %w", err)
	}

	logger.LogDebug(cmd.Context(), "Interlynk system is up and running.")

	return nil
}

// FetchSBOMs retrieves SBOMs lazily
func (i *InterlynkAdapter) FetchSBOMs(ctx *tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	return nil, fmt.Errorf("Interlynk adapter does not support SBOM Fetching")
}

func (i *InterlynkAdapter) UploadSBOMs(ctx *tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Starting SBOM upload", "mode", i.settings.ProcessingMode)

	if i.settings.ProcessingMode != "sequential" {
		return fmt.Errorf("unsupported processing mode: %s", i.settings.ProcessingMode) // Future-proofed for parallel & batch
	}

	switch i.settings.ProcessingMode {

	case UploadParallel:
		// TODO: cuncurrent upload: As soon as we get the SBOM, upload it
		// i.uploadParallel()
		return fmt.Errorf("processing mode %q not yet implemented", i.settings.ProcessingMode)

	case UploadBatching:
		// TODO: hybrid of sequential + parallel
		// i.uploadBatch()
		return fmt.Errorf("processing mode %q not yet implemented", i.settings.ProcessingMode)

	case UploadSequential:
		// Sequential Processing: Fetch SBOM → Upload → Repeat
		i.uploadSequential(ctx, iterator)

	default:
		//
		return fmt.Errorf("invalid processing mode: %q", i.settings.ProcessingMode)
	}

	return nil
}

// uploadSequential handles sequential SBOM processing and uploading
func (i *InterlynkAdapter) uploadSequential(ctx *tcontext.TransferMetadata, sboms iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Uploading SBOMs in sequential mode")

	// Initialize Interlynk API client
	client := NewClient(Config{
		Token:       i.ApiKey,
		APIURL:      i.BaseURL,
		ProjectName: i.ProjectName,
		ProjectEnv:  i.ProjectEnv,
	})

	errorCount := 0
	maxRetries := 5

	for {
		sbom, err := sboms.Next(ctx.Context)
		if err == io.EOF {
			logger.LogDebug(ctx.Context, "All SBOMs uploaded successfully, no more SBOMs left.")
			break
		}
		if err != nil {
			logger.LogInfo(ctx.Context, "Failed to retrieve SBOM from iterator", err)
			errorCount++
			if errorCount >= maxRetries {
				logger.LogInfo(ctx.Context, "Exceeded maximum retries", err)
				break
			}
			continue
		}
		errorCount = 0 // Reset error counter on successful iteration

		logger.LogDebug(ctx.Context, "Uploading SBOM", "repo", sbom.Repo, "version", sbom.Version, "data size", len(sbom.Data))

		projectID, err := client.FindOrCreateProjectGroup(ctx, sbom.Repo)
		if err != nil {
			logger.LogInfo(ctx.Context, "error", err)
			continue
		}

		// Upload SBOM content (stored in memory)
		err = client.UploadSBOM(ctx, projectID, sbom.Data)
		if err != nil {
			logger.LogInfo(ctx.Context, "Failed to upload SBOM", "repo", sbom.Repo, "version", sbom.Version)
		} else {
			logger.LogDebug(ctx.Context, "Successfully uploaded SBOM", "repo", sbom.Repo, "version", sbom.Version)
		}
	}

	return nil
}

// // DryRun for Output Adapter: Displays all SBOMs that to be uploaded by output adapter
// func (i *InterlynkAdapter) DryRun(ctx *tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
// 	logger.LogDebug(ctx.Context, "Dry-run mode: Displaying SBOMs fetched from output adapter")
// 	// TODO: Need to add core functionality
// 	return nil
// }

// DryRunUpload simulates SBOM upload to Interlynk without actually performing the upload
// DryRunUpload simulates SBOM upload to Interlynk without actual data transfer.
func (i *InterlynkAdapter) DryRun(ctx *tcontext.TransferMetadata, sbomIterator iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "🔄 Dry-Run Mode: Simulating Upload to Interlynk...")

	// Step 1: Validate Interlynk Connection
	err := ValidateInterlynkConnection(i.BaseURL, i.ApiKey)
	if err != nil {
		return fmt.Errorf("interlynk validation failed: %w", err)
	}

	// Step 2: Initialize SBOM Processor
	processor := sbom.NewSBOMProcessor("", false)

	// Step 3: Organize SBOMs into Projects
	projectSBOMs := make(map[string][]sbom.SBOMDocument)
	totalSBOMs := 0
	uniqueFormats := make(map[string]struct{})

	for {
		sbom, err := sbomIterator.Next(ctx.Context)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			continue
		}

		// Process SBOM to extract metadata
		doc, err := processor.ProcessSBOMs(sbom.Data, sbom.Repo, sbom.Path)
		if err != nil {
			logger.LogError(ctx.Context, err, "Failed to process SBOM")
			continue
		}

		// Identify project name (repo-version)
		projectKey := fmt.Sprintf("%s-%s", sbom.Repo, sbom.Version)
		projectSBOMs[projectKey] = append(projectSBOMs[projectKey], doc)
		totalSBOMs++
		uniqueFormats[string(doc.Format)] = struct{}{}
	}

	// Step 4: Print Dry-Run Summary
	fmt.Println("")
	fmt.Printf("📦 Interlynk API Endpoint: %s/vendor/products/upload\n", i.BaseURL)
	fmt.Printf("📂 Project Groups Total: %d\n", len(projectSBOMs))
	fmt.Printf("📊 Total SBOMs to be Uploaded: %d\n", totalSBOMs)
	fmt.Printf("📦 INTERLYNK_SECURITY_TOKEN is valid\n")
	fmt.Printf("📦 Unique Formats: %s\n", formatSetToString(uniqueFormats))
	fmt.Println()

	// Step 5: Print Project Details
	for project, sboms := range projectSBOMs {
		fmt.Printf("📌 **Project: %s** → %d SBOMs\n", project, len(sboms))
		for _, doc := range sboms {
			fmt.Printf("   - 📁  | Format: %s | SpecVersion: %s | Size: %d KB | Filename: %s\n",
				doc.Format, doc.SpecVersion, len(doc.Content)/1024, doc.Filename)
		}
	}

	fmt.Println("\n✅ **Dry-run completed**. No data was uploaded to Interlynk.")
	return nil
}

// formatSetToString converts a map of unique formats to a comma-separated string
func formatSetToString(formatSet map[string]struct{}) string {
	var formats []string
	for format := range formatSet {
		formats = append(formats, format)
	}
	return strings.Join(formats, ", ")
}
