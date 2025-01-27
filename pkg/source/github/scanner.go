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

package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/logger"
)

// SBOMAsset represents a found SBOM file
type SBOMAsset struct {
	Release     string
	Name        string
	DownloadURL string
	Size        int
}

// SBOMScanner scans GitHub releases for SBOM files
type SBOMScanner struct {
	client *Client
}

// NewScanner creates a new SBOM scanner
func NewScanner() *SBOMScanner {
	return &SBOMScanner{
		client: NewClient(),
	}
}

// FindSBOMs scans a repository's releases for SBOM files
func (s *SBOMScanner) FindSBOMs(ctx context.Context, url, version string) ([]SBOMAsset, error) {
	owner, repo, err := ParseGitHubURL(url)
	if err != nil {
		return nil, fmt.Errorf("parsing GitHub URL: %w", err)
	}

	logger.LogInfo(ctx, "Parsed github URL values", "url", url, "owner ", owner, "repo", repo)

	releases, err := s.client.GetReleases(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("fetching releases: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found in repository")
	}

	// Get latest release (first in the list as GitHub returns them in descending order)
	// latestRelease := releases[0]

	var targetRelease *Release
	if version != "" {
		// Find the release with the specified version
		for _, release := range releases {
			if release.TagName == version {
				targetRelease = &release
				break
			}
		}
		if targetRelease == nil {
			return nil, fmt.Errorf("release with version %s not found", version)
		}
	} else {
		// Default to the latest release
		targetRelease = &releases[0]
	}

	var sboms []SBOMAsset

	// Find all SBOM files in the latest release
	for _, asset := range targetRelease.Assets {
		if isSBOMFile(asset.Name) {
			sboms = append(sboms, SBOMAsset{
				Release:     targetRelease.TagName,
				Name:        asset.Name,
				DownloadURL: asset.DownloadURL,
				Size:        asset.Size,
			})
		}
	}
	if len(sboms) == 0 {
		return nil, fmt.Errorf("no SBOM files found in releases")
	}

	return sboms, nil
}

// isSBOMFile checks if a filename appears to be an SBOM
func isSBOMFile(name string) bool {
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
