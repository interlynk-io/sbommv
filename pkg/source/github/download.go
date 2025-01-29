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

package github

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/interlynk-io/sbommv/pkg/logger"
)

// Create a worker pool for concurrent downloads
type downloadWork struct {
	sbom   SBOMAsset
	output string
}

// DownloadSBOM downloads and saves all SBOM files found in the repository
func (c *Client) GetSBOMs(ctx context.Context, url, version, outputDir string) (map[string][]string, error) {
	// Find SBOMs in releases
	sboms, err := c.FindSBOMs(ctx, url, version)
	if err != nil {
		return nil, fmt.Errorf("finding SBOMs: %w", err)
	}

	if len(sboms) == 0 {
		return nil, fmt.Errorf("no SBOMs found in repository")
	}

	logger.LogDebug(ctx, "Total SBOMs found in the repository", "version", version, "total sboms", len(sboms))

	// Create output directory if specified and doesn't exist
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return nil, fmt.Errorf("creating output directory: %w", err)
		}
	}

	numWorkers := 3 // Configure number of concurrent downloads
	workChan := make(chan downloadWork)
	errChan := make(chan error)
	var wg sync.WaitGroup

	// Versioned SBOMs
	versionedSBOMs := make(VersionedSBOMs)

	// Start worker pool
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workChan {
				// Download the SBOM
				reader, err := c.DownloadAsset(ctx, work.sbom.DownloadURL)
				if err != nil {
					errChan <- fmt.Errorf("downloading SBOM %s: %w", work.sbom.Name, err)
					continue
				}

				// Handle output based on whether we're writing to file or stdout
				var output io.Writer
				var file *os.File

				if work.output == "" {
					// Write to stdout with header
					fmt.Printf("\n=== SBOM: %s ===\n", work.sbom.Name)
					output = os.Stdout
				} else {
					// Create output file
					file, err = os.Create(work.output)
					if err != nil {
						reader.Close()
						errChan <- fmt.Errorf("creating output file %s: %w", work.sbom.Name, err)
						continue
					}
					output = file
					defer file.Close()
				}

				// Copy content
				if _, err := io.Copy(output, reader); err != nil {
					reader.Close()
					errChan <- fmt.Errorf("writing SBOM %s: %w", work.sbom.Name, err)
					continue
				}
				reader.Close()

				if work.output != "" {
					logger.LogDebug(ctx, "SBOM file", "name", work.sbom.Name, "saved to ", work.output)
					// Group SBOMs by release version
					versionedSBOMs[work.sbom.Release] = append(versionedSBOMs[work.sbom.Release], work.output)
				}
			}
		}()
	}

	// Error collector goroutine
	var errors []error
	var errWg sync.WaitGroup
	errWg.Add(1)
	go func() {
		defer errWg.Done()
		for err := range errChan {
			errors = append(errors, err)
		}
	}()

	// Submit work
	for _, sbom := range sboms {
		var outputPath string
		if outputDir != "" {
			outputPath = filepath.Join(outputDir, sbom.Name)
		}
		select {
		case workChan <- downloadWork{sbom: sbom, output: outputPath}:
		case <-ctx.Done():
			close(workChan)
			return nil, ctx.Err()
		}
	}

	// Close channels and wait
	close(workChan)
	wg.Wait()
	close(errChan)
	errWg.Wait()

	// Check for errors
	if len(errors) > 0 {
		return nil, fmt.Errorf("encountered %d download errors: %v", len(errors), errors[0])
	}

	return versionedSBOMs, nil
}

// // DownloadSBOM downloads and saves all SBOM files found in the repository
// func (c *Client) GetSBOMs(ctx context.Context, url, version, outputDir string) (map[string][]string, error) {
// 	// Find SBOMs in releases
// 	sboms, err := c.FindSBOMs(ctx, url, version)
// 	if err != nil {
// 		return nil, fmt.Errorf("finding SBOMs: %w", err)
// 	}

// 	if len(sboms) == 0 {
// 		return nil, fmt.Errorf("no SBOMs found in repository")
// 	}

