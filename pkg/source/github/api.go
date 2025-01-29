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

	// SPDX JSON parsing library

	"github.com/interlynk-io/sbommv/pkg/logger"
)

const githubSBOMEndpoint = "repos/%s/%s/dependency-graph/sbom"

// GitHubSBOMResponse holds the JSON structure returned by GitHub API
type GitHubSBOMResponse struct {
	SBOM json.RawMessage `json:"sbom"` // Extract SBOM as raw JSON
}

func (c *Client) FetchSBOMFromAPI(ctx context.Context) ([]byte, error) {
	owner, repo, err := ParseGitHubURL(c.repoURL)
	if err != nil {
		return nil, fmt.Errorf("parsing GitHub URL: %w", err)
	}
	// Construct the API URL for the SBOM export
	url := fmt.Sprintf("%s/%s", c.baseURL, fmt.Sprintf(githubSBOMEndpoint, owner, repo))
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

	logger.LogDebug(ctx, "Fetched SBOM successfully", "repository", c.repoURL)

	// Return the raw SBOM JSON as bytes
	return response.SBOM, nil
}
