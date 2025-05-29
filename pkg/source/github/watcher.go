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
	"time"

	githublib "github.com/google/go-github/v62/github"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/source"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type GithubWatcherFetcher struct{}

func NewWatcherFetcher() *GithubWatcherFetcher {
	return &GithubWatcherFetcher{}
}

func (f *GithubWatcherFetcher) Fetch(ctx tcontext.TransferMetadata, config *GithubConfig) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Starting GitHub watcher", "repo", config.Repo, "version", config.Version)

	outputAdapter := ctx.Value("destination").(string)
	method := config.Method

	// Initialize cache with SQLite and in-memory caching
	cache := NewCache()
	if err := cache.InitCache(ctx, outputAdapter, method); err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	if err := cache.LoadCache(ctx, outputAdapter, method); err != nil {
		return nil, fmt.Errorf("failed to load cache: %w", err)
	}

	// Ensure cache paths for all methods
	cache.EnsureCachePath(ctx, outputAdapter, "github")

	sbomChan := make(chan *iterator.SBOM, 10)

	client, err := config.GetGitHubClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	var finalRepoList []string

	if config.Repo == "" && config.Owner != "" {

		// get all repos under that organization/owner
		repos, err := GetAllOrgRepositories(ctx, client, config.Owner)
		if err != nil {
			return nil, fmt.Errorf("failed to get repositories: %w", err)
		}

		if len(repos) == 0 {
			return nil, fmt.Errorf("no repositories found under organization/owner: %s", config.Owner)
		}

		// filter repos based on the provided icluded/excluded repos
		finalRepoList = config.applyRepoFilters(ctx, repos)
		if len(finalRepoList) == 0 {
			return nil, fmt.Errorf("no repositories found post filtering")
		}
	}

	finalRepoList = append(finalRepoList, config.Repo)
	if len(finalRepoList) == 0 {
		return nil, fmt.Errorf("no repositories found")
	}

	logger.LogDebug(ctx.Context, "Final repositories list to watch out", "repos", finalRepoList)

	// start polling loop in a goroutine
	go func() {
		defer close(sbomChan)
		ticker := time.NewTicker(time.Duration(config.Poll) * time.Second)
		logger.LogDebug(ctx.Context, "Started polling", "interval", config.Poll)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Context.Done():
				logger.LogInfo(ctx.Context, "Polling stopped")
				return
			case <-ticker.C:

				newReleaseDetected := false

				for _, repo := range finalRepoList {
					if err := pollRepository(ctx, client, repo, config.Owner, config.Method, config.BinaryPath, config.AssetWaitDelay, cache, sbomChan, &newReleaseDetected); err != nil {
						logger.LogError(ctx.Context, err, "Failed to poll repository", "repo", repo)
					}
				}
			}
		}
	}()

	return &GithubWatcherIterator{sbomChan: sbomChan}, nil
}

