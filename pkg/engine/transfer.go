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
	logger.LogDebug(ctx, "Starting SBOM transfer process")

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
		return fmt.Errorf("input adapter error: %w", err)
	}

	logger.LogDebug(transferCtx.Context, "input adapter instance config", "value", inputAdapterInstance)

	// Parse and validate output adapter parameters
	if err := outputAdapterInstance.ParseAndValidateParams(cmd); err != nil {
		return fmt.Errorf("output adapter error: %w", err)
	}

	logger.LogDebug(transferCtx.Context, "output adapter instance config", "value", outputAdapterInstance)

	// Fetch SBOMs lazily using the iterator
	sbomIterator, err := inputAdapterInstance.FetchSBOMs(transferCtx)
	if err != nil {
		return fmt.Errorf("failed to fetch SBOMs: %w", err)
	}

	// Dry-Run Mode: Display SBOMs Without Uploading
	if config.DryRun {
		logger.LogDebug(transferCtx.Context, "Dry-run mode enabled: Displaying retrieved SBOMs", "values", config.DryRun)

		if err := dryMode(transferCtx.Context, sbomIterator, ""); err != nil {
			return fmt.Errorf("failed to execute dry-run mode: %v", err)
		}
		return nil
	}

	// Process & Upload SBOMs Sequentially
	if err := outputAdapterInstance.UploadSBOMs(transferCtx, sbomIterator); err != nil {
		return fmt.Errorf("failed to output SBOMs: %w", err)
	}

	logger.LogDebug(ctx, "SBOM transfer process completed successfully ✅")
	return nil
}

func dryMode(ctx context.Context, iterator iterator.SBOMIterator, outputDir string) error {
	logger.LogDebug(ctx, "Dry-run mode enabled. Preparing to display SBOM details.")

	processor := sbom.NewSBOMProcessor(outputDir, false) // No need for output directory in dry-run mode
	sbomCount := 0

	for {
		sbom, err := iterator.Next(ctx)
		if err == io.EOF {
			break // No more SBOMs
		}
		if err != nil {
			logger.LogError(ctx, err, "Error retrieving SBOM from iterator")
			continue
		}

		logger.LogDebug(ctx, "Processing SBOM from memory", "repo", sbom.Repo, "version", sbom.Version)

		doc, err := processor.ProcessSBOMs(sbom.Data, sbom.Repo)
		if err != nil {
			logger.LogError(ctx, err, "Failed to process SBOM")
			continue
		}

		// If outputDir is provided, save the SBOM file
		if outputDir != "" {
			if err := processor.WriteSBOM(doc, sbom.Repo); err != nil {
				logger.LogError(ctx, err, "Failed to write SBOM to output directory")
			}
		}

		sbomCount++
		logger.LogDebug(ctx, fmt.Sprintf("%d. Repo: %s | Format: %s | SpecVersion: %s", sbomCount, sbom.Repo, doc.Format, doc.SpecVersion))
	}

	logger.LogDebug(ctx, "Dry-run mode completed", "total_sboms_processed", sbomCount)
	return nil
}
