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
	"errors"
	"fmt"
)

// SBOMFormat represents supported SBOM document formats
type SBOMFormat string

const (
	FormatCycloneDXJSON SBOMFormat = "CycloneDX-JSON"
	FormatCycloneDXXML  SBOMFormat = "CycloneDX-XML"
	FormatSPDXJSON      SBOMFormat = "SPDX-JSON"
	FormatSPDXYAML      SBOMFormat = "SPDX-YAML"
	FormatSPDXTag       SBOMFormat = "SPDX-Tag"
	FormatUnknown       SBOMFormat = "Unknown"
)

// SBOMDocument represents a processed SBOM file
type SBOMDocument struct {
	Filename    string
	Format      SBOMFormat
	Content     []byte
	SpecVersion string
}

// SBOMProcessor handles SBOM document processing
type SBOMProcessor struct {
	outputDir string
	verbose   bool
	data      []byte
	repo      string
	path      string
}

// NewSBOMProcessor creates a new SBOM processor
// TODO: outputDir will be used to save files in it
// TODO: verbose will be used to o/p sbom content on stdout
func NewSBOMProcessor(outputDir string, verbose bool) *SBOMProcessor {
	return &SBOMProcessor{
		// outputdir: represent writing all sbom files inside directory
		outputDir: outputDir,

		// verbose: represent wrtitng all sbom content to the terminal itself
		verbose: verbose,
	}
}

func (p *SBOMProcessor) Update(content []byte, repoName, filePath string) {
	p.data = content
	p.repo = repoName
	p.path = filePath
}

// ProcessSBOMFromBytes processes an SBOM directly from memory
func (p *SBOMProcessor) ProcessSBOMs() (SBOMDocument, error) {
	if len(p.data) == 0 {
		return SBOMDocument{}, errors.New("empty SBOM content")
	}
	if p.path == "" {
		p.path = "N/A"
	}

	doc := SBOMDocument{
		Filename: p.path,
		Content:  p.data,
	}

	// Detect format and parse content
	if err := p.detectAndParse(&doc); err != nil {
		return doc, fmt.Errorf("detecting format: %w", err)
	}

	return doc, nil
}
