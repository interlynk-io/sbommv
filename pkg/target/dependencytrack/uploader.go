// Copyright 2025 Interlynk.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"sync"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type SBOMUploader interface {
	Upload(ctx *tcontext.TransferMetadata, config *DependencyTrackConfig, client *DependencyTrackClient, iter iterator.SBOMIterator) error
}

type SequentialUploader struct {
	createdProjects map[string]bool // Cache of created project names
	mu              sync.Mutex      // Protect map access
}

func NewSequentialUploader() *SequentialUploader {
	return &SequentialUploader{
		createdProjects: make(map[string]bool), // Initialize map
	}
}

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

		projectName := config.ProjectName
		if projectName == "" {
			if sbom.Namespace == "" {
				return fmt.Errorf("no project name specified and SBOM namespace is empty")
			}
			projectName = sbom.Namespace
		}

		u.mu.Lock()
		if !u.createdProjects[projectName] {
			_, err = client.FindOrCreateProject(ctx, projectName, config.ProjectVersion)
			if err != nil {
				logger.LogInfo(ctx.Context, "Failed to find or create project", "project", projectName, "error", err)
				u.mu.Unlock()
				continue
			}
			u.createdProjects[projectName] = true
			logger.LogDebug(ctx.Context, "Project created", "project", projectName)
		}
		u.mu.Unlock()

		// Log SBOM filename before upload
		logger.LogDebug(ctx.Context, "Attempting to upload SBOM", "project", projectName, "file", sbom.Path)

		err = client.UploadSBOM(ctx, projectName, config.ProjectVersion, sbom.Data)
		if err != nil {
			logger.LogInfo(ctx.Context, "Failed to upload SBOM", "project", projectName, "file", sbom.Path, "error", err)
			continue
		}
		logger.LogDebug(ctx.Context, "Successfully uploaded SBOM", "project", projectName, "file", sbom.Path)
	}
	return nil
}

var uploaderFactory = map[string]SBOMUploader{
	"sequential": NewSequentialUploader(),
}
