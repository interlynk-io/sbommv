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

// GetSBOMs downloads and saves all SBOM files found in the repository
func (c *Client) GetSBOMs(ctx context.Context, outputDir string) (VersionedSBOMs, error) {
	// Find SBOMs in releases
	sboms, err := c.FindSBOMs(ctx)
	if err != nil {
		return nil, fmt.Errorf("finding SBOMs: %w", err)
	}
	if len(sboms) == 0 {
		return nil, fmt.Errorf("no SBOMs found in repository")
	}

	logger.LogDebug(ctx, "Total SBOMs found in the repository", "version", c.Version, "total sboms", len(sboms))

	// Create output directory if needed
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return nil, fmt.Errorf("creating output directory: %w", err)
		}
	}

	return c.downloadSBOMs(ctx, sboms, outputDir)
}

// downloadSBOMs handles the concurrent downloading of multiple SBOM files
func (c *Client) downloadSBOMs(ctx context.Context, sboms []SBOMAsset, outputDir string) (VersionedSBOMs, error) {
	var (
		wg             sync.WaitGroup                        // Coordinates all goroutines
		mu             sync.Mutex                            // Protects shared resources
		versionedSBOMs = make(VersionedSBOMs)                // Stores results
		errors         []error                               // Collects errors
		maxConcurrency = 3                                   // Maximum parallel downloads
		semaphore      = make(chan struct{}, maxConcurrency) // Controls concurrency
	)

	// Process each SBOM
	for _, sbom := range sboms {
		// Context cancellation check
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		wg.Add(1)
		go func(sbom SBOMAsset) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			// Download and save the SBOM
			outputPath := ""
			if outputDir != "" {
				outputPath = filepath.Join(outputDir, sbom.Name)
			}

			err := c.downloadSingleSBOM(ctx, sbom, outputPath)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("downloading %s: %w", sbom.Name, err))
				mu.Unlock()
				return
			}

			if outputPath != "" {
				mu.Lock()
				versionedSBOMs[sbom.Release] = append(versionedSBOMs[sbom.Release], outputPath)
				mu.Unlock()
				logger.LogDebug(ctx, "SBOM file", "name", sbom.Name, "saved to", outputPath)
			}
		}(sbom)
	}

	wg.Wait()

	if len(errors) > 0 {
		return nil, fmt.Errorf("encountered %d download errors: %v", len(errors), errors[0])
	}

	return versionedSBOMs, nil
}

// downloadSingleSBOM downloads and saves a single SBOM file
func (c *Client) downloadSingleSBOM(ctx context.Context, sbom SBOMAsset, outputPath string) error {
	reader, err := c.DownloadAsset(ctx, sbom.DownloadURL)
	if err != nil {
		return fmt.Errorf("downloading asset: %w", err)
	}
	defer reader.Close()

	var output io.Writer
	if outputPath == "" {
		// Write to stdout with header
		fmt.Printf("\n=== SBOM: %s ===\n", sbom.Name)
		output = os.Stdout
	} else {
		// Create and write to file
		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer file.Close()
		output = file
	}

	if _, err := io.Copy(output, reader); err != nil {
		return fmt.Errorf("writing SBOM: %w", err)
	}

	return nil
}
