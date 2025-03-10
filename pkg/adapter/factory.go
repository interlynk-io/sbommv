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
	"fmt"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/target/dependencytrack"
	ofolder "github.com/interlynk-io/sbommv/pkg/target/folder"

	ifolder "github.com/interlynk-io/sbommv/pkg/source/folder"
	"github.com/interlynk-io/sbommv/pkg/source/github"
	"github.com/interlynk-io/sbommv/pkg/target/interlynk"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/spf13/cobra"
)

// Adapter defines the interface for all adapters
type Adapter interface {
	// Adds CLI flags to the commands
	AddCommandParams(cmd *cobra.Command)

	// Parses & validates input params
	ParseAndValidateParams(cmd *cobra.Command) error

	// Fetch SBOMs lazily using iterator
	FetchSBOMs(ctx *tcontext.TransferMetadata) (iterator.SBOMIterator, error)

	// Outputs SBOMs (uploading)
	UploadSBOMs(ctx *tcontext.TransferMetadata, iterator iterator.SBOMIterator) error

	// Dry-Run: to be used to display fetched and uploaded SBOMs by input and output adapter respectively.
	DryRun(ctx *tcontext.TransferMetadata, iterator iterator.SBOMIterator) error
}

// NewAdapter initializes and returns the correct adapters (both input & output)
func NewAdapter(ctx *tcontext.TransferMetadata, config types.Config) (map[types.AdapterRole]Adapter, string, string, error) {
	adapters := make(map[types.AdapterRole]Adapter)
	var inputAdp, outputAdp string

	// Initialize Input Adapter
	if config.SourceAdapter != "" {
		logger.LogDebug(ctx.Context, "Initializing Input Adapter", "InputAdapter", config.SourceAdapter)

		processingMode := types.ProcessingMode(config.ProcessingStrategy)
		switch types.AdapterType(config.SourceAdapter) {

		case types.GithubAdapterType:
			adapters[types.InputAdapterRole] = &github.GitHubAdapter{Role: types.InputAdapterRole, ProcessingMode: processingMode}
			inputAdp = "github"

		case types.FolderAdapterType:
			adapters[types.InputAdapterRole] = &ifolder.FolderAdapter{Role: types.InputAdapterRole, Fetcher: &ifolder.SequentialFetcher{}}
			inputAdp = "folder"

		case types.InterlynkAdapterType:
			adapters[types.InputAdapterRole] = &interlynk.InterlynkAdapter{Role: types.InputAdapterRole}
			inputAdp = "interlynk"

		default:
			return nil, "", "", fmt.Errorf("unsupported input adapter type: %s", config.SourceAdapter)
		}
	}

	// Initialize Output Adapter
	if config.DestinationAdapter != "" {
		logger.LogDebug(ctx.Context, "Initializing Output Adapter", "OutputAdapter", config.DestinationAdapter)

		switch types.AdapterType(config.DestinationAdapter) {

		case types.FolderAdapterType:
			adapters[types.OutputAdapterRole] = &ofolder.FolderAdapter{Role: types.OutputAdapterRole, Uploader: &ofolder.SequentialUploader{}}
			outputAdp = "folder"

		case types.InterlynkAdapterType:
			adapters[types.OutputAdapterRole] = &interlynk.InterlynkAdapter{Role: types.OutputAdapterRole}
			outputAdp = "interlynk"

		case types.DtrackAdapterType:
			adapters[types.OutputAdapterRole] = &dependencytrack.DependencyTrackAdapter{Role: types.OutputAdapterRole, Uploader: dependencytrack.NewSequentialUploader()}
			outputAdp = "dtrack"
		default:
			return nil, "", "", fmt.Errorf("unsupported output adapter type: %s", config.DestinationAdapter)
		}
	}

	if len(adapters) == 0 {
		return nil, "", "", fmt.Errorf("no valid adapters found")
	}

	return adapters, inputAdp, outputAdp, nil
}
