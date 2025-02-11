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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/interlynk-io/sbommv/pkg/logger"
)

// PrettyPrintSBOM prints an SBOM in formatted JSON
func PrettyPrintSBOM(w io.Writer, Content []byte) error {
	// First try to unmarshal into a generic interface{} to get the structure
	var data interface{}
	if err := json.Unmarshal(Content, &data); err != nil {
		return fmt.Errorf("failed to parse SBOM content: %w", err)
	}

	// Create a buffer for pretty printing
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to format SBOM: %w", err)
	}

	// Write the formatted JSON
	_, err := w.Write(buf.Bytes())
	return err
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
