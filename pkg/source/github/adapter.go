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

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/interlynk-io/sbommv/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GitHubAdapter handles fetching SBOMs from GitHub releases
type GitHubAdapter struct {
	URL         string
	Repo        string
	Owner       string
	Version     string
	Branch      string
	Method      string
	BinaryPath  string
	client      *Client
	GithubToken string
	Role        types.AdapterRole

	// Comma-separated list (e.g., "repo1,repo2")
	IncludeRepos []string
	ExcludeRepos []string
}

type ProcessingMode string

const (
	FetchParallel   ProcessingMode = "parallel"
	FetchSequential ProcessingMode = "sequential"
)

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
	cmd.Flags().String("in-github-url", "", "GitHub repository URL")
	cmd.Flags().String("in-github-method", "api", "GitHub method: release, api, or tool")
	cmd.Flags().String("in-github-branch", "", "Github repository branch")

	// Updated to StringSlice to support multiple values (comma-separated)
	cmd.Flags().StringSlice("in-github-include-repos", nil, "Include only these repositories e.g sbomqs,sbomasm")
	cmd.Flags().StringSlice("in-github-exclude-repos", nil, "Exclude these repositories e.g sbomqs,sbomasm")

	// (Optional) If you plan to fetch **all versions** of a repo
	// cmd.Flags().Bool("in-github-all-versions", false, "Fetch SBOMs from all versions")
}

// ParseAndValidateParams validates the GitHub adapter params
func (g *GitHubAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var urlFlag, methodFlag, includeFlag, excludeFlag, githubBranchFlag string

	if g.Role == types.InputAdapter {
		urlFlag = "in-github-url"
		methodFlag = "in-github-method"
		includeFlag = "in-github-include-repos"
		excludeFlag = "in-github-exclude-repos"
		githubBranchFlag = "in-github-branch"
	}

	// Extract GitHub URL
	githubURL, _ := cmd.Flags().GetString(urlFlag)
	if githubURL == "" {
		return fmt.Errorf("missing or invalid flag: %s", urlFlag)
	}

	method, _ := cmd.Flags().GetString(methodFlag)
	if method != "release" && method != "api" && method != "tool" {
		return fmt.Errorf("missing or invalid flag: %s", methodFlag)
	}

	branch, _ := cmd.Flags().GetString(githubBranchFlag)

	includeRepos, _ := cmd.Flags().GetStringSlice(includeFlag)
	excludeRepos, _ := cmd.Flags().GetStringSlice(excludeFlag)

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

	// Parse URL into owner, repo, and version
	owner, repo, version, err := utils.ParseGithubURL(githubURL)
	if err != nil {
		return fmt.Errorf("invalid GitHub URL format: %w", err)
	}

	if version != "" && method == "api" {
		return fmt.Errorf("version flag is not supported for GitHub API method")
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
	)
	return nil
}

