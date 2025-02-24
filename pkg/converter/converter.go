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
	"github.com/protobom/protobom/pkg/writer"
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

	sourceFormat, err := detectFormat(sbomData)
	if err != nil {
		return nil, fmt.Errorf("detecting source format: %w", err)
	}
	if sourceFormat == targetFormat {
		logger.LogDebug(ctx.Context, "No conversion needed", "format", sourceFormat)
		return sbomData, nil
	}

	// Parse the input SBOM
	r := reader.New()

	doc, err := r.ParseStream(bytes.NewReader(sbomData))
	if err != nil {
		return nil, fmt.Errorf("parsing SBOM: %w", err)
	}

	// Serialize to target format (CycloneDX for DTrack)
	if targetFormat == FormatCycloneDX {
		// Set serialNumber if missing or invalid
		if doc.Metadata.Id == "" || !isValidCycloneDXSerialNumber(doc.Metadata.Id) {
			doc.Metadata.Id = "urn:uuid:" + uuid.New().String()
		}
		doc.Metadata.Version = "1" // Default version

		w := writer.New()
		buf := newBufferWriteCloser()
		if err := w.WriteStreamWithOptions(doc, buf, &writer.Options{Format: formats.CDX16JSON}); err != nil {
			return nil, fmt.Errorf("writing CycloneDX: %w", err)
		}
		logger.LogDebug(ctx.Context, "Converted SBOM", "from", sourceFormat, "to", targetFormat)
		return buf.Bytes(), nil
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
