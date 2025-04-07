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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/utils"
)

type SBOMUploader interface {
	Upload(ctx tcontext.TransferMetadata, config *DependencyTrackConfig, client *DependencyTrackClient, iter iterator.SBOMIterator) error
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

func (u *SequentialUploader) Upload(ctx tcontext.TransferMetadata, config *DependencyTrackConfig, client *DependencyTrackClient, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Initializing SBOMs uploading to Dependency-Track sequentially")

	totalSBOMs := 0
	successfullyUploaded := 0
	for {
		sbom, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}

		totalSBOMs++

		if err != nil {
			logger.LogDebug(ctx.Context, "Next: failed to get next SBOM continuing", "error", err)
			continue
		}

		sourceAdapter := ctx.Value("source")

		// if project name is provided that's well and good, else we need to construct project name from:
		// SBOM primary component name and lasly from file name
		projectName, projectVersion := utils.ConstructProjectName(ctx, config.ProjectName, config.ProjectVersion, sbom.Namespace, sbom.Version, sbom.Data, sourceAdapter.(string))
		if projectName == "" {
			// THIS CASE OCCURS WHEN SBOM IS NOT IN JSON FORMAT
			// when a JSON SBOM has empty primary comp and version, use the file name as project name

			projectName = filepath.Base(sbom.Path)
			projectName = projectName[:len(projectName)-len(filepath.Ext(projectName))]
			projectVersion = "latest"
		}

		finalProjectName := fmt.Sprintf("%s-%s", projectName, projectVersion)
		logger.LogDebug(ctx.Context, "Project Details", "name", finalProjectName, "version", projectVersion)

		var projectUUID string
		if !u.createdProjects[finalProjectName] {

			// find or create project using project name and project version
			projectUUID, err = client.FindOrCreateProject(ctx, finalProjectName, projectVersion)
			if err != nil {
				logger.LogInfo(ctx.Context, "Failed to find or create project", "project", projectName, "error", err)
				continue
			}
			u.createdProjects[finalProjectName] = true
		}

		logger.LogDebug(ctx.Context, "Initializing uploading SBOM content", "size", len(sbom.Data), "file", sbom.Path)

		if config.Overwrite {
			// Check if the SBOM file already exists in Dependency-Track
			parsedUUID, err := uuid.Parse(projectUUID)
			if err != nil {
				logger.LogDebug(ctx.Context, "Failed to parse project UUID", "projectUUID", projectUUID, "error", err)
				continue
			}

			// Check if project exists and has an SBOM (components)
			project, err := client.Client.Project.Get(ctx.Context, parsedUUID)
			if err != nil {
				logger.LogDebug(ctx.Context, "Failed to fetch project, assuming it’s new", "project", finalProjectName, "error", err)
			} else {

				// BOM import occurs when you upload an SBOM file
				// therefore, LastBomImport is non-zero)
				hasSBOM := project.LastBOMImport != 0

				// Optionally, check metrics if available
				if project.Metrics.Components > 0 {
					hasSBOM = true
				}

				logger.LogDebug(ctx.Context, "Project exists", "project", finalProjectName, "uuid", projectUUID)
				logger.LogDebug(ctx.Context, "Project metrics", "components", project.Metrics.Components, "last_bom_import", project.LastBOMImport)
				logger.LogDebug(ctx.Context, "Project active status", "active", project.Active)
				logger.LogDebug(ctx.Context, "Project has SBOM", "has_sbom", hasSBOM)

				if project.Active && hasSBOM {
					logger.LogInfo(ctx.Context, "Project exists and has an SBOM, skipping upload",
						"project", finalProjectName,
						"uuid", projectUUID,
						"last_bom_import", project.LastBOMImport,
						"component_count", project.Metrics.Components)
					successfullyUploaded++ // Count as successful since it’s an intentional skip
					continue
				}
				logger.LogDebug(ctx.Context, "Project exists but no SBOM detected, proceeding with upload", "project", finalProjectName)
			}

		}
		err = client.UploadSBOM(ctx, finalProjectName, projectVersion, sbom.Data)
		if err != nil {
			logger.LogDebug(ctx.Context, "Upload Failed for", "project", finalProjectName, "size", len(sbom.Data), "file", sbom.Path, "error", err)
			continue
		}
		successfullyUploaded++
		logger.LogInfo(ctx.Context, "Successfully uploaded SBOM to dependency track", "project", finalProjectName, "version", projectVersion, "file", sbom.Path)
		logger.LogDebug(ctx.Context, "Successfully uploaded SBOM file", "size", len(sbom.Data), "file", sbom.Path)
	}
	logger.LogInfo(ctx.Context, "Successfully Uploaded", "Total count", totalSBOMs, "Success", successfullyUploaded, "Failed", totalSBOMs-successfullyUploaded)
	return nil
}

