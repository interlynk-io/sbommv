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
	"strings"

	"github.com/interlynk-io/sbommv/pkg/types"
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
	GithubToken    string
	IncludeRepos   []string
	ExcludeRepos   []string
	ProcessingMode types.ProcessingMode
	Daemon         bool
}

func NewGithubConfig() *GithubConfig {
	return &GithubConfig{
		Method:         "",
		BinaryPath:     "",
		client:         nil,
		GithubToken:    "",
		IncludeRepos:   []string{},
		ExcludeRepos:   []string{},
		ProcessingMode: types.FetchSequential,
		Daemon:         false,
	}
}

// applyRepoFilters filters repositories based on inclusion/exclusion flags
func (g *GithubConfig) applyRepoFilters(repos []string) []string {
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
