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
	"net/http"
	"net/url"
	"os"
	"strings"

	adapter "github.com/interlynk-io/sbommv/pkg/adapter"
	adapters "github.com/interlynk-io/sbommv/pkg/adapters"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/mvtypes"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/spf13/cobra"
)

func TransferRun(ctx context.Context, cmd *cobra.Command, config mvtypes.Config) error {
	logger.LogInfo(ctx, "Starting SBOM transfer process")

	var inputAdapterInstance adapter.Adapter
	var outputAdapterInstance adapter.Adapter

	inputAdapterInstance, err := adapter.NewAdapter(ctx, config.SourceType)
	if err != nil {
		logger.LogError(ctx, err, "Failed to initialize source adapter")
		return fmt.Errorf("failed to get source adapter: %w", err)
	}

	outputAdapterInstance, err = adapter.NewAdapter(ctx, config.DestinationType)
	if err != nil {
		logger.LogError(ctx, err, "Failed to initialize destination adapter")
		return fmt.Errorf("failed to get a destination adapter %v", err)
	}

	// Parse & Validate Parameters
	if err := inputAdapterInstance.ParseAndValidateParams(cmd); err != nil {
		logger.LogError(ctx, err, "Input adapter error")
	}

	logger.LogDebug(ctx, "input adapter instance config", "value", inputAdapterInstance)

	if err := outputAdapterInstance.ParseAndValidateParams(cmd); err != nil {
		logger.LogError(ctx, err, "Output adapter error")
	}

	logger.LogDebug(ctx, "output adapter instance config", "value", outputAdapterInstance)

	os.Exit(0)
	///-----------------------------------------------------------

	sourceAdapter, err := adapters.NewSourceAdapter(ctx, config)
	if err != nil {
		logger.LogError(ctx, err, "Failed to initialize source adapter")
		return fmt.Errorf("failed to get source adapter: %w", err)
	}

	logger.LogDebug(ctx, "Fetching SBOMs from source using source adapters")
	allSBOMs, err := sourceAdapter.GetSBOMs(ctx)
	if err != nil {
		logger.LogError(ctx, err, "Failed to retrieve SBOMs")
		return fmt.Errorf("failed to get SBOMs: %w", err)
	}
	logger.LogInfo(ctx, "Successfully fetched all SBOMs")

	if config.DryRun {
		logger.LogInfo(ctx, "Dry-run mode enabled: Displaying SBOMs which are retrieved", "values", config.DryRun)
		err := dryMode(ctx, allSBOMs)
		if err != nil {
			logger.LogError(ctx, err, "Dry-run mode failed")
			return fmt.Errorf("failed to execute dry-run mode: %v", err)
		}
		return nil
	}

	destAdapter, err := adapters.NewDestAdapter(ctx, config)
	if err != nil {
		logger.LogError(ctx, err, "Failed to initialize destination adapter")
		return fmt.Errorf("failed to get a destination adapter %v", err)
	}

	logger.LogDebug(ctx, "Uploading SBOMs to destination")
	err = destAdapter.UploadSBOMs(ctx, allSBOMs)
	if err != nil {
		logger.LogError(ctx, err, "Failed to upload SBOMs")
		return fmt.Errorf("failed to upload SBOMs %v", err)
	}
	logger.LogInfo(ctx, "Successfully uploaded all SBOMs")
	return nil
}

// // registerAdapterFlags ensures flags are added only once per adapter type.
// func registerAdapterFlags(cmd *cobra.Command, adapters ...adapter.Adapter) {
// 	seenFlags := make(map[string]bool) // Track already registered flags

// 	for _, adapter := range adapters {
// 		adapter.AddCommandParams(cmd, seenFlags)
// 	}
// }

func dryMode(ctx context.Context, allSBOMs map[string][]string) error {
	logger.LogDebug(ctx, "Dry-run mode enabled. Preparing to print SBOM details.", "total_versions", len(allSBOMs))

	processor := sbom.NewSBOMProcessor("", false)
	sbomCount := 0

	fmt.Println("=========================================")
	fmt.Println(" List of SBOM files fetched by Input Adapter (Grouped by Version)")
	fmt.Println("=========================================")

	for version, sboms := range allSBOMs {
		fmt.Printf("\n--- Version: %s ---\n", version)

		for _, sbomPath := range sboms {
			logger.LogDebug(ctx, "Processing SBOM file", "path", sbomPath)

			doc, err := processor.ProcessSBOM(sbomPath)
			if err != nil {
				logger.LogError(ctx, err, "Failed to process SBOM", "path", sbomPath)
				continue
			}

			sbomCount++
			fmt.Printf("%d. File: %s | Format: %s | SpecVersion: %s\n", sbomCount, doc.Filename, doc.Format, doc.SpecVersion)

			// Uncomment if you want to pretty print the SBOM content
			// if err := sbom.PrettyPrintSBOM(os.Stdout, doc.Content); err != nil {
			//     logger.LogError(ctx, err, "Failed to pretty-print SBOM content", "file", doc.Filename)
			// }
		}
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
