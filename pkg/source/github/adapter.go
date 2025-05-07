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
	"strconv"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/interlynk-io/sbommv/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GitHubAdapter handles fetching SBOMs from GitHub releases
type GitHubAdapter struct {
	Config  *GithubConfig
	Role    types.AdapterRole
	Fetcher SBOMFetcher
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
	cmd.Flags().String("in-github-token", "", "GitHub token (required for more than 5000/hour rate limit)")
	cmd.Flags().String("in-github-poll-interval", "60", "Polling interval to check GitHub Releases (default: 60s)")

	// Updated to StringSlice to support multiple values (comma-separated)
	cmd.Flags().StringSlice("in-github-include-repos", nil, "Include only these repositories e.g sbomqs,sbomasm")
	cmd.Flags().StringSlice("in-github-exclude-repos", nil, "Exclude these repositories e.g sbomqs,sbomasm")

	// (Optional) If you plan to fetch **all versions** of a repo
	// cmd.Flags().Bool("in-github-all-versions", false, "Fetch SBOMs from all versions")
}

// ParseAndValidateParams validates the GitHub adapter params
func (g *GitHubAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var (
		urlFlag, methodFlag, includeFlag, excludeFlag,
		githubBranchFlag, githubVersionFlag,
		githubToken, githubPoll string
		missingFlags []string
		invalidFlags []string
	)

	switch g.Role {
	case types.InputAdapterRole:
		urlFlag = "in-github-url"
		methodFlag = "in-github-method"
		includeFlag = "in-github-include-repos"
		excludeFlag = "in-github-exclude-repos"
		githubBranchFlag = "in-github-branch"
		githubVersionFlag = "in-github-version"
		githubToken = "in-github-token"
		githubPoll = "in-github-poll-interval"

	case types.OutputAdapterRole:
		return fmt.Errorf("The GitHub adapter doesn't support output adapter functionalities.")

	default:
		return fmt.Errorf("The adapter is neither an input type nor an output type")
	}
	// validate flags for respective adapters
	err := utils.FlagValidation(cmd, types.GithubAdapterType, types.InputAdapterFlagPrefix)
	if err != nil {
		return fmt.Errorf("github flag validation failed: %w", err)
	}

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

	poll, _ := cmd.Flags().GetString(githubPoll)

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

	var fetcher SBOMFetcher
	daemon := g.Config.Daemon

	if daemon {
		// daemon fetcher initialized
		// fetcher = NewWatcherFetcher()
	} else if g.Config.ProcessingMode == types.FetchSequential {
		fetcher = &SequentialFetcher{}
	} else if g.Config.ProcessingMode == types.FetchParallel {
		fetcher = &ParallelFetcher{}
	}

	cfg := NewGithubConfig()
	cfg.SetIncludeRepos(includeRepos)
	cfg.SetExcludeRepos(excludeRepos)

	// Validate that both include & exclude are not used together
	if len(cfg.IncludeRepos) > 0 && len(cfg.ExcludeRepos) > 0 {
		return fmt.Errorf("cannot use both --in-github-include-repos and --in-github-exclude-repos together")
	}

	if GitHubMethod(method) == MethodTool {
		binaryPath, err := utils.GetBinaryPath()
		if err != nil {
			return fmt.Errorf("failed to get Syft binary: %w", err)
		}

		cfg.BinaryPath = binaryPath
	}

	token := viper.GetString("GITHUB_TOKEN")
	if token == "" {
		token, _ = cmd.Flags().GetString(githubToken)
		logger.LogDebug(cmd.Context(), "GitHub Token not found in environment")
	}

	if method == "api" && version != "latest" {
		fmt.Println("Github API method calculates SBOM for a complete repo not for any particular version: ", version)
	}

	// Assign extracted values to struct
	if version == "" {
		version = "latest"
		cfg.URL = githubURL
	} else {
		cfg.URL = fmt.Sprintf("https://github.com/%s/%s", owner, repo)
	}

	cfg.Owner = owner
	// cfg.SetOwner(owner)
	// cfg.SetRepo(repo)
	cfg.Repo = repo
	cfg.Branch = branch

	cfg.Version = version
	// cfg.SetBranch(branch)
	// cfg.SetVersion(version)
	cfg.Method = method
	// cfg.SetMethod(method)
	cfg.Token = token
	// cfg.SetToken(token)
	pollInterval, err := strconv.Atoi(poll)
	if err != nil {
		return fmt.Errorf("invalid poll interval format: %w", err)
	}
	cfg.Poll = int64(pollInterval)

	// cfg.SetPollInterval(int64(pollInterval))

	// Initialize GitHub client
	cfg.client = NewClient(cfg)

	g.Config = cfg
	g.Fetcher = fetcher
	return nil
}

// FetchSBOMs initializes the GitHub SBOM iterator using the unified method
func (g *GitHubAdapter) FetchSBOMs(ctx tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Intializing SBOM fetching process", g.Config.ProcessingMode)
	return g.Fetcher.Fetch(ctx, g.Config)
}

func (g *GitHubAdapter) Monitor(ctx tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	logger.LogInfo(ctx.Context, "monitoring", "repo", g.Config.Repo)
	return g.Fetcher.Fetch(ctx, g.Config)
}

// OutputSBOMs should return an error since GitHub does not support SBOM uploads
func (g *GitHubAdapter) UploadSBOMs(ctx tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	return fmt.Errorf("GitHub adapter does not support SBOM uploading")
}

// DryRun for Input Adapter: Displays all fetched SBOMs from input adapter
func (g *GitHubAdapter) DryRun(ctx tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	reporter := NewGithubReporter(false, "")
	return reporter.DryRun(ctx, iterator)
}