// pollRepository checks a single repository for new releases and fetches SBOMs based on the configured method.
func pollRepository(ctx tcontext.TransferMetadata, client *githublib.Client, repo, owner, method, binaryPath string, assetWaitDelay int64, cache *Cache, sbomChan chan *iterator.SBOM, newReleaseDetected *bool) error {
	logger.LogDebug(ctx.Context, "Polling repository", "repo", repo, "time", time.Now().Format(time.RFC3339))

	outputAdapter := ctx.Value("destination").(string)

	var releases []*githublib.RepositoryRelease
	var resp *githublib.Response
	var err error

	// get all releases
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

	// extract the release ID, published date and tag name from the latest release
	releaseID := fmt.Sprintf("%d", latestRelease.GetID())
	publishedAt := latestRelease.GetPublishedAt().Format(time.RFC3339)
	tagName := latestRelease.GetTagName()

	// now compare these release infor with the cached one
	// check cache whether this repo has been processed for this release
	cache.RLock()
	repoInfo, exists := cache.Data[outputAdapter]["github"][method].Repos[repo]
	cache.RUnlock()

	if exists && repoInfo.PublishedAt == publishedAt && repoInfo.ReleaseID == releaseID {
		logger.LogDebug(ctx.Context, "No new release found", "repo", repo)
		return nil
	}

	logger.LogInfo(ctx.Context, "New release detected", "repo", repo, "tag", tagName, "release_id", releaseID, "published_at", publishedAt)

	// *newReleaseDetected = true

	// wait before fetching assets to allow GitHub Actions/workflows to upload them
	if assetWaitDelay > 0 {
		logger.LogDebug(ctx.Context, "Waiting before fetching assets", "repo", repo, "tag", tagName, "delay_seconds", assetWaitDelay)
		select {
		case <-time.After(time.Duration(assetWaitDelay) * time.Second):
			// Continue after delay
		case <-ctx.Context.Done():
			logger.LogInfo(ctx.Context, "Context cancelled during asset wait", "repo", repo)
			return ctx.Context.Err()
		}
	}

	// after the new released is confirmed, fetch SBOMs based on the configured method
	switch method {
	case string(MethodAPI):
		if err := fetchSBOMFromDependencyGraph(ctx, client, owner, repo, releaseID, publishedAt, tagName, cache, sbomChan); err != nil {
			logger.LogError(ctx.Context, err, "Failed to fetch SBOM from Dependency Graph API", "repo", repo)
		}

	case string(MethodReleases):
		if err := fetchSBOMFromReleaseAssets(ctx, client, owner, repo, latestRelease, releaseID, publishedAt, tagName, cache, sbomChan); err != nil {
			logger.LogError(ctx.Context, err, "Failed to fetch SBOM from release assets", "repo", repo)
		}

	case string(MethodTool):
		if err := fetchSBOMUsingTool(ctx, client, owner, repo, latestRelease, releaseID, publishedAt, tagName, binaryPath, cache, sbomChan); err != nil {
			logger.LogError(ctx.Context, err, "Failed to generate SBOM with tool", "repo", repo)
		}

	default:
		return fmt.Errorf("unsupported GitHub method: %s", method)
	}

	// update cache with latest repository release info
	cache.Lock()
	repoState := cache.Data[outputAdapter]["github"][method].Repos[repo]
	repoState.PublishedAt = publishedAt
	repoState.ReleaseID = releaseID
	cache.Data[outputAdapter]["github"][method].Repos[repo] = repoState
	cache.Unlock()

	// cache.Lock()
	// cache.ensureCachePathFor(outputAdapter, "github", method) // Initialize path
	// cache.Data[outputAdapter]["github"][method].Repos[repo] = RepoState{
	// 	PublishedAt: publishedAt,
	// 	ReleaseID:   releaseID,
	// }
	// cache.Unlock()

	// Save cache immediately to persist this daemon's update
	if err := cache.SaveCache(ctx, outputAdapter, method); err != nil {
		logger.LogError(ctx.Context, err, "Failed to save cache after new release", "repo", repo)
	}

	logger.LogDebug(ctx.Context, "Updated cache for repository", "repo", repo, "tag", tagName, "published_at", publishedAt, "release_id", releaseID)
	return nil
}

func processAsset(ctx tcontext.TransferMetadata, client *githublib.Client, owner, repo, releaseID, tagName string, asset *githublib.ReleaseAsset, cache *Cache, sbomChan chan *iterator.SBOM) error {
	logger.LogDebug(ctx.Context, "Processing asset", "repo", repo, "tag", tagName, "asset", asset.GetName())
	assetName := asset.GetName()

	if !source.DetectSBOMsFile(assetName) {
		logger.LogDebug(ctx.Context, "asset is not a SBOM from it's extention", "repo", repo, "asset", assetName)
		return nil
	}

	// download SBOMs
	reader, _, err := client.Repositories.DownloadReleaseAsset(ctx.Context, owner, repo, asset.GetID(), http.DefaultClient)
	if err != nil {
		return fmt.Errorf("failed to download asset %s: %w", assetName, err)
	}
	defer reader.Close()
	logger.LogDebug(ctx.Context, "downloaded asset", "repo", repo, "tag", tagName, "asset", assetName)

	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read asset %s: %w", assetName, err)
	}

	// Validate SBOM
	if !source.IsSBOMFile(content) {
		logger.LogDebug(ctx.Context, "asset content is not a SBOM", "repo", repo, "asset", assetName)
		return nil
	}

	logger.LogDebug(ctx.Context, "Valid SBOM found", "repo", repo, "tag", tagName, "asset", assetName)

	outputAdapter := ctx.Value("destination").(string)

	// create unique cache key for the SBOM (owner:repo:tagName:filename)
	sbomCacheKey := fmt.Sprintf("%s:%s:%s:%s", owner, repo, tagName, assetName)
	processed := cache.IsSBOMProcessed(ctx, outputAdapter, "github", string(MethodReleases), sbomCacheKey, repo)
	if processed {
		logger.LogDebug(ctx.Context, "SBOM already processed", "repo", "sbom_key", sbomCacheKey, "method", MethodReleases)
		return nil
	}

	// pass SBOM to the channel
	logger.LogInfo(ctx.Context, "Found new SBOM", "repo", repo, "tag", tagName, "asset", assetName)
	sbomChan <- &iterator.SBOM{
		Data:      content,
		Path:      assetName,
		Version:   tagName,
		Namespace: fmt.Sprintf("%s-%s", owner, repo),
	}

	// update SBOM cache
	cache.MarkSBOMProcessed(ctx, outputAdapter, "github", string(MethodReleases), sbomCacheKey, repo)
	return nil
}

