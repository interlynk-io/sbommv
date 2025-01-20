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
func (s *SBOMScanner) FindSBOMs(ctx context.Context, url string) ([]SBOMAsset, error) {
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
	latestRelease := releases[0]
	var sboms []SBOMAsset

	// Find all SBOM files in the latest release
	for _, asset := range latestRelease.Assets {
		if isSBOMFile(asset.Name) {
			sboms = append(sboms, SBOMAsset{
				Release:     latestRelease.TagName,
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
		"sbom.",
		"bom.",
		"cyclonedx",
		"spdx",
		".cdx.",
	}

	// Common SBOM file extensions
	extensions := []string{
		"sbom",
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
