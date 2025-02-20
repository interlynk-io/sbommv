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
// -------------------------------------------------------------------------

// pkg/target/interlynk/reporter.go
package interlynk

import (
	"context"
	"fmt"
	"io"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/sbom"
)

type InterlynkReporter struct {
	baseURL string
	apiKey  string
}

func NewInterlynkReporter(baseURL, apiKey string) *InterlynkReporter {
	return &InterlynkReporter{
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

func (r *InterlynkReporter) DryRun(ctx context.Context, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx, "🔄 Dry-Run Mode: Simulating Upload to Interlynk...")

	// Step 1: Validate Interlynk Connection
	if err := ValidateInterlynkConnection(r.baseURL, r.apiKey); err != nil {
		return fmt.Errorf("interlynk validation failed: %w", err)
	}

	// Step 2: Initialize SBOM Processor
	processor := sbom.NewSBOMProcessor("", false)

	// Step 3: Organize SBOMs into Projects
	projectSBOMs := make(map[string][]sbom.SBOMDocument)
	totalSBOMs := 0
	uniqueFormats := make(map[string]struct{})

	for {
		sbom, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.LogError(ctx, err, "Error retrieving SBOM from iterator")
			return err // Propagate error
		}

		processor.Update(sbom.Data, sbom.Namespace, sbom.Path)
		doc, err := processor.ProcessSBOMs()
		if err != nil {
			logger.LogError(ctx, err, "Failed to process SBOM")
			return err
		}

		projectKey := fmt.Sprintf("%s", sbom.Namespace)
		projectSBOMs[projectKey] = append(projectSBOMs[projectKey], doc)
		totalSBOMs++
		uniqueFormats[string(doc.Format)] = struct{}{}
	}

	// Step 4: Print Dry-Run Summary
	fmt.Println("")
	fmt.Printf("📦 Interlynk API Endpoint: %s\n", r.baseURL)
	fmt.Printf("📂 Project Groups Total: %d\n", len(projectSBOMs))
	fmt.Printf("📊 Total SBOMs to be Uploaded: %d\n", totalSBOMs)
	fmt.Printf("📦 INTERLYNK_SECURITY_TOKEN is valid\n")
	fmt.Printf("📦 Unique Formats: %s\n", formatSetToString(uniqueFormats))
	fmt.Println()

	for project, sboms := range projectSBOMs {
		fmt.Printf("📌 **Project: %s** → %d SBOMs\n", project, len(sboms))
		for _, doc := range sboms {
			fmt.Printf("   - 📁  | Format: %s | SpecVersion: %s | Size: %d KB | Filename: %s\n",
				doc.Format, doc.SpecVersion, len(doc.Content)/1024, doc.Filename)
		}
	}

	fmt.Println("\n✅ **Dry-run completed**. No data was uploaded to Interlynk.")
	return nil
}
