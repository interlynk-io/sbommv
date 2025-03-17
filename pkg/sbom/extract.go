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
	"strings"
)

type PrimaryComponent struct {
	Name    string
	Version string
}

func ExtractPrimaryComponentName(content []byte) (PrimaryComponent, error) {
	// get primaryComp for cyclonedx
	var cdx struct {
		Metadata struct {
			Component struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"component"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal(content, &cdx); err == nil && cdx.Metadata.Component.Name != "" {
		return PrimaryComponent{
			Name:    cdx.Metadata.Component.Name,
			Version: cdx.Metadata.Component.Version,
		}, nil
	}

	// get primaryComp for cyclonedx
	var spdx struct {
		Packages []struct {
			SPDXID      string `json:"SPDXID"`
			Name        string `json:"name"`
			VersionInfo string `json:"versionInfo"`
		} `json:"packages"`
		Relationships []struct {
			SPDXElementID      string `json:"spdxElementId"`
			RelationshipType   string `json:"relationshipType"`
			RelatedSPDXElement string `json:"relatedSpdxElement"`
		} `json:"relationships"`
	}

	if err := json.Unmarshal(content, &spdx); err == nil {
		var targetID string

		// Find DESCRIBES relationship from document
		for _, rel := range spdx.Relationships {
			if rel.SPDXElementID == "SPDXRef-DOCUMENT" && strings.ToUpper(rel.RelationshipType) == "DESCRIBES" {
				targetID = rel.RelatedSPDXElement
				break
			}
		}

		// Match targetID to a package
		for _, pkg := range spdx.Packages {
			if pkg.SPDXID == targetID && pkg.Name != "" {
				return PrimaryComponent{
					Name:    pkg.Name,
					Version: pkg.VersionInfo,
				}, nil
			}
		}
	}
	return PrimaryComponent{}, fmt.Errorf("no primary component found in JSON SBOM")
}
