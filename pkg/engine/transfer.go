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
	"log"

	adapter "github.com/interlynk-io/sbommv/pkg/adapters"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/mvtypes"
	"github.com/interlynk-io/sbommv/pkg/sbom"
)

func TransferRun(ctx context.Context, config mvtypes.Config) error {
	// sourceType, err := utils.DetectSourceType(sourceAdpCfg.URL)
	// if err != nil {
	// 	return fmt.Errorf("input URL is invalid source type")
	// }

	logger.LogDebug(ctx, "input adapter", "source", config.SourceType)

	sourceAdapter, err := adapter.NewSourceAdapter(config)
	if err != nil {
		return fmt.Errorf("Failed to get an Source Adapter")
	}

	allSBOMs, err := sourceAdapter.GetSBOMs(ctx)
	if err != nil {
		return fmt.Errorf("Failed to get SBOMs %v", err)
	}
	logger.LogDebug(ctx, "List of retieved SBOMs from source", "sboms", allSBOMs)

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
		return fmt.Errorf("Failed to get an Destination Adapter %v", err)
	}

	err = destAdapter.UploadSBOMs(ctx, allSBOMs)
	if err != nil {
		return fmt.Errorf("Failed to upload SBOMs %v", err)
	}
	return nil
}

func dryMode(ctx context.Context, allSBOMs []string) error {
	// Handle dry-run mode

	logger.LogDebug(ctx, "Dry-run mode enabled. Printing SBOM details:")

	processor := sbom.NewSBOMProcessor("", false)
	docs, err := processor.ProcessSBOMs(allSBOMs)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("List of SBOM files fetched by Input Adapter")
	number := 0
	for _, doc := range docs {
		number++
		fmt.Printf("%v. %s  ", number, doc.Filename)
		fmt.Printf("%s  ", doc.Format)
		fmt.Printf("%s  \n", doc.SpecVersion)

		// if err := sbom.PrettyPrintSBOM(os.Stdout, doc.Content); err != nil {
		// 	log.Fatal(err)
		// }
	}

	return nil
}
