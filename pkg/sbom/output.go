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
	"fmt"
	"os"
	"path/filepath"

	"github.com/interlynk-io/sbommv/pkg/logger"
)

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
