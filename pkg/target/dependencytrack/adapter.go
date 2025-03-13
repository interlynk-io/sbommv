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

package dependencytrack

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

type DependencyTrackAdapter struct {
	Config         *DependencyTrackConfig
	client         *DependencyTrackClient
	Uploader       SBOMUploader
	Role           types.AdapterRole
	ProcessingMode types.ProcessingMode
}

// func NewDependencyTrackAdapter(config *DependencyTrackConfig, client *DependencyTrackClient) *DependencyTrackAdapter {
// 	uploader := uploaderFactory["sequential"]
// 	return &DependencyTrackAdapter{
// 		Config:   config,
// 		client:   client,
// 		Uploader: uploader,
// 	}
// }

func (d *DependencyTrackAdapter) AddCommandParams(cmd *cobra.Command) {
	cmd.Flags().String("out-dtrack-url", "", "Dependency Track API URL")
	cmd.Flags().String("out-dtrack-project-name", "", "Project name to upload SBOMs to")
	cmd.Flags().String("out-dtrack-project-version", "latest", "Project version (default: latest)")
}

// ParseAndValidateParams validates the Dependency-Track adapter params
func (d *DependencyTrackAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var (
		urlFlag, projectNameFlag, projectVersionFlag string
		missingFlags                                 []string
		invalidFlags                                 []string
	)

	switch d.Role {
	case types.InputAdapterRole:
		return fmt.Errorf("The Dependency-Track adapter doesn't support input adapter functionalities.")

	case types.OutputAdapterRole:
		urlFlag = "out-dtrack-url"
		projectNameFlag = "out-dtrack-project-name"
		projectVersionFlag = "out-dtrack-project-version"

	default:
		return fmt.Errorf("The adapter is neither an input type nor an output type")
	}

	// Extract flags
	apiURL := viper.GetString("DTRACK_API_URL")

	if apiURL == "" {
		apiURL, _ = cmd.Flags().GetString(urlFlag)
	}

	if !utils.IsValidURL(apiURL) {
		invalidFlags = append(invalidFlags, fmt.Sprintf("invalid DTrack API URL format: %s", apiURL))
	}

	// Check if DTRACK_API_KEY is set
	token := viper.GetString("DTRACK_API_KEY")
	if token == "" {
		return fmt.Errorf("missing DTRACK_API_KEY: authentication required")
	}
	projectName, _ := cmd.Flags().GetString(projectNameFlag)
	projectVersion, _ := cmd.Flags().GetString(projectVersionFlag)

	// Check missing flags
	if len(missingFlags) > 0 {
		return fmt.Errorf("missing required flags: %v\nUse 'sbommv transfer --help' for usage details.", missingFlags)
	}
	if len(invalidFlags) > 0 {
		return fmt.Errorf("invalid flag usage:\n- %s\nUse 'sbommv transfer --help' for correct usage.", strings.Join(invalidFlags, "\n- "))
	}

	var uploader SBOMUploader
	// SequentialFetcher
	if d.ProcessingMode == types.FetchSequential {
		uploader = NewSequentialUploader()
	} else if d.ProcessingMode == types.FetchParallel {
		uploader = NewParallelUploader()
	}

	cfg := NewDependencyTrackConfig()
	cfg.APIURL = apiURL
	cfg.APIKey = token
	cfg.ProjectName = projectName
	cfg.ProjectVersion = projectVersion

	// Set values to struct
	d.Config = cfg

	// Initialize the DependencyTrack client
	client := NewDependencyTrackClient(d.Config)
	d.client = client
	d.Uploader = uploader

	logger.LogDebug(cmd.Context(), "Dependency-Track parameters validated and assigned",
		"url", d.Config.APIURL,
		"apiKey", d.Config.APIKey,
		"project_name", d.Config.ProjectName,
		"project_version", d.Config.ProjectVersion,
	)
	return nil
}

// FetchSBOMs returns an error since Dependency-Track is an output adapter
func (d *DependencyTrackAdapter) FetchSBOMs(ctx tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	return nil, fmt.Errorf("Dependency-Track adapter does not support SBOM fetching")
}

func (d *DependencyTrackAdapter) UploadSBOMs(ctx tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	return d.Uploader.Upload(ctx, d.Config, d.client, iter)
}

func (d *DependencyTrackAdapter) DryRun(ctx tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	reporter := NewDependencyTrackReporter(d.Config.APIURL, d.Config.ProjectName)
	return reporter.DryRun(ctx, iter)
}
