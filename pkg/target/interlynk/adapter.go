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
	"strings"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/interlynk-io/sbommv/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// InterlynkAdapter manages SBOM uploads to the Interlynk service.
type InterlynkAdapter struct {
	client   InterlynkAPI
	config   *InterlynkConfig
	uploader SBOMUploader

	Role types.AdapterRole
}

func NewInterlynkAdapter(config *InterlynkConfig, client InterlynkAPI) *InterlynkAdapter {
	uploader, ok := uploaderFactory[string(config.Settings.ProcessingMode)]
	if !ok {
		uploader = uploaderFactory[string(types.UploadSequential)]
	}
	return &InterlynkAdapter{
		config:   config,
		client:   client,
		uploader: uploader,
	}
}

// AddCommandParams adds GitHub-specific CLI flags
func (i *InterlynkAdapter) AddCommandParams(cmd *cobra.Command) {
	cmd.Flags().String("out-interlynk-url", "https://api.interlynk.io/lynkapi", "Interlynk API URL")
	cmd.Flags().String("out-interlynk-project-name", "", "Interlynk Project Name")
	cmd.Flags().String("out-interlynk-project-env", "default", "Interlynk Project Environment")
}

// ParseAndValidateParams validates the Interlynk adapter params
func (i *InterlynkAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var urlFlag, projectNameFlag, projectEnvFlag string
	var missingFlags []string
	var invalidFlags []string

	switch i.Role {

	case types.InputAdapterRole:
		return fmt.Errorf("The Interlynk adapter doesn't support input adapter functionalities.")

	case types.OutputAdapterRole:
		urlFlag = "out-interlynk-url"
		projectNameFlag = "out-interlynk-project-name"
		projectEnvFlag = "out-interlynk-project-env"

	default:
		return fmt.Errorf("The adapter is neither an input type nor an output type")
	}

	// Get flags
	url, _ := cmd.Flags().GetString(urlFlag)
	projectName, _ := cmd.Flags().GetString(projectNameFlag)
	projectEnv, _ := cmd.Flags().GetString(projectEnvFlag)

	// Check if INTERLYNK_SECURITY_TOKEN is set
	token := viper.GetString("INTERLYNK_SECURITY_TOKEN")
	if token == "" {
		return fmt.Errorf("missing INTERLYNK_SECURITY_TOKEN: authentication required")
	}

	// Validate Interlynk URL
	if !utils.IsValidURL(url) {
		invalidFlags = append(invalidFlags, fmt.Sprintf("invalid Interlynk API URL format: %s", url))
	}

	// Restrict `--out-interlynk-project-env` to only allowed values
	allowedEnvs := map[string]bool{"default": true, "development": true, "production": true}
	if !allowedEnvs[projectEnv] {
		invalidFlags = append(invalidFlags, fmt.Sprintf("invalid project environment: %s (allowed values: default, development, production)", projectEnv))
	}

	// Validate Interlynk connectivity before proceeding
	if err := ValidateInterlynkConnection(url, token); err != nil {
		return fmt.Errorf("Interlynk validation failed: %w", err)
	}

	// Show missing/invalid flags
	if len(missingFlags) > 0 {
		return fmt.Errorf("missing output adapter required flags: %v\n\nUse 'sbommv transfer --help' for usage details.", missingFlags)
	}
	if len(invalidFlags) > 0 {
		return fmt.Errorf("invalid output adapter flag usage:\n- %s\n\nUse 'sbommv transfer --help' for correct usage.", strings.Join(invalidFlags, "\n- "))
	}

	// Assign values to struct
	i.config.BaseURL = url
	i.config.ProjectName = projectName
	i.config.ProjectEnv = projectEnv
	i.config.ApiKey = token
	i.config.Settings = types.UploadSettings{ProcessingMode: types.UploadSequential}

	logger.LogDebug(cmd.Context(), "Interlynk parameters validated and assigned",
		"url", i.config.BaseURL,
		"project_name", i.config.ProjectName,
		"project_env", i.config.ProjectEnv,
	)
	return nil
}

// FetchSBOMs retrieves SBOMs lazily
func (i *InterlynkAdapter) FetchSBOMs(ctx *tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	return nil, fmt.Errorf("Interlynk adapter does not support SBOM Fetching")
}

func (i *InterlynkAdapter) UploadSBOMs(ctx *tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Starting SBOM upload", "mode", i.config.Settings.ProcessingMode)
	return i.uploader.Upload(ctx, i.client, iter)
}

// func (i *InterlynkAdapter) UploadSBOMs(ctx *tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
// 	logger.LogDebug(ctx.Context, "Starting SBOM upload", "mode", i.config.Settings.ProcessingMode)

// 	if i.config.Settings.ProcessingMode != "sequential" {
// 		return fmt.Errorf("unsupported processing mode: %s", i.config.Settings.ProcessingMode) // Future-proofed for parallel & batch
// 	}

// 	switch i.config.Settings.ProcessingMode {

// 	case types.UploadParallel:
// 		// TODO: cuncurrent upload: As soon as we get the SBOM, upload it
// 		// i.uploadParallel()
// 		return fmt.Errorf("processing mode %q not yet implemented", i.config.Settings.ProcessingMode)

// 	case types.UploadBatching:
// 		// TODO: hybrid of sequential + parallel
// 		// i.uploadBatch()
// 		return fmt.Errorf("processing mode %q not yet implemented", i.config.Settings.ProcessingMode)

// 	case types.UploadSequential:
// 		// Sequential Processing: Fetch SBOM → Upload → Repeat
// 		i.uploadSequential(ctx, iterator)

