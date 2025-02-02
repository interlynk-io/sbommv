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

package source

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/mvtypes"
	"github.com/interlynk-io/sbommv/pkg/source/github"
)

// GitHubAdapter implements InputAdapter for GitHub repositories
type GitHubAdapter struct {
	URL     string
	Version string
	Tool    string
	Binary  string
	// repo    string
	// token   string
	method  GitHubMethod
	client  *http.Client
	options InputOptions
}

// GitHubMethod specifies how to retrieve/generate SBOMs from GitHub
type GitHubMethod string

const (
	// MethodReleases searches for SBOMs in GitHub releases
	MethodReleases GitHubMethod = "release"

	// // MethodReleases searches for SBOMs in GitHub releases
	MethodAPI GitHubMethod = "api"

	// MethodGenerate clones the repo and generates SBOMs using external Tools
	MethodTool GitHubMethod = "tool"
)

// NewGitHubAdapter creates a new GitHub adapter
func NewGitHubAdapter(config mvtypes.Config) *GitHubAdapter {
	url, _ := config.SourceConfigs["url"].(string)
	version, _ := config.SourceConfigs["version"].(string)
	method, _ := config.SourceConfigs["method"].(string)
	binary, _ := config.SourceConfigs["binary"].(string)

	return &GitHubAdapter{
		URL:     url,
		Version: version,
		method:  GitHubMethod(method),
		client:  &http.Client{},
		Binary:  binary,
	}
}

// GitHubAdapter implements GetSBOMs. Therefore, it implements InputAdapter.
func (a *GitHubAdapter) GetSBOMs(ctx context.Context) (map[string][]string, error) {
	logger.LogDebug(ctx, "Executing GetSBOMs function", "method", a.method)

	switch a.method {
	case MethodReleases:
		logger.LogDebug(ctx, "Fetching SBOMs from GitHub Release Page", "method", MethodReleases)
		return a.getSBOMsFromReleases(ctx)

	case MethodAPI:
		return a.getSBOMsFromAPI(ctx)

	case MethodTool:
		logger.LogDebug(ctx, "Generating SBOMs using tools", "method", MethodTool)
		return a.getSBOMsFromTool(ctx)

	default:
		err := fmt.Errorf("unsupported GitHub method: %v", a.method)
		logger.LogError(ctx, err, "Invalid GitHub SBOM retrieval method", "method", a.method)
		return nil, err
	}
}

func (a *GitHubAdapter) getSBOMsFromReleases(ctx context.Context) (map[string][]string, error) {
	logger.LogDebug(ctx, "Fetching SBOMs from GitHub using", "url", a.URL)

	client := github.NewClient(a.URL, a.Version, string(a.method))

	sbomFiles, err := client.GetSBOMs(ctx, "sboms")
	if err != nil {
		logger.LogError(ctx, err, "Failed to retrieve SBOMs from GitHub releases", "url", a.URL, "version", a.Version)
		return nil, fmt.Errorf("error retrieving SBOMs from releases: %w", err)
	}

	return sbomFiles, nil
}

func (a *GitHubAdapter) getSBOMsFromAPI(ctx context.Context) (map[string][]string, error) {
	logger.LogDebug(ctx, "Fetching SBOM from GitHub API", "repository", a.URL)

	client := github.NewClient(a.URL, a.Version, string(a.method))
	sbomData, err := client.FetchSBOMFromAPI(ctx)
	if err != nil {
		logger.LogError(ctx, err, "Failed to fetch SBOM from GitHub API", "repository", a.URL)
		return nil, fmt.Errorf("error retrieving SBOM from GitHub API: %w", err)
	}

	logger.LogDebug(ctx, "Successfully retrieved SBOM from GitHub API", "repository", a.URL)

	// Define SBOM file path
	sbomFilePath := fmt.Sprintf("sboms/github_api_sbom_%s.json", sanitizeRepoName(a.URL))

	// Ensure the directory exists
	if err := os.MkdirAll("sboms", 0o755); err != nil {
		logger.LogError(ctx, err, "Failed to create SBOM output directory")
		return nil, fmt.Errorf("error creating SBOM output directory: %w", err)
	}

	// Write SBOM data to file
	if err := os.WriteFile(sbomFilePath, sbomData, 0o644); err != nil {
		logger.LogError(ctx, err, "Failed to write SBOM to file", "file", sbomFilePath)
		return nil, fmt.Errorf("error writing SBOM to file: %w", err)
	}
	logger.LogDebug(ctx, "SBOM successfully written to file", "file", sbomFilePath)

	return map[string][]string{"main": {sbomFilePath}}, nil
}

func (a *GitHubAdapter) getSBOMsFromTool(ctx context.Context) (map[string][]string, error) {
	logger.LogDebug(ctx, "Generating SBOM using tool", "repository", a.URL)

	// Clone the repository for which SBOM has to be generated
	repoDir := filepath.Join(os.TempDir(), fmt.Sprintf("sbommv_%d", time.Now().UnixNano()))
	defer os.RemoveAll(repoDir) // Cleanup cloned repo after execution

	if err := CloneRepoWithGit(ctx, a.URL, repoDir); err != nil {
		logger.LogError(ctx, err, "Failed to clone repository", "repository", a.URL)
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	logger.LogDebug(ctx, "Repository cloned successfully", "path", repoDir)

	// Generate SBOM using Syft or another tool
	sbomFile, err := github.GenerateSBOM(ctx, repoDir, a.Binary)
	if err != nil {
		logger.LogError(ctx, err, "Failed to generate SBOM", "repository", a.URL)
		return nil, fmt.Errorf("failed to generate SBOM: %w", err)
	}

	// Ensure SBOM output directory exists before renaming
	outputDir := "sboms"
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		logger.LogError(ctx, err, "Failed to create SBOM output directory", "directory", outputDir)
		return nil, fmt.Errorf("failed to create SBOM output directory: %w", err)
	}

	// Define S	// Define final SBOM file path BOM file path
	sbomFilePath := filepath.Join(outputDir, fmt.Sprintf("github_tool_sbom_%s.json", sanitizeRepoName(a.URL)))

	// Move SBOM file to final location
	if err := os.Rename(sbomFile, sbomFilePath); err != nil {
		logger.LogError(ctx, err, "Failed to move SBOM file", "source", sbomFile, "destination", sbomFilePath)
		return nil, fmt.Errorf("failed to move SBOM file: %w", err)
	}

	// Ensure the directory exists
	if err := os.MkdirAll("sboms", 0o755); err != nil {
		logger.LogError(ctx, err, "Failed to create SBOM output directory")
		return nil, fmt.Errorf("error creating SBOM output directory: %w", err)
	}

	logger.LogDebug(ctx, "SBOM successfully written to file", "file", sbomFilePath)

	// Return the generated SBOM file
	return map[string][]string{"main": {sbomFilePath}}, nil
}

// CloneRepoWithGit clones a GitHub repository using the Git command-line tool.
func CloneRepoWithGit(ctx context.Context, repoURL, targetDir string) error {
	// Ensure Git is installed
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed, install Git or use --method=api")
	}

	fmt.Println("🚀 Cloning repository using Git:", repoURL)

	// Run `git clone --depth=1` for faster shallow cloning
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", repoURL, targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	fmt.Println("✅ Repository successfully cloned using Git.")
	return nil
}
