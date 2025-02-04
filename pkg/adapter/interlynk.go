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

package adapter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/target/interlynk"
	"github.com/spf13/cobra"
)

type InterlynkAdapter struct {
	projectID       string
	baseURL         string
	apiKey          string
	client          *http.Client
	pushAllVersions bool

	// repoURL
	URL     string
	Version string
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
		return fmt.Errorf("missing or invalid flag: : in-github-url")
	}

	projectID, _ := cmd.Flags().GetString("out-interlynk-project-id")
	if projectID == "" {
		fmt.Println("New projects ID will be created as provided empty")
	}

	i.baseURL = url
	i.projectID = projectID

	return nil
}

// FetchSBOMs retrieves SBOMs lazily
func (i *InterlynkAdapter) FetchSBOMs(ctx context.Context) (iterator.SBOMIterator, error) {
	return nil, fmt.Errorf("GitHub adapter does not support SBOM uploading")
}

func (a *InterlynkAdapter) UploadSBOMs(ctx context.Context, iterator iterator.SBOMIterator) error {
	mode := "sequential"
	logger.LogDebug(ctx, "Starting SBOM upload", "mode", mode)

	if mode != "sequential" {
		return fmt.Errorf("unsupported processing mode: %s", mode) // Future-proofed for parallel & batch
	}

	switch mode {

	case "parallel":
		// TODO: cuncurrent upload: As soon as we get the SBOM, upload it

	case "batch":
		// TODO: hybrid of sequential + parallel
		// As soon as we get the batch size sbom, upload it.

	case "sequential":
		// Sequential Processing: Fetch SBOM → Upload → Repeat
		for {
			sbom, err := iterator.Next(ctx)
			if err == io.EOF {
				logger.LogDebug(ctx, "All SBOMs processed")
				break
			}
			if err != nil {
				logger.LogError(ctx, err, "Failed to fetch SBOM")
				continue
			}

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

			// Upload the SBOM file
			err = client.UploadSBOM(ctx, sbom.Path)
			if err != nil {
				logger.LogError(ctx, err, "Failed to upload SBOM", "path", sbom.Path)
				return fmt.Errorf("failed to upload SBOM: %w", err)
			}
			logger.LogDebug(ctx, "SBOM uploaded successfully", "path", sbom.Path)
		}
	default:
		//
	}

	return nil
}

func sanitizeRepoName(repoURL string) string {
	repoParts := strings.Split(repoURL, "/")
	if len(repoParts) < 2 {
		return "unknown"
	}
	return repoParts[len(repoParts)-1] // Extracts "cosign" from URL
}