// 	default:
// 		//
// 		return fmt.Errorf("invalid processing mode: %q", i.config.Settings.ProcessingMode)
// 	}

// 	return nil
// }

// // uploadSequential handles sequential SBOM processing and uploading
// func (i *InterlynkAdapter) uploadSequential(ctx *tcontext.TransferMetadata, sboms iterator.SBOMIterator) error {
// 	logger.LogDebug(ctx.Context, "Uploading SBOMs in sequential mode")

// 	// // Initialize Interlynk API client
// 	// client := NewClient(Config{
// 	// 	Token:       i.config.ApiKey,
// 	// 	APIURL:      i.config.BaseURL,
// 	// 	ProjectName: i.config.ProjectName,
// 	// 	ProjectEnv:  i.config.ProjectEnv,
// 	// })

// 	errorCount := 0
// 	maxRetries := 5

// 	for {
// 		sbom, err := sboms.Next(ctx.Context)
// 		if err == io.EOF {
// 			logger.LogDebug(ctx.Context, "All SBOMs uploaded successfully, no more SBOMs left.")
// 			break
// 		}
// 		if err != nil {
// 			logger.LogInfo(ctx.Context, "Failed to retrieve SBOM from iterator", err)
// 			errorCount++
// 			if errorCount >= maxRetries {
// 				logger.LogInfo(ctx.Context, "Exceeded maximum retries", err)
// 				break
// 			}
// 			continue
// 		}
// 		errorCount = 0 // Reset error counter on successful iteration

// 		logger.LogDebug(ctx.Context, "Uploading SBOM", "repo", sbom.Namespace, "version", sbom.Version, "data size", len(sbom.Data))

// 		projectID, err := i.client.FindOrCreateProjectGroup(ctx, sbom.Namespace)
// 		if err != nil {
// 			logger.LogInfo(ctx.Context, "error", err)
// 			continue
// 		}

// 		// Upload SBOM content (stored in memory)
// 		err = i.client.UploadSBOM(ctx, projectID, sbom.Data)
// 		if err != nil {
// 			logger.LogInfo(ctx.Context, "Failed to upload SBOM", "repo", sbom.Namespace, "version", sbom.Version)
// 		} else {
// 			logger.LogDebug(ctx.Context, "Successfully uploaded SBOM", "repo", sbom.Namespace, "version", sbom.Version)
// 		}
// 	}

// 	return nil
// }

// DryRunUpload simulates SBOM upload to Interlynk without actual data transfer.
func (i *InterlynkAdapter) DryRun(ctx *tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	reporter := NewInterlynkReporter(i.config.BaseURL, i.config.ApiKey)
	return reporter.DryRun(ctx.Context, iter)
}

// // DryRunUpload simulates SBOM upload to Interlynk without actual data transfer.
// func (i *InterlynkAdapter) DryRun(ctx *tcontext.TransferMetadata, sbomIterator iterator.SBOMIterator) error {
// 	logger.LogDebug(ctx.Context, "🔄 Dry-Run Mode: Simulating Upload to Interlynk...")

// 	// Step 1: Validate Interlynk Connection
// 	err := ValidateInterlynkConnection(i.config.BaseURL, i.config.ApiKey)
// 	if err != nil {
// 		return fmt.Errorf("interlynk validation failed: %w", err)
// 	}

// 	// Step 2: Initialize SBOM Processor
// 	processor := sbom.NewSBOMProcessor("", false)

// 	// Step 3: Organize SBOMs into Projects
// 	projectSBOMs := make(map[string][]sbom.SBOMDocument)
// 	totalSBOMs := 0
// 	uniqueFormats := make(map[string]struct{})

// 	for {
// 		sbom, err := sbomIterator.Next(ctx.Context)
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
// 			continue
// 		}

// 		// Update processor with current SBOM data
// 		processor.Update(sbom.Data, sbom.Namespace, sbom.Path)

// 		// Process SBOM to extract metadata
// 		doc, err := processor.ProcessSBOMs()
// 		if err != nil {
// 			logger.LogError(ctx.Context, err, "Failed to process SBOM")
// 			continue
// 		}

// 		// Identify project name (repo-version)
// 		projectKey := fmt.Sprintf("%s", sbom.Namespace)
// 		projectSBOMs[projectKey] = append(projectSBOMs[projectKey], doc)
// 		totalSBOMs++
// 		uniqueFormats[string(doc.Format)] = struct{}{}
// 	}

// 	// Step 4: Print Dry-Run Summary
// 	fmt.Println("")
// 	fmt.Printf("📦 Interlynk API Endpoint: %s\n", i.config.BaseURL)
// 	fmt.Printf("📂 Project Groups Total: %d\n", len(projectSBOMs))
// 	fmt.Printf("📊 Total SBOMs to be Uploaded: %d\n", totalSBOMs)
// 	fmt.Printf("📦 INTERLYNK_SECURITY_TOKEN is valid\n")
// 	fmt.Printf("📦 Unique Formats: %s\n", formatSetToString(uniqueFormats))
// 	fmt.Println()

// 	// Step 5: Print Project Details
// 	for project, sboms := range projectSBOMs {
// 		fmt.Printf("📌 **Project: %s** → %d SBOMs\n", project, len(sboms))
// 		for _, doc := range sboms {
// 			fmt.Printf("   - 📁  | Format: %s | SpecVersion: %s | Size: %d KB | Filename: %s\n",
// 				doc.Format, doc.SpecVersion, len(doc.Content)/1024, doc.Filename)
// 		}
// 	}

// 	fmt.Println("\n✅ **Dry-run completed**. No data was uploaded to Interlynk.")
// 	return nil
// }
