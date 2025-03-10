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

package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/protobom/protobom/pkg/formats"
	"github.com/protobom/protobom/pkg/reader"
	"github.com/protobom/protobom/pkg/sbom"
	"github.com/protobom/protobom/pkg/writer"
	"github.com/sirupsen/logrus"
)

type FormatSpec string

const (
	FormatCycloneDX FormatSpec = "cyclonedx"
	FormatSPDX      FormatSpec = "spdx"
)

// bufferWriteCloser wraps *bytes.Buffer to implement io.WriteCloser
type bufferWriteCloser struct {
	*bytes.Buffer
}

func (b *bufferWriteCloser) Close() error {
	return nil // No-op for in-memory buffer
}

func newBufferWriteCloser() *bufferWriteCloser {
	return &bufferWriteCloser{&bytes.Buffer{}}
}

// ConvertSBOM converts SBOM data to the target format using protobom
func ConvertSBOM(ctx tcontext.TransferMetadata, sbomData []byte, targetFormat FormatSpec) ([]byte, error) {
	// Detect source format
	logger.LogDebug(ctx.Context, "Iniatializing for SBOM conversion from spdx to cdx")

	// Mute protobom warnings of data lost
	originalLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.ErrorLevel)   // Only ERROR and above from protobom
	defer logrus.SetLevel(originalLevel) // Restore after

	// Preprocess SPDX SBOM to remove licenseInfoInFiles: ["NOASSERTION"] from files section
	cleanedData, err := preprocessSBOM(sbomData)
	if err != nil {
		return nil, fmt.Errorf("preprocessing SBOM: %w", err)
	}

	sourceFormat, err := detectFormat(cleanedData)
	if err != nil {
		return nil, fmt.Errorf("detecting source format: %w", err)
	}

	if sourceFormat == targetFormat {
		logger.LogDebug(ctx.Context, "No conversion needed", "format", sourceFormat)
		return sbomData, nil
	}

	doc, err := parseSBOM(cleanedData)
	if err != nil {
		return sbomData, err
	}

	// Serialize to CycloneDX format from SPDX format
	if targetFormat == FormatCycloneDX {
		enrichedDoc := enrichCycloneDXSBOM(doc)
		return serializeToCycloneDX(ctx, enrichedDoc)
	}

	return nil, fmt.Errorf("unsupported conversion to %s", targetFormat)
}

// detectFormat identifies the SBOM format from raw data
func detectFormat(data []byte) (FormatSpec, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("unmarshaling SBOM: %w", err)
	}

	if _, ok := raw["bomFormat"]; ok {
		return FormatCycloneDX, nil
	}
	if _, ok := raw["spdxVersion"]; ok {
		return FormatSPDX, nil
	}
	return "", fmt.Errorf("unknown SBOM format")
}

// isValidCycloneDXSerialNumber checks if the serial number matches the required UUID pattern
func isValidCycloneDXSerialNumber(serial string) bool {
	pattern := `^urn:uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	matched, _ := regexp.MatchString(pattern, serial)
	return matched
}

// protobom on converting from spdx to cdx, puts wp invalid serial number and version.
// enrichCycloneDXSBOM() function will update the correct serial number and version
func enrichCycloneDXSBOM(doc *sbom.Document) *sbom.Document {
	if doc.Metadata.Id == "" || !isValidCycloneDXSerialNumber(doc.Metadata.Id) {
		doc.Metadata.Id = "urn:uuid:" + uuid.New().String()
	}

	doc.Metadata.Version = "1" // Default version

	return doc
}

func parseSBOM(sbomData []byte) (*sbom.Document, error) {
	// Parse the input SBOM
	r := reader.New()

	// remove licenseInfoInFiles

	// parse a sbom document from a sbom data using protobom
	doc, err := r.ParseStream(bytes.NewReader(sbomData))
	if err != nil {
		return nil, fmt.Errorf("parsing SBOM: %w", err)
	}
	return doc, nil
}

// preprocessSBOM for the time it does the job of protobom of removing NOASSERTION license field.
// as these changes is added in the protobom, will remove this function.
func preprocessSBOM(data []byte) ([]byte, error) {
	var sbom map[string]interface{}
	if err := json.Unmarshal(data, &sbom); err != nil {
		return nil, fmt.Errorf("unmarshaling SBOM: %w", err)
	}
	if files, ok := sbom["files"].([]interface{}); ok {
		for _, file := range files {
			if f, ok := file.(map[string]interface{}); ok {
				if licInfo, ok := f["licenseInfoInFiles"].([]interface{}); ok {
					if len(licInfo) == 1 && licInfo[0] == "NOASSERTION" {
						delete(f, "licenseInfoInFiles") // Remove if only "NOASSERTION"
					}
				}
			}
		}
	}
	return json.Marshal(sbom)
}

func serializeToCycloneDX(ctx tcontext.TransferMetadata, doc *sbom.Document) ([]byte, error) {
	w := writer.New()
	buf := newBufferWriteCloser()

	if err := w.WriteStreamWithOptions(doc, buf, &writer.Options{Format: formats.CDX15JSON}); err != nil {
		return nil, fmt.Errorf("writing CycloneDX: %w", err)
	}
	logger.LogDebug(ctx.Context, "SPDX SBOM converted to CycloneDX SBOM")

	return buf.Bytes(), nil
}
