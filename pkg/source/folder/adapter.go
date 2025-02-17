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
	"io"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/spf13/cobra"
)

// FolderAdapter handles fetching SBOMs from folders
type FolderAdapter struct {
	FolderPath     string // Folder path where SBOMs exist or will be stored
	Recursive      bool   // Scan subdirectories (for input mode)
	ProcessingMode types.ProcessingMode

	Role types.AdapterRole // "input" or "output" adapter type
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

	f.FolderPath = folderPath
	f.Recursive = folderRecurse
	f.ProcessingMode = types.ProcessingMode(mode)

	return nil
}

// FetchSBOMs initializes the Folder SBOM iterator using the unified method
func (f *FolderAdapter) FetchSBOMs(ctx *tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Initializing SBOM fetching process")
	logger.LogDebug(ctx.Context, "Scanning folder for SBOMs", "path", f.FolderPath, "recursive", f.Recursive)

	var sbomIterator iterator.SBOMIterator
	var err error

	switch f.ProcessingMode {
	case types.FetchParallel:
		sbomIterator, err = f.fetchSBOMsConcurrently(ctx)
	case types.FetchSequential:
		sbomIterator, err = f.fetchSBOMsSequentially(ctx)
	default:
		return nil, fmt.Errorf("Unsupported Processing Mode !!")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch SBOMs: %w", err)
	}

	return sbomIterator, nil
}

// OutputSBOMs should return an error since Folder does not support SBOM uploads
func (f *FolderAdapter) UploadSBOMs(ctx *tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	return fmt.Errorf("Folder adapter does not support SBOM uploading")
}

// DryRun for Folder Adapter: Displays all fetched SBOMs from folder adapter
func (f *FolderAdapter) DryRun(ctx *tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Dry-run mode: Displaying SBOMs fetched from folder input adapter")

	var outputDir string
	var verbose bool

	processor := sbom.NewSBOMProcessor(outputDir, verbose)
	sbomCount := 0
	fmt.Println()
	fmt.Printf("📦 Details of all Fetched SBOMs by Folder Input Adapter\n")

	for {

		sbom, err := iterator.Next(ctx.Context)
		if err == io.EOF {
			break // No more sboms
		}

		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
		}

		// update processor with current SBOM data
		processor.Update(sbom.Data, "", sbom.Path)

		doc, err := processor.ProcessSBOMs()
		if err != nil {
			logger.LogError(ctx.Context, err, "Failed to process SBOM")
			continue
		}

		// if outputDir is provided, save the SBOM file
		if outputDir != "" {
			if err := processor.WriteSBOM(doc, ""); err != nil {
				logger.LogError(ctx.Context, err, "Failed to write SBOM to output directory")
			}
		}

		// Print SBOM content if verbose mode is enabled
		if verbose {
			fmt.Println("\n-------------------- 📜 SBOM Content --------------------")
			fmt.Printf("📂 Filename: %s\n", doc.Filename)
			fmt.Printf("📦 Format: %s | SpecVersion: %s\n\n", doc.Format, doc.SpecVersion)
			fmt.Println(string(doc.Content))
			fmt.Println("------------------------------------------------------")
			fmt.Println()
		}

		sbomCount++
		fmt.Printf(" - 📁 Folder: %s | Format: %s | SpecVersion: %s | Filename: %s \n", sbom.Namespace, doc.Format, doc.SpecVersion, doc.Filename)

	}

	return nil
}
