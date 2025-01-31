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
	"os"

	adapter "github.com/interlynk-io/sbommv/pkg/adapters"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/mvtypes"
	"github.com/interlynk-io/sbommv/pkg/sbom"
)

func TransferRun(ctx context.Context, config mvtypes.Config) error {
	logger.LogInfo(ctx, "Starting SBOM transfer process")

	sourceAdapter, err := adapter.NewSourceAdapter(ctx, config)
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
	os.Exit(0)

	if config.DryRun {
		logger.LogInfo(ctx, "Dry-run mode enabled: Displaying SBOMs which are retrieved", "values", config.DryRun)
		err := dryMode(ctx, allSBOMs)
		if err != nil {
			logger.LogError(ctx, err, "Dry-run mode failed")
			return fmt.Errorf("failed to execute dry-run mode: %v", err)
		}
		return nil
	}

	destAdapter, err := adapter.NewDestAdapter(ctx, config)
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
