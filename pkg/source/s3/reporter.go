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

package s3

import (
	"fmt"
	"io"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type S3Reporter struct {
	verbose    bool
	inputDir   string
	bucketName string
	prefix     string
}

func NewS3Reporter(verbose bool, inputDir, bucketName, prefix string) *S3Reporter {
	return &S3Reporter{
		verbose:    verbose,
		inputDir:   inputDir,
		bucketName: bucketName,
		prefix:     prefix,
	}
}

func (s *S3Reporter) DryRun(ctx tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Dry-run mode: Displaying SBOMs fetched from S3")
	processor := sbom.NewSBOMProcessor(s.inputDir, s.verbose)
	sbomCount := 0
	fmt.Println("\n📦 Details of all Fetched SBOMs by S3 Input Adapter")
	for {
		sbom, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			return err
		}
		processor.Update(sbom.Data, "", sbom.Path)
		doc, err := processor.ProcessSBOMs()
		if err != nil {
			logger.LogError(ctx.Context, err, "Failed to process SBOM")
			return err
		}

		if s.inputDir != "" {
			if err := processor.WriteSBOM(doc, ""); err != nil {
				logger.LogError(ctx.Context, err, "Failed to write SBOM")
				return err
			}
		}

		if s.verbose {
			fmt.Printf("\n-------------------- 📜 SBOM Content --------------------\n")
			fmt.Printf("📂 Filename: %s\n", doc.Filename)
			fmt.Printf("📦 Format: %s | SpecVersion: %s\n\n", doc.Format, doc.SpecVersion)
			fmt.Println(string(doc.Content))
			fmt.Println("------------------------------------------------------")
		}

		sbomCount++
		fmt.Printf(" - 📁 Folder: %s | Format: %s | SpecVersion: %s | Filename: %s\n",
			s.bucketName, doc.Format, doc.SpecVersion, doc.Filename)
	}
	fmt.Printf("\n📦 Total SBOMs fetched: %d\n", sbomCount)
	return nil
}
