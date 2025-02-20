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
	"github.com/spf13/cobra"
)

// FolderAdapter handles fetching SBOMs from folders
type FolderAdapter struct {
	config  *FolderConfig
	Role    types.AdapterRole // "input" or "output" adapter type
	Fetcher SBOMFetcher
}

// AddCommandParams adds Folder-specific CLI flags
func (f *FolderAdapter) AddCommandParams(cmd *cobra.Command) {
	cmd.Flags().String("in-folder-path", "", "Folder path")
	cmd.Flags().Bool("in-folder-recursive", false, "Folder recurssive (default: false)")
	cmd.Flags().String("in-folder-processing-mode", "sequential", "Folder processing mode (sequential/parallel)")
}

// ParseAndValidateParams validates the Folder adapter params
func (f *FolderAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var (
		pathFlag, recursiveFlag, processingModeFlag string
		missingFlags                                []string
		invalidFlags                                []string
	)

	switch f.Role {
	case types.InputAdapterRole:
		pathFlag = "in-folder-path"
		recursiveFlag = "in-folder-recursive"
		processingModeFlag = "in-folder-processing-mode"

	case types.OutputAdapterRole:
		return fmt.Errorf("The Folder adapter doesn't support output adapter functionalities.")

	default:
		return fmt.Errorf("The adapter is neither an input type nor an output type")

	}

	// Extract Folder Path
	folderPath, _ := cmd.Flags().GetString(pathFlag)
	if folderPath == "" {
		missingFlags = append(missingFlags, "--"+pathFlag)
	}

	// Extract Folder Path
	folderRecurse, _ := cmd.Flags().GetBool(recursiveFlag)

	validModes := map[string]bool{"sequential": true, "parallel": true}

	// Extract the processing mode: sequential/parallel
	mode, _ := cmd.Flags().GetString(processingModeFlag)
	if !validModes[mode] {
		invalidFlags = append(invalidFlags, fmt.Sprintf("%s=%s (must be one of: sequential, parallel mode)", processingModeFlag, mode))
	}

	// Validate required flags
	if len(missingFlags) > 0 {
		return fmt.Errorf("missing input adapter required flags: %v\n\nUse 'sbommv transfer --help' for usage details.", missingFlags)
	}

	// Validate incorrect flag usage
	if len(invalidFlags) > 0 {
		return fmt.Errorf("invalid input adapter flag usage:\n %s\n\nUse 'sbommv transfer --help' for correct usage.", strings.Join(invalidFlags, "\n "))
	}

	cfg := FolderConfig{
		FolderPath:     folderPath,
		Recursive:      folderRecurse,
		ProcessingMode: types.ProcessingMode(mode),
	}

	f.config = &cfg

	return nil
}

// FetchSBOMs initializes the Folder SBOM iterator using the unified method
func (f *FolderAdapter) FetchSBOMs(ctx *tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Initializing SBOM fetching", "mode", f.config.ProcessingMode)
	return f.Fetcher.Fetch(ctx, f.config)
}

// OutputSBOMs should return an error since Folder does not support SBOM uploads
func (f *FolderAdapter) UploadSBOMs(ctx *tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	return fmt.Errorf("Folder adapter does not support SBOM uploading")
}

// DryRun for Folder Adapter: Displays all fetched SBOMs from folder adapter
func (f *FolderAdapter) DryRun(ctx *tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	reporter := NewFolderReporter(false, "")
	return reporter.DryRun(ctx.Context, iter)
}
