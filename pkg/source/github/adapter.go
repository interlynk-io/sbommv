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
	"github.com/interlynk-io/sbommv/pkg/reporter"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/interlynk-io/sbommv/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GitHubAdapter handles fetching SBOMs from GitHub releases
type GitHubAdapter struct {
	config *GitHubConfig
	client *Client
	Role   types.AdapterRole
}

// NewGitHubAdapter creates a new GitHub adapter with the given config and client
func NewGitHubAdapter(config *GitHubConfig, client *Client) *GitHubAdapter {
	return &GitHubAdapter{
		config: config,
		client: client,
	}
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

	// Updated to StringSlice to support multiple values (comma-separated)
	cmd.Flags().StringSlice("in-github-include-repos", nil, "Include only these repositories e.g sbomqs,sbomasm")
	cmd.Flags().StringSlice("in-github-exclude-repos", nil, "Exclude these repositories e.g sbomqs,sbomasm")

	// (Optional) If you plan to fetch **all versions** of a repo
	// cmd.Flags().Bool("in-github-all-versions", false, "Fetch SBOMs from all versions")
}

// ParseAndValidateParams validates the GitHub adapter params
func (g *GitHubAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var (
		urlFlag, methodFlag, includeFlag, excludeFlag, githubBranchFlag string
		missingFlags                                                    []string
		invalidFlags                                                    []string
	)

	switch g.Role {
	case types.InputAdapterRole:
		urlFlag = "in-github-url"
		methodFlag = "in-github-method"
		includeFlag = "in-github-include-repos"
		excludeFlag = "in-github-exclude-repos"
		githubBranchFlag = "in-github-branch"

	case types.OutputAdapterRole:
		return fmt.Errorf("The GitHub adapter doesn't support output adapter functionalities.")

	default:
		return fmt.Errorf("The adapter is neither an input type nor an output type")
	}

	// Extract GitHub URL
	githubURL, _ := cmd.Flags().GetString(urlFlag)
	if githubURL == "" {
		missingFlags = append(missingFlags, "--"+urlFlag)
	}

	includeRepos, _ := cmd.Flags().GetStringSlice(includeFlag)
	excludeRepos, _ := cmd.Flags().GetStringSlice(excludeFlag)

	// Validate GitHub URL to determine if it's an org or repo
	owner, repo, version, err := utils.ParseGithubURL(githubURL)
	if err != nil {
		return fmt.Errorf("invalid GitHub URL format: %w", err)
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

	g.config.IncludeRepos = includeRepos
	g.config.ExcludeRepos = excludeRepos

	// Validate that both include & exclude are not used together
	if len(g.config.IncludeRepos) > 0 && len(g.config.ExcludeRepos) > 0 {
		return fmt.Errorf("cannot use both --in-github-include-repos and --in-github-exclude-repos together")
	}

	if method == "tool" {
		binaryPath, err := utils.GetBinaryPath()
		if err != nil {
			return fmt.Errorf("failed to get Syft binary: %w", err)
		}

		g.config.BinaryPath = binaryPath
		logger.LogDebug(context.Background(), "Binary Path", "value", g.config.BinaryPath)
	}

	token := viper.GetString("GITHUB_TOKEN")
	if token == "" {
		logger.LogDebug(cmd.Context(), "GitHub Token not found in environment")
	}

	if version != "" && method == "api" {
		return fmt.Errorf("version flag is not supported for GitHub API method")
	}

	// Assign extracted values to struct
	if version == "" {
		version = "latest"
		g.config.URL = githubURL
	} else {
		g.config.URL = fmt.Sprintf("https://github.com/%s/%s", owner, repo)
	}

	g.config.Owner = owner
	g.config.Repo = repo
	g.config.Branch = branch
	g.config.Version = version
	g.config.Method = method
	g.config.GithubToken = token

	// Initialize GitHub client
	g.client = NewClient(g)

	// Debugging logs for tracking
	logger.LogDebug(cmd.Context(), "Parsed GitHub parameters",
		"url", g.config.URL,
		"owner", g.config.Owner,
		"branch", g.config.Branch,
		"repo", g.config.Repo,
		"version", g.config.Version,
		"include_repos", g.config.IncludeRepos,
		"exclude_repos", g.config.ExcludeRepos,
		"method", g.config.Method,
		"token", g.config.GithubToken,
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

	processingMode := types.FetchSequential
	var sbomIterator iterator.SBOMIterator

	switch processingMode {
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

// OutputSBOMs should return an error since GitHub does not support SBOM uploads
func (g *GitHubAdapter) UploadSBOMs(ctx *tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	return fmt.Errorf("GitHub adapter does not support SBOM uploading")
}

// pkg/source/github/adapter.go (update)
func (g *GitHubAdapter) DryRun(ctx *tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	reporter := reporter.NewSBOMReporter(false, "") // Configurable later
	return reporter.DryRun(ctx.Context, iter)
}

// // DryRun for Input Adapter: Displays all fetched SBOMs from input adapter
// func (g *GitHubAdapter) DryRun(ctx *tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
// 	logger.LogDebug(ctx.Context, "Dry-run mode: Displaying SBOMs fetched from input adapter")

// 	var outputDir string
// 	var verbose bool

// 	processor := sbom.NewSBOMProcessor(outputDir, verbose)
// 	sbomCount := 0
// 	fmt.Println()
// 	fmt.Printf("📦 Details of all Fetched SBOMs by Input Adapter\n")

// 	for {

// 		sbom, err := iterator.Next(ctx.Context)
// 		if err == io.EOF {
// 			break // No more SBOMs
// 		}
// 		if err != nil {
// 			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
// 			continue
// 		}
// 		// Update processor with current SBOM data
// 		processor.Update(sbom.Data, sbom.Namespace, sbom.Path)

// 		doc, err := processor.ProcessSBOMs()
// 		if err != nil {
// 			logger.LogError(ctx.Context, err, "Failed to process SBOM")
// 			continue
// 		}

// 		// If outputDir is provided, save the SBOM file
// 		if outputDir != "" {
// 			if err := processor.WriteSBOM(doc, sbom.Namespace); err != nil {
// 				logger.LogError(ctx.Context, err, "Failed to write SBOM to output directory")
// 			}
// 		}

// 		// Print SBOM content if verbose mode is enabled
// 		if verbose {
// 			fmt.Println("\n-------------------- 📜 SBOM Content --------------------")
// 			fmt.Printf("📂 Filename: %s\n", doc.Filename)
// 			fmt.Printf("📦 Format: %s | SpecVersion: %s\n\n", doc.Format, doc.SpecVersion)
// 			fmt.Println(string(doc.Content))
// 			fmt.Println("------------------------------------------------------")
// 			fmt.Println()
// 		}

// 		sbomCount++
// 		fmt.Printf(" - 📁 Repo: %s | Format: %s | SpecVersion: %s | Filename: %s \n", sbom.Namespace, doc.Format, doc.SpecVersion, doc.Filename)

// 		// logger.LogInfo(ctx.Context, fmt.Sprintf("%d. Repo: %s | Format: %s | SpecVersion: %s | Filename: %s",
// 		// 	sbomCount, sbom.Repo, doc.Format, doc.SpecVersion, doc.Filename))
// 	}
// 	fmt.Printf("📊 Total SBOMs are: %d\n", sbomCount)

// 	logger.LogDebug(ctx.Context, "Dry-run mode completed for input adapter", "total_sboms", sbomCount)
// 	return nil
// }

// applyRepoFilters filters repositories based on inclusion/exclusion flags
func (g *GitHubAdapter) applyRepoFilters(repos []string) []string {
	includedRepos := make(map[string]bool)
	excludedRepos := make(map[string]bool)

	for _, repo := range g.config.IncludeRepos {
		if repo != "" {
			includedRepos[strings.TrimSpace(repo)] = true
		}
	}

	for _, repo := range g.config.ExcludeRepos {
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
			g.config.Repo = repo
			g.client.Repo = repo

			iter := NewGitHubIterator(ctx, g, repo)

			// Fetch SBOMs separately
			err := iter.HandleSBOMFetchingViaIterator(ctx, GitHubMethod(g.config.Method))
			if err != nil {
				logger.LogDebug(ctx.Context, "Failed to fetch SBOMs for repo", "repo", repo)
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
		g.config.Repo = repo // Set current repository

		logger.LogDebug(ctx.Context, "Fetching SBOMs sequentially", "repo", repo)

		iter := NewGitHubIterator(ctx, g, repo)

		// Fetch SBOMs separately
		err := iter.HandleSBOMFetchingViaIterator(ctx, GitHubMethod(g.config.Method))
		if err != nil {
			logger.LogInfo(ctx.Context, "Failed to fetch SBOMs for", "repo", repo)
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
