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
	"github.com/interlynk-io/sbommv/pkg/source/github"
)

// GitHubAdapter implements InputAdapter for GitHub repositories
type GitHubAdapter struct {
	URL string
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
func NewGitHubAdapter(URL string, method GitHubMethod, opts InputOptions) *GitHubAdapter {
	return &GitHubAdapter{
		URL: URL,
		// token:   token,
		method:  method,
		client:  &http.Client{},
		options: opts,
	}
}

// GetSBOMs implements InputAdapter
func (a *GitHubAdapter) GetSBOMs(ctx context.Context) ([]string, error) {
	switch a.method {
	case MethodReleases:
		logger.LogInfo(ctx, "Get SBOMs from Release Page", "method", MethodReleases)
		return a.getSBOMsFromReleases(ctx)
	case MethodGenerate:
		logger.LogInfo(ctx, "Get SBOMs from tools", "method", MethodGenerate)
		return a.generateSBOMs(ctx)
	default:
		return nil, fmt.Errorf("unsupported GitHub method: %v", a.method)
	}
}

func (a *GitHubAdapter) getSBOMsFromReleases(ctx context.Context) ([]string, error) {
	sboms, err := github.GetSBOMs(ctx, a.URL, "sboms")
	if err != nil {
		return nil, err
	}

	return sboms, nil
}

func (a *GitHubAdapter) generateSBOMs(ctx context.Context) ([]string, error) {
	// TODO: Implement SBOM generation using tools like cdxgen
	return nil, fmt.Errorf("not implemented")
}
