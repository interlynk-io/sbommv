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
	"strings"
	"sync"
	"time"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/interlynk-io/sbommv/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

// GitHubAdapter handles fetching SBOMs from GitHub releases
type GitHubAdapter struct {
	URL            string
	Repo           string
	Owner          string
	Version        string
	Branch         string
	Method         string
	BinaryPath     string
	client         *Client
	GithubToken    string
	Role           types.AdapterRole
	ProcessingMode types.ProcessingMode
	Daemon         bool

	// Comma-separated list (e.g., "repo1,repo2")
	IncludeRepos []string
	ExcludeRepos []string
}

type GitHubMethod string

const (
	// MethodReleases searches for SBOMs in GitHub releases
	MethodReleases GitHubMethod = "release"

	// // MethodReleases searches for SBOMs in GitHub releases
	MethodAPI GitHubMethod = "api"

	// MethodGenerate clones the repo and generates SBOMs using external Tools
	MethodTool GitHubMethod = "tool"
)

// AddCommandParams adds GitHub-specific CLI flags
func (g *GitHubAdapter) AddCommandParams(cmd *cobra.Command) {
	cmd.Flags().String("in-github-url", "", "GitHub organization or repository URL")
	cmd.Flags().String("in-github-method", "api", "GitHub method: release, api, or tool")
	cmd.Flags().String("in-github-branch", "", "Github repository branch")
	cmd.Flags().String("in-github-version", "", "github repo version")

	// Updated to StringSlice to support multiple values (comma-separated)
	cmd.Flags().StringSlice("in-github-include-repos", nil, "Include only these repositories e.g sbomqs,sbomasm")
	cmd.Flags().StringSlice("in-github-exclude-repos", nil, "Exclude these repositories e.g sbomqs,sbomasm")

	// (Optional) If you plan to fetch **all versions** of a repo
	// cmd.Flags().Bool("in-github-all-versions", false, "Fetch SBOMs from all versions")
}

