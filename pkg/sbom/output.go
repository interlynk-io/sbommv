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

package sbom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// PrettyPrintSBOM prints an SBOM in formatted JSON
func PrettyPrintSBOM(w io.Writer, Content []byte) error {
	// First try to unmarshal into a generic interface{} to get the structure
	var data interface{}
	if err := json.Unmarshal(Content, &data); err != nil {
		return fmt.Errorf("failed to parse SBOM content: %w", err)
	}

	// Create a buffer for pretty printing
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to format SBOM: %w", err)
	}

	// Write the formatted JSON
	_, err := w.Write(buf.Bytes())
	return err
}
