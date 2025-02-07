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
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// InterlynkAdapter manages SBOM uploads to the Interlynk service.
type InterlynkAdapter struct {
	// Config fields
	ProjectID string
	BaseURL   string
	ApiKey    string
	Role      types.AdapterRole

	// HTTP client for API requests
	client *http.Client

	// Repository info
	RepoURL  string
	Version  string
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
	cmd.Flags().String("out-interlynk-url", "", "Interlynk API URL")
	cmd.Flags().String("out-interlynk-project-id", "", "Interlynk Project ID")
}

// ParseAndValidateParams validates the GitHub adapter params
func (i *InterlynkAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var urlFlag, projectIDFlag string

	if i.Role == types.InputAdapter {
		urlFlag = "in-interlynk-url"
		projectIDFlag = "in-interlynk-project-id"
	} else {
		urlFlag = "out-interlynk-url"
		projectIDFlag = "out-interlynk-project-id"
	}

	url, _ := cmd.Flags().GetString(urlFlag)
	if url == "" {
		return fmt.Errorf("missing or invalid flag: %s", urlFlag)
	}

	projectID, _ := cmd.Flags().GetString(projectIDFlag)
	if projectID == "" {
		fmt.Println("Warning: No project ID provided, a new project will be created")
	}

	token := viper.GetString("INTERLYNK_SECURITY_TOKEN")
	if token == "" {
		return fmt.Errorf("INTERLYNK_SECURITY_TOKEN environment variable is required")
	}

	i.BaseURL = url
	i.ProjectID = projectID
	i.ApiKey = token
	i.settings = UploadSettings{
		ProcessingMode: UploadSequential,
	}

	// 🔹 Validate Interlynk connection before proceeding
	if err := ValidateInterlynkConnection(i.BaseURL, i.ApiKey); err != nil {
		return fmt.Errorf("Interlynk validation failed: %w", err)
	}

	fmt.Println("✅ Interlynk system is up and running.")

	return nil
}

// FetchSBOMs retrieves SBOMs lazily
func (i *InterlynkAdapter) FetchSBOMs(ctx *tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	return nil, fmt.Errorf("GitHub adapter does not support SBOM uploading")
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
		Token:     i.ApiKey,
		APIURL:    i.BaseURL,
		ProjectID: i.ProjectID,
	})

	// Retrieve metadata from context
	repoURL, _ := ctx.Value("repo_url").(string)
	repoVersion, _ := ctx.Value("repo_version").(string)
	totalSBOMs, _ := ctx.Value("total_sboms").(int)

	if repoVersion == "" {
		repoVersion = "all-version"
	}
	fmt.Println("repoVersion: ", repoVersion)
	fmt.Println("totalSBOMs: ", totalSBOMs)

	repoName := sanitizeRepoName(repoURL)
	fmt.Println("repoName: ", repoName)

	// Create project if needed
	if client.ProjectID == "" {
		projectName := fmt.Sprintf("%s-%s", repoName, repoVersion)
		logger.LogDebug(ctx.Context, "Creating new project", "name", projectName)

		projectID, err := client.CreateProjectGroup(ctx, projectName, "Project for SBOM", true)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		logger.LogDebug(ctx.Context, "New project created successfully", "name", projectName)
		client.SetProjectID(projectID)
	}

	// Initialize progress bar
	bar := progressbar.Default(int64(totalSBOMs), "🚀 Uploading SBOMs")

	for {
		sbom, err := sboms.Next(ctx.Context)
		if err == io.EOF {
			logger.LogDebug(ctx.Context, "All SBOMs uploaded successfully, no more SBOMs left.")
			break
		}
		if err != nil {
			logger.LogError(ctx.Context, err, "Failed to retrieve SBOM from iterator")
			continue
		}

		logger.LogDebug(ctx.Context, "Uploading SBOM", "repo", sbom.Repo, "version", sbom.Version)

		// Upload SBOM content (stored in memory)
		err = client.UploadSBOM(ctx, sbom.Data)
		if err != nil {
			logger.LogDebug(ctx.Context, "Failed to upload SBOM", "repo", sbom.Repo, "version", sbom.Version)
		} else {
			logger.LogDebug(ctx.Context, "Successfully uploaded SBOM", "repo", sbom.Repo, "version", sbom.Version)
		}

		// Update progress bar
		if err := bar.Add(1); err != nil {
			logger.LogError(ctx.Context, err, "Error updating progress bar")
		}
	}
	logger.LogInfo(ctx.Context, "✅ All SBOMs uploaded successfully!")

	return nil
}

func sanitizeRepoName(repoURL string) string {
	repoParts := strings.Split(repoURL, "/")
	if len(repoParts) < 2 {
		return "unknown"
	}
	return repoParts[len(repoParts)-1]
}
