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

	if config.DryRun {
		logger.LogDebug(transferCtx.Context, "Dry-run mode enabled: Displaying retrieved SBOMs", "values", config.DryRun)
		dryRun(*transferCtx, sbomIterator, inputAdapterInstance, outputAdapterInstance)
	}

	// Process & Upload SBOMs Sequentially
	if err := outputAdapterInstance.UploadSBOMs(transferCtx, sbomIterator); err != nil {
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
