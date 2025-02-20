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

// pkg/source/github/config.go
package github

// GitHubConfig holds all configuration data for the GitHub adapter
type GitHubConfig struct {
	URL          string
	Repo         string
	Owner        string
	Version      string
	Branch       string
	Method       string
	BinaryPath   string
	GithubToken  string
	IncludeRepos []string
	ExcludeRepos []string
}

func NewGitHubConfig() *GitHubConfig {
	return &GitHubConfig{
		Version: "latest", // Default value
	}
}
