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

package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	githublib "github.com/google/go-github/v62/github"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/source"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/utils"
)

type GithubWatcherFetcher struct{}

func NewWatcherFetcher() *GithubWatcherFetcher {
	return &GithubWatcherFetcher{}
}

// Cache holds polling state for repositories and SBOMs.
type Cache struct {
	Repos map[string]RepoState `json:"repos"`
	SBOMs map[string]bool      `json:"sboms"`
	sync.RWMutex
}

// RepoState stores release information.
type RepoState struct {
	PublishedAt string `json:"published_at"`
	ReleaseID   string `json:"release_id"`
}

// NewCache initializes a cache.
func NewCache() *Cache {
	return &Cache{
		Repos: make(map[string]RepoState),
		SBOMs: make(map[string]bool),
	}
}

// LoadCache reads the cache from file.
func (c *Cache) LoadCache(path string) error {
	c.Lock()
	defer c.Unlock()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read cache: %w", err)
	}
	return json.Unmarshal(data, c)
}

// SaveCache writes the cache to file.
func (c *Cache) SaveCache(path string) error {
	c.RLock()
	defer c.RUnlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

func (f *GithubWatcherFetcher) Fetch(ctx tcontext.TransferMetadata, config *GithubConfig) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Starting GitHub watcher", "repo", config.Repo, "branch", config.Branch)

	repos, err := config.client.GetAllRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	// filtering to include/exclude repos
	repos = config.applyRepoFilters(repos)

	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories left after applying filters")
	}

	// Initialize GitHub client
	client, err := config.GetGitHubClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Initiate cache
	cache := NewCache()
	cachePath := ".sbommv/cache.json"
	if err := cache.LoadCache(cachePath); err != nil {
		return nil, fmt.Errorf("failed to load cache: %w", err)
	}

	// Create SBOM channel for iterator
	sbomChan := make(chan *iterator.SBOM, 10)

	// Start polling loop in a goroutine
	go func() {
		defer close(sbomChan)
		ticker := time.NewTicker(time.Duration(config.Poll) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Context.Done():
				logger.LogInfo(ctx.Context, "Polling stopped")
				return
			case <-ticker.C:
				logger.LogDebug(ctx.Context, "Polling repositories", "time", time.Now().Format(time.RFC3339))
				for _, repo := range repos {
					if err := pollRepository(ctx, client, repo, config.Owner, cache, sbomChan); err != nil {
						logger.LogError(ctx.Context, err, "Failed to poll repository", "repo", repo)
					}
				}
				if err := cache.SaveCache(cachePath); err != nil {
					logger.LogError(ctx.Context, err, "Failed to save cache")
				}
			}
		}
	}()

	return &GithubWatcherIterator{sbomChan: sbomChan}, nil
}

func pollRepository(ctx tcontext.TransferMetadata, client *githublib.Client, repo, owner string, cache *Cache, sbomChan chan *iterator.SBOM) error {
	logger.LogDebug(ctx.Context, "Polling repository", "repo", repo)
	var releases []*githublib.RepositoryRelease

	var resp *githublib.Response
	var err error
	releases, resp, err = client.Repositories.ListReleases(ctx.Context, owner, repo, &githublib.ListOptions{PerPage: 100})
	if err != nil {
		if resp != nil && resp.StatusCode == 429 {
			logger.LogDebug(ctx.Context, "Rate limit hit, retrying", "repo", repo)
		}
		return err
	}

	if len(releases) == 0 {
		logger.LogDebug(ctx.Context, "No releases found for repository", "repo", repo)
		return nil
	}

	// collect latest release and related information
	latestRelease := releases[0]
	releaseID := fmt.Sprintf("%d", latestRelease.GetID())
	publishedAt := latestRelease.GetPublishedAt().Format(time.RFC3339)

	cached, exists := cache.Repos[repo]

	if exists && cached.PublishedAt == publishedAt && cached.ReleaseID == releaseID {
		logger.LogDebug(ctx.Context, "No new release found", "repo", repo)
		return nil
	}

	logger.LogInfo(ctx.Context, "New release detected", "repo", repo, "release_id", releaseID, "published_at", publishedAt)

	// Now fetch the new assets
	assets, _, err := client.Repositories.ListReleaseAssets(ctx.Context, owner, repo, latestRelease.GetID(), nil)
	if err != nil {
		return fmt.Errorf("failed to list assets: %w", err)
	}

	for _, asset := range assets {
		if err := processAsset(ctx, client, owner, repo, asset, cache, sbomChan); err != nil {
			logger.LogError(ctx.Context, err, "Failed to process asset", "repo", repo, "asset", asset.GetName())
		}
	}
	// Process assets
	// processor := sbom.NewSBOMProcessor("", false)

	return nil
}

func processAsset(ctx tcontext.TransferMetadata, client *githublib.Client, owner, repo string, asset *githublib.ReleaseAsset, cache *Cache, sbomChan chan *iterator.SBOM) error {
	name := asset.GetName()
	if !source.DetectSBOMsFile(name) {
		return nil // Skip non-SBOM extensions
	}

	// download SBOMs
	reader, _, err := client.Repositories.DownloadReleaseAsset(ctx.Context, owner, repo, asset.GetID(), http.DefaultClient)
	if err != nil {
		return fmt.Errorf("failed to download asset %s: %w", name, err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read asset %s: %w", name, err)
	}

	// Validate SBOM
	if !source.IsSBOMFile(content) {
		logger.LogDebug(ctx.Context, "Asset is not a valid SBOM", "repo", repo, "asset", name)
		return nil
	}

	sourceAdapter := ctx.Value("source")
	// Check uniqueness
	primaryCompName, primaryCompVersion := utils.ConstructProjectName(ctx, "", "", "", "", content, sourceAdapter.(string))
	componentVersion := fmt.Sprintf("%s:%s", primaryCompName, primaryCompVersion)

	cache.RLock()
	if cache.SBOMs[componentVersion] {
		logger.LogDebug(ctx.Context, "SBOM already processed", "repo", repo, "asset", name, "component", componentVersion)
		cache.RUnlock()
		return nil
	}
	cache.RUnlock()

	// Log and yield SBOM
	logger.LogInfo(ctx.Context, "Found new SBOM", "repo", repo, "asset", name, "component", primaryCompName, "version", primaryCompVersion)
	sbomChan <- &iterator.SBOM{
		Data:      content,
		Path:      name,
		Namespace: repo,
	}

	// Update SBOM cache
	cache.Lock()
	cache.SBOMs[componentVersion] = true
	cache.Unlock()

	return nil
}
