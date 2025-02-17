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
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/spf13/cobra"
)

// FolderAdapter handles storing SBOMs in a local folder
type FolderAdapter struct {
	Role       types.AdapterRole
	FolderPath string
	settings   types.UploadSettings
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

	f.FolderPath = folderPath
	f.settings.ProcessingMode = types.UploadMode(mode)

	logger.LogDebug(cmd.Context(), "Folder Output Adapter Initialized", "path", f.FolderPath)
	return nil
}

// FetchSBOMs retrieves SBOMs lazily
func (i *FolderAdapter) FetchSBOMs(ctx *tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	return nil, fmt.Errorf("Folder adapter does not support SBOM Fetching")
}

// UploadSBOMs writes SBOMs to the output folder
func (f *FolderAdapter) UploadSBOMs(ctx *tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Starting SBOM upload", "mode", f.settings.ProcessingMode)

	if f.settings.ProcessingMode != "sequential" {
		return fmt.Errorf("unsupported processing mode: %s", f.settings.ProcessingMode) // Future-proofed for parallel & batch
	}

	switch f.settings.ProcessingMode {

	case types.UploadParallel:
		// TODO: cuncurrent upload: As soon as we get the SBOM, upload it
		// f.uploadParallel()
		return fmt.Errorf("processing mode %q not yet implemented", f.settings.ProcessingMode)

	case types.UploadBatching:
		// TODO: hybrid of sequential + parallel
		// f.uploadBatch()
		return fmt.Errorf("processing mode %q not yet implemented", f.settings.ProcessingMode)

	case types.UploadSequential:
		// Sequential Processing: Fetch SBOM → Upload → Repeat
		f.uploadSequential(ctx, iterator)

	default:
		//
		return fmt.Errorf("invalid processing mode: %q", f.settings.ProcessingMode)
	}

	logger.LogDebug(ctx.Context, "All SBOMs have been successfully saved in directory", "value", f.FolderPath)
	return nil
}

// DryRun for Output Adapter: Simulates writing SBOMs to a folder
func (f *FolderAdapter) DryRun(ctx *tcontext.TransferMetadata, sbomIter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Dry-run mode: Displaying SBOMs that would be stored in folder")

	fmt.Println("\n📦 **Folder Output Adapter Dry-Run**")

	sbomCount := 0

	for {
		sbom, err := sbomIter.Next(ctx.Context)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			continue
		}

		namespace := filepath.Base(sbom.Namespace)
		if namespace == "" {
			namespace = fmt.Sprintf("sbom_%s.json", uuid.New().String()) // Generate unique filename
		}

		outputPath := filepath.Join(f.FolderPath, namespace)
		outputFile := filepath.Join(outputPath, sbom.Path)

		fmt.Printf("- 📂 Would write: %s\n", outputFile)
		sbomCount++
	}

	fmt.Printf("\n📊 Total SBOMs to be stored: %d\n", sbomCount)
	logger.LogDebug(ctx.Context, "Dry-run mode completed for folder output adapter", "total_sboms", sbomCount)
	return nil
}

func (f *FolderAdapter) uploadSequential(ctx *tcontext.TransferMetadata, sbomIter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Writing SBOMs in sequential mode", "folder", f.FolderPath)

	// Process SBOMs
	for {
		sbom, err := sbomIter.Next(ctx.Context)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			continue
		}

		namespace := filepath.Base(sbom.Namespace)
		if namespace == "" {
			namespace = fmt.Sprintf("sbom_%s.json", uuid.New().String()) // Generate unique filename
		}

		// Construct output path (preserve filename if available)
		outputDir := filepath.Join(f.FolderPath, namespace)
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			logger.LogError(ctx.Context, err, "Failed to create folder", "path", outputDir)
			continue
		}

		outputFile := filepath.Join(outputDir, sbom.Path)
		if sbom.Path == "" {
			outputFile = filepath.Join(outputDir, fmt.Sprintf("%s.sbom.json", uuid.New().String()))
		}

		// Write SBOM file
		if err := os.WriteFile(outputFile, sbom.Data, 0o644); err != nil {
			logger.LogError(ctx.Context, err, "Failed to write SBOM file", "path", outputFile)
			continue
		}

		logger.LogDebug(ctx.Context, "Successfully written SBOM", "path", outputFile)
	}
	return nil
}
