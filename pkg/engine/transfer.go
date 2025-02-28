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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	adapter "github.com/interlynk-io/sbommv/pkg/adapter"
	"github.com/interlynk-io/sbommv/pkg/converter"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/spf13/cobra"
)

func TransferRun(ctx context.Context, cmd *cobra.Command, config types.Config) error {
	logger.LogDebug(ctx, "Starting SBOM transfer process")

	// Initialize shared context with metadata support
	transferCtx := tcontext.NewTransferMetadata(ctx)

	var inputAdapterInstance adapter.Adapter
	var outputAdapterInstance adapter.Adapter
	var err error

	adapters, err := adapter.NewAdapter(transferCtx, config)
	if err != nil {
		return fmt.Errorf("failed to initialize adapters: %v", err)
	}

	// Extract input and output adapters using predefined roles
	inputAdapterInstance = adapters[types.InputAdapterRole]
	outputAdapterInstance = adapters[types.OutputAdapterRole]

	if inputAdapterInstance == nil || outputAdapterInstance == nil {
		return fmt.Errorf("failed to initialize both input and output adapters")
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

	var convertedIterator iterator.SBOMIterator
	convertedIterator = sbomProcessing(transferCtx, config, sbomIterator)

	if config.DryRun {
		logger.LogDebug(transferCtx.Context, "Dry-run mode enabled: Displaying retrieved SBOMs", "values", config.DryRun)
		dryRun(*transferCtx, sbomIterator, inputAdapterInstance, outputAdapterInstance)
	}

	// Process & Upload SBOMs Sequentially
	if err := outputAdapterInstance.UploadSBOMs(transferCtx, convertedIterator); err != nil {
		return fmt.Errorf("failed to output SBOMs: %w", err)
	}

	logger.LogDebug(ctx, "SBOM transfer process completed successfully ✅")
	return nil
}

func dryRun(ctx tcontext.TransferMetadata, sbomIterator iterator.SBOMIterator, input, output adapter.Adapter) error {
	// Step 1: Store SBOMs in memory (avoid consuming iterator)
	var sboms []*iterator.SBOM
	for {
		sbom, err := sbomIterator.Next(ctx.Context)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			continue
		}
		sboms = append(sboms, sbom)
	}
	fmt.Println()

	fmt.Println("-----------------🌐 INPUT ADAPTER DRY-RUN OUTPUT 🌐-----------------")
	// Step 2: Use stored SBOMs for input dry-run
	if err := input.DryRun(&ctx, iterator.NewMemoryIterator(sboms)); err != nil {
		return fmt.Errorf("failed to execute dry-run mode for input adapter: %v", err)
	}
	fmt.Println()
	fmt.Println("-----------------🌐 OUTPUT ADAPTER DRY-RUN OUTPUT 🌐-----------------")

	// Step 3: Use the same stored SBOMs for output dry-run
	if err := output.DryRun(&ctx, iterator.NewMemoryIterator(sboms)); err != nil {
		return fmt.Errorf("failed to execute dry-run mode for output adapter: %v", err)
	}

	return nil
}

func sbomConversion(sbomIterator iterator.SBOMIterator, transferCtx tcontext.TransferMetadata) []*iterator.SBOM {
	logger.LogDebug(transferCtx.Context, "Processing SBOM conversion")

	var convertedSBOMs []*iterator.SBOM

	for {
		sbom, err := sbomIterator.Next(transferCtx.Context)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.LogError(transferCtx.Context, err, "Error retrieving SBOM from iterator")
			continue // Skip erroring SBOMs, proceed with next
		}

		// Convert SBOM to CycloneDX for Dependency-Track
		convertedData, err := converter.ConvertSBOM(transferCtx, sbom.Data, converter.FormatCycloneDX)
		if err != nil {
			logger.LogInfo(transferCtx.Context, "Failed to convert SBOM to CycloneDX", "file", sbom.Path, "error", err)
			continue // Skip unconverted SBOMs
		}

		// let's check minimfied SBOM
		sbom.Data, err = convertMinifiedJSON(transferCtx, convertedData)

		// Update SBOM data with converted content
		sbom.Data = convertedData

		if strings.Contains(sbom.Path, "spdx") {
			sbom.Path = strings.Replace(sbom.Path, "spdx", "spdxtocdx", 1)
			// transferCtx.FilePath = sbom.Path // Sync FilePath for logging
		}

		err = os.WriteFile(sbom.Path, sbom.Data, 0o644)
		if err != nil {
			fmt.Println("Error writing formatted JSON:", err)
		}

		convertedSBOMs = append(convertedSBOMs, sbom)
	}
	logger.LogDebug(transferCtx.Context, "Successfully SBOM conversion")

	return convertedSBOMs
}

func sbomProcessing(transferCtx *tcontext.TransferMetadata, config types.Config, sbomIterator iterator.SBOMIterator) iterator.SBOMIterator {
	logger.LogDebug(transferCtx.Context, "Checking adapter eligibility for undergoing conversion layer", "adapter type", config.DestinationType)

	// convert sbom to cdx for DTrack adapter only
	if types.AdapterType(config.DestinationType) == types.DtrackAdapterType {
		logger.LogDebug(transferCtx.Context, "Adapter eligible for conversion layer", "adapter type", config.DestinationType)

		logger.LogDebug(transferCtx.Context, "SBOM conversion will take place")
		convertedSBOMs := sbomConversion(sbomIterator, *transferCtx)

		return iterator.NewMemoryIterator(convertedSBOMs)
	} else {
		logger.LogDebug(transferCtx.Context, "Adapter not eligible for conversion layer", "adapter type", config.DestinationType)
		logger.LogDebug(transferCtx.Context, "SBOM conversion will not take place")
		return sbomIterator
	}
}

func isMinifiedJSON(data []byte) (bool, []byte, []byte, error) {
	// Try parsing the JSON
	var jsonData map[string]interface{}
	err := json.Unmarshal(data, &jsonData)
	if err != nil {
		return false, nil, nil, err
	}

	// Pretty-print the JSON
	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return false, nil, nil, err
	}

	// Check if original file is minified by comparing bytes
	if bytes.Equal(data, prettyJSON) {
		return false, data, prettyJSON, nil // Already formatted
	}

	return true, data, prettyJSON, nil // Minified JSON detected
}

func convertMinifiedJSON(transferCtx tcontext.TransferMetadata, data []byte) ([]byte, error) {
	minified, original, formatted, err := isMinifiedJSON(data)
	if err != nil {
		logger.LogError(transferCtx.Context, err, "Error while isMinifiedJSON")
		return original, nil
	}

	if minified {
		fmt.Println("Minified JSON detected! Converting to pretty format...")
		// Write formatted JSON back to file
		fmt.Println("isMinifiedJSON Converted successfully!")
		return formatted, nil

	} else {
		fmt.Println("isMinifiedJSON is already pretty-formatted.")
		return original, nil
	}
}
