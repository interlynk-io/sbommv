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
	"strings"

	githublib "github.com/google/go-github/v62/github"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"golang.org/x/oauth2"
)

type GithubConfig struct {
	URL            string
	Repo           string
	Owner          string
	Version        string
	Branch         string
	Method         string
	BinaryPath     string
	client         *Client
	Token          string
	IncludeRepos   []string
	ExcludeRepos   []string
	ProcessingMode types.ProcessingMode
	Daemon         bool
	Poll           int64
}

func NewGithubConfig() *GithubConfig {
	return &GithubConfig{
		Method:         "",
		BinaryPath:     "",
		client:         nil,
		Token:          "",
		IncludeRepos:   []string{},
		ExcludeRepos:   []string{},
		ProcessingMode: types.FetchSequential,
		Daemon:         false,
	}
}

func (c *GithubConfig) GetRepo() string {
	return c.Repo
}

// SetRepos sets the list of repositories.
func (c *GithubConfig) SetRepo(repo string) {
	c.Repo = repo
}

// SetOrg sets the organization name.
func (c *GithubConfig) SetOwner(org string) {
	c.Owner = org
}

func (c *GithubConfig) SetBranch(branch string) {
	c.Branch = branch
}

func (c *GithubConfig) SetVersion(version string) {
	c.Version = version
}

func (c *GithubConfig) SetMethod(method string) {
	c.Method = method
}

// SetToken sets the GitHub token.
func (c *GithubConfig) SetToken(token string) {
	c.Token = token
}

// SetPollInterval sets the polling interval in seconds.
func (c *GithubConfig) SetPollInterval(interval int64) {
	c.Poll = interval
}

// SetIncludePattern sets the include pattern for repository filtering.
func (c *GithubConfig) SetIncludeRepos(repos []string) {
	c.IncludeRepos = repos
}

// SetExcludePattern sets the exclude pattern for repository filtering.
func (c *GithubConfig) SetExcludeRepos(repos []string) {
	c.ExcludeRepos = repos
}

// SetProcessingMode sets the processing mode (Sequential, Parallel, Watcher).
func (c *GithubConfig) SetProcessingMode(mode types.ProcessingMode) {
	c.ProcessingMode = mode
}

// GetGitHubClient initializes and returns a GitHub API client.
func (c *GithubConfig) GetGitHubClient(ctx tcontext.TransferMetadata) (*githublib.Client, error) {
	logger.LogDebug(ctx.Context, "Initializing GitHub client")

	if c.Token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.Token})
	tc := oauth2.NewClient(ctx.Context, ts)
	client := githublib.NewClient(tc)

	// Verify token by making a simple API call
	_, _, err := client.Users.Get(ctx.Context, "")
	if err != nil {
		logger.LogError(ctx.Context, err, "Failed to validate GitHub token")
		return nil, fmt.Errorf("invalid GitHub token: %w", err)
	}

	return client, nil
}

// applyRepoFilters filters repositories based on inclusion/exclusion flags
func (g *GithubConfig) applyRepoFilters(ctx tcontext.TransferMetadata, repos []string) []string {
	includedRepos := make(map[string]bool)
	excludedRepos := make(map[string]bool)

	logger.LogDebug(ctx.Context, "Applying repository filters", "include", g.IncludeRepos, "exclude", g.ExcludeRepos)
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

	logger.LogDebug(ctx.Context, "Filtered repositories", "filtered", filteredRepos)
	return filteredRepos
}
