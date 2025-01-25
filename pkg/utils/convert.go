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

package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConvertToJSON converts SBOM content to JSON format if needed and saves with .json extension.
// It only converts SPDX files and removes the original file after conversion.
// For JSON and YAML files, it preserves them as-is.
func ConvertToJSON(inputPath, outputDir string) (string, error) {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(inputPath))

	// Skip JSON and YAML files
	if ext == ".json" || ext == ".yaml" || ext == ".yml" {
		// For these formats, just copy the file to output directory
		return inputPath, nil
	}

	// Only convert SPDX files
	if ext != ".spdx" {
		// For non-SPDX files, just copy to output directory
		return copyFile(inputPath, outputDir)
	}

	// Try to convert using jq
	output, err := executeJQConversion(inputPath)
	if err != nil {
		return "", fmt.Errorf("converting to JSON using jq: %w", err)
	}

	// Create output filename with .json extension
	baseName := strings.TrimSuffix(filepath.Base(inputPath), ext)
	outputPath := filepath.Join(outputDir, baseName+".spdx.json")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	// Marshal to JSON with indentation
	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonBytes, 0o644); err != nil {
		return "", fmt.Errorf("writing output file: %w", err)
	}

	// Remove the original file after successful conversion
	if err := os.Remove(inputPath); err != nil {
		return "", fmt.Errorf("removing original file: %w", err)
	}

	return outputPath, nil
}

// executeJQConversion uses jq to convert the input file to JSON
func executeJQConversion(inputPath string) (interface{}, error) {
	cmd := exec.Command("jq", ".", inputPath)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("jq conversion failed: %s", exitErr.Stderr)
		}
		return nil, fmt.Errorf("running jq: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, fmt.Errorf("parsing jq output: %w", err)
	}

	return data, nil
}

// copyFile copies a file to the output directory preserving its extension
func copyFile(inputPath, outputDir string) (string, error) {
	// Read input file
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	// Create output path preserving original filename
	outputPath := filepath.Join(outputDir, filepath.Base(inputPath))

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	// Write to output file
	if err := os.WriteFile(outputPath, content, 0o644); err != nil {
		return "", fmt.Errorf("writing output file: %w", err)
	}

	return outputPath, nil
}
