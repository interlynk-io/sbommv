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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SBOMFormat represents supported SBOM document formats
type SBOMFormat string

const (
	FormatCycloneDXJSON SBOMFormat = "CycloneDX-JSON"
	FormatCycloneDXXML  SBOMFormat = "CycloneDX-XML"
	FormatSPDXJSON      SBOMFormat = "SPDX-JSON"
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
}

// NewSBOMProcessor creates a new SBOM processor
func NewSBOMProcessor(outputDir string, verbose bool) *SBOMProcessor {
	return &SBOMProcessor{
		// outputdir: represent writing all sbom files inside directory
		outputDir: outputDir,

		// verbose: represent wrtitng all sbom content to the terminal itself
		verbose: verbose,
	}
}

// ProcessSBOMs processes multiple SBOM files
func (p *SBOMProcessor) ProcessSBOMs(filenames []string) ([]SBOMDocument, error) {
	var documents []SBOMDocument
	var errs []error

	for _, filename := range filenames {
		doc, err := p.ProcessSBOM(filename)
		if err != nil {
			errs = append(errs, fmt.Errorf("processing %s: %w", filename, err))
			continue
		}
		documents = append(documents, doc)
	}

	if len(errs) > 0 {
		return documents, fmt.Errorf("encountered %d errors: %v", len(errs), errs[0])
	}

	return documents, nil
}

// ProcessSBOM processes a single SBOM file
func (p *SBOMProcessor) ProcessSBOM(filename string) (SBOMDocument, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return SBOMDocument{}, fmt.Errorf("reading file: %w", err)
	}

	doc := SBOMDocument{
		Filename: filename,
		Content:  content,
	}

	// Detect format and parse content
	if err := p.detectAndParse(&doc); err != nil {
		return doc, fmt.Errorf("detecting format: %w", err)
	}

	// Write processed document if output directory is specified
	if p.outputDir != "" {
		if err := p.writeProcessedSBOM(doc); err != nil {
			return doc, fmt.Errorf("writing output: %w", err)
		}
	}

	return doc, nil
}

// detectAndParse detects the SBOM format and parses its content
func (p *SBOMProcessor) detectAndParse(doc *SBOMDocument) error {
	// Try JSON formats first
	if isJSON(doc.Content) {
		if format, ok := p.detectJSONFormat(doc.Content); ok {
			doc.Format = format
			return p.parseJSONContent(doc)
		}
	}

	// Try XML formats
	if isXML(doc.Content) {
		if format, ok := p.detectXMLFormat(doc.Content); ok {
			doc.Format = format
			return p.parseXMLContent(doc)
		}
	}

	// Try SPDX tag-value format
	if isSPDXTag(doc.Content) {
		doc.Format = FormatSPDXTag
		return p.parseSPDXTagContent(doc)
	}

	doc.Format = FormatUnknown
	return errors.New("unknown SBOM format")
}

func (p *SBOMProcessor) writeProcessedSBOM(doc SBOMDocument) error {
	if err := os.MkdirAll(p.outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	outPath := filepath.Join(p.outputDir, filepath.Base(doc.Filename))
	return os.WriteFile(outPath, doc.Content, 0o644)
}

// Helper functions to detect formats
func isJSON(content []byte) bool {
	var js json.RawMessage
	return json.Unmarshal(content, &js) == nil
}

func isXML(content []byte) bool {
	return xml.Unmarshal(content, new(interface{})) == nil
}

func isSPDXTag(content []byte) bool {
	return strings.Contains(string(content), "SPDXVersion:") ||
		strings.Contains(string(content), "SPDX-License-Identifier:")
}
