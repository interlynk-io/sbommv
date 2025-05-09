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
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	githublib "github.com/google/go-github/v62/github"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/source"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

const (
	CACHE_PATH = ".sbommv/cache.json"
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
func (c *Cache) SaveCache(ctx tcontext.TransferMetadata, path string) error {
	logger.LogDebug(ctx.Context, "Saving cache to file", "path", path)
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
	logger.LogDebug(ctx.Context, "Starting GitHub watcher", "repo", config.Repo, "version", config.Version)

	repos, err := config.client.GetAllRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	filterdRepos := config.applyRepoFilters(ctx, repos)

	logger.LogDebug(ctx.Context, "Filtered repositories to watch out", "repos", filterdRepos)

	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories left after applying filters")
	}

	client, err := config.GetGitHubClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// initiate cache
	cache := NewCache()
	if err := cache.LoadCache(CACHE_PATH); err != nil {
		return nil, fmt.Errorf("failed to load cache: %w", err)
	}

	sbomChan := make(chan *iterator.SBOM, 10)

	// start polling loop in a goroutine
	go func() {
		defer close(sbomChan)
		ticker := time.NewTicker(time.Duration(config.Poll) * time.Second)
		logger.LogDebug(ctx.Context, "Starting polling loop", "interval", config.Poll)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Context.Done():
				logger.LogInfo(ctx.Context, "Polling stopped")
				return
			case <-ticker.C:
				for _, repo := range filterdRepos {
					if err := pollRepository(ctx, client, repo, config.Owner, config.Method, config.BinaryPath, cache, sbomChan); err != nil {
						logger.LogError(ctx.Context, err, "Failed to poll repository", "repo", repo)
					}
				}
				if err := cache.SaveCache(ctx, CACHE_PATH); err != nil {
					logger.LogError(ctx.Context, err, "Failed to save cache")
				}
			}
		}
	}()

	return &GithubWatcherIterator{sbomChan: sbomChan}, nil
}

// pollRepository checks a single repository for new releases and fetches SBOMs based on the configured method.
func pollRepository(ctx tcontext.TransferMetadata, client *githublib.Client, repo, owner, method, binaryPath string, cache *Cache, sbomChan chan *iterator.SBOM) error {
	logger.LogDebug(ctx.Context, "Polling repository", "repo", repo, "time", time.Now().Format(time.RFC3339))

	var releases []*githublib.RepositoryRelease
	var resp *githublib.Response
	var err error

	// list all releases
	releases, resp, err = client.Repositories.ListReleases(ctx.Context, owner, repo, &githublib.ListOptions{PerPage: 1})
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

	// extract latest release
	latestRelease := releases[0]

	// get the release ID and published date
	releaseID := fmt.Sprintf("%d", latestRelease.GetID())
	publishedAt := latestRelease.GetPublishedAt().Format(time.RFC3339)

	// compare with cache
	cache.RLock()
	cached, exists := cache.Repos[repo]
	cache.RUnlock()

	if exists && cached.PublishedAt == publishedAt && cached.ReleaseID == releaseID {
		logger.LogDebug(ctx.Context, "No new release found", "repo", repo)
		return nil
	}

	logger.LogInfo(ctx.Context, "New release detected", "repo", repo, "release_id", releaseID, "published_at", publishedAt)

	switch method {
	case string(MethodAPI):
		if err := fetchSBOMFromDependencyGraph(ctx, client, owner, repo, releaseID, publishedAt, cache, sbomChan); err != nil {
			logger.LogError(ctx.Context, err, "Failed to fetch SBOM from Dependency Graph API", "repo", repo)
		}
	case string(MethodReleases):
		if err := fetchSBOMFromReleaseAssets(ctx, client, owner, repo, latestRelease, releaseID, publishedAt, cache, sbomChan); err != nil {
			logger.LogError(ctx.Context, err, "Failed to fetch SBOM from release assets", "repo", repo)
		}
	case string(MethodTool):
		if err := fetchSBOMFromTool(ctx, client, owner, repo, latestRelease, releaseID, publishedAt, binaryPath, cache, sbomChan); err != nil {
			logger.LogError(ctx.Context, err, "Failed to generate SBOM with tool", "repo", repo)
		}
	default:
		return fmt.Errorf("unsupported GitHub method: %s", method)
	}

	// update repository cache with latest release info
	cache.Lock()
	cache.Repos[repo] = RepoState{
		PublishedAt: publishedAt,
		ReleaseID:   releaseID,
	}
	cache.Unlock()

	logger.LogDebug(ctx.Context, "Updated cache for repository", "repo", repo, "published_at", publishedAt, "release_id", releaseID)

	return nil
}

