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

package source

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// isSBOMFile checks if a filename appears to be an SBOM
func IsSBOMFile(name string) bool {
	name = strings.ToLower(name)

	// Common SBOM file patterns
	patterns := []string{
		".spdx.",
		".sbom",
		"bom.",
		"cyclonedx",
		"spdx",
		".cdx.",
	}

	// Common SBOM file extensions
	extensions := []string{
		".sbom",
		".json",
		".xml",
		".yaml",
		".yml",
		".txt", // for SPDX tag-value
	}

	// Check if name contains any SBOM pattern
	hasPattern := false
	for _, pattern := range patterns {
		if strings.Contains(name, pattern) {
			hasPattern = true
			break
		}
	}

	// Check if name has a valid extension
	hasExt := false
	for _, ext := range extensions {
		if strings.HasSuffix(name, ext) {
			hasExt = true
			break
		}
	}

	return hasPattern && hasExt
}

// ParseGitHubURL parses a GitHub URL into owner and repository
func ParseGitHubURL(url string) (owner, repo string, err error) {
	// Remove protocol and domain
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "github.com/")

	// Split remaining path
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub URL format: %s", url)
	}

	return parts[0], parts[1], nil
}

// decodeBase64 decodes base64 encoded SBOM data
func DecodeBase64(encoded string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 SBOM: %w", err)
	}
	return string(decodedBytes), nil
}

func sanitizeRepoName(repoURL string) string {
	repoParts := strings.Split(repoURL, "/")
	if len(repoParts) < 2 {
		return "unknown"
	}
	return repoParts[len(repoParts)-1] // Extracts "cosign" from URL
}
