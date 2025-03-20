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
		if err := os.WriteFile(outputFile, sbom.Data, 0o644); err != nil {
			logger.LogError(ctx.Context, err, "Failed to write SBOM file", "path", outputFile)
			return err
		}
		successfullyUploaded++
		logger.LogDebug(ctx.Context, "Successfully written SBOM", "path", outputFile)
	}

	logger.LogInfo(ctx.Context, "SBOM uploading processing done, no more SBOMs left")
	logger.LogInfo(ctx.Context, "Total SBOMs", "count", totalSBOMs)
	logger.LogInfo(ctx.Context, "Successfully Uploaded", "count", successfullyUploaded)

	return nil
}