func processAsset(ctx tcontext.TransferMetadata, client *githublib.Client, owner, repo, releaseID string, asset *githublib.ReleaseAsset, cache *Cache, sbomChan chan *iterator.SBOM) error {
	logger.LogDebug(ctx.Context, "Processing asset", "repo", repo, "asset", asset.GetName())
	name := asset.GetName()

	if !source.DetectSBOMsFile(name) {
		// skip non-SBOM extensions
		return nil
	}

	// download SBOMs
	reader, _, err := client.Repositories.DownloadReleaseAsset(ctx.Context, owner, repo, asset.GetID(), http.DefaultClient)
	if err != nil {
		return fmt.Errorf("failed to download asset %s: %w", name, err)
	}
	defer reader.Close()
	logger.LogDebug(ctx.Context, "Downloaded asset", "repo", repo, "asset", name)

	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read asset %s: %w", name, err)
	}

	// Validate SBOM
	if !source.IsSBOMFile(content) {
		logger.LogDebug(ctx.Context, "Asset is not a valid SBOM", "repo", repo, "asset", name)
		return nil
	}

	logger.LogDebug(ctx.Context, "Valid SBOM found", "repo", repo, "asset", name)

	// create unique cache key for the SBOM (repo:release_id:filename)
	cacheKey := fmt.Sprintf("%s:%s:%s", repo, releaseID, name)

	cache.RLock()
	if cache.SBOMs[cacheKey] {
		logger.LogDebug(ctx.Context, "SBOM already processed", "repo", repo, "asset", name, "cache_key", cacheKey)
		cache.RUnlock()
		return nil
	}
	cache.RUnlock()

	logger.LogDebug(ctx.Context, "Valid SBOM found", "repo", repo, "asset", name)

	// pass SBOM to the channel
	logger.LogInfo(ctx.Context, "Found new SBOM", "repo", repo, "asset", name)
	sbomChan <- &iterator.SBOM{
		Data:      content,
		Path:      name,
		Namespace: repo,
	}

	// update SBOM cache
	cache.Lock()
	cache.SBOMs[cacheKey] = true
	logger.LogDebug(ctx.Context, "Updated SBOM cache", "repo", repo, "asset", name, "cache_key", cacheKey)
	cache.Unlock()

	return nil
}

// fetchSBOMFromReleaseAssets fetches SBOMs from the release assets.
func fetchSBOMFromReleaseAssets(ctx tcontext.TransferMetadata, client *githublib.Client, owner, repo string, release *githublib.RepositoryRelease, releaseID, publishedAt string, cache *Cache, sbomChan chan *iterator.SBOM) error {
	logger.LogDebug(ctx.Context, "Fetching SBOMs from release assets", "repo", repo)

	// fetch assets
	assets, _, err := client.Repositories.ListReleaseAssets(ctx.Context, owner, repo, release.GetID(), nil)
	if err != nil {
		return fmt.Errorf("failed to list assets: %w", err)
	}

	// process assets
	for _, asset := range assets {
		if err := processAsset(ctx, client, owner, repo, releaseID, asset, cache, sbomChan); err != nil {
			logger.LogError(ctx.Context, err, "Failed to process asset", "repo", repo, "asset", asset.GetName())
		}
	}

	return nil
}

