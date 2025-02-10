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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
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
// type VersionedSBOMs map[string][]string
type VersionedSBOMs map[string][][]byte

// Client interacts with the GitHub API
type Client struct {
	httpClient   *http.Client
	BaseURL      string
	RepoURL      string
	Organization string
	Owner        string
	Repo         string
	Version      string
	Method       string
	token        string
}

// NewClient initializes a GitHub client
func NewClient(g *GitHubAdapter) *Client {
	return &Client{
		httpClient: &http.Client{},
		BaseURL:    "https://api.github.com",
		RepoURL:    g.URL,
		Version:    g.Version,
		Method:     g.Method,
		Owner:      g.Owner,
		Repo:       g.Repo,
	}
}

// FindSBOMs gets all releases assets from github release page
// filter out the particular provided release asset and
// extract SBOMs from that
func (c *Client) FindSBOMs(ctx *tcontext.TransferMetadata) ([]SBOMAsset, error) {
	logger.LogDebug(ctx.Context, "Fetching GitHub releases", "repo_url", c.RepoURL, "owner", c.Owner, "repo", c.Repo)

	releases, err := c.GetReleases(ctx, c.Owner, c.Repo)
	if err != nil {
		return nil, fmt.Errorf("error retrieving releases: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found for repository %s/%s", c.Owner, c.Repo)
	}

	// Select target releases (single version or all versions)
	targetReleases := c.filterReleases(releases, c.Version)
	if len(targetReleases) == 0 {
		return nil, fmt.Errorf("no matching release found for version: %s", c.Version)
	}
	logger.LogDebug(ctx.Context, "Total number of Releases", "value", len(targetReleases))

	// Extract SBOM assets from target release
	sboms := c.extractSBOMs(targetReleases)

	if len(sboms) == 0 {
		return nil, fmt.Errorf("no SBOM files found in releases for repository %s/%s", c.Owner, c.Repo)
	}
	logger.LogDebug(ctx.Context, "Successfully retrieved SBOMs", "total_sboms", len(sboms), "repo_url", c.RepoURL)

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
func (c *Client) GetReleases(ctx *tcontext.TransferMetadata, owner, repo string) ([]Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases", c.BaseURL, owner, repo)
	logger.LogDebug(ctx.Context, "Constructed GitHub Releases", "url", url)

	req, err := http.NewRequestWithContext(ctx.Context, "GET", url, nil)
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
func (c *Client) DownloadAsset(ctx *tcontext.TransferMetadata, downloadURL string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx.Context, "GET", downloadURL, nil)
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

// DownloadSBOM fetches an SBOM from its download URL
func (c *Client) DownloadSBOM(ctx *tcontext.TransferMetadata, asset SBOMAsset) ([]byte, error) {
	logger.LogDebug(ctx.Context, "Downloading SBOM", "url", asset.DownloadURL)

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
func (c *Client) GetSBOMs(ctx *tcontext.TransferMetadata) (VersionedSBOMs, error) {
	// Find SBOMs in releases
	sboms, err := c.FindSBOMs(ctx)
	if err != nil {
		return nil, fmt.Errorf("finding SBOMs: %w", err)
	}
	if len(sboms) == 0 {
		return nil, fmt.Errorf("no SBOMs found in repository")
	}

	logger.LogDebug(ctx.Context, "Total SBOMs found in the repository", "version", c.Version, "total sboms", len(sboms))
	ctx.WithValue("total_sboms", len(sboms))

	return c.downloadSBOMs(ctx, sboms)
}

// downloadSBOMs handles the concurrent downloading of multiple SBOM files
func (c *Client) downloadSBOMs(ctx *tcontext.TransferMetadata, sboms []SBOMAsset) (VersionedSBOMs, error) {
	var (
		wg             sync.WaitGroup                        // Coordinates all goroutines
		mu             sync.Mutex                            // Protects shared resources
		versionedSBOMs = make(VersionedSBOMs)                // Stores results in memory
		errors         []error                               // Collects errors
		maxConcurrency = 3                                   // Maximum parallel downloads
		semaphore      = make(chan struct{}, maxConcurrency) // Controls concurrency
	)

	// Initialize progress bar
	// bar := progressbar.Default(int64(len(sboms)), "📥 Fetching SBOMs")

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

			// Download the SBOM and store it in memory
			sbomData, err := c.downloadSingleSBOM(ctx, sbom)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("downloading %s: %w", sbom.Name, err))
				mu.Unlock()
				return
			}

			// Store SBOM content in memory
			mu.Lock()
			versionedSBOMs[sbom.Release] = append(versionedSBOMs[sbom.Release], sbomData)
			mu.Unlock()

			logger.LogDebug(ctx.Context, "SBOM fetched and stored in memory", "name", sbom.Name)
			// _ = bar.Add(1) // Update progress bar
		}(sbom)
	}

	wg.Wait()
	// _ = bar.Finish() // Close progress bar on completion

	if len(errors) > 0 {
		return nil, fmt.Errorf("encountered %d download errors: %v", len(errors), errors[0])
	}

	return versionedSBOMs, nil
}

