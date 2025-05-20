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

type Cache struct {
	Data map[string]AdapterCache `json:"data"`
	sync.RWMutex
}

type AdapterCache map[string]GitHubDaemonCache

type GitHubDaemonCache map[string]MethodCache

type MethodCache struct {
	Repos map[string]RepoState `json:"repos"`
	SBOMs map[string]bool      `json:"sboms"`
}

// RepoState stores release information.
type RepoState struct {
	PublishedAt string `json:"published_at"`
	ReleaseID   string `json:"release_id"`
}

// NewCache initializes a cache.
func NewCache() *Cache {
	return &Cache{
		Data: make(map[string]AdapterCache),
	}
}

func (c *Cache) LoadCache(ctx tcontext.TransferMetadata, path string) error {
	c.Lock()
	defer c.Unlock()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		logger.LogDebug(ctx.Context, "Cache file does not exist, starting with empty cache", "path", path)
		return nil // Cache doesn't exist
	}

	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		logger.LogDebug(ctx.Context, "Failed to parse cache file, starting with empty cache", "path", path, "error", err)
		return nil // Invalid cache
	}

	logger.LogDebug(ctx.Context, "Successfully loaded cache", "path", path)
	return nil
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

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	logger.LogDebug(ctx.Context, "Successfully saved cache", "path", path)

	return nil
}

func (c *Cache) EnsureCachePath(ctx tcontext.TransferMetadata, outputAdapter, inputAdapter string) {
	c.Lock()
	defer c.Unlock()

	if _, exists := c.Data[outputAdapter]; !exists {
		c.Data[outputAdapter] = make(AdapterCache)
	}

	if _, exists := c.Data[outputAdapter][inputAdapter]; !exists {
		c.Data[outputAdapter][inputAdapter] = make(GitHubDaemonCache)
	}

	// intialize all methods
	for _, method := range []string{string(MethodAPI), string(MethodReleases), string(MethodTool)} {
		if _, exists := c.Data[outputAdapter][inputAdapter][method]; !exists {
			c.Data[outputAdapter][inputAdapter][method] = MethodCache{
				Repos: make(map[string]RepoState),
				SBOMs: make(map[string]bool),
			}
		}
	}
	logger.LogDebug(ctx.Context, "Initialized cache paths", "output", outputAdapter, "input", inputAdapter, "methods", []string{"release", "api", "tool"})
}

