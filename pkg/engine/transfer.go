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
)

func TransferRun(ctx context.Context, config mvtypes.Config) error {
	// sourceType, err := utils.DetectSourceType(sourceAdpCfg.URL)
	// if err != nil {
	// 	return fmt.Errorf("input URL is invalid source type")
	// }

	logger.LogInfo(ctx, "input adapter", "source", config.SourceType)

	sourceAdapter, err := adapter.NewSourceAdapter(config)
	if err != nil {
		return fmt.Errorf("Failed to get an Source Adapter")
	}

	allSBOMs, err := sourceAdapter.GetSBOMs(ctx)
	if err != nil {
		return fmt.Errorf("Failed to get SBOMs %v", err)
	}
	logger.LogInfo(ctx, "List of retieved SBOMs from source", "sboms", allSBOMs)

	// destType, err := utils.DetectDestinationType(destAdpCfg.BaseURL)
	// if err != nil {
	// 	return fmt.Errorf("destination URL is invalid destination type %v", err)
	// }

	logger.LogInfo(ctx, "output adapter", "destination", config.DestinationType)

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
