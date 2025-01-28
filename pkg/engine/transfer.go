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

	adapter "github.com/interlynk-io/sbommv/pkg/adapters"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/mvtypes"
	"github.com/interlynk-io/sbommv/pkg/sbom"
)

func TransferRun(ctx context.Context, config mvtypes.Config) error {
	logger.LogDebug(ctx, "input adapter", "source", config.SourceType)

	sourceAdapter, err := adapter.NewSourceAdapter(config)
	if err != nil {
		return fmt.Errorf("Failed to get an Source Adapter")
	}

	allSBOMs, err := sourceAdapter.GetSBOMs(ctx)
	if err != nil {
		return fmt.Errorf("Failed to get SBOMs %v", err)
	}
	logger.LogDebug(ctx, "List of retrieved SBOMs from source", "sboms", allSBOMs)

	if config.DryRun {
		err := dryMode(ctx, allSBOMs)
		if err != nil {
			return fmt.Errorf("failed to execute dry-run mode: %v", err)
		}
		return nil
	}
	logger.LogDebug(ctx, "output adapter", "destination", config.DestinationType)

	destAdapter, err := adapter.NewDestAdapter(config)
	if err != nil {
		return fmt.Errorf("Failed to get a Destination Adapter %v", err)
	}
	fmt.Println("destAdapter: ", destAdapter)

	err = destAdapter.UploadSBOMs(ctx, allSBOMs)
	if err != nil {
		return fmt.Errorf("Failed to upload SBOMs %v", err)
	}
	return nil
}

func dryMode(ctx context.Context, allSBOMs map[string][]string) error {
	logger.LogDebug(ctx, "Dry-run mode enabled. Printing SBOM details:")

	processor := sbom.NewSBOMProcessor("", false)
	number := 0

	fmt.Println("List of SBOM files fetched by Input Adapter (Grouped by Version):")
	for version, sboms := range allSBOMs {
		fmt.Printf("\nVersion: %s\n", version)
		for _, sbomPath := range sboms {
			doc, err := processor.ProcessSBOM(sbomPath)
			if err != nil {
				logger.LogError(ctx, fmt.Errorf("Failed to process SBOM %v", err), sbomPath)
				continue
			}

			number++
			fmt.Printf("%v. %s  %s  %s\n", number, doc.Filename, doc.Format, doc.SpecVersion)

			// Uncomment if you want to pretty print the SBOM content
			// if err := sbom.PrettyPrintSBOM(os.Stdout, doc.Content); err != nil {
			//     logger.LogError(ctx, err, "Failed to pretty-print SBOM content")
			// }
		}
	}
	return nil
}