func (f *GithubWatcherFetcher) Fetch(ctx tcontext.TransferMetadata, config *GithubConfig) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Starting GitHub watcher", "repo", config.Repo, "version", config.Version)

	client, err := config.GetGitHubClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	var filterdRepos []string
	if config.Repo == "" {

		repos, err := GetAllOrgRepositories(ctx, client, config.Owner)
		if err != nil {
			return nil, fmt.Errorf("failed to get repositories: %w", err)
		}

		if len(repos) == 0 {
			return nil, fmt.Errorf("no repositories left after applying filters")
		}

		filterdRepos = config.applyRepoFilters(ctx, repos)

	}
	filterdRepos = append(filterdRepos, config.Repo)
	if len(filterdRepos) == 0 {
		return nil, fmt.Errorf("no repositories found")
	}

	logger.LogDebug(ctx.Context, "Filtered repositories to watch out", "repos", filterdRepos)

	// initiate cache
	cache := NewCache()
	if err := cache.LoadCache(ctx, CACHE_PATH); err != nil {
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

	outputAdapter := ctx.Value("destination").(string)
	fmt.Println("adapter", outputAdapter)

	// Ensure cache paths for all methods
	cache.EnsureCachePath(ctx, outputAdapter, "github")

	// Compare with cache
	cache.RLock()
	cached, exists := cache.Data[outputAdapter]["github"][method].Repos[repo]
	cache.RUnlock()

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

	if exists && cached.PublishedAt == publishedAt && cached.ReleaseID == releaseID {
		logger.LogDebug(ctx.Context, "No new release found", "repo", repo)
		return nil
	}

	logger.LogInfo(ctx.Context, "New release detected", "repo", repo, "release_id", releaseID, "published_at", publishedAt)

	// once the new released is out, fetch SBOMs based on the configured method
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
		if err := fetchSBOMUsingTool(ctx, client, owner, repo, latestRelease, releaseID, publishedAt, binaryPath, cache, sbomChan); err != nil {
			logger.LogError(ctx.Context, err, "Failed to generate SBOM with tool", "repo", repo)
		}
	default:
		return fmt.Errorf("unsupported GitHub method: %s", method)
	}

	// update repository cache with latest release info
	cache.Lock()
	cache.Data[outputAdapter]["github"][method].Repos[repo] = RepoState{
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
		logger.LogDebug(ctx.Context, "Asset is not a SBOM file via it's extention", "repo", repo, "asset", name)
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
		logger.LogDebug(ctx.Context, "Asset is not a valid SBOM from it's content", "repo", repo, "asset", name)
		return nil
	}

	logger.LogDebug(ctx.Context, "Valid SBOM found", "repo", repo, "asset", name)

	// create unique cache key for the SBOM (repo:release_id:filename)
	sbomCacheKey := fmt.Sprintf("%s:%s:%s", repo, releaseID, name)
	outputAdapter := ctx.Value("destination").(string)

	cache.RLock()
	if cache.Data[outputAdapter]["github"][string(MethodReleases)].SBOMs[sbomCacheKey] {
		logger.LogDebug(ctx.Context, "SBOM already processed", "repo", repo, "asset", name, "cache_key", sbomCacheKey)
		cache.RUnlock()
		return nil
	}
	cache.RUnlock()

	// pass SBOM to the channel
	logger.LogInfo(ctx.Context, "Found new SBOM", "repo", repo, "asset", name)
	sbomChan <- &iterator.SBOM{
		Data:      content,
		Path:      name,
		Version:   releaseID,
		Namespace: fmt.Sprintf("%s-%s", owner, repo),
	}

	// update SBOM cache
	cache.Lock()
	cache.Data[outputAdapter]["github"][string(MethodReleases)].SBOMs[sbomCacheKey] = true
	logger.LogDebug(ctx.Context, "Updated SBOM cache", "repo", repo, "asset", name, "cache_key", sbomCacheKey)
	cache.Unlock()

	return nil
}

// fetchSBOMFromReleaseAssets fetches SBOMs from the release assets.
func fetchSBOMFromReleaseAssets(ctx tcontext.TransferMetadata, client *githublib.Client, owner, repo string, release *githublib.RepositoryRelease, releaseID, publishedAt string, cache *Cache, sbomChan chan *iterator.SBOM) error {
	logger.LogDebug(ctx.Context, "Fetching SBOMs from release assets", "repo", repo)

	opt := &githublib.ListOptions{PerPage: 100}
	var allAssets []*githublib.ReleaseAsset
	page := 1

	for {
		assets, resp, err := client.Repositories.ListReleaseAssets(ctx.Context, owner, repo, release.GetID(), opt)
		if err != nil {
			logger.LogError(ctx.Context, err, "Failed to fetch release assets", "repo", repo, "page", page)
			return fmt.Errorf("failed to list release assets: %w", err)
		}
		allAssets = append(allAssets, assets...)
		logger.LogDebug(ctx.Context, "Fetched release assets", "repo", repo, "page", page, "assets_fetched", len(assets), "total_so_far", len(allAssets))

		if resp.NextPage == 0 {
			logger.LogInfo(ctx.Context, "Completed fetching all release assets", "repo", repo, "total_assets", len(allAssets))
			break
		}
		opt.Page = resp.NextPage
		page++
	}

	logger.LogDebug(ctx.Context, "Fetched assets", "repo", repo, "count", len(allAssets))

	// process assets
	for _, asset := range allAssets {
		if err := processAsset(ctx, client, owner, repo, releaseID, asset, cache, sbomChan); err != nil {
			logger.LogError(ctx.Context, err, "Failed to process asset", "repo", repo, "asset", asset.GetName())
		}
	}

	return nil
}

