package iterator

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/converter"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

// ConvertingSBOMIterator wraps an SBOMIterator and converts SBOMs lazily
type ConvertingSBOMIterator struct {
	original          SBOMIterator
	totalMinifiedSBOM int
	totalSBOM         int
}

// NewConvertingSBOMIterator creates a new ConvertingSBOMIterator
func NewConvertingSBOMIterator(original SBOMIterator) SBOMIterator {
	return &ConvertingSBOMIterator{
		original: original,
	}
}

// Next fetches the next SBOM, converts it, and returns it
func (it *ConvertingSBOMIterator) Next(transferCtx tcontext.TransferMetadata) (*SBOM, error) {
	logger.LogDebug(transferCtx.Context, "Next iteration for ConvertingSBOMIterator")
	sbom, err := it.original.Next(transferCtx)
	if err != nil {
		if err == io.EOF {
			logger.LogDebug(transferCtx.Context, "SBOM conversion summary", "total SBOMs", it.totalSBOM, "total minified JSON SBOMs converted to pretty JSON", it.totalMinifiedSBOM)
		}
		return nil, err
	}

	// Convert SBOM to CycloneDX
	convertedData, err := converter.ConvertSBOM(transferCtx, sbom.Data, converter.FormatCycloneDX)
	if err != nil {
		logger.LogInfo(transferCtx.Context, "Failed to convert SBOM to CycloneDX", "file", sbom.Path, "error", err)
		return it.Next(transferCtx) // Skip to the next SBOM
	}

	// Handle minified JSON conversion
	sbom.Data, it.totalMinifiedSBOM, err = convertMinifiedJSON(transferCtx, convertedData, it.totalMinifiedSBOM)
	if err != nil {
		logger.LogInfo(transferCtx.Context, "Failed to handle minified JSON", "file", sbom.Path, "error", err)
		return it.Next(transferCtx) // Skip to the next SBOM
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

func isMinifiedJSON(data []byte) (bool, []byte, []byte, error) {
	// Try parsing the JSON
	var jsonData map[string]interface{}
	err := json.Unmarshal(data, &jsonData)
	if err != nil {
		return false, nil, nil, err
	}

	// Pretty-print the JSON
	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return false, nil, nil, err
	}

	// Check if original file is minified by comparing bytes
	if bytes.Equal(data, prettyJSON) {
		return false, data, prettyJSON, nil // Already formatted
	}

	return true, data, prettyJSON, nil // Minified JSON detected
}

func convertMinifiedJSON(transferCtx tcontext.TransferMetadata, data []byte, totalMinifiedSBOM int) ([]byte, int, error) {
	minified, original, formatted, err := isMinifiedJSON(data)
	if err != nil {
		logger.LogError(transferCtx.Context, err, "Error while isMinifiedJSON")
		return original, totalMinifiedSBOM, nil
	}

	if minified {
		totalMinifiedSBOM++
		return formatted, totalMinifiedSBOM, nil
	}
	return original, totalMinifiedSBOM, nil
}