// 	logger.LogDebug(ctx, "Total SBOMs found in the repository", "version", version, "total sboms", len(sboms))

// 	// Create output directory if it doesn't exist
// 	if err := ensureOutputDir(outputDir); err != nil {
// 		return nil, err
// 	}

// 	// Process SBOMs concurrently
// 	return c.processSBOMDownloadConcurrently(ctx, sboms, outputDir)
// }

// processSBOMsConcurrently handles concurrent SBOM downloads.
func (c *Client) processSBOMDownloadConcurrently(ctx context.Context, sboms []SBOMAsset, outputDir string) (VersionedSBOMs, error) {
	const numWorkers = 3
	workChan := make(chan downloadWork)
	errChan := make(chan error)
	var wg sync.WaitGroup

	// Versioned SBOMs storage
	versionedSBOMs := make(VersionedSBOMs)

	// Start worker pool
	c.startWorkerPool(ctx, &wg, workChan, errChan, versionedSBOMs)

	// Error collector goroutine
	errors := collectErrors(errChan)

	// Submit work
	if err := submitDownloadTasks(ctx, sboms, outputDir, workChan); err != nil {
		close(workChan)
		return nil, err
	}

	// Close channels and wait for workers
	close(workChan)
	wg.Wait()
	close(errChan)

	// Handle errors after all workers are done
	if len(errors) > 0 {
		return nil, fmt.Errorf("encountered %d download errors: %v", len(errors), errors[0])
	}

	return versionedSBOMs, nil
}

// ensureOutputDir creates the output directory if it does not exist.
func ensureOutputDir(outputDir string) error {
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}
	return nil
}

// startWorkerPool initializes worker goroutines for concurrent SBOM downloads.
func (c *Client) startWorkerPool(ctx context.Context, wg *sync.WaitGroup, workChan chan downloadWork, errChan chan error, versionedSBOMs VersionedSBOMs) {
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workChan {
				if err := c.downloadSBOM(ctx, work, versionedSBOMs); err != nil {
					errChan <- err
				}
			}
		}()
	}
}

// downloadSBOM handles downloading and saving a single SBOM file.
func (c *Client) downloadSBOM(ctx context.Context, work downloadWork, versionedSBOMs VersionedSBOMs) error {
	reader, err := c.DownloadAsset(ctx, work.sbom.DownloadURL)
	if err != nil {
		return fmt.Errorf("downloading SBOM %s: %w", work.sbom.Name, err)
	}
	defer reader.Close()

	var output io.Writer
	var file *os.File

	// Write to stdout or file
	if work.output == "" {
		fmt.Printf("\n=== SBOM: %s ===\n", work.sbom.Name)
		output = os.Stdout
	} else {
		file, err = os.Create(work.output)
		if err != nil {
			return fmt.Errorf("creating output file %s: %w", work.sbom.Name, err)
		}
		defer file.Close()
		output = file
	}

	// Copy content
	if _, err := io.Copy(output, reader); err != nil {
		return fmt.Errorf("writing SBOM %s: %w", work.sbom.Name, err)
	}

	// If file output, log and update versionedSBOMs
	if work.output != "" {
		logger.LogDebug(ctx, "SBOM file saved", "name", work.sbom.Name, "path", work.output)
		versionedSBOMs[work.sbom.Release] = append(versionedSBOMs[work.sbom.Release], work.output)
	}

	return nil
}

// collectErrors aggregates errors from the error channel.
func collectErrors(errChan chan error) []error {
	var errors []error
	var errWg sync.WaitGroup
	errWg.Add(1)

	go func() {
		defer errWg.Done()
		for err := range errChan {
			errors = append(errors, err)
		}
	}()

	errWg.Wait()
	return errors
}

// submitDownloadTasks sends SBOM download tasks to the worker pool.
func submitDownloadTasks(ctx context.Context, sboms []SBOMAsset, outputDir string, workChan chan downloadWork) error {
	for _, sbom := range sboms {
		var outputPath string
		if outputDir != "" {
			outputPath = filepath.Join(outputDir, sbom.Name)
		}
		select {
		case workChan <- downloadWork{sbom: sbom, output: outputPath}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
