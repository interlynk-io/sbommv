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
	"os"
	"path/filepath"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

// SBOMFetcher defines the strategy for fetching SBOMs
type SBOMFetcher interface {
	Fetch(ctx *tcontext.TransferMetadata, client GitHubAPI, config *GitHubConfig) (iterator.SBOMIterator, error)
}

// APIFetcher fetches SBOMs using GitHub's dependency graph API
type APIFetcher struct{}

func (f *APIFetcher) Fetch(ctx *tcontext.TransferMetadata, client GitHubAPI, config *GitHubConfig) (iterator.SBOMIterator, error) {
	iter := NewGitHubIterator(ctx, client, config)
	if err := iter.fetchSBOMFromAPI(ctx); err != nil {
		return nil, err
	}
	return iter, nil
}

// ReleasesFetcher fetches SBOMs from GitHub release assets
type ReleasesFetcher struct{}

func (f *ReleasesFetcher) Fetch(ctx *tcontext.TransferMetadata, client GitHubAPI, config *GitHubConfig) (iterator.SBOMIterator, error) {
	iter := NewGitHubIterator(ctx, client, config)
	if err := iter.fetchSBOMFromReleases(ctx); err != nil {
		return nil, err
	}
	return iter, nil
}

// ToolFetcher generates SBOMs using an external tool like Syft
type ToolFetcher struct {
	generator SBOMGenerator
}

func NewToolFetcher(generator SBOMGenerator) *ToolFetcher {
	return &ToolFetcher{generator: generator}
}

func (f *ToolFetcher) Fetch(ctx *tcontext.TransferMetadata, client GitHubAPI, config *GitHubConfig) (iterator.SBOMIterator, error) {
	iter := NewGitHubIterator(ctx, client, config)
	repoDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s", config.Repo, config.Version))
	defer os.RemoveAll(repoDir)

	if err := CloneRepoWithGit(ctx, config.URL, config.Branch, repoDir); err != nil {
		return nil, err
	}

	sbomFile, err := f.generator.Generate(ctx, repoDir, config.BinaryPath)
	if err != nil {
		return nil, err
	}

	sbomBytes, err := os.ReadFile(sbomFile)
	if err != nil {
		return nil, err
	}

	iter.sboms = append(iter.sboms, &iterator.SBOM{
		Data:      sbomBytes,
		Namespace: fmt.Sprintf("%s/%s", config.Owner, config.Repo),
		Version:   config.Version,
		Branch:    config.Branch,
	})
	return iter, nil
}

// fetcherFactory maps method names to their fetcher implementations
var fetcherFactory = map[string]SBOMFetcher{
	MethodAPI:      &APIFetcher{},
	MethodReleases: &ReleasesFetcher{},
	MethodTool:     NewToolFetcher(&SyftGenerator{}),
}
