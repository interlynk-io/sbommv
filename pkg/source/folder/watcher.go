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

	"github.com/fsnotify/fsnotify"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/source"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type WatcherFetcher struct{}

func NewWatcherFetcher() *WatcherFetcher {
	return &WatcherFetcher{}
}

func (f *WatcherFetcher) Fetch(ctx tcontext.TransferMetadata, config *FolderConfig) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Starting folder watcher", "path", config.FolderPath, "recurssive", config.ProcessingMode)

	processor := sbom.NewSBOMProcessor("", false)

	// Create new watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	sbomChan := make(chan *iterator.SBOM, 10)

	// add to watch more sub-directories if recurssive is true
	err = filepath.Walk(config.FolderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.LogError(ctx.Context, err, "Error accessing path", "path", path)
			return nil
		}
		if info.IsDir() {
			if !config.Recursive && path != config.FolderPath {
				return filepath.SkipDir
			}

			// add it to the watcher
			if err := watcher.Add(path); err != nil {
				logger.LogError(ctx.Context, err, "Failed to watch directory", "path", path)
			} else {
				logger.LogDebug(ctx.Context, "Watching directory", "path", path)
			}

			return nil
		}
		return nil
	})
	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Start listening for events.
	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					close(sbomChan)
					return
				}

				logger.LogDebug(ctx.Context, "Event Triggered", "name", event)

				// handle removal or renaming events explicitly.
				if event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove) {
					// just log and continue.
					logger.LogDebug(ctx.Context, "Resource removed from watched folder", "path", event.Name)
					continue
				}

				info, err := os.Stat(event.Name)
				if err != nil {
					logger.LogDebug(ctx.Context, "err", "Failed to stat path", "path", event.Name)
					continue
				}
				// || event.Has(fsnotify.Create)
				if event.Has(fsnotify.Write) {

					var allFiles []string
					if info.IsDir() {
						logger.LogDebug(ctx.Context, "New directory created", "path", event.Name)

						// if recurssive is true, add subdirectory to the watcher created during real-time monitoring.
						if config.Recursive {
							if err := watcher.Add(event.Name); err != nil {
								logger.LogError(ctx.Context, err, "Failed to watch new directory", "path", event.Name)
							} else {
								logger.LogInfo(ctx.Context, "monitoring", "path", event.Name)
							}
						}

						dirEntries, err := os.ReadDir(event.Name)
						if err != nil {
							logger.LogDebug(ctx.Context, "err", "Failed to read directory", "path", event.Name)
							continue
						}

						if len(dirEntries) == 0 {
							logger.LogDebug(ctx.Context, "No files in new directory, skipping", "path", event.Name)
							continue
						}

						for _, entry := range dirEntries {
							if !entry.IsDir() {
								logger.LogDebug(ctx.Context, "Found file in new directory", "path", entry.Name())
								allFiles = append(allFiles, filepath.Join(event.Name, entry.Name()))
							}
						}
					} else {
						// add file directly to allFiles.
						allFiles = append(allFiles, event.Name)
					}

					for _, filePath := range allFiles {
						content, err := os.ReadFile(filePath)
						if err != nil {
							logger.LogDebug(ctx.Context, "err", "Failed to read SBOM", "path", filePath)
							continue
						}

						if source.IsSBOMFile(content) {
							logger.LogDebug(ctx.Context, "Locally SBOM located folder", "path", config.FolderPath)

							fileName := getFilePath(config.FolderPath, filePath)
							processor.Update(content, "", fileName)

							sbomChan <- &iterator.SBOM{
								Data:      content,
								Path:      fileName,
								Namespace: config.FolderPath,
							}

						} else {
							logger.LogInfo(ctx.Context, "not-found", "path", filePath)
						}

					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					close(sbomChan)
					return
				}
				logger.LogError(ctx.Context, err, "Watcher error")

			case <-ctx.Done():
				close(sbomChan)
				return
			}
		}
	}()

	return &WatcherIterator{sbomChan: sbomChan}, nil
}
