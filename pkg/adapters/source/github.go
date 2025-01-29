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
	logger.LogDebug(ctx, "Fetching SBOMs from GitHub releases", "url", a.URL, "version", a.Version)

	client := github.NewClient()

	sbomFiles, err := client.GetSBOMs(ctx, a.URL, a.Version, "sboms")
	if err != nil {
		logger.LogError(ctx, err, "Failed to retrieve SBOMs from GitHub releases", "url", a.URL, "version", a.Version)
		return nil, fmt.Errorf("error retrieving SBOMs from releases: %w", err)
	}

	return sbomFiles, nil
}

func (a *GitHubAdapter) generateSBOMs(ctx context.Context) (map[string][]string, error) {
	// TODO: Implement SBOM generation using tools like cdxgen
	return nil, fmt.Errorf("not implemented")
}
