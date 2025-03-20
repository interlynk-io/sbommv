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

package folder

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type FolderOutputReporter struct {
	folderPath string
}

func NewFolderOutputReporter(folderPath string) *FolderOutputReporter {
	return &FolderOutputReporter{folderPath: folderPath}
}

func (r *FolderOutputReporter) DryRun(ctx tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Dry-run mode: Displaying SBOMs for folder output")
	fmt.Println("\n📦 Folder Output Adapter Dry-Run")
	sbomCount := 0

	for {
		sbom, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			return err
		}

		namespace := filepath.Base(sbom.Namespace)
		if namespace == "" {
			namespace = fmt.Sprintf("sbom_%s.json", uuid.New().String())
		}

		// outputPath := filepath.Join(r.folderPath, namespace)
		outputPath := r.folderPath

		outputFile := filepath.Join(outputPath, sbom.Path)
		if sbom.Path == "" {
			outputFile = filepath.Join(outputPath, fmt.Sprintf("%s.sbom.json", uuid.New().String()))
		}

		fmt.Printf("- 📂 Would write: %s\n", outputFile)
		sbomCount++
	}

	fmt.Printf("\n📊 Total SBOMs to be stored: %d\n", sbomCount)
	logger.LogDebug(ctx.Context, "Dry-run completed", "total_sboms", sbomCount)
	return nil
}
