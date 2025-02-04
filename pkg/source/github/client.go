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
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/interlynk-io/sbommv/pkg/logger"
)

// Asset represents a GitHub release asset (e.g., SBOM files)
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int    `json:"size"`
}

// Release represents a GitHub release containing assets
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// SBOMAsset represents an SBOM file found in a GitHub release
type SBOMAsset struct {
	Release     string
	Name        string
	DownloadURL string
	Size        int
}

// VersionedSBOMs maps versions to their respective SBOMs in that version
type VersionedSBOMs map[string][]string

// Client interacts with the GitHub API
type Client struct {
	httpClient *http.Client
	BaseURL    string
	RepoURL    string
	Version    string
	Method     string
	token      string
}

// NewClient initializes a GitHub client
func NewClient(repoURL, version, method string) *Client {
	return &Client{
		httpClient: &http.Client{},
		BaseURL:    "https://api.github.com",
		RepoURL:    repoURL,
		Version:    version,
		Method:     method,
	}
}

// FindSBOMs gets all releases assets from github release page
// filter out the particular provided release asset and
// extract SBOMs from that
func (c *Client) FindSBOMs(ctx context.Context) ([]SBOMAsset, error) {
	owner, repo, err := ParseGitHubURL(c.RepoURL)
	if err != nil {
		return nil, fmt.Errorf("parsing GitHub URL: %w", err)
	}

	logger.LogDebug(ctx, "Fetching GitHub releases", "repo_url", c.RepoURL, "owner", owner, "repo", repo)

	releases, err := c.GetReleases(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("error retrieving releases: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found for repository %s/%s", owner, repo)
	}

	// Select target releases (single version or all versions)
	targetReleases := c.filterReleases(releases, c.Version)
	if len(targetReleases) == 0 {
		return nil, fmt.Errorf("no matching release found for version: %s", c.Version)
	}

	// Extract SBOM assets from target release
	sboms := c.extractSBOMs(targetReleases)

	if len(sboms) == 0 {
		return nil, fmt.Errorf("no SBOM files found in releases for repository %s/%s", owner, repo)
	}
	logger.LogDebug(ctx, "Successfully retrieved SBOMs", "total_sboms", len(sboms), "repo_url", c.RepoURL)

	return sboms, nil
}

// filterReleases filters releases based on version input
func (c *Client) filterReleases(releases []Release, version string) []Release {
	if version == "" {
		// Return all releases
		return releases
	}
	if version == "latest" {
		// Return latest release
		return []Release{releases[0]}
	}

	// Return the matching release version
	for _, release := range releases {
		if release.TagName == version {
			return []Release{release}
		}
	}
	return nil
}

// extractSBOMs extracts SBOM assets from releases
func (c *Client) extractSBOMs(releases []Release) []SBOMAsset {
	var sboms []SBOMAsset
	for _, release := range releases {
		for _, asset := range release.Assets {
			if isSBOMFile(asset.Name) {
				sboms = append(sboms, SBOMAsset{
					Release:     release.TagName,
					Name:        asset.Name,
					DownloadURL: asset.DownloadURL,
					Size:        asset.Size,
				})
			}
		}
	}
	return sboms
}

// GetReleases fetches all releases for a repository
func (c *Client) GetReleases(ctx context.Context, owner, repo string) ([]Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases", c.BaseURL, owner, repo)
	// logger.LogDebug(ctx, "Fetching GitHub Releases", "url", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	// logger.LogDebug(ctx, "Response ", "body", resp.Body)

	// Read response body for error reporting
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body failed: %w", err)
	}

	// Handle different status codes with specific error messages
	switch resp.StatusCode {

	case http.StatusOK:
		var releases []Release
		if err := json.Unmarshal(body, &releases); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}
		return releases, nil

	case http.StatusNotFound:
		return nil, fmt.Errorf("repository %s/%s not found or no releases available", owner, repo)

	case http.StatusUnauthorized:
		return nil, fmt.Errorf("authentication required or invalid token for %s/%s", owner, repo)

	case http.StatusForbidden:
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return nil, fmt.Errorf("GitHub API rate limit exceeded")
		}
		return nil, fmt.Errorf("access forbidden to %s/%s", owner, repo)

	default:
		// Try to parse GitHub error message
		var ghErr struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &ghErr); err == nil && ghErr.Message != "" {
			return nil, fmt.Errorf("GitHub API error: %s", ghErr.Message)
		}
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
}

// DownloadAsset downloads a release asset from download url of SBOM
func (c *Client) DownloadAsset(ctx context.Context, downloadURL string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request execution failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// FetchSBOMsFromReleases fetches and downloads SBOMs from GitHub releases
func (c *Client) FetchSBOMsFromReleases(ctx context.Context) (map[string][]byte, error) {
	logger.LogDebug(ctx, "Fetching SBOMs from GitHub Releases", "repo", c.RepoURL)

	// Step 1: Get All Releases
	sbomAssets, err := c.FindSBOMs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error finding SBOMs in releases: %w", err)
	}

	// Step 2: Download Each SBOM
	versionedSBOMs := make(map[string][]byte)
	for _, asset := range sbomAssets {
		sbomData, err := c.DownloadSBOM(ctx, asset)
		if err != nil {
			logger.LogError(ctx, err, "Failed to download SBOM", "file", asset.Name)
			continue
		}

		versionedSBOMs[asset.Release] = sbomData
		logger.LogDebug(ctx, "Downloaded SBOM successfully", "version", asset.Release, "file", asset.Name)
	}

	// Step 3: Return All Downloaded SBOMs
	return versionedSBOMs, nil
}

// DownloadSBOM fetches an SBOM from its download URL
func (c *Client) DownloadSBOM(ctx context.Context, asset SBOMAsset) ([]byte, error) {
	logger.LogDebug(ctx, "Downloading SBOM", "url", asset.DownloadURL)

	resp, err := c.httpClient.Get(asset.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SBOM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	sbomData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read SBOM data: %w", err)
	}

	return sbomData, nil
}
