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

package folder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/source"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type SBOMFetcher interface {
	Fetch(ctx tcontext.TransferMetadata, config *FolderConfig) (iterator.SBOMIterator, error)
}

type SequentialFetcher struct{}

// SequentialFetcher Fetch() scans the folder for SBOMs one-by-one
// 1. Walks through the folder file-by-file
// 2. Detects valid SBOMs using source.IsSBOMFile().
// 3. Reads the content & adds it to the iterator along with path.
func (f *SequentialFetcher) Fetch(ctx tcontext.TransferMetadata, config *FolderConfig) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Fetching SBOMs Sequentially")

	var sbomList []*iterator.SBOM
	err := filepath.Walk(config.FolderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.LogError(ctx.Context, err, "Error accessing file", "path", path)
			return nil
		}
		if info.IsDir() && !config.Recursive && path != config.FolderPath {
			return filepath.SkipDir
		}

		content, err := os.ReadFile(path)
		if err != nil {
			logger.LogError(ctx.Context, err, "Failed to read SBOM", "path", path)
			return nil
		}

		if source.IsSBOMFile(content) {

			projectName, projectVersion := getProjectNameAndVersion(ctx, path, content)
			logger.LogDebug(ctx.Context, "Project Details", "name", projectName, "version", projectVersion)

			fileName := getFilePath(config.FolderPath, path)
			sbomList = append(sbomList, &iterator.SBOM{
				Data:      content,
				Path:      fileName,
				Namespace: projectName,
				Version:   projectVersion,
			})
		} else {
			logger.LogDebug(ctx.Context, "Skipping non-SBOM file", "path", getFilePath(config.FolderPath, path))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(sbomList) == 0 {
		return nil, fmt.Errorf("no SBOMs found in folder")
	}
	return NewFolderIterator(sbomList), nil
}

type ParallelFetcher struct{}

// Fetch scans the folder for SBOMs concurrently.
// It walks through the directory to collect file paths, then spawns a fixed number of worker goroutines
// to read and process those files concurrently.
func (f *ParallelFetcher) Fetch(ctx tcontext.TransferMetadata, config *FolderConfig) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Fetching SBOMs Parallely")
	filePaths := make(chan string, 100)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var sbomList []*iterator.SBOM

	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range filePaths {

				// skip directories.
				info, err := os.Stat(path)
				if err != nil {
					logger.LogError(ctx.Context, err, "Failed to stat file", "path", path)
					continue
				}
				if info.IsDir() {
					continue
				}

				content, err := os.ReadFile(path)
				if err != nil {
					logger.LogError(ctx.Context, err, "Failed to read SBOM", "path", path)
					continue
				}

				if !source.IsSBOMFile(content) {
					continue
				}

				projectName, projectVersion := getProjectNameAndVersion(ctx, path, content)
				logger.LogDebug(ctx.Context, "Project Details", "name", projectName, "version", projectVersion)

				//  get a relative file path.
				fileName := getFilePath(config.FolderPath, path)

				mu.Lock()
				sbomList = append(sbomList, &iterator.SBOM{
					Data:      content,
					Path:      fileName,
					Namespace: projectName,
					Version:   projectVersion,
				})
				mu.Unlock()
			}
		}()
	}

	// walk the folder and send each file path into the channel.
	err := filepath.Walk(config.FolderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.LogError(ctx.Context, err, "Error accessing file", "path", path)
			return nil
		}

		// if not recursive and the current path is a subdirectory, skip it.
		if info.IsDir() && !config.Recursive && path != config.FolderPath {
			return filepath.SkipDir
		}

		filePaths <- path
		return nil
	})
	close(filePaths)
	wg.Wait()

	if err != nil {
		return nil, err
	}
	if len(sbomList) == 0 {
		return nil, fmt.Errorf("no SBOMs found in folder")
	}
	return NewFolderIterator(sbomList), nil
}

// getFilePath returns file path
func getFilePath(basePath, fullPath string) string {
	relPath, err := filepath.Rel(basePath, fullPath)
	if err != nil {
		logger.LogDebug(context.Background(), "Path resolution failed", "base", basePath, "full", fullPath, "error", err)
		return filepath.Base(fullPath)
	}

	// Split and grab the last part—always the filename
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) > 0 {
		logger.LogDebug(context.Background(), "Path structure", "path", parts[len(parts)-1])
		return parts[len(parts)-1]
	}

	logger.LogDebug(context.Background(), "Unexpected path structure", "base", basePath, "full", fullPath)
	return filepath.Base(fullPath)
}
