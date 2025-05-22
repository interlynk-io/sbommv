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

	adapter "github.com/interlynk-io/sbommv/pkg/adapter"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/monitor"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/spf13/cobra"
)

func TransferRun(ctx context.Context, cmd *cobra.Command, config types.Config) error {
	logger.LogDebug(ctx, "Starting SBOM transfer process....")

	// Initialize shared context with metadata support
	transferCtx := tcontext.NewTransferMetadata(ctx)

	var inputAdapterInstance, outputAdapterInstance adapter.Adapter
	var err error

	if config.SourceAdapter == "github" && config.Daemon {
		config.Overwrite = true
		logger.LogDebug(transferCtx.Context, "overwrite flag set to true for github adapter", "overwrite_value", config.Overwrite)
	}

	adapters, iAdp, oAdp, err := adapter.NewAdapter(*transferCtx, config)
	if err != nil {
		return fmt.Errorf("failed to initialize adapters: %v", err)
	}

	// store source adapter type and destination adapter using ctx for later use
	transferCtx.WithValue("source", iAdp)
	transferCtx.WithValue("destination", oAdp)

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

	logger.LogDebug(transferCtx.Context, "Input adapter instance config", "value", inputAdapterInstance)

	// Parse and validate output adapter parameters
	if err := outputAdapterInstance.ParseAndValidateParams(cmd); err != nil {
		return fmt.Errorf("output adapter error: %w", err)
	}

	logger.LogDebug(transferCtx.Context, "Output adapter instance config", "value", outputAdapterInstance)

	var sbomIterator iterator.SBOMIterator

	// fetch SBOMs in daemon mode
	if config.Daemon {
		if ma, ok := inputAdapterInstance.(monitor.MonitorAdapter); ok {
			sbomIterator, err = ma.Monitor(*transferCtx)
		} else {
			return fmt.Errorf("input adapter %s does not support daemon mode", config.SourceAdapter)
		}
	} else {
		// fetch SBOMs in one go
		sbomIterator, err = inputAdapterInstance.FetchSBOMs(*transferCtx)
		if err != nil {
			return fmt.Errorf("failed to fetch SBOMs: %w", err)
		}
	}

	// process SBOMs for conversion
	convertedIterator := sbomProcessing(*transferCtx, config, sbomIterator)

	if config.DryRun {
		if config.Daemon {
		}
		logger.LogDebug(transferCtx.Context, "Dry-run mode enabled: Displaying retrieved SBOMs", "values", config.DryRun)
		dryRun(*transferCtx, convertedIterator, inputAdapterInstance, outputAdapterInstance, config)
		return nil
	}

	// Process & Upload SBOMs Sequentially
	if err := outputAdapterInstance.UploadSBOMs(*transferCtx, convertedIterator); err != nil {
		return fmt.Errorf("%w", err)
	}

	logger.LogDebug(ctx, "SBOM transfer process completed successfully ✅")
	return nil
}

func dryRun(ctx tcontext.TransferMetadata, sbomIterator iterator.SBOMIterator, input, output adapter.Adapter, config types.Config) error {
	// dry-run mode for daemon
	if config.Daemon {
		logger.LogDebug(ctx.Context, "Dry-run mode in daemon: Previewing SBOMs in real-time")
		fmt.Println("\n------------------------------------------                                 ------------------------------------------")
		fmt.Println("------------------------------------------🌐 DAEMON MODE DRY-RUN PREVIEW 🌐------------------------------------------")
		fmt.Println("------------------------------------------                                 ------------------------------------------\n")
		fmt.Println()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("\n✅ Dry-run stopped due to context cancellation")
				return ctx.Err()

			default:
				sbom, err := sbomIterator.Next(ctx)
				if err != nil {
					if err == context.Canceled || err == context.DeadlineExceeded {
						fmt.Println("\n✅ Dry-run stopped due to context cancellation")
						return err
					}
					logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
					continue
				}
				fmt.Println()
				fmt.Println("------------------------------------------🌐 INPUT ADAPTER DRY-RUN OUTPUT 🌐------------------------------------------")

				// preview single SBOM for input
				inputIter := iterator.NewMemoryIterator([]*iterator.SBOM{sbom})
				if err := input.DryRun(ctx, inputIter); err != nil {
					logger.LogError(ctx.Context, err, "Input dry-run failed")
					continue
				}

				fmt.Println()
				fmt.Println("------------------------------------------🌐 OUTPUT ADAPTER DRY-RUN OUTPUT 🌐------------------------------------------")

				// preview single SBOM for output
				outputIter := iterator.NewMemoryIterator([]*iterator.SBOM{sbom})
				if err := output.DryRun(ctx, outputIter); err != nil {
					logger.LogError(ctx.Context, err, "Output dry-run failed")
					continue
				}

				fmt.Println("\n                              +-+-+-+-+-+-+ SBOM DRY-RUN COMPLETED +-+-+-+-+\n")
			}
		}
	} else {
		// Step 1: Store SBOMs in memory (avoid consuming iterator)
		var sboms []*iterator.SBOM
		for {
			sbom, err := sbomIterator.Next(ctx)
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
		if err := input.DryRun(ctx, iterator.NewMemoryIterator(sboms)); err != nil {
			return fmt.Errorf("failed to execute dry-run mode for input adapter: %v", err)
		}
		fmt.Println()
		fmt.Println("-----------------🌐 OUTPUT ADAPTER DRY-RUN OUTPUT 🌐-----------------")

		// Step 3: Use the same stored SBOMs for output dry-run
		if err := output.DryRun(ctx, iterator.NewMemoryIterator(sboms)); err != nil {
			return fmt.Errorf("failed to execute dry-run mode for output adapter: %v", err)
		}
	}
	return nil
}

func sbomProcessing(ctx tcontext.TransferMetadata, config types.Config, sbomIterator iterator.SBOMIterator) iterator.SBOMIterator {
	logger.LogDebug(ctx.Context, "Checking adapter eligibility for undergoing conversion layer", "adapter type", config.DestinationAdapter)

	// convert sbom to cdx for DTrack adapter only
	if types.AdapterType(config.DestinationAdapter) == types.DtrackAdapterType {

		logger.LogDebug(ctx.Context, "Adapter is eligible for SBOM conversion", "adapter type", config.DestinationAdapter)
		// convertedSBOMs := sbomConversion(sbomIterator, ctx)
		return iterator.NewConvertedIterator(sbomIterator, sbom.FormatSpecCycloneDX)
	} else {
		logger.LogDebug(ctx.Context, "Adapter is not eligible for SBOM conversion", "adapter type", config.DestinationAdapter)
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

func convertMinifiedJSON(transferCtx tcontext.TransferMetadata, data []byte, totalMinifiedSBOM int) ([]byte, int, error) {
	minified, original, formatted, err := isMinifiedJSON(data)
	if err != nil {
		logger.LogError(transferCtx.Context, err, "Error while isMinifiedJSON")
		return original, totalMinifiedSBOM, nil
	}

	if minified {
		totalMinifiedSBOM++
		return formatted, totalMinifiedSBOM, nil
	}
	return original, totalMinifiedSBOM, nil
}
