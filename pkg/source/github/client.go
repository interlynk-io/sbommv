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
	"strings"
	"sync"

	githublib "github.com/google/go-github/v62/github"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/source"
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
type VersionedSBOMs map[string][]SBOMData

type SBOMData struct {
	Content  []byte
	Filename string
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
func NewClient(g *GithubConfig) *Client {
	return &Client{
		httpClient: &http.Client{},
		BaseURL:    "https://api.github.com",
		RepoURL:    g.URL,
		Version:    g.Version,
		Method:     g.Method,
		Owner:      g.Owner,
		Repo:       g.Repo,
		Branch:     g.Branch,
		Token:      g.Token,
	}
}

// FindSBOMs gets all releases assets from github release page
// filter out the particular provided release asset and
// extract SBOMs from that
func (c *Client) FindSBOMs(ctx tcontext.TransferMetadata) ([]SBOMAsset, error) {
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

	// Extract SBOM assets from target release
	sboms := c.extractSBOMs(targetReleases)

	if len(sboms) == 0 {
		logger.LogInfo(ctx.Context, "error", "sboms", 0, "repo", c.Repo, "owner", c.Owner)
		return nil, nil
	}
	logger.LogDebug(ctx.Context, "Successfully retrieved SBOMs", "total_sboms", len(sboms), "repo_url", c.RepoURL)

	return sboms, nil
}

// filterReleases filters releases based on version input
func (c *Client) filterReleases(releases []Release, version string) []Release {
	if version == "*" {
		// Return all refilterReleasesleases
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
			if source.DetectSBOMsFile(asset.Name) {
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

// GetSBOMs downloads and saves all SBOM files found in the repository
func (c *Client) FetchSBOMFromReleases(ctx tcontext.TransferMetadata) (VersionedSBOMs, error) {
	logger.LogDebug(ctx.Context, "Initializing fetching of SBOMs from repo", "repository", c.Repo, "version", c.Version)
	// Find SBOMs in releases
	sboms, err := c.FindSBOMs(ctx)
	if err != nil {
		return nil, fmt.Errorf("finding SBOMs: %w", err)
	}
	if len(sboms) == 0 {
		return nil, fmt.Errorf("no SBOMs found in repository")
	}

	logger.LogDebug(ctx.Context, "Total SBOMs found in the repository release page", "version", c.Version, "total sboms", len(sboms))
	ctx.WithValue("total_sboms", len(sboms))

	return c.downloadSBOMs(ctx, sboms)
}

// downloadSBOMs handles the concurrent downloading of multiple SBOM files
func (c *Client) downloadSBOMs(ctx tcontext.TransferMetadata, sboms []SBOMAsset) (VersionedSBOMs, error) {
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

	var totalDownloadedSBOMs int
	var totalSBOMsWithCorrectFormatAndSpec int

	// Process each SBOM
	for _, sbom := range sboms {
		totalDownloadedSBOMs++
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

			// now check the spec and format of a downloaded SBOM files from github
			if source.IsSBOMFile(sbomData) {
				totalSBOMsWithCorrectFormatAndSpec++
				versionedSBOM := SBOMData{
					Content:  sbomData,
					Filename: sbom.Name,
				}

				mu.Lock()
				versionedSBOMs[sbom.Release] = append(versionedSBOMs[sbom.Release], versionedSBOM)
				mu.Unlock()

				logger.LogDebug(ctx.Context, "SBOM fetched and stored in memory", "name", sbom.Name)
			}
			// _ = bar.Add(1) // Update progress bar
		}(sbom)
	}

	wg.Wait()

	if len(errors) > 0 {
		return nil, fmt.Errorf("encountered %d download errors: %v", len(errors), errors[0])
	}

	logger.LogDebug(ctx.Context, "Total SBOMs fetched and stored in memory", "total_downloaded_sboms", totalDownloadedSBOMs, "correct_sboms_with_format_and_spec", totalSBOMsWithCorrectFormatAndSpec)
	return versionedSBOMs, nil
}

// downloadSingleSBOM downloads a single SBOM and stores it in memory
func (c *Client) downloadSingleSBOM(ctx tcontext.TransferMetadata, sbom SBOMAsset) ([]byte, error) {
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

// GetAllRepositories fetches all repositories for the organization specified in c.Owner.
// It also handles pagination to ensure all repositories are retrieved.
func (c *Client) GetAllRepositories(ctx tcontext.TransferMetadata) ([]string, error) {
	if c.Repo != "" {
		return []string{c.Repo}, nil
	}
	logger.LogDebug(ctx.Context, "Fetching all repositories for an organization", "name", c.Owner)

	baseURL := fmt.Sprintf("https://api.github.com/orgs/%s/repos", c.Owner)
	apiURL := baseURL + "?per_page=100"

	var allRepos []map[string]interface{}
	page := 1
	for {
		logger.LogDebug(ctx.Context, "Fetching repository page", "org", c.Owner, "page", page)

		req, err := http.NewRequestWithContext(ctx.Context, "GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request for page %d: %w", page, err)
		}

		if c.Token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching repositories for page %d: %w", page, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("GitHub API returned status %d for page %d: %s", resp.StatusCode, page, string(body))
		}

		var repos []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			return nil, fmt.Errorf("decoding response for page %d: %w", page, err)
		}

		logger.LogDebug(ctx.Context, "Fetched repository page", "org", c.Owner, "page", page, "repos_fetched", len(repos), "total_so_far", len(allRepos)+len(repos))
		allRepos = append(allRepos, repos...)

		// Check for pagination via Link header
		linkHeader := resp.Header.Get("Link")
		if linkHeader == "" || !strings.Contains(linkHeader, `rel="next"`) {
			// No more pages
			break
		}

		// Extract the next page URL from the Link header
		links := parseLinkHeader(linkHeader)
		if nextURL, ok := links["next"]; ok {
			apiURL = nextURL
			page++
		} else {
			break
		}
	}

	logger.LogInfo(ctx.Context, "Completed fetching repositories", "org", c.Owner, "total_repos", len(allRepos))

	var repoNames []string
	for _, r := range allRepos {
		if name, ok := r["name"].(string); ok {
			repoNames = append(repoNames, name)
		}
	}

	if len(repoNames) == 0 {
		return nil, fmt.Errorf("no repositories found for organization %s", c.Owner)
	}

	logger.LogDebug(ctx.Context, "Total available repos in an organization", "count", len(repoNames), "in organization", c.Owner)

	return repoNames, nil
}

// parseLinkHeader parses the GitHub Link header to extract pagination URLs.
// Example: <https://api.github.com/orgs/interlynk-io/repos?page=2>; rel="next", <https://api.github.com/orgs/interlynk-io/repos?page=2>; rel="last"
func parseLinkHeader(header string) map[string]string {
	links := make(map[string]string)
	if header == "" {
		return links
	}

	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !strings.Contains(part, ";") {
			continue
		}

		sections := strings.Split(part, ";")
		if len(sections) < 2 {
			continue
		}

		url := strings.Trim(sections[0], "<>")
		rel := strings.TrimSpace(sections[1])
		rel = strings.Trim(rel, `rel="`)
		rel = strings.Trim(rel, `"`)

		links[rel] = url
	}

	return links
}

