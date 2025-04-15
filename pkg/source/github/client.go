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
	"github.com/interlynk-io/sbommv/pkg/source"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

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

type SBOMData struct {
	Content  []byte
	Filename string
	Release  string
}

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
	Branch       string
	Token        string
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
		Branch:     g.Branch,
		Token:      g.GithubToken,
	}
}

// FindSBOMs fetches SBOMs from particular repository release page with configurable concurrency
func (c *Client) FindSBOMs(ctx tcontext.TransferMetadata, concurrency int) ([]SBOMData, error) {
	logger.LogDebug(ctx.Context, "Fetching SBOMs from GitHub releases", "repo_url", c.RepoURL, "owner", c.Owner, "repo", c.Repo)

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
	logger.LogDebug(ctx.Context, "Total Releases from SBOM is fetched", "value", len(targetReleases))

	var sbomDataList []SBOMData
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	for _, release := range targetReleases {
		for _, asset := range release.Assets {
			if !source.DetectSBOMsFile(asset.Name) {
				continue
			}

			if concurrency == 0 {
				// download sequentially
				reader, err := c.DownloadAsset(ctx, asset.DownloadURL)
				if err != nil {
					logger.LogDebug(ctx.Context, "Failed to download", "file", asset.Name, "error", err)
					continue
				}

				content, err := io.ReadAll(reader)
				if err != nil {
					return nil, fmt.Errorf("reading SBOM content: %w", err)
				}

				if !source.IsSBOMFile(content) {
					logger.LogDebug(ctx.Context, "Skipping non-SBOM", "file", asset.Name)
					continue
				}

				sbomDataList = append(sbomDataList, SBOMData{
					Content:  content,
					Filename: asset.Name,
					Release:  release.TagName,
				})

				logger.LogDebug(ctx.Context, "Fetched SBOM", "file", asset.Name)

			} else {
				// download concurrently
				wg.Add(1)
				semaphore <- struct{}{}
				go func(asset Asset, releaseTag string) {
					defer wg.Done()
					defer func() { <-semaphore }()

					reader, err := c.DownloadAsset(ctx, asset.DownloadURL)
					if err != nil {
						logger.LogDebug(ctx.Context, "Failed to download", "file", asset.Name, "error", err)
						return
					}

					content, err := io.ReadAll(reader)
					if err != nil {
						logger.LogDebug(ctx.Context, "Error in reading SBOM content", "file", asset.Name)
						return
					}

					if !source.IsSBOMFile(content) {
						logger.LogDebug(ctx.Context, "Skipping non-SBOM", "file", asset.Name)
						return
					}
					mu.Lock()
					sbomDataList = append(sbomDataList, SBOMData{
						Content:  content,
						Filename: asset.Name,
						Release:  releaseTag,
					})
					mu.Unlock()
					logger.LogDebug(ctx.Context, "Fetched SBOM", "file", asset.Name)
				}(asset, release.TagName)
			}

		}
	}

	if concurrency > 0 {
		wg.Wait()
	}

	if len(sbomDataList) == 0 {
		logger.LogInfo(ctx.Context, "No SBOMs found", "repo", c.Repo, "owner", c.Owner)
		return nil, nil
	}

	logger.LogDebug(ctx.Context, "Successfully retrieved SBOMs", "total_sboms", len(sbomDataList), "repo_url", c.RepoURL)

	return sbomDataList, nil
}

// filterReleases filters releases based on version
func (c *Client) filterReleases(releases []Release, version string) []Release {
	// return all Releases
	if version == "*" {
		return releases
	}

	// return latest release
	if version == "latest" {
		return []Release{releases[0]}
	}

	// return the matching release version
	for _, release := range releases {
		if release.TagName == version {
			return []Release{release}
		}
	}

	return nil
}

// GetReleases fetches all releases for a repository
func (c *Client) GetReleases(ctx tcontext.TransferMetadata, owner, repo string) ([]Release, error) {
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
func (c *Client) DownloadAsset(ctx tcontext.TransferMetadata, downloadURL string) (io.ReadCloser, error) {
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

// FetchSBOMFromAPI fetches SBOM from GitHub Dependency Graph API
func (c *Client) FetchSBOMFromAPI(ctx tcontext.TransferMetadata) ([]byte, error) {
	owner, repo, err := source.ParseGitHubURL(c.RepoURL)
	if err != nil {
		return nil, fmt.Errorf("parsing GitHub URL: %w", err)
	}

	logger.LogDebug(ctx.Context, "Fetching SBOM Details", "repository", repo, "owner", owner, "repo_url", c.RepoURL)

	// Construct the API URL for the SBOM export
	url := fmt.Sprintf("%s/%s", c.BaseURL, fmt.Sprintf(githubSBOMEndpoint, owner, repo))
	logger.LogDebug(ctx.Context, "Fetching SBOM via GitHub API", "url", url)

	// Create request
	req, err := http.NewRequestWithContext(ctx.Context, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication only if a token is provided
	if c.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
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

// GetAllRepositories fetches all repositories for the organization
func (c *Client) GetAllRepositories(ctx tcontext.TransferMetadata) ([]string, error) {
	if c.Repo != "" {
		return []string{c.Repo}, nil
	}
	logger.LogDebug(ctx.Context, "Fetching all repositories for an organization", "name", c.Owner)

	apiURL := fmt.Sprintf("https://api.github.com/orgs/%s/repos", c.Owner)

	logger.LogDebug(ctx.Context, "Constructed API URL for repositories", "value", apiURL)

	req, err := http.NewRequestWithContext(ctx.Context, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Add authentication only if a token is provided
	if c.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching repositories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Decode the JSON response
	var repos []map[string]interface{}
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

	logger.LogDebug(ctx.Context, "Total available repos in an organization", "count", len(repos), "in organization", c.Owner)

	return repoNames, nil
}
