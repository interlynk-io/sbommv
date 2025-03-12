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

// FolderAdapter handles storing SBOMs in a local folder
type FolderAdapter struct {
	Role     types.AdapterRole
	config   *FolderConfig
	Uploader SBOMUploader
}

// AddCommandParams defines folder adapter CLI flags
func (f *FolderAdapter) AddCommandParams(cmd *cobra.Command) {
	cmd.Flags().String("out-folder-path", "", "The folder where SBOMs should be stored")
	cmd.Flags().String("out-folder-processing-mode", "sequential", "Folder processing mode (sequential/parallel)")
}

// ParseAndValidateParams validates the folder path
func (f *FolderAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var pathFlag string
	var processingModeFlag string
	var missingFlags []string
	var invalidFlags []string

	switch f.Role {
	case types.InputAdapterRole:
		return fmt.Errorf("The Folder adapter doesn't support output adapter functionalities.")

	case types.OutputAdapterRole:
		pathFlag = "out-folder-path"
		processingModeFlag = "out-folder-processing-mode"

	default:
		return fmt.Errorf("The adapter is neither an input type nor an output type")

	}

	// validate flags for respective adapters
	utils.FlagValidation(cmd, types.FolderAdapterType, types.OutputAdapterFlagPrefix)

	// Extract Folder Path
	folderPath, _ := cmd.Flags().GetString(pathFlag)
	if folderPath == "" {
		missingFlags = append(missingFlags, "--"+pathFlag)
	}

	validModes := map[string]bool{"sequential": true, "parallel": true}
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
		FolderPath: folderPath,
		Settings:   types.UploadSettings{ProcessingMode: types.UploadMode(mode)},
	}
	f.config = &cfg

	logger.LogDebug(cmd.Context(), "Folder Output Adapter Initialized", "path", f.config.FolderPath)
	return nil
}

// FetchSBOMs retrieves SBOMs lazily
func (i *FolderAdapter) FetchSBOMs(ctx tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	return nil, fmt.Errorf("Folder adapter does not support SBOM Fetching")
}

// UploadSBOMs writes SBOMs to the output folder
func (f *FolderAdapter) UploadSBOMs(ctx tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Starting SBOM upload", "mode", f.config.Settings.ProcessingMode)
	return f.Uploader.Upload(ctx, f.config, iter)
}

// DryRun for Output Adapter: Simulates writing SBOMs to a folder
func (f *FolderAdapter) DryRun(ctx tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	reporter := NewFolderOutputReporter(f.config.FolderPath)
	return reporter.DryRun(ctx, iter)
}
