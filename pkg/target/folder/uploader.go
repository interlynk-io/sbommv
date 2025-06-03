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
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
)

type SBOMUploader interface {
	Upload(ctx tcontext.TransferMetadata, config *FolderConfig, iter iterator.SBOMIterator) error
}

var uploaderFactory = map[types.UploadMode]SBOMUploader{
	types.UploadSequential: &SequentialUploader{},
	// Add parallel uploader later
}

type SequentialUploader struct{}

func (u *SequentialUploader) Upload(ctx tcontext.TransferMetadata, config *FolderConfig, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Writing SBOMs sequentially", "folder", config.FolderPath)
	totalSBOMs := 0
	successfullyUploaded := 0
	failed := 0

	// space for proper logging
	fmt.Println()

	for {
		sbom, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		totalSBOMs++
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			return err
		}
		outputDir := config.FolderPath

		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			logger.LogError(ctx.Context, err, "Failed to create folder", "path", outputDir)
			return err
		}
		outputFile := filepath.Join(outputDir, sbom.Path)
		if sbom.Path == "" {
			outputFile = filepath.Join(outputDir, fmt.Sprintf("%s.sbom.json", uuid.New().String()))
		}

		if !config.Overwrite {

			// skip if file exists(default behavior)
			if _, err := os.Stat(outputFile); err == nil {

				// file exists, skip writing
				logger.LogDebug(ctx.Context, "File already exists, skipping write (overwrite=false)", "path", outputFile)
				successfullyUploaded++
				continue

			} else if !os.IsNotExist(err) {

				// unexpected error (not just "file doesn’t exist")
				logger.LogError(ctx.Context, err, "Failed to check file existence", "path", outputFile)
				continue
			}

			logger.LogDebug(ctx.Context, "Written to file", "path", outputFile)
		}

		// write the SBOM file (either overwrite is true or file doesn’t exist)
		if err := os.WriteFile(outputFile, sbom.Data, 0o644); err != nil {
			logger.LogError(ctx.Context, err, "Failed to write SBOM file", "path", outputFile)
			failed++
			continue // Continue to next SBOM instead of returning error
		}

		successfullyUploaded++
		logger.LogInfo(ctx.Context, "wrote", "path", outputFile)
	}

	logger.LogInfo(ctx.Context, "wrote", "sboms", totalSBOMs, "success", successfullyUploaded, "failed", failed)

	return nil
}