// fetchSBOMFromReleaseAssets fetches SBOMs from the release assets.
func fetchSBOMFromReleaseAssets(ctx tcontext.TransferMetadata, client *githublib.Client, owner, repo string, release *githublib.RepositoryRelease, releaseID, publishedAt, tagName string, cache *Cache, sbomChan chan *iterator.SBOM) error {
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
		logger.LogDebug(ctx.Context, "Fetched release assets", "repo", repo, "tag", tagName, "page", page, "assets_fetched", len(assets), "total_so_far", len(allAssets))

		if resp.NextPage == 0 {
			logger.LogInfo(ctx.Context, "Completed fetching all release assets", "repo", repo, "total_assets", len(allAssets))
			break
		}
		opt.Page = resp.NextPage
		page++
	}

	logger.LogDebug(ctx.Context, "Fetched assets", "repo", repo, "tag", tagName, "count", len(allAssets))

	// process each assets
	for _, asset := range allAssets {
		if err := processAsset(ctx, client, owner, repo, releaseID, tagName, asset, cache, sbomChan); err != nil {
			logger.LogError(ctx.Context, err, "Failed to process asset", "repo", repo, "asset", asset.GetName())
		}
	}

	return nil
}

// fetchSBOMFromDependencyGraph fetches an SBOM from the GitHub Dependency Graph API.
func fetchSBOMFromDependencyGraph(ctx tcontext.TransferMetadata, client *githublib.Client, owner, repo, releaseID, publishedAt, tagName string, cache *Cache, sbomChan chan *iterator.SBOM) error {
	logger.LogDebug(ctx.Context, "Fetching SBOM from Dependency Graph API", "repo", repo, "tag", tagName)

	sbomCacheKey := fmt.Sprintf("%s:%s:%s:dependency-graph-sbom.json", owner, repo, tagName)
	outputAdapter := ctx.Value("destination").(string)

	processed := cache.IsSBOMProcessed(ctx, outputAdapter, "github", string(MethodAPI), sbomCacheKey, repo)
	if processed {
		logger.LogDebug(ctx.Context, "SBOM already processed", "repo", "sbom_key", sbomCacheKey, "method", MethodAPI)
		return nil
	}

	// get SBOM from Dependency Graph API
	dependencyGraph, _, err := client.DependencyGraph.GetSBOM(ctx.Context, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to fetch SBOM from Dependency Graph API: %w", err)
	}

	sbomData, err := json.Marshal(dependencyGraph.SBOM)
	if err != nil {
		return fmt.Errorf("failed to marshal SBOM: %w", err)
	}

	filepath := "dependency-graph-sbom.json"
	logger.LogInfo(ctx.Context, "Found new SBOM from Dependency Graph API", "repo", repo)
	sbomChan <- &iterator.SBOM{
		Data:      sbomData,
		Path:      filepath,
		Version:   tagName,
		Namespace: fmt.Sprintf("%s-%s", owner, repo),
	}

	cache.MarkSBOMProcessed(ctx, outputAdapter, "github", string(MethodAPI), sbomCacheKey, repo)
	return nil
}

// fetchSBOMUsingTool generates an SBOM using the Syft tool for the repository at the release's commit.
func fetchSBOMUsingTool(ctx tcontext.TransferMetadata, client *githublib.Client, owner, repo string, release *githublib.RepositoryRelease, releaseID, publishedAt, tagName, binaryPath string, cache *Cache, sbomChan chan *iterator.SBOM) error {
	logger.LogDebug(ctx.Context, "Generating SBOM with Syft tool", "repo", repo, "tag", tagName)

	sbomCacheKey := fmt.Sprintf("%s:%s:%s:syft-generated-sbom.json", owner, repo, tagName)
	outputAdapter := ctx.Value("destination").(string)

	processed := cache.IsSBOMProcessed(ctx, outputAdapter, "github", string(MethodTool), sbomCacheKey, repo)
	if processed {
		logger.LogDebug(ctx.Context, "SBOM already processed", "repo", "sbom_key", sbomCacheKey, "method", MethodTool)
		return nil
	}

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

	filepath := "syft-generated-sbom.json"
	logger.LogInfo(ctx.Context, "Generated new SBOM with Syft", "repo", repo, "tag", tagName)
	sbomChan <- &iterator.SBOM{
		Data:      sbomData,
		Path:      filepath,
		Version:   tagName,
		Namespace: fmt.Sprintf("%s-%s", owner, repo),
	}

	cache.MarkSBOMProcessed(ctx, outputAdapter, "github", string(MethodTool), sbomCacheKey, repo)
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
