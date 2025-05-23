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

package github

import (
	"fmt"
	"io"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type GithubReporter struct {
	verbose  bool
	inputDir string
}

func NewGithubReporter(verbose bool, inputDir string) *GithubReporter {
	return &GithubReporter{
		verbose:  verbose,
		inputDir: inputDir,
	}
}

func (r *GithubReporter) DryRun(ctx tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Dry-run mode: Displaying SBOMs fetched from input adapter")

	processor := sbom.NewSBOMProcessor(r.inputDir, r.verbose)
	sbomCount := 0
	fmt.Println()
	fmt.Printf("📦 Details of all Fetched SBOMs by Github Input Adapter\n")

	for {
		sbom, err := iter.Next(ctx)
		if err == io.EOF {
			break // No more SBOMs
		}
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			continue
		}
		// Update processor with current SBOM data
		processor.Update(sbom.Data, sbom.Namespace, sbom.Path)

		doc, err := processor.ProcessSBOMs()
		if err != nil {
			logger.LogError(ctx.Context, err, "Failed to process SBOM")
			continue
		}

		// If outputDir is provided, save the SBOM file
		if r.inputDir != "" {
			if err := processor.WriteSBOM(doc, sbom.Namespace); err != nil {
				logger.LogError(ctx.Context, err, "Failed to write SBOM to output directory")
			}
		}

		// Print SBOM content if verbose mode is enabled
		if r.verbose {
			fmt.Println("\n-------------------- 📜 SBOM Content --------------------")
			fmt.Printf("📂 Filename: %s\n", doc.Filename)
			fmt.Printf("📦 Format: %s | SpecVersion: %s\n\n", doc.Format, doc.SpecVersion)
			fmt.Println(string(doc.Content))
			fmt.Println("------------------------------------------------------")
			fmt.Println()
		}

		sbomCount++
		fmt.Printf(" - 📁 Repo: %s | Format: %s | SpecVersion: %s | Filename: %s \n", sbom.Namespace, doc.Format, doc.SpecVersion, doc.Filename)

		// logger.LogInfo(ctx.Context, fmt.Sprintf("%d. Repo: %s | Format: %s | SpecVersion: %s | Filename: %s",
		// 	sbomCount, sbom.Repo, doc.Format, doc.SpecVersion, doc.Filename))
	}
	fmt.Printf("📊 Total SBOMs are: %d\n", sbomCount)

	logger.LogDebug(ctx.Context, "Dry-run mode completed for input adapter", "total_sboms", sbomCount)
	return nil
}