// ParseAndValidateParams validates the GitHub adapter params
func (g *GitHubAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var (
		urlFlag, methodFlag, includeFlag, excludeFlag, githubBranchFlag, githubVersionFlag string
		missingFlags                                                                       []string
		invalidFlags                                                                       []string
	)

	switch g.Role {
	case types.InputAdapterRole:
		urlFlag = "in-github-url"
		methodFlag = "in-github-method"
		includeFlag = "in-github-include-repos"
		excludeFlag = "in-github-exclude-repos"
		githubBranchFlag = "in-github-branch"
		githubVersionFlag = "in-github-version"

	case types.OutputAdapterRole:
		return fmt.Errorf("The GitHub adapter doesn't support output adapter functionalities.")

	default:
		return fmt.Errorf("The adapter is neither an input type nor an output type")
	}
	// validate flags for respective adapters
	utils.FlagValidation(cmd, types.GithubAdapterType, types.InputAdapterFlagPrefix)

	// Extract GitHub URL
	githubURL, _ := cmd.Flags().GetString(urlFlag)
	if githubURL == "" {
		missingFlags = append(missingFlags, "--"+urlFlag)
	}

	includeRepos, _ := cmd.Flags().GetStringSlice(includeFlag)
	excludeRepos, _ := cmd.Flags().GetStringSlice(excludeFlag)

	// Validate GitHub URL to determine if it's an org or repo
	owner, repo, err := utils.ParseGithubURL(githubURL)
	if err != nil {
		return fmt.Errorf("invalid GitHub URL format: %w", err)
	}

	version, _ := cmd.Flags().GetString(githubVersionFlag)
	if version == "" {
		version = "latest"
	}

	// If repo is present (i.e., single repo URL), filtering flags should NOT be used
	if repo != "" {
		if len(includeRepos) > 0 || len(excludeRepos) > 0 {
			return fmt.Errorf(
				"Filtering flags (--in-github-include-repos / --in-github-exclude-repos) can only be used with an organization URL(i.e. https://github.com/<organization>), not a single repository(i.e https://github.com/<organization>/<repo>)",
			)
		}
	}

	validMethods := map[string]bool{"release": true, "api": true, "tool": true}

	// Extract GitHub method
	method, _ := cmd.Flags().GetString(methodFlag)
	if !validMethods[method] {
		invalidFlags = append(invalidFlags, fmt.Sprintf("%s=%s (must be one of: release, api, tool)", methodFlag, method))
	}

	// Extract branch (only valid for "tool" method)
	branch, _ := cmd.Flags().GetString(githubBranchFlag)
	if branch != "" && method != "tool" {
		invalidFlags = append(invalidFlags, fmt.Sprintf("--%s is only supported for --in-github-method=tool, whereas it's not supported for --in-github-method=api and --in-github-method=release", githubBranchFlag))
	}

	// Validate include & exclude repos cannot be used together
	if len(includeRepos) > 0 && len(excludeRepos) > 0 {
		invalidFlags = append(invalidFlags, fmt.Sprintf("Cannot use both %s and %s together", includeFlag, excludeFlag))
	}

	// Validate required flags
	if len(missingFlags) > 0 {
		return fmt.Errorf("missing input adapter required flags: %v\n\nUse 'sbommv transfer --help' for usage details.", missingFlags)
	}

	// Validate incorrect flag usage
	if len(invalidFlags) > 0 {
		return fmt.Errorf("invalid input adapter flag usage:\n %s\n\nUse 'sbommv transfer --help' for correct usage.", strings.Join(invalidFlags, "\n "))
	}

	g.IncludeRepos = includeRepos
	g.ExcludeRepos = excludeRepos

	// Validate that both include & exclude are not used together
	if len(g.IncludeRepos) > 0 && len(g.ExcludeRepos) > 0 {
		return fmt.Errorf("cannot use both --in-github-include-repos and --in-github-exclude-repos together")
	}

	if method == "tool" {
		binaryPath, err := utils.GetBinaryPath()
		if err != nil {
			return fmt.Errorf("failed to get Syft binary: %w", err)
		}

		g.BinaryPath = binaryPath
		logger.LogDebug(context.Background(), "Binary Path", "value", g.BinaryPath)
	}

	token := viper.GetString("GITHUB_TOKEN")
	if token == "" {
		logger.LogDebug(cmd.Context(), "GitHub Token not found in environment")
	}

	if method == "api" && version != "latest" {
		fmt.Println("Github API method calculates SBOM for a complete repo not for any particular version: ", version)
	}

	// Assign extracted values to struct
	if version == "" {
		version = "latest"
		g.URL = githubURL
	} else {
		g.URL = fmt.Sprintf("https://github.com/%s/%s", owner, repo)
	}

	g.Owner = owner
	g.Repo = repo
	g.Branch = branch
	g.Version = version
	g.Method = method
	g.GithubToken = token

	// Initialize GitHub client
	g.client = NewClient(g)

	// Debugging logs for tracking
	logger.LogDebug(cmd.Context(), "Parsed GitHub parameters",
		"url", g.URL,
		"owner", g.Owner,
		"branch", g.Branch,
		"repo", g.Repo,
		"version", g.Version,
		"include_repos", g.IncludeRepos,
		"exclude_repos", g.ExcludeRepos,
		"method", g.Method,
		"token", g.GithubToken,
		"processing_mode", g.ProcessingMode,
	)
	return nil
}

// FetchSBOMs initializes the GitHub SBOM iterator using the unified method
func (g *GitHubAdapter) FetchSBOMs(ctx tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Intializing SBOM fetching process")

	// Org Mode: Fetch all repositories
	repos, err := g.client.GetAllRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	// filtering to include/exclude repos
	repos = g.applyRepoFilters(repos)

	if len(repos) == 1 {
	}
	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories left after applying filters")
	}

	logger.LogDebug(ctx.Context, "SBOMs will be fetched from these repos", "values", repos, "count", len(repos))

	logger.LogDebug(ctx.Context, "Processing Mode", "strategy", g.ProcessingMode)

	var sbomIterator iterator.SBOMIterator

	switch g.ProcessingMode {
	case types.FetchParallel:
		sbomIterator, err = g.fetchSBOMsConcurrently(ctx, repos)
	case types.FetchSequential:
		sbomIterator, err = g.fetchSBOMsSequentially(ctx, repos)
	default:
		return nil, fmt.Errorf("Unsupported Processing Mode !!")
	}

	if err != nil {
		logger.LogError(ctx.Context, err, "Failed to fetch SBOMs via Processing Mode")
		return nil, err
	}

	return sbomIterator, err
}

func (g *GitHubAdapter) Monitor(ctx tcontext.TransferMetadata) (iterator.SBOMIterator, tcontext.TransferMetadata, error) {
	return nil, ctx, fmt.Errorf("Currently gitHub adapter does not support monitoring")
}

