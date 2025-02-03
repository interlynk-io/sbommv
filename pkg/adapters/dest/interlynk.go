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
// -------------------------------------------------------------------------

package dest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/mvtypes"
	"github.com/interlynk-io/sbommv/pkg/target/interlynk"
)

// InterlynkAdapter implements OutputAdapter for the Interlynk platform
type InterlynkAdapter struct {
	projectID       string
	baseURL         string
	apiKey          string
	client          *http.Client
	options         OutputOptions
	pushAllVersions bool

	// repoURL
	URL     string
	Version string
}

// NewInterlynkAdapter creates a new Interlynk adapter
func NewInterlynkAdapter(config mvtypes.Config) *InterlynkAdapter {
	url := config.SourceConfigs["url"].(string)
	version := config.SourceConfigs["version"].(string)

	baseurl := config.DestinationConfigs["url"].(string)
	projectID := config.DestinationConfigs["projectID"].(string)
	token := config.DestinationConfigs["token"].(string)
	pushAllVersion := config.DestinationConfigs["pushAllVersion"]

	return &InterlynkAdapter{
		projectID:       projectID,
		baseURL:         baseurl,
		apiKey:          token,
		client:          &http.Client{},
		pushAllVersions: pushAllVersion.(bool),
		URL:             url,
		Version:         version,
		// options:   config.OutputOptions,
	}
}

// InterlynkAdapter implements UploadSBOMs. Hence it implement OutputAdapter.
func (a *InterlynkAdapter) UploadSBOMs(ctx context.Context, allSBOMs map[string][]string) error {
	// Initialize Interlynk client
	client := interlynk.NewClient(interlynk.Config{
		Token:     a.apiKey,
		APIURL:    a.baseURL,
		ProjectID: a.projectID,
	})

	if client.ProjectID == "" {
		// Create a new project
		projectName := fmt.Sprintf("%s-%s", sanitizeRepoName(a.URL), a.Version)

		projectID, err := client.CreateProjectGroup(ctx, projectName, fmt.Sprintf("Project for SBOM"), true)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}
		logger.LogDebug(ctx, "Created project", "projectID", projectID, "project Name", projectName)
		client.SetProjectID(projectID)
	}

	if a.pushAllVersions {
		for version, sboms := range allSBOMs {
			logger.LogDebug(ctx, "Processing SBOMs for version", "version", version)

			// Initialize upload service
			uploadService := interlynk.NewUploadService(client, interlynk.UploadOptions{})

			logger.LogDebug(ctx, "Current version uploading", "value", version)

			// Upload SBOMs for the current version
			results := uploadService.UploadSBOMs(ctx, sboms)
			for _, result := range results {
				if result.Error != nil {
					logger.LogDebug(ctx, "Failed to upload SBOM", "file", result.Path, "error", result.Error)
				} else {
					logger.LogDebug(ctx, "Successfully uploaded SBOM", "file", result.Path)
				}
			}
		}
	} else {

		// Initialize upload service
		uploadService := interlynk.NewUploadService(client, interlynk.UploadOptions{})

		for _, sboms := range allSBOMs {
			// Upload SBOMs
			results := uploadService.UploadSBOMs(ctx, sboms)

			// Log results
			for _, result := range results {
				if result.Error != nil {
					logger.LogDebug(ctx, "Failed to upload SBOMs", "response", result.Error)
				} else {
					logger.LogDebug(ctx, "SBOM uploaded successfully", "file", result.Path)
				}
			}
		}
	}
	return nil
}
