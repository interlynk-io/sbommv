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

package iterator

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/source/github"
)

// GitHubAPIIterator iterates over SBOMs fetched from GitHub API
type GitHubAPIIterator struct {
	ctx     context.Context
	client  *github.Client
	sbom    *SBOM // Store the fetched SBOM
	fetched bool  // Tracks if the SBOM has already been returned
}

// NewGitHubAPIIterator initializes the iterator and fetches the SBOM at creation
func NewGitHubAPIIterator(ctx context.Context, client *github.Client) (*GitHubAPIIterator, error) {
	logger.LogDebug(ctx, "Initializing GitHub API Iterator", "repo", client.RepoURL)

	// Fetch SBOM immediately at initialization
	sbomData, err := client.FetchSBOMFromAPI(ctx)
	if err != nil {
		logger.LogError(ctx, err, "Failed to fetch SBOM from GitHub API")
		return nil, err
	}

	logger.LogDebug(ctx, "Successfully retrieved SBOM from GitHub API", "repository", "sbommv")

	// Define SBOM file path
	sbomFilePath := fmt.Sprintf("sboms/github_api_sbom_%s.json", sanitizeRepoName("SBOMMV"))

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

	logger.LogDebug(ctx, "SBOM fetched and stored", "file", sbomFilePath)

	// Create and return the iterator
	return &GitHubAPIIterator{
		ctx:    ctx,
		client: client,
		sbom: &SBOM{
			Path: sbomFilePath,
			Data: sbomData,
		},
		fetched: false, // Mark as not returned yet
	}, nil
}

// Next returns the fetched SBOM on the first call, then signals EOF
func (it *GitHubAPIIterator) Next(ctx context.Context) (*SBOM, error) {
	if it.fetched {
		return nil, io.EOF // No more SBOMs left
	}

	it.fetched = true // Mark as returned
	return it.sbom, nil
}

func sanitizeRepoName(repoURL string) string {
	repoParts := strings.Split(repoURL, "/")
	if len(repoParts) < 2 {
		return "unknown"
	}
	return repoParts[len(repoParts)-1] // Extracts "cosign" from URL
}
