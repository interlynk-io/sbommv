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
	"strings"

	"github.com/interlynk-io/sbommv/pkg/logger"
)

// Asset represents a release asset
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int    `json:"size"`
}

// Release represents a GitHub release
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Client handles GitHub API interactions
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new GitHub client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{},
		baseURL:    "https://api.github.com",
	}
}

// GetReleases fetches all releases for a repository
func (c *Client) GetReleases(ctx context.Context, owner, repo string) ([]Release, error) {
	// url := fmt.Sprintf("https://github.com/%s/%s/releases", owner, repo)
	url := fmt.Sprintf("%s/repos/%s/%s/releases", c.baseURL, owner, repo)
	logger.LogInfo(ctx, "Fetching GitHub Releases", "url", url)

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
	logger.LogInfo(ctx, "Response ", "body", resp.Body)

	// Read response body for error reporting
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
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

// DownloadAsset downloads a release asset
func (c *Client) DownloadAsset(ctx context.Context, downloadURL string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	return resp.Body, nil
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
