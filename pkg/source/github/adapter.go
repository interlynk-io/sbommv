package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/utils"
	"github.com/spf13/cobra"
)

// GitHubAdapter handles fetching SBOMs from GitHub releases
type GitHubAdapter struct {
	URL        string
	Repo       string
	Owner      string
	Version    string
	Branch     string
	Method     string
	BinaryPath string
	client     *http.Client
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
	url, _ := cmd.Flags().GetString("in-github-url")
	if url == "" {
		return fmt.Errorf("missing or invalid flag: in-github-url")
	}

	method, _ := cmd.Flags().GetString("in-github-method")
	if method != "release" && method != "api" && method != "tool" {
		return fmt.Errorf("missing or invalid flag: in-github-method")
	}

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