// fetchSBOMFromDependencyGraph fetches an SBOM from the GitHub Dependency Graph API.
func fetchSBOMFromDependencyGraph(ctx tcontext.TransferMetadata, client *githublib.Client, owner, repo, releaseID, publishedAt string, cache *Cache, sbomChan chan *iterator.SBOM) error {
	logger.LogDebug(ctx.Context, "Fetching SBOM from Dependency Graph API", "repo", repo)

	// get SBOM from Dependency Graph API
	dependencyGraph, _, err := client.DependencyGraph.GetSBOM(ctx.Context, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to fetch SBOM from Dependency Graph API: %w", err)
	}

	sbomData, err := json.Marshal(dependencyGraph.SBOM)
	if err != nil {
		return fmt.Errorf("failed to marshal SBOM: %w", err)
	}

	cacheKey := fmt.Sprintf("%s:%s:api", repo, releaseID)

	cache.RLock()
	if cache.SBOMs[cacheKey] {
		logger.LogDebug(ctx.Context, "SBOM already processed", "repo", repo, "cache_key", cacheKey)
		cache.RUnlock()
		return nil
	}
	cache.RUnlock()

	filepath := fmt.Sprintf("%s-%s-dependency-graph-sbom.json", owner, repo)
	logger.LogInfo(ctx.Context, "Found new SBOM from Dependency Graph API", "repo", repo)
	sbomChan <- &iterator.SBOM{
		Data:      sbomData,
		Path:      filepath,
		Namespace: repo,
	}

	cache.Lock()
	cache.SBOMs[cacheKey] = true
	logger.LogDebug(ctx.Context, "Updated SBOM cache", "repo", repo, "cache_key", cacheKey)
	cache.Unlock()

	return nil
}

// fetchSBOMFromTool generates an SBOM using the Syft tool for the repository at the release's commit.
func fetchSBOMFromTool(ctx tcontext.TransferMetadata, client *githublib.Client, owner, repo string, release *githublib.RepositoryRelease, releaseID, publishedAt, binaryPath string, cache *Cache, sbomChan chan *iterator.SBOM) error {
	logger.LogDebug(ctx.Context, "Generating SBOM with Syft tool", "repo", repo)

	// get release commit SHA
	releaseCommit, _, err := client.Repositories.GetCommit(ctx.Context, owner, repo, release.GetTargetCommitish(), nil)
	if err != nil {
		return fmt.Errorf("failed to get release commit: %w", err)
	}
	commitSHA := releaseCommit.GetSHA()

	cacheKey := fmt.Sprintf("%s:%s:tool", repo, releaseID)

	cache.RLock()
	if cache.SBOMs[cacheKey] {
		logger.LogDebug(ctx.Context, "SBOM already generated", "repo", repo, "cache_key", cacheKey)
		cache.RUnlock()
		return nil
	}
	cache.RUnlock()

	// clone repository at the release commit
	repoDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s-%s", owner, repo, releaseID))
	defer os.RemoveAll(repoDir)

	if err := cloneRepoWithGit(ctx, repo, owner, commitSHA, repoDir); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// generate SBOM
	sbomData, err := GenerateSBOM(ctx, repoDir, binaryPath)
	if err != nil {
		return fmt.Errorf("failed to generate SBOM: %w", err)
	}

	filepath := fmt.Sprintf("%s-%s-syft-generated-sbom.json", owner, repo)
	logger.LogInfo(ctx.Context, "Generated new SBOM with Syft", "repo", repo)
	sbomChan <- &iterator.SBOM{
		Data:      sbomData,
		Path:      filepath,
		Namespace: repo,
	}

	cache.Lock()
	cache.SBOMs[cacheKey] = true
	logger.LogDebug(ctx.Context, "Updated SBOM cache", "repo", repo, "cache_key", cacheKey)
	cache.Unlock()

	return nil
}

// cloneRepoWithGit clones a GitHub repository at the specified commit using git.
func cloneRepoWithGit(ctx tcontext.TransferMetadata, repo, owner, commitSHA, targetDir string) error {
	logger.LogDebug(ctx.Context, "Cloning repository", "repo", repo, "commit", commitSHA, "directory", targetDir)

	// ensure git is installed
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed")
	}

	// Clone repository
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
	cmd := exec.CommandContext(ctx.Context, "git", "clone", "--depth=1", repoURL, targetDir)
	var stderr strings.Builder
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w, stderr: %s", err, stderr.String())
	}

	// Checkout specific commit
	cmd = exec.CommandContext(ctx.Context, "git", "checkout", commitSHA)
	cmd.Dir = targetDir
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout commit %s: %w, stderr: %s", commitSHA, err, stderr.String())
	}

	logger.LogDebug(ctx.Context, "Repository cloned successfully", "repo", repo, "commit", commitSHA)
	return nil
}
