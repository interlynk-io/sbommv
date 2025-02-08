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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

// // GitHubIterator iterates over SBOMs fetched from GitHub (API, Release, Tool)
type GitHubIterator struct {
	client     *Client
	sboms      []*iterator.SBOM // Stores all fetched SBOMs
	position   int              // Tracks iteration position
	binaryPath string
}

// NewGitHubIterator initializes the iterator based on the GitHub method
func NewGitHubIterator(ctx *tcontext.TransferMetadata, g *GitHubAdapter, repo string) (*GitHubIterator, error) {
	logger.LogDebug(ctx.Context, "Initializing GitHub Iterator", "repo", g.URL, "method", g.Method, "repo", repo)

	g.client.updateRepo(repo)

	iterator := &GitHubIterator{
		client:     g.client,
		sboms:      []*iterator.SBOM{},
		binaryPath: g.BinaryPath,
	}

	// Fetch SBOMs based on method
	var err error

	switch GitHubMethod(g.Method) {
	case MethodAPI:
		err = iterator.fetchSBOMFromAPI(ctx)

	case MethodReleases:
		err = iterator.fetchSBOMFromReleases(ctx)

	case MethodTool:
		err = iterator.fetchSBOMFromTool(ctx)

	default:
		return nil, fmt.Errorf("unsupported GitHub method: %s", g.Method)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch SBOMs: %w", err)
	}

	if len(iterator.sboms) == 0 {
		return nil, fmt.Errorf("no SBOMs found for repository")
	}

	logger.LogDebug(ctx.Context, "Total SBOMs fetched", "count", len(iterator.sboms))
	return iterator, nil
}

// Next returns the next SBOM from the stored list
func (it *GitHubIterator) Next(ctx context.Context) (*iterator.SBOM, error) {
	if it.position >= len(it.sboms) {
		return nil, io.EOF // No more SBOMs left
	}

	sbom := it.sboms[it.position]
	it.position++ // Move to the next SBOM
	return sbom, nil
}

// Fetch SBOM via GitHub API
func (it *GitHubIterator) fetchSBOMFromAPI(ctx *tcontext.TransferMetadata) error {
	sbomData, err := it.client.FetchSBOMFromAPI(ctx)
	if err != nil {
		return err
	}

	it.sboms = append(it.sboms, &iterator.SBOM{
		Path:    "",
		Data:    sbomData,
		Repo:    fmt.Sprintf("%s/%s", it.client.Owner, it.client.Repo),
		Version: "latest",
	})
	return nil
}

// Fetch SBOMs from GitHub Releases
func (it *GitHubIterator) fetchSBOMFromReleases(ctx *tcontext.TransferMetadata) error {
	sbomFiles, err := it.client.GetSBOMs(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving SBOMs from releases: %w", err)
	}

	for version, sbomDataList := range sbomFiles {
		for _, sbomData := range sbomDataList { // sbomPath is a string (file path)
			it.sboms = append(it.sboms, &iterator.SBOM{
				Path:    "", // No file path, storing in memory
				Data:    sbomData,
				Repo:    fmt.Sprintf("%s/%s", it.client.Owner, it.client.Repo),
				Version: version,
			})
		}
	}

	return nil
}

// Fetch SBOM by running a tool (Syft)
func (it *GitHubIterator) fetchSBOMFromTool(ctx *tcontext.TransferMetadata) error {
	logger.LogDebug(ctx.Context, "Generating SBOM using tool", "repository", it.client.RepoURL)

	// Clone the repository
	repoDir := filepath.Join(os.TempDir(), fmt.Sprintf("sbommv_%d", time.Now().UnixNano()))
	defer os.RemoveAll(repoDir)

	if err := CloneRepoWithGit(ctx, it.client.RepoURL, repoDir); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Generate SBOM
	sbomFile, err := GenerateSBOM(ctx, repoDir, it.binaryPath)
	if err != nil {
		return fmt.Errorf("failed to generate SBOM: %w", err)
	}

	// Ensure the "sboms" directory exists
	sbomDir := "sboms"
	if err := os.MkdirAll(sbomDir, 0o755); err != nil {
		return fmt.Errorf("failed to create SBOM output directory: %w", err)
	}

	// Move SBOM to final location
	sbomFilePath := fmt.Sprintf("%s/github_tool_sbom_%s.json", sbomDir, sanitizeRepoName(it.client.RepoURL))
	if err := os.Rename(sbomFile, sbomFilePath); err != nil {
		return fmt.Errorf("failed to move SBOM file: %w", err)
	}

	it.sboms = append(it.sboms, &iterator.SBOM{
		Path: sbomFilePath,
		Data: nil, // SBOM stored in file, no need for in-memory data
	})
	return nil
}
