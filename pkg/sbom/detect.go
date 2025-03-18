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

package sbom

import (
	"encoding/json"
	"fmt"
)

type FormatSpec string

const (
	FormatSpecCycloneDX FormatSpec = "cyclonedx"
	FormatSpecSPDX      FormatSpec = "spdx"
	FormatSpecUnknown   FormatSpec = "unknown"
)

type FormatSpecVersion string

const (
	FormatSpecVersionCycloneDXV1_3 FormatSpecVersion = "1.3"
	FormatSpecVersionCycloneDXV1_4 FormatSpecVersion = "1.4"
	FormatSpecVersionCycloneDXV1_5 FormatSpecVersion = "1.5"
	FormatSpecVersionCycloneDXV1_6 FormatSpecVersion = "1.6"
	FormatSpecVersionSPDXV2_1      FormatSpecVersion = "SPDX-2.1"
	FormatSpecVersionSPDXV2_2      FormatSpecVersion = "SPDX-2.2"
	FormatSpecVersionSPDXV2_3      FormatSpecVersion = "SPDX-2.3"
)

func DetectSBOMSpecAndVersion(data []byte) (FormatSpec, string, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", "", fmt.Errorf("unmarshaling SBOM: %w", err)
	}

	if version, ok := raw["specVersion"].(string); ok {
		return FormatSpecCycloneDX, version, nil
	}

	if version, ok := raw["spdxVersion"].(string); ok {
		return FormatSpecSPDX, version, nil
	}

	return FormatSpecUnknown, "", nil
}