// fetchSBOMFromDependencyGraph fetches an SBOM from the GitHub Dependency Graph API.
func fetchSBOMFromDependencyGraph(ctx tcontext.TransferMetadata, client *githublib.Client, owner, repo, releaseID, publishedAt string, cache *Cache, sbomChan chan *iterator.SBOM) error {
	logger.LogDebug(ctx.Context, "Fetching SBOM from Dependency Graph API", "repo", repo)

	sbomCacheKey := fmt.Sprintf("%s:%s:dependency-graph-sbom.json", repo, releaseID)
	outputAdapter := ctx.Value("destination").(string)

	cache.RLock()
	if cache.Data[outputAdapter]["github"][string(MethodAPI)].SBOMs[sbomCacheKey] {
		logger.LogDebug(ctx.Context, "SBOM already processed", "repo", repo, "cache_key", sbomCacheKey)
		cache.RUnlock()
		return nil
	}
	cache.RUnlock()

	// get SBOM from Dependency Graph API
	dependencyGraph, _, err := client.DependencyGraph.GetSBOM(ctx.Context, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to fetch SBOM from Dependency Graph API: %w", err)
	}

	sbomData, err := json.Marshal(dependencyGraph.SBOM)
	if err != nil {
		return fmt.Errorf("failed to marshal SBOM: %w", err)
	}

	filepath := fmt.Sprintf("%s-%s-dependency-graph-sbom.json", owner, repo)
	logger.LogInfo(ctx.Context, "Found new SBOM from Dependency Graph API", "repo", repo)
	sbomChan <- &iterator.SBOM{
		Data:      sbomData,
		Path:      filepath,
		Version:   releaseID,
		Namespace: fmt.Sprintf("%s-%s", owner, repo),
	}

	cache.Lock()
	cache.Data[outputAdapter]["github"][string(MethodAPI)].SBOMs[sbomCacheKey] = true
	logger.LogDebug(ctx.Context, "Updated SBOM cache", "repo", repo, "cache_key", sbomCacheKey)
	cache.Unlock()

	return nil
}

// fetchSBOMUsingTool generates an SBOM using the Syft tool for the repository at the release's commit.
func fetchSBOMUsingTool(ctx tcontext.TransferMetadata, client *githublib.Client, owner, repo string, release *githublib.RepositoryRelease, releaseID, publishedAt, binaryPath string, cache *Cache, sbomChan chan *iterator.SBOM) error {
	logger.LogDebug(ctx.Context, "Generating SBOM with Syft tool", "repo", repo)

	sbomCacheKey := fmt.Sprintf("%s:%s:syft-generated-sbom.json", repo, releaseID)
	outputAdapter := ctx.Value("destination").(string)
	cache.RLock()
	if cache.Data[outputAdapter]["github"][string(MethodTool)].SBOMs[sbomCacheKey] {
		logger.LogDebug(ctx.Context, "SBOM already generated", "repo", repo, "cache_key", sbomCacheKey)
		cache.RUnlock()
		return nil
	}
	cache.RUnlock()

	// get release commit SHA
	releaseCommit, _, err := client.Repositories.GetCommit(ctx.Context, owner, repo, release.GetTargetCommitish(), nil)
	if err != nil {
		return fmt.Errorf("failed to get release commit: %w", err)
	}
	commitSHA := releaseCommit.GetSHA()

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
		Version:   releaseID,
		Namespace: fmt.Sprintf("%s-%s", owner, repo),
	}

	cache.Lock()
	cache.Data[outputAdapter]["github"][string(MethodTool)].SBOMs[sbomCacheKey] = true
	logger.LogDebug(ctx.Context, "Updated SBOM cache", "repo", repo, "cache_key", sbomCacheKey)
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