// OutputSBOMs should return an error since GitHub does not support SBOM uploads
func (g *GitHubAdapter) UploadSBOMs(ctx tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	return fmt.Errorf("GitHub adapter does not support SBOM uploading")
}

// DryRun for Input Adapter: Displays all fetched SBOMs from input adapter
func (g *GitHubAdapter) DryRun(ctx tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Dry-run mode: Displaying SBOMs fetched from input adapter")

	var outputDir string
	var verbose bool

	processor := sbom.NewSBOMProcessor(outputDir, verbose)
	sbomCount := 0
	fmt.Println()
	fmt.Printf("📦 Details of all Fetched SBOMs by Input Adapter\n")

	for {

		sbom, err := iterator.Next(ctx)
		if err == io.EOF {
			break // No more SBOMs
		}
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			continue
		}
		// Update processor with current SBOM data
		processor.Update(sbom.Data, sbom.Namespace, sbom.Path)

		doc, err := processor.ProcessSBOMs()
		if err != nil {
			logger.LogError(ctx.Context, err, "Failed to process SBOM")
			continue
		}

		// If outputDir is provided, save the SBOM file
		if outputDir != "" {
			if err := processor.WriteSBOM(doc, sbom.Namespace); err != nil {
				logger.LogError(ctx.Context, err, "Failed to write SBOM to output directory")
			}
		}

		// Print SBOM content if verbose mode is enabled
		if verbose {
			fmt.Println("\n-------------------- 📜 SBOM Content --------------------")
			fmt.Printf("📂 Filename: %s\n", doc.Filename)
			fmt.Printf("📦 Format: %s | SpecVersion: %s\n\n", doc.Format, doc.SpecVersion)
			fmt.Println(string(doc.Content))
			fmt.Println("------------------------------------------------------")
			fmt.Println()
		}

		sbomCount++
		fmt.Printf(" - 📁 Repo: %s | Format: %s | SpecVersion: %s | Filename: %s \n", sbom.Namespace, doc.Format, doc.SpecVersion, doc.Filename)

		// logger.LogInfo(ctx.Context, fmt.Sprintf("%d. Repo: %s | Format: %s | SpecVersion: %s | Filename: %s",
		// 	sbomCount, sbom.Repo, doc.Format, doc.SpecVersion, doc.Filename))
	}
	fmt.Printf("📊 Total SBOMs are: %d\n", sbomCount)

	logger.LogDebug(ctx.Context, "Dry-run mode completed for input adapter", "total_sboms", sbomCount)
	return nil
}

// applyRepoFilters filters repositories based on inclusion/exclusion flags
func (g *GitHubAdapter) applyRepoFilters(repos []string) []string {
	includedRepos := make(map[string]bool)
	excludedRepos := make(map[string]bool)

	for _, repo := range g.IncludeRepos {
		if repo != "" {
			includedRepos[strings.TrimSpace(repo)] = true
		}
	}

	for _, repo := range g.ExcludeRepos {
		if repo != "" {
			excludedRepos[strings.TrimSpace(repo)] = true
		}
	}

	var filteredRepos []string

	for _, repo := range repos {
		if _, isExcluded := excludedRepos[repo]; isExcluded {
			continue // Skip excluded repositories
		}

		// Include only if in the inclusion list (if provided)
		if len(includedRepos) > 0 {
			if _, isIncluded := includedRepos[repo]; !isIncluded {
				continue // Skip repos that are not in the include list
			}
		}
		// filtered repo are added to the final list
		filteredRepos = append(filteredRepos, repo)
	}

	return filteredRepos
}

func (g *GitHubAdapter) fetchWatcher(ctx tcontext.TransferMetadata, repos []string) (iterator.SBOMIterator, error) {
	logger.LogInfo(ctx.Context, "Monitoring SBOM via github adapter currently doesn't support")
	return nil, nil
}