// normalizeJSON removes insignificant whitespace and ensures consistent serialization
func normalizeJSON(data []byte) ([]byte, error) {
	// Parse JSON into a generic interface{}
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	// Re-serialize without indentation
	return json.Marshal(obj)
}

// ComputeFileHash computes the SHA-256 hash of a file
func ComputeContentHash(ctx tcontext.TransferMetadata, content []byte) (string, error) {
	// Check if content is valid JSON
	normalized, err := normalizeJSON(content)
	if err != nil {
		// If not JSON (e.g., XML), hash the raw bytes as fallback
		logger.LogDebug(ctx.Context, "Content is not JSON, hashing raw bytes", "error", err)
		hash := sha256.New()
		_, err := hash.Write(content)
		if err != nil {
			return "", fmt.Errorf("failed to compute hash: %v", err)
		}
		return hex.EncodeToString(hash.Sum(nil)), nil
	}

	hash := sha256.New()
	_, err = hash.Write(normalized)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash: %v", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ParallelUploader uploads SBOMs to Dependency-Track concurrently.
type ParallelUploader struct {
	createdProjects map[string]bool
	mu              sync.Mutex // Protects access to createdProjects.
}

// NewParallelUploader returns a new instance of ParallelUploader.
func NewParallelUploader() *ParallelUploader {
	return &ParallelUploader{
		createdProjects: make(map[string]bool),
	}
}

// Upload implements the SBOMUploader interface for ParallelUploader.
func (u *ParallelUploader) Upload(ctx tcontext.TransferMetadata, config *DependencyTrackConfig, client *DependencyTrackClient, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Initializing SBOMs uploading to Dependency-Track parallely")

	sbomChan := make(chan *iterator.SBOM, 100)
	totalSBOMs := 0
	successfullyUploaded := 0

	// multiple goroutines will read SBOMs from the iterator.
	go func() {
		for {
			sbom, err := iter.Next(ctx)
			if err == io.EOF {
				break
			}
			totalSBOMs++
			if err != nil {
				logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
				continue
			}
			sbomChan <- sbom
		}
		close(sbomChan)
	}()

	const numWorkers = 5 // no. of worker goroutines to process SBOM uploads.
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sbom := range sbomChan {

				// if project name is provided that's well and good, else we need to construct project name from:
				// SBOM primary component name
				// and lasly from file name
				sourceAdapter := ctx.Value("source")
				projectName, projectVersion := utils.ConstructProjectName(ctx, config.ProjectName, config.ProjectVersion, sbom.Namespace, sbom.Version, sbom.Data, sourceAdapter.(string))
				if projectName == "" {
					// THIS CASE OCCURS WHEN SBOM IS NOT IN JSON FORMAT
					// when a JSON SBOM has empty primary comp and version, use the file name as project name

					projectName = filepath.Base(sbom.Path)
					projectName = projectName[:len(projectName)-len(filepath.Ext(projectName))]
					projectVersion = "latest"
				}

				finalProjectName := fmt.Sprintf("%s-%s", projectName, projectVersion)
				logger.LogDebug(ctx.Context, "Project Details", "name", finalProjectName, "version", projectVersion)

				// Ensure the project exists (using a shared cache to avoid duplicate creation).
				u.mu.Lock()
				if !u.createdProjects[finalProjectName] {
					_, err := client.FindOrCreateProject(ctx, finalProjectName, projectVersion)
					if err != nil {
						logger.LogInfo(ctx.Context, "Failed to find or create project", "project", finalProjectName, "error", err)
						u.mu.Unlock()
						continue
					}
					u.createdProjects[finalProjectName] = true
				}
				u.mu.Unlock()

				logger.LogDebug(ctx.Context, "Uploading SBOM file", "file", sbom.Path)

				// Upload the SBOM.
				err := client.UploadSBOM(ctx, finalProjectName, projectVersion, sbom.Data)
				if err != nil {
					logger.LogDebug(ctx.Context, "Failed to upload SBOM", "project", finalProjectName, "file", sbom.Path, "error", err)
					continue
				}
				successfullyUploaded++
				logger.LogDebug(ctx.Context, "Successfully uploaded SBOM file", "file", sbom.Path)
			}
		}()
	}

	// wait for all workers to complete.
	wg.Wait()
	logger.LogInfo(ctx.Context, "Successfully Uploaded", "Total count", totalSBOMs, "Success", successfullyUploaded, "Failed", totalSBOMs-successfullyUploaded)
	return nil
}
