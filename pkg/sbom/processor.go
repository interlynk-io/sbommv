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
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/logger"
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

// ProcessSBOMFromBytes processes an SBOM directly from memory
func (p *SBOMProcessor) ProcessSBOMs(content []byte, repoName, filePath string) (SBOMDocument, error) {
	if len(content) == 0 {
		return SBOMDocument{}, errors.New("empty SBOM content")
	}
	if filePath == "" {
		filePath = "N/A"
	}

	doc := SBOMDocument{
		// Filename: fmt.Sprintf("%s.sbom.json", repoName), // Use repo name as filename
		Filename: filePath,
		Content:  content,
	}

	// Detect format and parse content
	if err := p.detectAndParse(&doc); err != nil {
		return doc, fmt.Errorf("detecting format: %w", err)
	}

	return doc, nil
}

// WriteSBOM writes an SBOM to the output directory
func (p *SBOMProcessor) WriteSBOM(doc SBOMDocument, repoName string) error {
	if p.outputDir == "" {
		return nil // No output directory specified, skip writing
	}

	// Construct full path: sboms/<org>/<repo>.sbom.json
	outputPath := filepath.Join(p.outputDir, repoName+".sbom.json")
	outputDir := filepath.Dir(outputPath) // Extract directory path

	// Ensure all parent directories exist
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Write SBOM file
	if err := os.WriteFile(outputPath, doc.Content, 0o644); err != nil {
		return fmt.Errorf("writing SBOM file: %w", err)
	}

	logger.LogDebug(context.Background(), "SBOM successfully written", "path", outputPath)
	return nil
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
