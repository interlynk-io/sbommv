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

package dependencytrack

import (
	"fmt"
	"io"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type DependencyTrackReporter struct {
	apiURL         string
	projectName    string
	projectVersion string
}

func NewDependencyTrackReporter(apiURL, projectName, projectVersion string) *DependencyTrackReporter {
	return &DependencyTrackReporter{
		apiURL:         apiURL,
		projectName:    projectName,
		projectVersion: projectVersion,
	}
}

func (r *DependencyTrackReporter) DryRun(ctx tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Dry-run mode: Simulating SBOM upload to Dependency-Track")
	fmt.Println("\n📦 Dependency-Track Output Adapter Dry-Run")
	fmt.Printf("📦 API Endpoint: %s\n", r.apiURL)
	sbomCount := 0

	processor := sbom.NewSBOMProcessor("", false)
	for {
		sbom, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM")
			return err
		}

		processor.Update(sbom.Data, sbom.Namespace, "")
		doc, err := processor.ProcessSBOMs()
		if err != nil {
			logger.LogError(ctx.Context, err, "Failed to process SBOM")
			return err
		}

		var projectName, projectVersion string

		if r.projectName != "" {
			projectName, projectVersion = getExplicitProjectVersion(ctx, r.projectName, r.projectVersion)
		} else if sbom.Namespace != "" {
			projectName, projectVersion = getImplicitProjectVersion(ctx, sbom.Namespace, sbom.Version)
		} else {
			continue
		}

		finalProjectName := fmt.Sprintf("%s-%s", projectName, projectVersion)

		fmt.Printf("- 📁 Would upload to project '%s' | Format: %s | SpecVersion: %s\n",
			finalProjectName, doc.Format, doc.SpecVersion)
		sbomCount++
	}
	fmt.Printf("📦 DTrack API Endpoint: %s\n", r.apiURL)
	fmt.Printf("\n 📊 Total SBOMs to upload: %d\n", sbomCount)
	return nil
}
