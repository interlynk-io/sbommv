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
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/interlynk-io/sbomasm/pkg/detect"
)

// Format-specific structs for basic parsing
type cycloneDXJSON struct {
	BOMFormat    string `json:"bomFormat"`
	SpecVersion  string `json:"specVersion"`
	Components   []any  `json:"components"`
	Dependencies []any  `json:"dependencies"`
}

type cycloneDXXML struct {
	XMLName      xml.Name `xml:"bom"`
	SpecVersion  string   `xml:"version,attr"`
	Components   []any    `xml:"components>component"`
	Dependencies []any    `xml:"dependencies>dependency"`
}

type spdxJSON struct {
	SPDXID        string `json:"SPDXID"`
	SpecVersion   string `json:"spdxVersion"`
	Packages      []any  `json:"packages"`
	Relationships []any  `json:"relationships"`
}

func (p *SBOMProcessor) detectAndParse(doc *SBOMDocument) error {
	// Convert SBOM content whih is in byte to an `io.ReadSeek
	sbomReader := bytes.NewReader(doc.Content)

	// Use sbomasms Detect function
	specFormat, fileFormat, err := detect.Detect(sbomReader)
	if err != nil {
		return fmt.Errorf("failed to detect SBOM format: %w", err)
	}

	// ✅ Map detected format to our SBOMFormat type
	switch specFormat {

	case detect.SBOMSpecSPDX:
		if fileFormat == detect.FileFormatJSON {
			doc.Format = FormatSPDXJSON
		} else if fileFormat == detect.FileFormatTagValue {
			doc.Format = FormatSPDXTag
		} else if fileFormat == detect.FileFormatYAML {
			doc.Format = FormatSPDXYAML
		}
	case detect.SBOMSpecCDX:
		if fileFormat == detect.FileFormatJSON {
			doc.Format = FormatCycloneDXJSON
		} else if fileFormat == detect.FileFormatXML {
			doc.Format = FormatCycloneDXXML
		}
	default:
		doc.Format = FormatUnknown
		return fmt.Errorf("unknown SBOM format")
	}
	return p.parseSBOMContent(doc)
}

func (p *SBOMProcessor) parseSBOMContent(doc *SBOMDocument) error {
	switch doc.Format {
	case FormatCycloneDXJSON, FormatCycloneDXXML:
		var cdx cycloneDXJSON
		if err := json.Unmarshal(doc.Content, &cdx); err == nil {
			doc.SpecVersion = cdx.SpecVersion
		}
	case FormatSPDXJSON, FormatSPDXYAML:
		var spdx spdxJSON
		if err := json.Unmarshal(doc.Content, &spdx); err == nil {
			doc.SpecVersion = spdx.SpecVersion
		}

	case FormatSPDXTag:
		lines := strings.Split(string(doc.Content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "SPDXVersion:") {
				doc.SpecVersion = strings.TrimSpace(strings.TrimPrefix(line, "SPDXVersion:"))
				break
			}
		}
	default:
		return fmt.Errorf("unsupported SBOM format for parsing: %s", doc.Format)
	}
	return nil
}
