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
	"os"
	"path/filepath"
	"sync"

	"github.com/interlynk-io/sbommv/pkg/logger"
)

type downloadWork struct {
	sbom   SBOMAsset
	output string
}

const githubSBOMEndpoint = "repos/%s/%s/dependency-graph/sbom"

// GitHubSBOMResponse holds the JSON structure returned by GitHub API
type GitHubSBOMResponse struct {
	SBOM json.RawMessage `json:"sbom"` // Extract SBOM as raw JSON
}

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

	if len(sbomAssets) == 0 {
		return nil, fmt.Errorf("no SBOMs found in repository")
	}

	logger.LogDebug(ctx, "Total SBOMs found in the repository", "version", c.Version, "total sboms", len(sbomAssets))

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

// GetSBOMs downloads and saves all SBOM files found in the repository
func (c *Client) GetSBOMs(ctx context.Context, outputDir string) (VersionedSBOMs, error) {
	// Find SBOMs in releases
	sboms, err := c.FindSBOMs(ctx)
	if err != nil {
		return nil, fmt.Errorf("finding SBOMs: %w", err)
	}
	if len(sboms) == 0 {
		return nil, fmt.Errorf("no SBOMs found in repository")
	}

	logger.LogDebug(ctx, "Total SBOMs found in the repository", "version", c.Version, "total sboms", len(sboms))

	// Create output directory if needed
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return nil, fmt.Errorf("creating output directory: %w", err)
		}
	}

	return c.downloadSBOMs(ctx, sboms, outputDir)
}

// downloadSBOMs handles the concurrent downloading of multiple SBOM files
func (c *Client) downloadSBOMs(ctx context.Context, sboms []SBOMAsset, outputDir string) (VersionedSBOMs, error) {
	var (
		wg             sync.WaitGroup                        // Coordinates all goroutines
		mu             sync.Mutex                            // Protects shared resources
		versionedSBOMs = make(VersionedSBOMs)                // Stores results
		errors         []error                               // Collects errors
		maxConcurrency = 3                                   // Maximum parallel downloads
		semaphore      = make(chan struct{}, maxConcurrency) // Controls concurrency
	)

	// Process each SBOM
	for _, sbom := range sboms {
		// Context cancellation check
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		wg.Add(1)
		go func(sbom SBOMAsset) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			// Download and save the SBOM
			outputPath := ""
			if outputDir != "" {
				outputPath = filepath.Join(outputDir, sbom.Name)
			}

			err := c.downloadSingleSBOM(ctx, sbom, outputPath)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("downloading %s: %w", sbom.Name, err))
				mu.Unlock()
				return
			}

			if outputPath != "" {
				mu.Lock()
				versionedSBOMs[sbom.Release] = append(versionedSBOMs[sbom.Release], outputPath)
				mu.Unlock()
				logger.LogDebug(ctx, "SBOM file", "name", sbom.Name, "saved to", outputPath)
			}
		}(sbom)
	}

	wg.Wait()

	if len(errors) > 0 {
		return nil, fmt.Errorf("encountered %d download errors: %v", len(errors), errors[0])
	}

	return versionedSBOMs, nil
}

// downloadSingleSBOM downloads and saves a single SBOM file
func (c *Client) downloadSingleSBOM(ctx context.Context, sbom SBOMAsset, outputPath string) error {
	reader, err := c.DownloadAsset(ctx, sbom.DownloadURL)
	if err != nil {
		return fmt.Errorf("downloading asset: %w", err)
	}
	defer reader.Close()

	var output io.Writer
	if outputPath == "" {
		// Write to stdout with header
		fmt.Printf("\n=== SBOM: %s ===\n", sbom.Name)
		output = os.Stdout
	} else {
		// Create and write to file
		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer file.Close()
		output = file
	}

	if _, err := io.Copy(output, reader); err != nil {
		return fmt.Errorf("writing SBOM: %w", err)
	}

	return nil
}

func (c *Client) FetchSBOMFromAPI(ctx context.Context) ([]byte, error) {
	owner, repo, err := ParseGitHubURL(c.RepoURL)
	if err != nil {
		return nil, fmt.Errorf("parsing GitHub URL: %w", err)
	}

	// Construct the API URL for the SBOM export
	url := fmt.Sprintf("%s/%s", c.BaseURL, fmt.Sprintf(githubSBOMEndpoint, owner, repo))
	logger.LogDebug(ctx, "Fetching SBOM via GitHub API", "url", url)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication only if a token is provided
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	// Set required headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Perform the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SBOM: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Extract SBOM field from response
	var response GitHubSBOMResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing SBOM response: %w", err)
	}

	// Ensure SBOM field is not empty
	if len(response.SBOM) == 0 {
		return nil, fmt.Errorf("empty SBOM data received from GitHub API")
	}

	logger.LogDebug(ctx, "Fetched SBOM successfully", "repository", c.RepoURL)

	// Return the raw SBOM JSON as bytes
	return response.SBOM, nil
}
