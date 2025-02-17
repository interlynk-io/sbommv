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
	"sync"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/source"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

// fetchSBOMsSequentially scans the folder for SBOMs one-by-one
// 1. Walks through the folder file-by-file
// 2. Detects valid SBOMs using source.IsSBOMFile().
// 3. Reads the content & adds it to the iterator along with path.
func (f *FolderAdapter) fetchSBOMsSequentially(ctx *tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Scanning folder sequentially for SBOMs", "path", f.FolderPath, "recursive", f.Recursive)

	var sbomList []*iterator.SBOM

	err := filepath.Walk(f.FolderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.LogError(ctx.Context, err, "Error accessing file", "path", path)
			return nil
		}

		// Skip directories (except the root folder)
		if info.IsDir() && !f.Recursive && path != f.FolderPath {
			return filepath.SkipDir
		}
		fmt.Println("path", path)

		// Check if the file is a valid SBOM
		if source.IsSBOMFile(path) {
			content, err := os.ReadFile(path)
			if err != nil {
				logger.LogError(ctx.Context, err, "Failed to read SBOM", "path", path)
				return nil
			}

			// Extract project name from the top-level directory
			projectName := getTopLevelDir(f.FolderPath, path)

			sbomList = append(sbomList, &iterator.SBOM{
				Data: content,
				// Format:  utils.DetectSBOMFormat(content),
				Path:      path,
				Namespace: projectName,
			})

			logger.LogDebug(ctx.Context, "SBOM Detected", "file", path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error scanning folder: %w", err)
	}

	logger.LogDebug(ctx.Context, "Total SBOMs fetched (Sequential Mode)", "count", len(sbomList))

	if len(sbomList) == 0 {
		return nil, fmt.Errorf("no SBOMs found in the specified folder")
	}

	return NewFolderIterator(sbomList), nil
}

// fetchSBOMsConcurrently scans the folder for SBOMs using parallel processing
// 1. Walks through the folder file-by-file.
// 2. Launch a goroutine for each file.
// 3. Detects valid SBOMs using source.IsSBOMFile().
// 4. Uses channels to store SBOMs & errors.
// 5. Reads the content & adds it to the iterator along with path.
func (f *FolderAdapter) fetchSBOMsConcurrently(ctx *tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Using PARALLEL processing mode")

	var wg sync.WaitGroup
	sbomsChan := make(chan *iterator.SBOM, 100)
	errChan := make(chan error, 10)

	// Walk the folder and process files in parallel
	err := filepath.Walk(f.FolderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errChan <- fmt.Errorf("error accessing file %s: %w", path, err)
			return nil
		}

		fmt.Println("path", path)
		// Skip directories (except root folder)
		if info.IsDir() && !f.Recursive && path != f.FolderPath {
			return filepath.SkipDir
		}

		// Launch a goroutine for each file
		wg.Add(1)

		go func(path string) {
			defer wg.Done()

			if source.IsSBOMFile(path) {
				content, err := os.ReadFile(path)
				if err != nil {
					logger.LogError(ctx.Context, err, "Failed to read SBOM", "path", path)
					errChan <- err
					return
				}

				// Extract project name from the top-level directory
				projectName := getTopLevelDir(f.FolderPath, path)

				sbomsChan <- &iterator.SBOM{
					Data:      content,
					Path:      path,
					Namespace: projectName,
				}

			}
		}(path)
		return nil
	})

	// Close channels after all goroutines complete
	go func() {
		wg.Wait()
		close(sbomsChan)
		close(errChan)
	}()

	// Collect SBOMs from channel
	var sboms []*iterator.SBOM
	for sbom := range sbomsChan {
		sboms = append(sboms, sbom)
	}

	// Check for errors
	for err := range errChan {
		logger.LogError(ctx.Context, err, "Error processing files in parallel mode")
	}

	if err != nil {
		return nil, fmt.Errorf("error scanning folder: %w", err)
	}

	logger.LogDebug(ctx.Context, "Total SBOMs fetched (Parallel Mode)", "count", len(sboms))
	return iterator.NewMemoryIterator(sboms), nil
}

// getTopLevelDir extracts the top-level directory from a given path
func getTopLevelDir(basePath, fullPath string) string {
	relPath, err := filepath.Rel(basePath, fullPath)
	if err != nil {
		return "unknown" // Fallback in case of an error
	}

	// Split the relative path and return the first directory
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) > 1 {
		return parts[0] // Return the top-level folder (e.g., "cdx" or "spdx")
	}

	return "unknown"
}
