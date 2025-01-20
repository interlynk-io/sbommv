package github

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// DownloadSBOM downloads and saves all SBOM files found in the repository
func DownloadSBOM(ctx context.Context, url, outputDir string) error {
	scanner := NewScanner()

	// Find SBOMs in releases
	sboms, err := scanner.FindSBOMs(ctx, url)
	if err != nil {
		return fmt.Errorf("finding SBOMs: %w", err)
	}

	if len(sboms) == 0 {
		return fmt.Errorf("no SBOMs found in repository")
	}

	fmt.Printf("Found %d SBOM(s) in latest release %s\n", len(sboms), sboms[0].Release)

	// Create output directory if specified and doesn't exist
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}

	// Create a worker pool for concurrent downloads
	type downloadWork struct {
		sbom   SBOMAsset
		output string
	}

	numWorkers := 3 // Configure number of concurrent downloads
	workChan := make(chan downloadWork)
	errChan := make(chan error)
	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workChan {
				// Download the SBOM
				reader, err := scanner.client.DownloadAsset(ctx, work.sbom.DownloadURL)
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
					fmt.Printf("Downloaded %s to %s\n", work.sbom.Name, work.output)
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
			return ctx.Err()
		}
	}

	// Close channels and wait
	close(workChan)
	wg.Wait()
	close(errChan)
	errWg.Wait()

	// Check for errors
	if len(errors) > 0 {
		return fmt.Errorf("encountered %d download errors: %v", len(errors), errors[0])
	}

	return nil
}
