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

package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	adapter "github.com/interlynk-io/sbommv/pkg/adapter"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/mvtypes"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/spf13/cobra"
)

func TransferRun(ctx context.Context, cmd *cobra.Command, config mvtypes.Config) error {
	logger.LogInfo(ctx, "Starting SBOM transfer process")

	// ✅ Initialize shared context with metadata support
	transferCtx := tcontext.NewTransferMetadata(ctx)

	var inputAdapterInstance adapter.Adapter
	var outputAdapterInstance adapter.Adapter
	var err error

	// Initialize source adapter
	inputAdapterInstance, err = adapter.NewAdapter(transferCtx, config.SourceType, types.AdapterRole("input"))
	if err != nil {
		logger.LogError(transferCtx.Context, err, "Failed to initialize source adapter")
		return fmt.Errorf("failed to get source adapter: %w", err)
	}

	// Initialize destination adapter
	outputAdapterInstance, err = adapter.NewAdapter(transferCtx, config.DestinationType, types.AdapterRole("output"))
	if err != nil {
		logger.LogError(transferCtx.Context, err, "Failed to initialize destination adapter")
		return fmt.Errorf("failed to get a destination adapter %v", err)
	}

	// Parse and validate input adapter parameters
	if err := inputAdapterInstance.ParseAndValidateParams(cmd); err != nil {
		logger.LogError(transferCtx.Context, err, "Input adapter error")
		return fmt.Errorf("input adapter error: %w", err)

	}
	logger.LogDebug(transferCtx.Context, "input adapter instance config", "value", inputAdapterInstance)

	// Parse and validate output adapter parameters
	if err := outputAdapterInstance.ParseAndValidateParams(cmd); err != nil {
		return fmt.Errorf("output adapter error: %w", err)
	}
	logger.LogDebug(transferCtx.Context, "output adapter instance config", "value", outputAdapterInstance)

	// Fetch SBOMs lazily using the iterator
	logger.LogInfo(transferCtx.Context, "Fetching SBOMs from input adapter...")

	sbomIterator, err := inputAdapterInstance.FetchSBOMs(transferCtx)
	if err != nil {
		logger.LogError(transferCtx.Context, err, "Failed to fetch SBOMs")
		return fmt.Errorf("failed to fetch SBOMs: %w", err)
	}

	logger.LogDebug(transferCtx.Context, "SBOM fetching started successfully", "sbomIterator", sbomIterator)

	// Dry-Run Mode: Display SBOMs Without Uploading
	if config.DryRun {
		logger.LogInfo(transferCtx.Context, "Dry-run mode enabled: Displaying retrieved SBOMs", "values", config.DryRun)

		if err := dryMode(transferCtx.Context, sbomIterator); err != nil {
			return fmt.Errorf("failed to execute dry-run mode: %v", err)
		}
		return nil
	}

	// Process & Upload SBOMs Sequentially
	if err := outputAdapterInstance.UploadSBOMs(transferCtx, sbomIterator); err != nil {
		logger.LogError(transferCtx.Context, err, "Failed to output SBOMs")
		return fmt.Errorf("failed to output SBOMs: %w", err)
	}

	logger.LogDebug(ctx, "SBOM transfer process completed successfully ✅")
	return nil
}

func dryMode(ctx context.Context, iterator iterator.SBOMIterator) error {
	logger.LogDebug(ctx, "Dry-run mode enabled. Preparing to print SBOM details.")

	processor := sbom.NewSBOMProcessor("", false)
	sbomCount := 0

	fmt.Println("=========================================")
	fmt.Println(" List of SBOM files fetched by Input Adapter (Grouped by Version)")
	fmt.Println("=========================================")

	for {
		sbom, err := iterator.Next(ctx)
		if err == io.EOF {
			break // no more sboms
		}

		if err != nil {
			logger.LogError(ctx, err, "Error retrieving SBOM from iterator")
			continue
		}

		logger.LogDebug(ctx, "Processing SBOM file", "path", sbom.Path)

		doc, err := processor.ProcessSBOM(sbom.Path)
		if err != nil {
			logger.LogError(ctx, err, "Failed to process SBOM", "path", sbom.Path)
			continue
		}

		sbomCount++
		fmt.Printf("%d. File: %s | Format: %s | SpecVersion: %s\n", sbomCount, doc.Filename, doc.Format, doc.SpecVersion)

	}

	logger.LogDebug(ctx, "Dry-run mode completed", "total_sboms_processed", sbomCount)
	return nil
}

// ValidateInterlynkConnection chesks whether Interlynk ssytem is up and running
func ValidateInterlynkConnection(ctx context.Context, url, token string) error {
	baseURL, err := extractBaseURL(url)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	logger.LogDebug(ctx, "Validating Interlynk running status for", "url", baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return fmt.Errorf("falied to create request for Interlynk: %w", err)
	}

	// INTERLYNK_SECURITY_TOKEN is required here
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Interlynk at %s: %w", baseURL, err)
	}
	defer resp.Body.Close()

	// provided token is invalid
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid API token: authentication failed")
	}

	// interlynk looks to down
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Interlynk API returned unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func extractBaseURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// construct base URL (protocol + host)
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// ensure it always ends with a single "/"
	return strings.TrimRight(baseURL, "/") + "/", nil
}
