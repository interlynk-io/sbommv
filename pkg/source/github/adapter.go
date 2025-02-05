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
	"net/http"

	"github.com/interlynk-io/sbommv/pkg/iterator"
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
	client      *http.Client
	GithubToken string
	Role        types.AdapterRole
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
	cmd.Flags().String("in-github-url", "", "GitHub repository URL")
	cmd.Flags().String("in-github-method", "release", "GitHub method: release, api, or tool")
}

// ParseAndValidateParams validates the GitHub adapter params
func (g *GitHubAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var urlFlag, methodFlag string

	if g.Role == types.InputAdapter {
		urlFlag = "in-github-url"
		methodFlag = "in-github-method"
	}

	url, _ := cmd.Flags().GetString(urlFlag)
	if url == "" {
		return fmt.Errorf("missing or invalid flag: in-github-url")
	}

	method, _ := cmd.Flags().GetString(methodFlag)
	if method != "release" && method != "api" && method != "tool" {
		return fmt.Errorf("missing or invalid flag: in-github-method")
	}

	token := viper.GetString("GITHUB_TOKEN")

	repoURL, version, err := utils.ParseRepoVersion(url)
	if err != nil {
		return fmt.Errorf("falied to parse github repo and version %v", err)
	}
	if repoURL == "" {
		return fmt.Errorf("failed to parse repo URL: %s", url)
	}
	if version == "" {
		version = "latest"
	}

	g.URL = url
	g.Method = method
	g.Repo = repoURL
	g.Version = version
	g.GithubToken = token

	return nil
}

// FetchSBOMs initializes the GitHub SBOM iterator using the unified method
func (g *GitHubAdapter) FetchSBOMs(ctx context.Context) (iterator.SBOMIterator, error) {
	return NewGitHubIterator(ctx, g)
}

// OutputSBOMs should return an error since GitHub does not support SBOM uploads
func (g *GitHubAdapter) UploadSBOMs(ctx context.Context, iterator iterator.SBOMIterator) error {
	return fmt.Errorf("GitHub adapter does not support SBOM uploading")
}
