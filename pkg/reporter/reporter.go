// pkg/reporter/reporter.go
package reporter

import (
	"context"
	"fmt"
	"io"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/sbom"
)

type SBOMReporter struct {
	verbose   bool
	outputDir string
}

func NewSBOMReporter(verbose bool, outputDir string) *SBOMReporter {
	return &SBOMReporter{verbose: verbose, outputDir: outputDir}
}

func (r *SBOMReporter) DryRun(ctx context.Context, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx, "Dry-run mode: Displaying SBOMs")
	processor := sbom.NewSBOMProcessor(r.outputDir, r.verbose)
	sbomCount := 0
	fmt.Println("\n📦 Details of all Fetched SBOMs")

	for {
		sbom, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.LogError(ctx, err, "Error retrieving SBOM from iterator")
			return err // Propagate instead of swallowing
		}
		processor.Update(sbom.Data, sbom.Namespace, sbom.Path)
		doc, err := processor.ProcessSBOMs()
		if err != nil {
			logger.LogError(ctx, err, "Failed to process SBOM")
			return err
		}
		if r.outputDir != "" {
			if err := processor.WriteSBOM(doc, sbom.Namespace); err != nil {
				logger.LogError(ctx, err, "Failed to write SBOM")
				return err
			}
		}
		if r.verbose {
			fmt.Printf("\n-------------------- 📜 SBOM Content --------------------\n")
			fmt.Printf("📂 Filename: %s\n", doc.Filename)
			fmt.Printf("📦 Format: %s | SpecVersion: %s\n\n", doc.Format, doc.SpecVersion)
			fmt.Println(string(doc.Content))
			fmt.Println("------------------------------------------------------")
		}
		sbomCount++
		fmt.Printf(" - 📁 Repo: %s | Format: %s | SpecVersion: %s | Filename: %s \n",
			sbom.Namespace, doc.Format, doc.SpecVersion, doc.Filename)
	}
	fmt.Printf("📊 Total SBOMs are: %d\n", sbomCount)
	logger.LogDebug(ctx, "Dry-run completed", "total_sboms", sbomCount)
	return nil
}