// FetchSBOMs initializes the GitHub SBOM iterator using the unified method
func (g *GitHubAdapter) FetchSBOMs(ctx *tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Intializing SBOM fetching process")

	// Org Mode: Fetch all repositories
	repos, err := g.client.GetAllRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	logger.LogDebug(ctx.Context, "Found repos", "number", len(repos))

	// filtering to include/exclude repos
	repos = g.applyRepoFilters(repos)

	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories left after applying filters")
	}

	logger.LogDebug(ctx.Context, "Listing repos of organization after filtering", "values", repos)

	processingMode := FetchSequential
	var sbomIterator iterator.SBOMIterator

	switch ProcessingMode(processingMode) {
	case FetchParallel:
		sbomIterator, err = g.fetchSBOMsConcurrently(ctx, repos)
	case FetchSequential:
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

// OutputSBOMs should return an error since GitHub does not support SBOM uploads
func (g *GitHubAdapter) UploadSBOMs(ctx *tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	return fmt.Errorf("GitHub adapter does not support SBOM uploading")
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

// fetchSBOMsConcurrently: fetch SBOMs from repositories concurrently
func (g *GitHubAdapter) fetchSBOMsConcurrently(ctx *tcontext.TransferMetadata, repos []string) (iterator.SBOMIterator, error) {
	var wg sync.WaitGroup
	sbomsChan := make(chan *iterator.SBOM, len(repos))

	for _, repo := range repos {
		wg.Add(1)
		go func(repo string) {
			defer wg.Done()
			g.Repo = repo
			g.client.Repo = repo
			iter, err := NewGitHubIterator(ctx, g, repo)
			if err != nil {
				logger.LogError(ctx.Context, err, "Failed to fetch SBOMs for repo", "repo", repo)
				return
			}
			for {
				sbom, err := iter.Next(ctx.Context)
				if err == io.EOF {
					break
				}
				if err != nil {
					logger.LogError(ctx.Context, err, "Error reading SBOM for", "repo", repo)
					break
				}
				sbomsChan <- sbom
			}
		}(repo)
	}

	wg.Wait()
	close(sbomsChan)

	// Collect SBOMs from channel
	var sbomList []*iterator.SBOM
	for sbom := range sbomsChan {
		sbomList = append(sbomList, sbom)
	}

	return &GitHubIterator{
		sboms: sbomList,
	}, nil
}

// fetchSBOMsSequentially: fetch SBOMs from repositories one at a time
func (g *GitHubAdapter) fetchSBOMsSequentially(ctx *tcontext.TransferMetadata, repos []string) (iterator.SBOMIterator, error) {
	var sbomList []*iterator.SBOM

	// Iterate over repositories one by one (sequential processing)
	for _, repo := range repos {
		g.Repo = repo // Set current repository

		logger.LogDebug(ctx.Context, "Fetching SBOMs sequentially", "repo", repo)

		// Fetch SBOMs for the current repository
		iter, err := NewGitHubIterator(ctx, g, repo)
		if err != nil {
			logger.LogError(ctx.Context, err, "Failed to fetch SBOMs for repo", "repo", repo)
			continue
		}

		// use iterator to add the SBOMs to the final sboms list
		for {
			sbom, err := iter.Next(ctx.Context)
			if err == io.EOF {
				break
			}
			if err != nil {
				logger.LogError(ctx.Context, err, "Error reading SBOM for", "repo", repo)
				break
			}
			sbomList = append(sbomList, sbom)
		}
	}

	if len(sbomList) == 0 {
		return nil, fmt.Errorf("no SBOMs found for any repository")
	}

	return &GitHubIterator{
		sboms: sbomList,
	}, nil
}

// DryRun for Input Adapter: Displays retrieved SBOMs without uploading
func (g *GitHubAdapter) DryRun(ctx *tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Dry-run mode: Displaying SBOMs fetched from input adapter")

	var outputDir string
	var verbose bool

	processor := sbom.NewSBOMProcessor(outputDir, verbose)
	sbomCount := 0
	fmt.Println()

	for {
		sbom, err := iterator.Next(ctx.Context)
		if err == io.EOF {
			break // No more SBOMs
		}
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			continue
		}

		doc, err := processor.ProcessSBOMs(sbom.Data, sbom.Repo, sbom.Path)
		if err != nil {
			logger.LogError(ctx.Context, err, "Failed to process SBOM")
			continue
		}

		// If outputDir is provided, save the SBOM file
		if outputDir != "" {
			if err := processor.WriteSBOM(doc, sbom.Repo); err != nil {
				logger.LogError(ctx.Context, err, "Failed to write SBOM to output directory")
			}
		}

		sbomCount++
		fmt.Printf("Repo: %s | Format: %s | SpecVersion: %s | Filename: %s \n", sbom.Repo, doc.Format, doc.SpecVersion, doc.Filename)

		// logger.LogInfo(ctx.Context, fmt.Sprintf("%d. Repo: %s | Format: %s | SpecVersion: %s | Filename: %s",
		// 	sbomCount, sbom.Repo, doc.Format, doc.SpecVersion, doc.Filename))
	}

	logger.LogDebug(ctx.Context, "Dry-run mode completed for input adapter", "total_sboms", sbomCount)
	return nil
}
