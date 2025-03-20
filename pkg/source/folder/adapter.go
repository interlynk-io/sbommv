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

package folder

import (
	"fmt"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/interlynk-io/sbommv/pkg/utils"
	"github.com/spf13/cobra"
)

// FolderAdapter handles fetching SBOMs from folders
type FolderAdapter struct {
	Config  *FolderConfig
	Role    types.AdapterRole // "input" or "output" adapter type
	Fetcher SBOMFetcher
}

// AddCommandParams adds Folder-specific CLI flags
func (f *FolderAdapter) AddCommandParams(cmd *cobra.Command) {
	cmd.Flags().String("in-folder-path", "", "Folder path")
	cmd.Flags().Bool("in-folder-recursive", false, "Folder recurssive (default: false)")
}

// ParseAndValidateParams validates the Folder adapter params
func (f *FolderAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var (
		pathFlag, recursiveFlag string
		missingFlags            []string
		invalidFlags            []string
	)

	switch f.Role {
	case types.InputAdapterRole:
		pathFlag = "in-folder-path"
		recursiveFlag = "in-folder-recursive"

	case types.OutputAdapterRole:
		return fmt.Errorf("The Folder adapter doesn't support output adapter functionalities.")

	default:
		return fmt.Errorf("The adapter is neither an input type nor an output type")

	}

	// validate flags for respective adapters
	err := utils.FlagValidation(cmd, types.FolderAdapterType, types.InputAdapterFlagPrefix)
	if err != nil {
		return fmt.Errorf("folder flag validation failed: %w", err)
	}

	// Extract Folder Path
	folderPath, _ := cmd.Flags().GetString(pathFlag)
	if folderPath == "" {
		missingFlags = append(missingFlags, "--"+pathFlag)
	}

	// Extract Folder Path
	folderRecurse, _ := cmd.Flags().GetBool(recursiveFlag)

	// Validate required flags
	if len(missingFlags) > 0 {
		return fmt.Errorf("missing input adapter required flags: %v\n\nUse 'sbommv transfer --help' for usage details.", missingFlags)
	}

	// Validate incorrect flag usage
	if len(invalidFlags) > 0 {
		return fmt.Errorf("invalid input adapter flag usage:\n %s\n\nUse 'sbommv transfer --help' for correct usage.", strings.Join(invalidFlags, "\n "))
	}
	var fetcher SBOMFetcher
	daemon := f.Config.Daemon

	if daemon {
		// daemon fether initialized
		fetcher = NewWatcherFetcher()
	} else if f.Config.ProcessingMode == types.FetchSequential {
		fetcher = &SequentialFetcher{}
	} else if f.Config.ProcessingMode == types.FetchParallel {
		fetcher = &ParallelFetcher{}
	}

	cfg := FolderConfig{
		FolderPath:     folderPath,
		Recursive:      folderRecurse,
		Daemon:         daemon,
		ProcessingMode: f.Config.ProcessingMode,
	}

	f.Config = &cfg
	f.Fetcher = fetcher

	return nil
}

// FetchSBOMs initializes the Folder SBOM iterator using the unified method
func (f *FolderAdapter) FetchSBOMs(ctx tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Initializing SBOM fetching", "mode", f.Config.ProcessingMode)
	return f.Fetcher.Fetch(ctx, f.Config)
}

func (f *FolderAdapter) Monitor(ctx tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	if !f.Config.Daemon {
		return nil, fmt.Errorf("daemon mode not enabled for folder adapter")
	}

	logger.LogDebug(ctx.Context, "Initializing SBOM monitoring")
	return f.Fetcher.Fetch(ctx, f.Config)
}

// OutputSBOMs should return an error since Folder does not support SBOM uploads
func (f *FolderAdapter) UploadSBOMs(ctx tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	return fmt.Errorf("Folder adapter does not support SBOM uploading")
}

// DryRun for Folder Adapter: Displays all fetched SBOMs from folder adapter
func (f *FolderAdapter) DryRun(ctx tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	reporter := NewFolderReporter(false, "", f.Config.FolderPath)
	return reporter.DryRun(ctx, iter)
}
