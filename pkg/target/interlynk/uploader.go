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

package interlynk

import (
	"fmt"
	"io"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
)

type SBOMUploader interface {
	Upload(ctx *tcontext.TransferMetadata, client *Client, iter iterator.SBOMIterator) error
}

type SequentialUploader struct{}

var uploaderFactory = map[string]SBOMUploader{
	string(types.UploadSequential): &SequentialUploader{},
	// Add parallel and batch uploaders later
}

func (u *SequentialUploader) Upload(ctx *tcontext.TransferMetadata, client *Client, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Uploading SBOMs sequentially")
	errorCount := 0
	maxRetries := 5

	for {
		sbom, err := iter.Next(ctx.Context)
		if err == io.EOF {
			logger.LogDebug(ctx.Context, "All SBOMs uploaded successfully")
			break
		}
		if err != nil {
			logger.LogInfo(ctx.Context, "Failed to retrieve SBOM", "error", err)
			errorCount++
			if errorCount >= maxRetries {
				return fmt.Errorf("exceeded max retries (%d): %w", maxRetries, err)
			}
			continue
		}
		errorCount = 0
		projectID, err := client.FindOrCreateProjectGroup(ctx, sbom.Namespace)
		if err != nil {
			logger.LogInfo(ctx.Context, "Failed to get project", "repo", sbom.Namespace, "error", err)
			continue
		}

		if err := client.UploadSBOM(ctx, projectID, sbom.Data); err != nil {
			logger.LogInfo(ctx.Context, "Failed to upload SBOM", "repo", sbom.Namespace, "error", err)
		} else {
			logger.LogDebug(ctx.Context, "Successfully uploaded SBOM", "repo", sbom.Namespace)
		}
	}
	return nil
}

// TODO:

/*
type ParallelUploader struct{}

func (u *ParallelUploader) Upload(ctx *tcontext.TransferMetadata, client *Client, iter iterator.SBOMIterator) error {
}

type BatchUploader struct{}
func (u *BatchUploader) Upload(ctx *tcontext.TransferMetadata, client *Client, iter iterator.SBOMIterator) error {
}
*/
