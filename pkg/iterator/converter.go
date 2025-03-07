package iterator

import (
	"context"
	"io"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/converter"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

// ConvertingSBOMIterator wraps an SBOMIterator and converts SBOMs lazily
type ConvertingSBOMIterator struct {
	original          SBOMIterator
	transferCtx       *tcontext.TransferMetadata
	totalMinifiedSBOM int
	totalSBOM         int
}

// NewConvertingSBOMIterator creates a new ConvertingSBOMIterator
func NewConvertingSBOMIterator(original SBOMIterator, transferCtx *tcontext.TransferMetadata) SBOMIterator {
	return &ConvertingSBOMIterator{
		original:    original,
		transferCtx: transferCtx,
	}
}

// Next fetches the next SBOM, converts it, and returns it
func (it *ConvertingSBOMIterator) Next(ctx context.Context) (*SBOM, error) {
	sbom, err := it.original.Next(ctx)
	if err != nil {
		if err == io.EOF {
			logger.LogDebug(it.transferCtx.Context, "SBOM conversion summary", "total SBOMs", it.totalSBOM, "total minified JSON SBOMs converted to pretty JSON", it.totalMinifiedSBOM)
		}
		return nil, err
	}

	// Convert SBOM to CycloneDX
	convertedData, err := converter.ConvertSBOM(*it.transferCtx, sbom.Data, converter.FormatCycloneDX)
	if err != nil {
		logger.LogInfo(it.transferCtx.Context, "Failed to convert SBOM to CycloneDX", "file", sbom.Path, "error", err)
		return it.Next(ctx) // Skip to the next SBOM
	}

	// Handle minified JSON conversion
	sbom.Data, it.totalMinifiedSBOM, err = convertMinifiedJSON(it.transferCtx, convertedData, it.totalMinifiedSBOM)
	if err != nil {
		logger.LogInfo(it.transferCtx.Context, "Failed to handle minified JSON", "file", sbom.Path, "error", err)
		return it.Next(ctx) // Skip to the next SBOM
	}

	// Update SBOM data with converted content
	sbom.Data = convertedData

	// Update path if it contains "spdx"
	if strings.Contains(sbom.Path, "spdx") {
		sbom.Path = strings.Replace(sbom.Path, "spdx", "spdxtocdx", 1)
	}

	it.totalSBOM++
	return sbom, nil
}
