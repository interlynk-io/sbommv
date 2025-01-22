package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// convertToJSON converts SBOM content to JSON format and saves with .json extension
func ConvertToJSON(inputPath, outputDir string) (string, error) {
	// Read input file
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	// Unmarshal to ensure valid JSON or convert to JSON
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		// If not valid JSON, use jq for conversion
		output, err := executeJQConversion(inputPath)
		if err != nil {
			return "", fmt.Errorf("converting to JSON using jq: %w", err)
		}
		data = output
	}

	// Create output filename with .json extension
	outputPath := filepath.Join(outputDir, filepath.Base(inputPath)+".json")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	// Marshal to JSON with indentation
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonBytes, 0o644); err != nil {
		return "", fmt.Errorf("writing output file: %w", err)
	}

	return outputPath, nil
}

// executeJQConversion uses jq to convert the input file to JSON
func executeJQConversion(inputPath string) (interface{}, error) {
	cmd := exec.Command("jq", ".", inputPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, fmt.Errorf("parsing jq output: %w", err)
	}

	return data, nil
}
