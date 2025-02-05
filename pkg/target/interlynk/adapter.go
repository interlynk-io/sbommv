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
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// InterlynkAdapter manages SBOM uploads to the Interlynk service.
type InterlynkAdapter struct {
	// Config fields
	ProjectID string
	BaseURL   string
	ApiKey    string

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
	url, _ := cmd.Flags().GetString("out-interlynk-url")
	if url == "" {
		return fmt.Errorf("missing or invalid flag: : out-interlynk-url")
	}

	projectID, _ := cmd.Flags().GetString("out-interlynk-project-id")
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
func (i *InterlynkAdapter) FetchSBOMs(ctx context.Context) (iterator.SBOMIterator, error) {
	return nil, fmt.Errorf("GitHub adapter does not support SBOM uploading")
}

func (i *InterlynkAdapter) UploadSBOMs(ctx context.Context, iterator iterator.SBOMIterator) error {
	logger.LogDebug(ctx, "Starting SBOM upload", "mode", i.settings.ProcessingMode)

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

func sanitizeRepoName(repoURL string) string {
	repoParts := strings.Split(repoURL, "/")
	if len(repoParts) < 2 {
		return "unknown"
	}
	return repoParts[len(repoParts)-1]
}

// uploadSequential handles sequential SBOM processing and uploading
func (i *InterlynkAdapter) uploadSequential(ctx context.Context, iterator iterator.SBOMIterator) error {
	// interlynk adapter client
	client := NewClient(Config{
		Token:     i.ApiKey,
		APIURL:    i.BaseURL,
		ProjectID: i.ProjectID,
	})

	// Create project if needed
	if client.ProjectID == "" {
		projectName := fmt.Sprintf("%s-%s", sanitizeRepoName(i.RepoURL), i.Version)
		projectID, err := client.CreateProjectGroup(ctx, projectName, "Project for SBOM", true)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}
		client.SetProjectID(projectID)
	}

	// upload SBOMs
	for {
		sbom, err := iterator.Next(ctx)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to fetch SBOM: %w", err)
		}

		if err := client.UploadSBOM(ctx, sbom.Path); err != nil {
			return fmt.Errorf("failed to upload SBOM %q: %w", sbom.Path, err)
		}
	}
}
