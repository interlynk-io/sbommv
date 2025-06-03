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
	"fmt"
	"io"
	"os"
	"path/filepath"

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

// NewGitHubIterator initializes and returns a new GitHubIterator instance
func NewGitHubIterator(ctx tcontext.TransferMetadata, g *GithubConfig, repo string) *GitHubIterator {
	logger.LogDebug(ctx.Context, "Initializing GitHub Iterator", "repo", g.URL, "method", g.Method, "repo", repo)

	g.client.updateRepo(repo)

	// Create and return the iterator instance without fetching SBOMs
	return &GitHubIterator{
		client:     g.client,
		sboms:      []*iterator.SBOM{},
		binaryPath: g.BinaryPath,
	}
}

// Next returns the next SBOM from the stored list
func (it *GitHubIterator) Next(ctx tcontext.TransferMetadata) (*iterator.SBOM, error) {
	if it.position >= len(it.sboms) {
		return nil, io.EOF // No more SBOMs left
	}

	sbom := it.sboms[it.position]
	it.position++ // Move to the next SBOM
	return sbom, nil
}

type GithubWatcherIterator struct {
	sbomChan chan *iterator.SBOM
}

func (it *GithubWatcherIterator) Next(ctx tcontext.TransferMetadata) (*iterator.SBOM, error) {
	select {
	case sbom, ok := <-it.sbomChan:
		if !ok {
			return nil, fmt.Errorf("watcher channel closed")
		}
		return sbom, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Fetch SBOM via GitHub API
func (it *GitHubIterator) fetchSBOMFromAPI(ctx tcontext.TransferMetadata) ([]*iterator.SBOM, error) {
	sbomData, err := it.client.FetchSBOMFromAPI(ctx)
	if err != nil {
		return nil, fmt.Errorf("error generating SBOMs from tool: %w", err)
	}

	var sbomSlice []*iterator.SBOM
	filepath := "dependency-graph-sbom.json"
	sbomSlice = append(sbomSlice, &iterator.SBOM{
		Path: filepath,
		Data: sbomData,

		// namespace as owner/repo, where SBOM are present
		Namespace: fmt.Sprintf("%s-%s", it.client.Owner, it.client.Repo),
		Version:   "latest",
	})
	logger.LogDebug(ctx.Context, "SBOM successfully fetched using API Method")

	return sbomSlice, nil
}

// Fetch SBOMs from GitHub Releases
func (it *GitHubIterator) fetchSBOMFromReleases(ctx tcontext.TransferMetadata) ([]*iterator.SBOM, error) {
	sbomFiles, err := it.client.FetchSBOMFromReleases(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving SBOMs from releases: %w", err)
	}

	var sbomSlice []*iterator.SBOM

	for version, sbomDataList := range sbomFiles {
		for _, sbomData := range sbomDataList { // sbomPath is a string (file path)
			sbomSlice = append(sbomSlice, &iterator.SBOM{
				Path: sbomData.Filename,
				Data: sbomData.Content,

				// namespace as owner/repo, where SBOM are present
				Namespace: fmt.Sprintf("%s-%s", it.client.Owner, it.client.Repo),
				Version:   version,
			})
		}
	}
	logger.LogDebug(ctx.Context, "SBOM successfully fetched using Release Method")
	return sbomSlice, nil
}

func (it *GitHubIterator) fetchSBOMFromTool(ctx tcontext.TransferMetadata) ([]*iterator.SBOM, error) {
	logger.LogDebug(ctx.Context, "Generating SBOM using Tool", "repository", it.client.RepoURL)

	var sbomSlice []*iterator.SBOM

	// Clone the repository
	repoDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s", it.client.Repo, it.client.Version))
	defer os.RemoveAll(repoDir)

	if err := CloneRepoWithGit(ctx, it.client.RepoURL, it.client.Branch, repoDir); err != nil {
		return nil, fmt.Errorf("failed to clone the repository: %w", err)
	}

	// Generate SBOM and save in memory
	sbomBytes, err := GenerateSBOM(ctx, repoDir, it.binaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SBOM: %w", err)
	}

	// TODO(cleanup later): Write SBOM to a file for testing to see its content
	// outputFile := fmt.Sprintf("syft-tool-%s.json", uuid.NewString())
	// if err := os.WriteFile(outputFile, sbomBytes, 0o644); err != nil {
	// 	return nil, fmt.Errorf("failed to write to file %s: %w", outputFile, err)
	// }

	if len(sbomBytes) == 0 {
		return nil, fmt.Errorf("generate SBOM with zero file data: %w", err)
	}

	filepath := "syft-generated-sbom.json"
	sbomSlice = append(sbomSlice, &iterator.SBOM{
		Path: filepath,
		Data: sbomBytes,

		// namespace as owner/repo, where SBOM are present
		Namespace: fmt.Sprintf("%s-%s", it.client.Owner, it.client.Repo),
		Version:   it.client.Version,
		Branch:    it.client.Branch,
	})
	logger.LogDebug(ctx.Context, "SBOM successfully fetched using Tool Method")
	return sbomSlice, nil
}
