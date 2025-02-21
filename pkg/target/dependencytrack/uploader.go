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
	"io"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type SBOMUploader interface {
	Upload(ctx *tcontext.TransferMetadata, config *DependencyTrackConfig, client *DependencyTrackClient, iter iterator.SBOMIterator) error
}

type SequentialUploader struct{}

func (u *SequentialUploader) Upload(ctx *tcontext.TransferMetadata, config *DependencyTrackConfig, client *DependencyTrackClient, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Uploading SBOMs to Dependency-Track sequentially")

	for {
		sbom, err := iter.Next(ctx.Context)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			return err
		}

		// Use config.ProjectName if provided, otherwise fall back to sbom.Namespace
		projectName := config.ProjectName
		if projectName == "" {
			if sbom.Namespace == "" {
				return fmt.Errorf("no project name specified and SBOM namespace is empty")
			}
			projectName = sbom.Namespace // e.g., "cosign", "fulcio", "rekor"
		}

		// Ensure project exists (idempotent PUT)
		_, err = client.CreateProject(ctx, projectName)
		if err != nil {
			logger.LogInfo(ctx.Context, "Failed to create project", "project", projectName, "error", err)
			continue
		}

		// Upload SBOM to the project
		err = client.UploadSBOM(ctx, projectName, sbom.Data)
		if err != nil {
			logger.LogInfo(ctx.Context, "Failed to upload SBOM", "project", projectName, "error", err)
			continue
		}
		logger.LogDebug(ctx.Context, "Successfully uploaded SBOM", "project", projectName)
	}
	return nil
}

var uploaderFactory = map[string]SBOMUploader{
	"sequential": &SequentialUploader{},
}
