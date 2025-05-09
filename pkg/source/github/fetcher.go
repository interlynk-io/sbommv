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
	"fmt"
	"sync"
	"time"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"golang.org/x/time/rate"
)

type SBOMFetcher interface {
	Fetch(ctx tcontext.TransferMetadata, config *GithubConfig) (iterator.SBOMIterator, error)
}

type SequentialFetcher struct{}

func (f *SequentialFetcher) Fetch(ctx tcontext.TransferMetadata, config *GithubConfig) (iterator.SBOMIterator, error) {
	// Implement the logic to fetch SBOMs sequentially
	logger.LogDebug(ctx.Context, "Fetching SBOMs Sequentially")

	repos, err := config.client.GetAllRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	// filtering to include/exclude repos
	repos = config.applyRepoFilters(ctx, repos)

	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories left after applying filters")
	}

	logger.LogDebug(ctx.Context, "Total repos from which SBOMs will be fetched", "count", len(repos), "repos", repos)
	logger.LogDebug(ctx.Context, "Processing Mode", "strategy", config.ProcessingMode)

	var sbomList []*iterator.SBOM
	giter := &GitHubIterator{client: config.client, binaryPath: config.BinaryPath}

	// Iterate over repositories one by one (sequential processing)
	for _, repo := range repos {
		giter.client.updateRepo(repo)

		logger.LogDebug(ctx.Context, "Repository", "value", repo)

		switch GitHubMethod(config.Method) {

		case MethodAPI:

			releaseSBOM, err := giter.fetchSBOMFromAPI(ctx)
			if err != nil {
				logger.LogDebug(ctx.Context, "Failed to fetch SBOMs from API Method for", "repo", repo, "error", err)
				continue
			}
			if len(releaseSBOM) > 0 {
				sbomList = append(sbomList, releaseSBOM...)
			}

		case MethodReleases:

			releaseSBOMs, err := giter.fetchSBOMFromReleases(ctx)
			if err != nil {
				logger.LogDebug(ctx.Context, "Failed to fetch SBOMs from Release Method for", "repo", repo, "error", err)
				continue
			}
			if len(releaseSBOMs) > 0 {
				sbomList = append(sbomList, releaseSBOMs...)
			}

		case MethodTool:

			releaseSBOM, err := giter.fetchSBOMFromTool(ctx)
			if err != nil {
				logger.LogDebug(ctx.Context, "Failed to generate SBOMs via Tool Method for", "repo", repo, "error", err)
				continue
			}

			if len(releaseSBOM) > 0 {
				sbomList = append(sbomList, releaseSBOM...)
			}

		default:
			return nil, fmt.Errorf("unsupported GitHub method: %s", config.Method)
		}

	}

	if len(sbomList) == 0 {
		return nil, fmt.Errorf("no SBOMs found for any repository")
	}
	logger.LogDebug(ctx.Context, "Total SBOMs fetched from all repos", "count", len(sbomList))

	return &GitHubIterator{
		sboms: sbomList,
	}, nil
}

type ParallelFetcher struct{}

func (f *ParallelFetcher) Fetch(ctx tcontext.TransferMetadata, config *GithubConfig) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Fetching SBOMs Parallely")

	repos, err := config.client.GetAllRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	// filtering to include/exclude repos
	repos = config.applyRepoFilters(ctx, repos)

	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories left after applying filters")
	}

	logger.LogDebug(ctx.Context, "Total repos from which SBOMs will be fetched", "count", len(repos), "repos", repos)
	logger.LogDebug(ctx.Context, "Processing Mode", "strategy", config.ProcessingMode)

	const maxWorkers = 5
	const requestsPerSecond = 5

	repoChan := make(chan string, len(repos))
	sbomsChan := make(chan []*iterator.SBOM, len(repos))

	var wg sync.WaitGroup

	// Rate limiter to respect GitHub API limits
	limiter := rate.NewLimiter(rate.Every(time.Second/requestsPerSecond), requestsPerSecond)

	// Start worker goroutines
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range repoChan {
				// Apply rate limiting
				if err := limiter.Wait(ctx.Context); err != nil {
					logger.LogDebug(ctx.Context, "Rate limiter error", "repo", repo, "error", err)
					continue
				}

				config.client.updateRepo(repo)
				iter := NewGitHubIterator(ctx, config, repo)

				var repoSboms []*iterator.SBOM
				var err error

				switch GitHubMethod(config.Method) {
				case MethodAPI:
					repoSboms, err = iter.fetchSBOMFromAPI(ctx)
					if err == nil {
						logger.LogDebug(ctx.Context, "Total SBOM fetched from API method", "count", len(repoSboms), "repo", repo, "error", err)
					}

				case MethodReleases:
					repoSboms, err = iter.fetchSBOMFromReleases(ctx)
					if err == nil {
						logger.LogDebug(ctx.Context, "Total SBOM fetched from release method", "count", len(repoSboms), "repo", repo, "error", err)
					}

				case MethodTool:
					repoSboms, err = iter.fetchSBOMFromTool(ctx)
					if err == nil {
						logger.LogDebug(ctx.Context, "Total SBOM fetched from tool method", "count", len(repoSboms), "repo", repo, "error", err)
					}

				default:
					logger.LogInfo(ctx.Context, "Unsupported method", "repo", repo, "method", config.Method)
					err = fmt.Errorf("unsupported method: %s", config.Method)
				}

				// only send SBOMs if fetch succeeded (no error)
				if err == nil && len(repoSboms) > 0 {
					logger.LogDebug(ctx.Context, "Fetched SBOMs", "repo", repo, "count", len(repoSboms))
					sbomsChan <- repoSboms
				} else {
					logger.LogInfo(ctx.Context, "Skipping SBOMs due to fetch error or no SBOMs found", "repo", repo, "error", err)
				}
			}
		}()
	}

	// Distribute repositories to workers
	for _, repo := range repos {
		repoChan <- repo
	}
	close(repoChan)

	// Wait for all workers to complete and close the results channel
	wg.Wait()
	close(sbomsChan)

	// Collect all SBOMs
	var finalSbomList []*iterator.SBOM
	for repoSboms := range sbomsChan {
		finalSbomList = append(finalSbomList, repoSboms...)
	}

	if len(finalSbomList) == 0 {
		return nil, fmt.Errorf("no SBOMs found for any repository")
	}
	logger.LogDebug(ctx.Context, "Total SBOMs fetched from all repos", "count", len(finalSbomList))

	return &GitHubIterator{sboms: finalSbomList}, nil
}
