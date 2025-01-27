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
	"encoding/xml"
	"fmt"
	"strings"
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

func (p *SBOMProcessor) detectJSONFormat(content []byte) (SBOMFormat, bool) {
	// Try CycloneDX
	var cdx cycloneDXJSON
	if err := json.Unmarshal(content, &cdx); err == nil {
		if strings.EqualFold(cdx.BOMFormat, "CycloneDX") {
			return FormatCycloneDXJSON, true
		}
	}

	// Try SPDX
	var spdx spdxJSON
	if err := json.Unmarshal(content, &spdx); err == nil {
		if strings.HasPrefix(spdx.SPDXID, "SPDXRef-") {
			return FormatSPDXJSON, true
		}
	}

	return FormatUnknown, false
}

func (p *SBOMProcessor) detectXMLFormat(content []byte) (SBOMFormat, bool) {
	var cdx cycloneDXXML
	if err := xml.Unmarshal(content, &cdx); err == nil {
		return FormatCycloneDXXML, true
	}
	return FormatUnknown, false
}

func (p *SBOMProcessor) parseJSONContent(doc *SBOMDocument) error {
	switch doc.Format {
	case FormatCycloneDXJSON:
		var cdx cycloneDXJSON
		if err := json.Unmarshal(doc.Content, &cdx); err != nil {
			return fmt.Errorf("parsing CycloneDX JSON: %w", err)
		}
		doc.SpecVersion = cdx.SpecVersion

	case FormatSPDXJSON:
		var spdx spdxJSON
		if err := json.Unmarshal(doc.Content, &spdx); err != nil {
			return fmt.Errorf("parsing SPDX JSON: %w", err)
		}
		doc.SpecVersion = spdx.SpecVersion
	}
	return nil
}

func (p *SBOMProcessor) parseXMLContent(doc *SBOMDocument) error {
	var cdx cycloneDXXML
	if err := xml.Unmarshal(doc.Content, &cdx); err != nil {
		return fmt.Errorf("parsing CycloneDX XML: %w", err)
	}
	doc.SpecVersion = cdx.SpecVersion
	return nil
}

func (p *SBOMProcessor) parseSPDXTagContent(doc *SBOMDocument) error {
	lines := strings.Split(string(doc.Content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "SPDXVersion:") {
			doc.SpecVersion = strings.TrimSpace(strings.TrimPrefix(line, "SPDXVersion:"))
			break
		}
	}
	return nil
}
