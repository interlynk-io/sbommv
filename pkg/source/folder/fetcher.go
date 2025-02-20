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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/source"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
)

type SBOMFetcher interface {
	Fetch(ctx *tcontext.TransferMetadata, config *FolderConfig) (iterator.SBOMIterator, error)
}

var fetcherFactory = map[types.ProcessingMode]SBOMFetcher{
	types.FetchSequential: &SequentialFetcher{},
	types.FetchParallel:   &ParallelFetcher{},
}

type SequentialFetcher struct{}

// SequentialFetcher Fetch() scans the folder for SBOMs one-by-one
// 1. Walks through the folder file-by-file
// 2. Detects valid SBOMs using source.IsSBOMFile().
// 3. Reads the content & adds it to the iterator along with path.
func (f *SequentialFetcher) Fetch(ctx *tcontext.TransferMetadata, config *FolderConfig) (iterator.SBOMIterator, error) {
	var sbomList []*iterator.SBOM
	err := filepath.Walk(config.FolderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.LogError(ctx.Context, err, "Error accessing file", "path", path)
			return nil
		}
		if info.IsDir() && !config.Recursive && path != config.FolderPath {
			return filepath.SkipDir
		}
		if source.IsSBOMFile(path) {
			content, err := os.ReadFile(path)
			if err != nil {
				logger.LogError(ctx.Context, err, "Failed to read SBOM", "path", path)
				return nil
			}
			projectName, path := getTopLevelDirAndFile(config.FolderPath, path)
			sbomList = append(sbomList, &iterator.SBOM{
				Data:      content,
				Path:      path,
				Namespace: projectName,
			})
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

// ParallelFetcher Fetch() scans the folder for SBOMs using parallel processing
// 1. Walks through the folder file-by-file.
// 2. Launch a goroutine for each file.
// 3. Detects valid SBOMs using source.IsSBOMFile().
// 4. Uses channels to store SBOMs & errors.
// 5. Reads the content & adds it to the iterator along with path.
func (f *ParallelFetcher) Fetch(ctx *tcontext.TransferMetadata, config *FolderConfig) (iterator.SBOMIterator, error) {
	var wg sync.WaitGroup
	sbomsChan := make(chan *iterator.SBOM, 100)
	errChan := make(chan error, 10)

	err := filepath.Walk(config.FolderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errChan <- err
			return nil
		}
		if info.IsDir() && !config.Recursive && path != config.FolderPath {
			return filepath.SkipDir
		}
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			if source.IsSBOMFile(path) {
				content, err := os.ReadFile(path)
				if err != nil {
					errChan <- err
					return
				}
				projectName, path := getTopLevelDirAndFile(config.FolderPath, path)
				sbomsChan <- &iterator.SBOM{
					Data:      content,
					Path:      path,
					Namespace: projectName,
				}
			}
		}(path)
		return nil
	})
	go func() {
		wg.Wait()
		close(sbomsChan)
		close(errChan)
	}()

	var sboms []*iterator.SBOM
	for sbom := range sbomsChan {
		sboms = append(sboms, sbom)
	}
	for err := range errChan {
		logger.LogError(ctx.Context, err, "Error in parallel fetch")
	}
	if err != nil {
		return nil, err
	}
	return iterator.NewMemoryIterator(sboms), nil
}

// getTopLevelDirAndFile extracts the first subdirectory after basePath and the filename.
func getTopLevelDirAndFile(basePath, fullPath string) (string, string) {
	// Get the relative path from basePath to fullPath
	relPath, err := filepath.Rel(basePath, fullPath)
	if err != nil {
		return "unknown", "unknown" // Fallback in case of error
	}

	// Split the relative path into directory components
	parts := strings.Split(relPath, string(filepath.Separator))

	// If there are at least two parts, return the first directory and the filename
	if len(parts) > 1 {
		return parts[0], parts[len(parts)-1] // First directory and last part (filename)
	}

	// If there's no subdirectory, return "unknown" for directory and actual filename
	return "unknown", relPath
}
