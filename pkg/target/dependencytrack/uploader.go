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
	logger.LogDebug(ctx.Context, "Uploading SBOMs to Dependency-Track sequentially")

	totalSBOMs := 0
	successfullyUploaded := 0
	for {
		sbom, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		totalSBOMs++
		if err != nil {
			return err
		}

		projectName, err := getProjectName(ctx, config.ProjectName, sbom.Namespace)
		if err != nil {
			continue
		}

		projectVersion := getProjectVersion(ctx, config.ProjectVersion, sbom.Version)
		finalProjectName := fmt.Sprintf("%s-%s", projectName, projectVersion)
		logger.LogDebug(ctx.Context, "Project Details", "name", finalProjectName, "version", projectVersion)

		if !u.createdProjects[finalProjectName] {

			// find or create project using project name and project version
			_, err = client.FindOrCreateProject(ctx, finalProjectName, projectVersion)
			if err != nil {
				logger.LogInfo(ctx.Context, "Failed to find or create project", "project", projectName, "error", err)
				// u.mu.Unlock()
				continue
			}
			u.createdProjects[finalProjectName] = true
		}

		logger.LogDebug(ctx.Context, "Initializing uploading SBOM file", "file", sbom.Path)

		err = client.UploadSBOM(ctx, finalProjectName, projectVersion, sbom.Data)
		if err != nil {
			logger.LogDebug(ctx.Context, "Failed to upload SBOM", "project", finalProjectName, "file", sbom.Path, "error", err)
			continue
		}
		successfullyUploaded++
		logger.LogDebug(ctx.Context, "Successfully uploaded SBOM file", "file", sbom.Path)
	}
	logger.LogInfo(ctx.Context, "Successfully Uploaded", "count", successfullyUploaded)
	return nil
}

// // ParallelUploader uploads SBOMs to Dependency-Track concurrently.
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
	logger.LogDebug(ctx.Context, "Uploading SBOMs to Dependency-Track in parallel mode")

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

				projectName, err := getProjectName(ctx, config.ProjectName, sbom.Namespace)
				if err != nil {
					continue
				}

				projectVersion := getProjectVersion(ctx, config.ProjectVersion, sbom.Version)
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
				err = client.UploadSBOM(ctx, finalProjectName, projectVersion, sbom.Data)
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
	logger.LogInfo(ctx.Context, "Successfully Uploaded", "count", successfullyUploaded)
	return nil
}

func getProjectName(ctx tcontext.TransferMetadata, providedProjectName string, namespace string) (string, error) {
	if providedProjectName == "" && namespace == "" {
		return "", fmt.Errorf("no project name specified and SBOM namespace is empty")
	}

	var projectName string
	if providedProjectName != "" {
		projectName = providedProjectName
		logger.LogDebug(ctx.Context, "Project Name is provided by the user", "name", projectName)
	} else {
		projectName = namespace
		logger.LogDebug(ctx.Context, "Project Name as sbom.Namespace will be used", "sbom.Namespace", namespace)
	}

	return projectName, nil
}

func getProjectVersion(ctx tcontext.TransferMetadata, providedProjectVersion string, version string) string {
	var projectVersion string
	if providedProjectVersion == "" && version == "" {
		projectVersion = "latest"
	}

	if providedProjectVersion != "" {
		projectVersion = providedProjectVersion
		logger.LogDebug(ctx.Context, "Project Version is provided by the user", "version", projectVersion)
	} else {
		projectVersion = version
		logger.LogDebug(ctx.Context, "Project Version as sbom.Version will be used", "sbom.Version", projectVersion)
	}

	return projectVersion
}
