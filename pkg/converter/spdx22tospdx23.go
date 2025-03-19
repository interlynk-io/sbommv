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

package converter

import (
	"encoding/json"
	"fmt"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/spdx/tools-golang/convert"
	"github.com/spdx/tools-golang/spdx/v2/v2_2"
	"github.com/spdx/tools-golang/spdx/v2/v2_3"
)

func ConvertSPDX22ToSPDX23(ctx tcontext.TransferMetadata, sbomData []byte) ([]byte, error) {
	logger.LogDebug(ctx.Context, "Converting SPDX 2.2 to 2.3")

	var newSBOMData []byte
	var err error

	// Convert SPDX 2.2 to 2.3
	var sourceDoc v2_2.Document
	var targetDoc v2_3.Document

	// Parse SPDX 2.2 JSON
	err = json.Unmarshal(sbomData, &sourceDoc)
	if err != nil {
		logger.LogDebug(ctx.Context, "Failed to unmarshal SPDX 2.2 JSON", "error", err)
		return nil, fmt.Errorf("unmarshaling SPDX 2.2 JSON: %w", err)
	}

	// Convert 2.2 to 2.3
	err = convert.Document(&sourceDoc, &targetDoc)
	if err != nil {
		logger.LogDebug(ctx.Context, "Failed to convert SPDX 2.2 to 2.3", "error", err)
		return nil, fmt.Errorf("converting SPDX 2.2 to 2.3: %w", err)
	}

	// Serialize back to JSON for Protobom
	newSBOMData, err = json.Marshal(&targetDoc)
	if err != nil {
		return nil, fmt.Errorf("marshaling converted SPDX 2.3: %w", err)
	}

	// // TODO: remove this later: only for testing
	// outputFile := fmt.Sprintf("spdx22tospdx23-%s.json", "foo-bar")
	// // Write bytes to a file
	// if err := os.WriteFile(outputFile, sbomData, 0o644); err != nil {
	// 	return nil, fmt.Errorf("failed to write to file %s: %w", outputFile, err)
	// }

	logger.LogDebug(ctx.Context, "Conversion successful from SPDX 2.2 to SPDX 2.3")

	return newSBOMData, err
}