func (g *GitHubAdapter) fetchSBOMsConcurrently(ctx tcontext.TransferMetadata, repos []string) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Fetching SBOMs concurrently")
	const maxWorkers = 5        // Number of concurrent workers (adjustable)
	const requestsPerSecond = 5 // Rate limit for GitHub API requests

	// Channels for distributing work and collecting results
	repoChan := make(chan string, len(repos))
	sbomsChan := make(chan []*iterator.SBOM, len(repos))

	// WaitGroup to track worker completion
	var wg sync.WaitGroup

	// Rate limiter to respect GitHub API limits
	limiter := rate.NewLimiter(rate.Every(time.Second/requestsPerSecond), requestsPerSecond)

	// Start worker goroutines
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range repoChan {

				repoURL := fmt.Sprintf("https://github.com/%s/%s", g.Owner, repo)
				logger.LogDebug(ctx.Context, "Fetching SBOMs", "repo", repo, "url", repoURL)

				g.client.Repo = repo
				g.client.RepoURL = repoURL

				iter := NewGitHubIterator(ctx, g, repo)

				var repoSboms []*iterator.SBOM

				// Apply rate limiting
				if err := limiter.Wait(ctx.Context); err != nil {
					logger.LogDebug(ctx.Context, "Rate limiter error", "repo", repo, "error", err)
					continue
				}

				// Fetch SBOMs with retry logic
				for attempt := 1; attempt <= 3; attempt++ {
					var err error

					switch GitHubMethod(g.Method) {

					case MethodAPI:
						repoSboms, err = iter.fetchSBOMFromAPI(ctx)
						logger.LogDebug(ctx.Context, "Total SBOM detched from API method", "count", len(repoSboms), "repo", repo)

					case MethodReleases:
						repoSboms, err = iter.fetchSBOMFromReleases(ctx)
						logger.LogDebug(ctx.Context, "Total SBOM detched from release method", "count", len(repoSboms), "repo", repo)

					case MethodTool:
						repoSboms, err = iter.fetchSBOMFromTool(ctx)
						logger.LogDebug(ctx.Context, "Total SBOM detched from release", "count", len(repoSboms), "repo", repo)

					default:
						logger.LogInfo(ctx.Context, "Unsupported method", "repo", repo, "method", g.Method)
						continue
					}

					if err == nil {
						break // Success, exit retry loop
					}
					logger.LogInfo(ctx.Context, "Retry attempt", "attempt", attempt, "repo", repo, "error", err)
					time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
				}

				if len(repoSboms) == 0 {
					logger.LogInfo(ctx.Context, "No SBOMs found", "repo", repo)
					continue
				}

				logger.LogDebug(ctx.Context, "Fetched SBOMs", "repo", repo, "count", len(repoSboms))
				sbomsChan <- repoSboms
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
	logger.LogDebug(ctx.Context, "Total SBOMs fetched from all repos", "count", len(finalSbomList))

	if len(finalSbomList) == 0 {
		return nil, fmt.Errorf("no SBOMs found for any repository")
	}

	// Return an iterator with the collected SBOMs
	return &GitHubIterator{sboms: finalSbomList}, nil
}

// fetchSBOMsSequentially: fetch SBOMs from repositories one at a time
func (g *GitHubAdapter) fetchSBOMsSequentially(ctx tcontext.TransferMetadata, repos []string) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Fetching SBOMs sequentially")
	fmt.Println("Fetching SBOMs sequentially")

	var sbomList []*iterator.SBOM
	giter := &GitHubIterator{client: g.client}

	// Iterate over repositories one by one (sequential processing)
	for _, repo := range repos {
		g.Repo = repo // Set current repository

		giter.client.Repo = repo

		logger.LogDebug(ctx.Context, "Repository", "value", repo)

		switch GitHubMethod(g.Method) {

		case MethodAPI:

			releaseSBOM, err := giter.fetchSBOMFromAPI(ctx)
			if err != nil {
				logger.LogInfo(ctx.Context, "Failed to fetch SBOMs from API Method for", "repo", repo)
				continue
			}
			sbomList = append(sbomList, releaseSBOM...)

		case MethodReleases:

			releaseSBOMs, err := giter.fetchSBOMFromReleases(ctx)
			if err != nil {
				logger.LogInfo(ctx.Context, "Failed to fetch SBOMs from Release Method for", "repo", repo)
				continue
			}
			fmt.Println("releaseSBOMs: ", len(releaseSBOMs))
			sbomList = append(sbomList, releaseSBOMs...)

		case MethodTool:

			releaseSBOM, err := giter.fetchSBOMFromTool(ctx)
			if err != nil {
				logger.LogInfo(ctx.Context, "Failed to fetch SBOMs from Tool Method for", "repo", repo)
				continue
			}
			sbomList = append(sbomList, releaseSBOM...)

		default:
			return nil, fmt.Errorf("unsupported GitHub method: %s", g.Method)
		}

	}

	if len(sbomList) == 0 {
		return nil, fmt.Errorf("no SBOMs found for any repository")
	}

	fmt.Println("finalSbomList: ", len(sbomList))

	return &GitHubIterator{
		sboms: sbomList,
	}, nil
}