// applyRepoFilters filters repositories based on inclusion/exclusion flags
func (c *Client) applyRepoFilters(ctx tcontext.TransferMetadata, repos, includeRepos, excludeRepos []string) []string {
	logger.LogDebug(ctx.Context, "Applying repository filters", "include", includeRepos, "exclude", excludeRepos)

	includedRepos := make(map[string]bool)
	excludedRepos := make(map[string]bool)

	for _, repo := range includeRepos {
		if repo != "" {
			includedRepos[strings.TrimSpace(repo)] = true
		}
	}

	for _, repo := range excludeRepos {
		if repo != "" {
			excludedRepos[strings.TrimSpace(repo)] = true
		}
	}

	var filteredRepos []string

	for _, repoName := range repos {

		if _, isExcluded := excludedRepos[repoName]; isExcluded {
			// skip excluded repositories
			continue
		}

		// Include only if in the inclusion list (if provided)
		if len(includedRepos) > 0 {
			if _, isIncluded := includedRepos[repoName]; !isIncluded {
				// skip repos that are not in the include list
				continue
			}
		}

		// filtered repo are added to the final list
		filteredRepos = append(filteredRepos, repoName)
	}

	logger.LogDebug(ctx.Context, "Filtered repositories", "filtered", filteredRepos)
	return filteredRepos
}

func GetAllOrgRepositories(ctx tcontext.TransferMetadata, client *githublib.Client, org string) ([]string, error) {
	logger.LogDebug(ctx.Context, "Fetching all repositories for organization", "org", org)

	var repoNames []string

	opt := &githublib.RepositoryListByOrgOptions{
		ListOptions: githublib.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx.Context, org, opt)
		if err != nil {
			logger.LogError(ctx.Context, err, "Failed to fetch repositories for organization", "org", org, "page", opt.Page+1)
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}

		for _, repo := range repos {
			repoNames = append(repoNames, fmt.Sprintf("%s/%s", org, repo.GetName()))
		}

		logger.LogDebug(ctx.Context, "Fetched repository page", "org", org, "page", opt.Page+1, "repos_fetched", len(repos), "total_so_far", len(repoNames))

		if resp.NextPage == 0 {
			break // No more pages
		}

		opt.Page = resp.NextPage

	}

	if len(repoNames) == 0 {
		return nil, fmt.Errorf("no repositories found for organization %s", org)
	}
	logger.LogInfo(ctx.Context, "Completed fetching repositories", "org", org, "total_repos", len(repoNames))
	return repoNames, nil
}
