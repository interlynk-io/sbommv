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

package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FolderAdapter implements InputAdapter specifically for directory sources
// with configurable concurrency and recursive options
type FolderAdapter struct {
	root      string
	options   InputOptions
	recursive bool
}

// NewFolderAdapter creates a new folder-based adapter with concurrent processing
func NewFolderAdapter(config AdapterConfig) (*FolderAdapter, error) {
	info, err := os.Stat(config.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat folder: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %s is not a directory", config.Path)
	}

	// Set reasonable default for concurrent operations
	if config.InputOptions.MaxConcurrent <= 0 {
		config.InputOptions.MaxConcurrent = 5
	}

	return &FolderAdapter{
		root:      config.Path,
		options:   config.InputOptions,
		recursive: config.Recursive,
	}, nil
}

// FolderAdapter implements GetSBOMs. Therefore  FolderAdapter implements for InputAdapter
func (a *FolderAdapter) GetSBOMs(ctx context.Context) ([]string, error) {
	// Channel for sending file paths
	found := make(chan string)

	// Channel for errors during walking
	walkErrs := make(chan error, 1)

	// Final results slice
	var paths []string

	// Mutex for thread-safe appending to paths
	var mu sync.Mutex

	// Start file discovery in a separate goroutine
	go func() {
		defer close(found)
		defer close(walkErrs)

		walkFn := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if info.IsDir() {
					if !a.recursive && path != a.root {
						return filepath.SkipDir
					}
					return nil
				}

				if isSBOMFile(info.Name()) {
					select {
					case found <- path:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				return nil
			}
		}

		if err := filepath.Walk(a.root, walkFn); err != nil {
			walkErrs <- err
		}
	}()

	// Collect results
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case err := <-walkErrs:
			if err != nil {
				return nil, fmt.Errorf("error walking directory: %w", err)
			}

		case path, ok := <-found:
			if !ok {
				// Channel closed, all paths collected
				return paths, nil
			}
			mu.Lock()
			paths = append(paths, path)
			mu.Unlock()
		}
	}
}
