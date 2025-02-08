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
	logger.LogDebug(transferCtx.Context, "Fetching SBOMs from input adapter...")
	sbomIterator, err := inputAdapterInstance.FetchSBOMs(transferCtx)
	if err != nil {
		return fmt.Errorf("failed to fetch SBOMs: %w", err)
	}

	logger.LogDebug(transferCtx.Context, "SBOM fetching started successfully", "sbomIterator", sbomIterator)

	// Dry-Run Mode: Display SBOMs Without Uploading
	if config.DryRun {
		logger.LogDebug(transferCtx.Context, "Dry-run mode enabled: Displaying retrieved SBOMs", "values", config.DryRun)

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

	logger.LogDebug(ctx, "Processing SBOMs in dry-run mode")

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
		logger.LogDebug(ctx, "%d. File: %s | Format: %s | SpecVersion: %s\n", sbomCount, doc.Filename, doc.Format, doc.SpecVersion)
	}

	logger.LogDebug(ctx, "Dry-run mode completed", "total_sboms_processed", sbomCount)
	return nil
}
