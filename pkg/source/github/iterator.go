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

// GitHubIterator iterates over SBOMs fetched from GitHub (API, Release, Tool)
type GitHubIterator struct {
	// ctx        context.Context
	client     *Client
	sboms      []*iterator.SBOM // Stores all fetched SBOMs
	position   int              // Tracks iteration position
	binaryPath string
}

// NewGitHubIterator initializes the iterator based on the GitHub method
func NewGitHubIterator(ctx *tcontext.TransferMetadata, g *GitHubAdapter) (*GitHubIterator, error) {
	logger.LogDebug(ctx.Context, "Initializing GitHub Iterator", "repo", g.URL, "method", g.Method)

	ctx.WithValue("repo_url", g.URL)
	ctx.WithValue("repo_version", g.Version)

	client := NewClient(g.URL, g.Version, string(g.Method))
	iterator := &GitHubIterator{
		// ctx:        ctx,
		client:     client,
		sboms:      []*iterator.SBOM{},
		binaryPath: g.BinaryPath,
	}

	// Fetch SBOMs based on method
	var err error

	switch GitHubMethod(g.Method) {
	case MethodAPI:
		logger.LogDebug(ctx.Context, "Github Method", "value", MethodAPI)
		err = iterator.fetchSBOMFromAPI(ctx)

	case MethodReleases:
		logger.LogDebug(ctx.Context, "Github Method", "value", MethodReleases)
		err = iterator.fetchSBOMFromReleases(ctx)

	case MethodTool:
		logger.LogDebug(ctx.Context, "Github Method", "value", MethodTool)
		err = iterator.fetchSBOMFromTool(ctx)

	default:
		return nil, fmt.Errorf("unsupported GitHub method: %s", g.Method)
	}

	if err != nil {
		logger.LogError(ctx.Context, err, "Failed to fetch SBOMs")
		return nil, err
	}

	if len(iterator.sboms) == 0 {
		return nil, fmt.Errorf("no SBOMs found for repository")
	}

	logger.LogDebug(ctx.Context, "Total SBOMs fetched", "count", len(iterator.sboms))

	return iterator, nil
}

// Next returns the next SBOM from the stored list
func (it *GitHubIterator) Next(ctx context.Context) (*iterator.SBOM, error) {
	logger.LogDebug(ctx, "Iterating via Next")
	if it.position >= len(it.sboms) {
		return nil, io.EOF // No more SBOMs left
	}

	sbom := it.sboms[it.position]
	it.position++ // Move to the next SBOM
	return sbom, nil
}

// Fetch SBOM via GitHub API
func (it *GitHubIterator) fetchSBOMFromAPI(ctx *tcontext.TransferMetadata) error {
	logger.LogDebug(ctx.Context, "Fetching SBOM from GitHub API", "repo", it.client.RepoURL)

	sbomData, err := it.client.FetchSBOMFromAPI(ctx)
	if err != nil {
		return err
	}

	// Save to file
	sbomFilePath := fmt.Sprintf("sboms/github_api_sbom_%s.json", sanitizeRepoName(it.client.RepoURL))
	if err := saveSBOMToFile(sbomFilePath, sbomData); err != nil {
		return err
	}

	it.sboms = append(it.sboms, &iterator.SBOM{
		Path: sbomFilePath,
		Data: sbomData,
	})
	return nil
}

// Fetch SBOMs from GitHub Releases
func (it *GitHubIterator) fetchSBOMFromReleases(ctx *tcontext.TransferMetadata) error {
	logger.LogDebug(ctx.Context, "Fetching SBOMs from GitHub Releases", "repo", it.client.RepoURL)

	repoURL, _ := ctx.Value("repo_url").(string)
	repoVersion, _ := ctx.Value("repo_version").(string)
	fmt.Println("repoURL: ", repoURL)
	fmt.Println("repoVersion: ", repoVersion)

	// sboms, err := it.client.FindSBOMs(ctx)
	ctx.WithValue("output_dir", "sboms")

	sbomFiles, err := it.client.GetSBOMs(ctx, "sboms")
	if err != nil {
		logger.LogError(ctx.Context, err, "Failed to retrieve SBOMs from GitHub releases", "url", it.client.RepoURL, "version", it.client.Version)
		return fmt.Errorf("error retrieving SBOMs from releases: %w", err)
	}

	for _, sbomFile := range sbomFiles {
		for _, sbom := range sbomFile {
			it.sboms = append(it.sboms, &iterator.SBOM{
				Path: sbom,
				Data: nil,
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

// saveSBOMToFile writes SBOM data to a file and returns its path.
func saveSBOMToFile(filePath string, sbomData []byte) error {
	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory for SBOM file: %w", err)
	}

	// Write SBOM data to file
	if err := os.WriteFile(filePath, sbomData, 0o644); err != nil {
		return fmt.Errorf("failed to write SBOM to file: %w", err)
	}

	logger.LogDebug(context.Background(), "SBOM successfully written to file", "file", filePath)
	return nil
}
