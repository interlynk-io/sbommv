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

func (it *GitHubIterator) fetchSBOMFromTool(ctx *tcontext.TransferMetadata) error {
	logger.LogDebug(ctx.Context, "Generating SBOM using Tool", "repository", it.client.RepoURL)

	// Clone the repository
	repoDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s", it.client.Repo, it.client.Version))
	defer os.RemoveAll(repoDir)

	if err := CloneRepoWithGit(ctx, it.client.RepoURL, it.client.Branch, repoDir); err != nil {
		return fmt.Errorf("failed to clone the repository: %w", err)
	}

	// Generate SBOM and save in memory
	sbomData, err := GenerateSBOM(ctx, repoDir, it.binaryPath)
	if err != nil {
		return fmt.Errorf("failed to generate SBOM: %w", err)
	}

	sbomBytes, err := os.ReadFile(sbomData)
	if err != nil {
		return fmt.Errorf("failed to read SBOM: %w", err)
	}

	if len(sbomBytes) == 0 {
		return fmt.Errorf("generate SBOM with zero file data: %w", err)
	}

	it.sboms = append(it.sboms, &iterator.SBOM{
		Path:    "",
		Data:    sbomBytes,
		Repo:    fmt.Sprintf("%s/%s", it.client.Owner, it.client.Repo),
		Version: it.client.Version,
		Branch:  it.client.Branch,
	})
	logger.LogDebug(ctx.Context, "SBOM successfully stored in memory", "repository", it.client.RepoURL)
	return nil
}
