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
// -------------------------------------------------------------------------

package source

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/mvtypes"
	"github.com/interlynk-io/sbommv/pkg/source/github"
)

// GitHubAdapter implements InputAdapter for GitHub repositories
type GitHubAdapter struct {
	URL     string
	Version string
	// repo    string
	// token   string
	method  GitHubMethod
	client  *http.Client
	options InputOptions
}

// GitHubMethod specifies how to retrieve/generate SBOMs from GitHub
type GitHubMethod string

const (
	// MethodReleases searches for SBOMs in GitHub releases
	MethodReleases GitHubMethod = "release"

	// // MethodReleases searches for SBOMs in GitHub releases
	MethodAPI GitHubMethod = "api"

	// MethodGenerate clones the repo and generates SBOMs using external Tools
	MethodGenerate GitHubMethod = "generate"
)

// NewGitHubAdapter creates a new GitHub adapter
func NewGitHubAdapter(config mvtypes.Config) *GitHubAdapter {
	url := config.SourceConfigs["url"].(string)
	version := config.SourceConfigs["version"].(string)
	method := config.SourceConfigs["method"].(string)

	return &GitHubAdapter{
		URL:     url,
		Version: version,
		method:  GitHubMethod(method),
		client:  &http.Client{},
		// options: config.InputOptions,
	}
}

// GitHubAdapter implements GetSBOMs. Therefore, it implements InputAdapter.
func (a *GitHubAdapter) GetSBOMs(ctx context.Context) (map[string][]string, error) {
	logger.LogDebug(ctx, "Executing GetSBOMs function", "method", a.method)

	switch a.method {
	case MethodReleases:
		logger.LogDebug(ctx, "Fetching SBOMs from GitHub Release Page", "method", MethodReleases)
		return a.getSBOMsFromReleases(ctx)

	case MethodAPI:
		return a.getSBOMsFromAPI(ctx)

	case MethodGenerate:
		logger.LogDebug(ctx, "Generating SBOMs using tools", "method", MethodGenerate)
		return a.generateSBOMs(ctx)

	default:
		err := fmt.Errorf("unsupported GitHub method: %v", a.method)
		logger.LogError(ctx, err, "Invalid GitHub SBOM retrieval method", "method", a.method)
		return nil, err
	}
}

func (a *GitHubAdapter) getSBOMsFromReleases(ctx context.Context) (map[string][]string, error) {
	logger.LogDebug(ctx, "Fetching SBOMs from GitHub using %s", a.method, "url", a.URL, "version", a.Version)

	client := github.NewClient(a.URL, a.Version, string(a.method))

	sbomFiles, err := client.GetSBOMs(ctx, "sboms")
	if err != nil {
		logger.LogError(ctx, err, "Failed to retrieve SBOMs from GitHub releases", "url", a.URL, "version", a.Version)
		return nil, fmt.Errorf("error retrieving SBOMs from releases: %w", err)
	}

	return sbomFiles, nil
}

func (a *GitHubAdapter) getSBOMsFromAPI(ctx context.Context) (map[string][]string, error) {
	logger.LogDebug(ctx, "Fetching SBOM from GitHub API", "repository", a.URL)

	client := github.NewClient(a.URL, a.Version, string(a.method))
	sbomData, err := client.FetchSBOMFromAPI(ctx)
	if err != nil {
		logger.LogError(ctx, err, "Failed to fetch SBOM from GitHub API", "repository", a.URL)
		return nil, fmt.Errorf("error retrieving SBOM from GitHub API: %w", err)
	}

	logger.LogDebug(ctx, "Successfully retrieved SBOM from GitHub API", "repository", a.URL)

	// Define SBOM file path
	sbomFilePath := fmt.Sprintf("sboms/github_api_sbom_%s.json", sanitizeRepoName(a.URL))

	// Ensure the directory exists
	if err := os.MkdirAll("sboms", 0o755); err != nil {
		logger.LogError(ctx, err, "Failed to create SBOM output directory")
		return nil, fmt.Errorf("error creating SBOM output directory: %w", err)
	}

	// Write SBOM data to file
	if err := os.WriteFile(sbomFilePath, sbomData, 0o644); err != nil {
		logger.LogError(ctx, err, "Failed to write SBOM to file", "file", sbomFilePath)
		return nil, fmt.Errorf("error writing SBOM to file: %w", err)
	}
	logger.LogDebug(ctx, "SBOM successfully written to file", "file", sbomFilePath)

	return map[string][]string{"main": {sbomFilePath}}, nil
}

func (a *GitHubAdapter) generateSBOMs(ctx context.Context) (map[string][]string, error) {
	// TODO: Implement SBOM generation using tools like cdxgen
	return nil, fmt.Errorf("not implemented")
}