// downloadSingleSBOM downloads a single SBOM and stores it in memory
func (c *Client) downloadSingleSBOM(ctx *tcontext.TransferMetadata, sbom SBOMAsset) ([]byte, error) {
	reader, err := c.DownloadAsset(ctx, sbom.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("downloading asset: %w", err)
	}
	defer reader.Close()

	// Read SBOM content into memory
	sbomData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading SBOM content: %w", err)
	}

	logger.LogDebug(ctx.Context, "SBOM fetched successfully", "file", sbom.Name)
	return sbomData, nil
}

func (c *Client) FetchSBOMFromAPI(ctx *tcontext.TransferMetadata) ([]byte, error) {
	owner, repo, err := ParseGitHubURL(c.RepoURL)
	if err != nil {
		return nil, fmt.Errorf("parsing GitHub URL: %w", err)
	}

	// Construct the API URL for the SBOM export
	url := fmt.Sprintf("%s/%s", c.BaseURL, fmt.Sprintf(githubSBOMEndpoint, owner, repo))
	logger.LogDebug(ctx.Context, "Fetching SBOM via GitHub API", "url", url)

	// Create request
	req, err := http.NewRequestWithContext(ctx.Context, "GET", url, nil)
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

	logger.LogDebug(ctx.Context, "Fetched SBOM successfully", "repository", c.RepoURL)

	// Return the raw SBOM JSON as bytes
	return response.SBOM, nil
}

func (c *Client) updateRepo(repo string) {
	c.Repo = repo
	c.RepoURL = fmt.Sprintf("https://github.com/%s/%s", c.Owner, repo)
}

func (c *Client) GetAllRepositories(ctx *tcontext.TransferMetadata) ([]string, error) {
	logger.LogDebug(ctx.Context, "Fetching all repositories for an organization", "name", c.Owner)

	if c.Repo != "" {
		return []string{c.Repo}, nil
	}

	apiURL := fmt.Sprintf("https://api.github.com/orgs/%s/repos", c.Owner)

	logger.LogDebug(ctx.Context, "Constructed API URL for repositories", "value", apiURL)

	req, err := http.NewRequestWithContext(ctx.Context, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching repositories: %w", err)
	}
	defer resp.Body.Close()

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Decode the JSON response
	var repos []map[string]interface{} // Handle dynamic JSON structure
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Extract repository names
	var repoNames []string
	for _, r := range repos {
		if name, ok := r["name"].(string); ok {
			repoNames = append(repoNames, name)
		}
	}

	// Check if repositories were found
	if len(repoNames) == 0 {
		return nil, fmt.Errorf("no repositories found for organization %s", c.Owner)
	}

	return repoNames, nil
}
