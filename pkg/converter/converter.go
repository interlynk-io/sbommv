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
	"fmt"
	"regexp"
	"time"

	"github.com/interlynk-io/sbommv/pkg/logger"
	sbomd "github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/protobom/protobom/pkg/formats"
	"github.com/protobom/protobom/pkg/reader"
	"github.com/protobom/protobom/pkg/sbom"
	"github.com/protobom/protobom/pkg/writer"
	"github.com/sirupsen/logrus"
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

// ConvertSBOM converts SBOM from SPDX to CDX format using protobom
func ConvertSBOM(ctx tcontext.TransferMetadata, sbomData []byte, targetFormat sbomd.FormatSpec) ([]byte, error) {
	logger.LogDebug(ctx.Context, "Iniatializing for SBOM conversion from SPDX to CDX")

	originalLevel := logrus.GetLevel()   // Mute protobom warnings of data lost
	logrus.SetLevel(logrus.ErrorLevel)   // Only ERROR and above from protobom
	defer logrus.SetLevel(originalLevel) // Restore after

	spec, version, err := sbomd.DetectSBOMSpecAndVersion(sbomData)
	if err != nil {
		return nil, fmt.Errorf("detecting SPDX SBOM: %w", err)
	}

	if spec == targetFormat {
		logger.LogDebug(ctx.Context, "No conversion needed", "format", spec, "spec_version", version)
		return sbomData, nil
	}

	if spec != sbomd.FormatSpecSPDX {
		return nil, fmt.Errorf("conversion layer is provided with SBOM other than SPDX, therefore no conversion will take place")
	}

	logger.LogDebug(ctx.Context, "Detected SPDX SBOM", "version", version)

	var spdx23SbomData []byte
	var doc *sbom.Document

	switch sbomd.FormatSpecVersion(version) {

	case sbomd.FormatSpecVersionSPDXV2_1:
		return nil, fmt.Errorf("unsupported conversion from SPDX 2.1 to %s", targetFormat)

	case sbomd.FormatSpecVersionSPDXV2_2:
		spdx23SbomData, err = ConvertSPDX22ToSPDX23(ctx, sbomData)
		if err != nil {
			return nil, fmt.Errorf("converting SPDX 2.2 to 2.3: %w", err)
		}

	case sbomd.FormatSpecVersionSPDXV2_3:
		spdx23SbomData = sbomData

	default:
		return nil, fmt.Errorf("unsupported SPDX version: %s", version)
	}

	// Parse the converted 2.3 SBOM with Protobom
	doc, err = parseSBOM(spdx23SbomData)
	if err != nil {
		return nil, fmt.Errorf("parsing converted SPDX 2.3: %w", err)
	}

	logger.LogDebug(ctx.Context, "Converting SBOM", "source", spec, "source version", version, "target", targetFormat)

	// Serialize to CycloneDX format from SPDX:2.3
	if targetFormat == sbomd.FormatSpecCycloneDX {
		// enrichedDoc := enrichCycloneDXSBOM(doc)
		return serializeToCycloneDX(ctx, doc)
	}

	return nil, fmt.Errorf("unsupported conversion to %s", targetFormat)
}

// isValidCycloneDXSerialNumber checks if the serial number matches the required UUID pattern
func isValidCycloneDXSerialNumber(serial string) bool {
	pattern := `^urn:uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	matched, _ := regexp.MatchString(pattern, serial)
	return matched
}

// // protobom on converting from spdx to cdx, puts wp invalid serial number and version.
// // enrichCycloneDXSBOM() function will update the correct serial number and version
// func enrichCycloneDXSBOM(doc *sbom.Document) *sbom.Document {
// 	if doc.Metadata.Id == "" || !isValidCycloneDXSerialNumber(doc.Metadata.Id) {
// 		doc.Metadata.Id = "urn:uuid:" + uuid.New().String()
// 	}

// 	doc.Metadata.Version = "1" // Default version

// 	return doc
// }

// parseSBOM parse the SBOM using Protobom
func parseSBOM(sbomData []byte) (*sbom.Document, error) {
	r := reader.New()

	// parse a sbom document from a sbom data using protobom
	doc, err := r.ParseStream(bytes.NewReader(sbomData))
	if err != nil {
		return nil, fmt.Errorf("protobom parsing SBOM: %w", err)
	}
	return doc, nil
}

// // preprocessSBOM for the time it does the job of protobom of removing NOASSERTION license field.
// // as these changes is added in the protobom, will remove this function.
// func preprocessSBOM(data []byte) ([]byte, error) {
// 	var sbom map[string]interface{}
// 	if err := json.Unmarshal(data, &sbom); err != nil {
// 		return nil, fmt.Errorf("unmarshaling SBOM: %w", err)
// 	}
// 	if files, ok := sbom["files"].([]interface{}); ok {
// 		for _, file := range files {
// 			if f, ok := file.(map[string]interface{}); ok {
// 				if licInfo, ok := f["licenseInfoInFiles"].([]interface{}); ok {
// 					if len(licInfo) == 1 && licInfo[0] == "NOASSERTION" {
// 						delete(f, "licenseInfoInFiles") // Remove if only "NOASSERTION"
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return json.Marshal(sbom)
// }

func serializeToCycloneDX(ctx tcontext.TransferMetadata, doc *sbom.Document) ([]byte, error) {
	logger.LogDebug(ctx.Context, "Initializing protobom serialization of SBOM from SPDX to CycloneDX")
	w := writer.New()
	buf := &bytes.Buffer{}

	// channel to receive result or error
	resultChan := make(chan struct {
		data []byte
		err  error
	}, 1)

	go func(buffer *bytes.Buffer) {
		logger.LogDebug(ctx.Context, "Starting WriteStreamWithOptions", "nodeCount", len(doc.NodeList.Nodes))
		err := w.WriteStreamWithOptions(doc, buffer, &writer.Options{Format: formats.CDX15JSON})
		data := buffer.Bytes()
		resultChan <- struct {
			data []byte
			err  error
		}{data, err}
	}(buf)

	// wait for result with timeout
	select {
	case res := <-resultChan:
		if res.err != nil {
			return nil, fmt.Errorf("writing protobom serialized CycloneDX: %w", res.err)
		}
		logger.LogDebug(ctx.Context, "Finished WriteStreamWithOptions")
		data := res.data
		if len(data) == 0 {
			return nil, fmt.Errorf("empty protobom serialized CycloneDX SBOM")
		}
		logger.LogDebug(ctx.Context, "Successfully protobom serialization of SBOM from SPDX to CycloneDX")
		return data, nil
	case <-time.After(30 * time.Second): // 30 seconds timeout
		return nil, fmt.Errorf("serialization timed out after 30 seconds")
	}
}

// func convertMinifiedJSON(data []byte) ([]byte, error) {
// 	// Try parsing the JSON
// 	var jsonData map[string]interface{}
// 	err := json.Unmarshal(data, &jsonData)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Pretty-print the JSON
// 	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Check if original file is minified by comparing bytes
// 	if bytes.Equal(data, prettyJSON) {
// 		return prettyJSON, nil // Already formatted
// 	}

// 	return prettyJSON, nil // Minified JSON detected
// }
