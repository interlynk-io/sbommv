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

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
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
