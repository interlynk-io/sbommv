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
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v50/github"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"golang.org/x/oauth2"
)

// SupportedTools maps tool names to their GitHub repositories
var SupportedTools = map[string]string{
	"syft":    "https://github.com/anchore/syft.git",
	"spdxgen": "https://github.com/spdx/spdx-sbom-generator.git",
}

// SBOMTool defines the structure for an SBOM tool
type SBOMTool struct {
	Name     string
	RepoURL  string
	CloneDir string
	Binary   string
}

// NewSBOMTool initializes a tool instance
func NewSBOMTool(toolName string) (*SBOMTool, error) {
	repoURL, exists := SupportedTools[toolName]
	if !exists {
		return nil, fmt.Errorf("unsupported SBOM tool: %s", toolName)
	}

	cloneDir := filepath.Join(os.TempDir(), toolName)
	binary := filepath.Join(cloneDir, toolName)

	return &SBOMTool{
		Name:     toolName,
		RepoURL:  repoURL,
		CloneDir: cloneDir,
		Binary:   binary,
	}, nil
}

// Clone clones the SBOM tool's repository if it is not already cloned
func (t *SBOMTool) Clone(ctx context.Context) error {
	if _, err := os.Stat(t.CloneDir); !os.IsNotExist(err) {
		logger.LogDebug(ctx, "SBOM tool already cloned", "path", t.CloneDir)
		return nil
	}

	logger.LogDebug(ctx, "Cloning SBOM tool repository", "repo", t.RepoURL)

	cmd := exec.Command("git", "clone", t.RepoURL, t.CloneDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone SBOM tool: %w", err)
	}

	return nil
}

// Build compiles the SBOM tool from source
func (t *SBOMTool) Build(ctx context.Context) error {
	logger.LogDebug(ctx, "Building SBOM tool", "tool", t.Name)

	cmd := exec.Command("go", "build", "-o", t.Binary)
	cmd.Dir = t.CloneDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build SBOM tool: %w", err)
	}

	return nil
}

// Run generates an SBOM for the specified repository
func (t *SBOMTool) Run(ctx context.Context, repoURL, outputPath string) error {
	logger.LogDebug(ctx, "Running SBOM tool", "tool", t.Name, "repo", repoURL)

	args := []string{repoURL, "-o", outputPath}
	cmd := exec.Command(t.Binary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate SBOM: %w", err)
	}

	logger.LogDebug(ctx, "SBOM generation successful", "output", outputPath)
	return nil
}

// Cleanup removes the cloned repository
func (t *SBOMTool) Cleanup(ctx context.Context) {
	logger.LogDebug(ctx, "Cleaning up SBOM tool directory", "path", t.CloneDir)
	os.RemoveAll(t.CloneDir)
}

// CloneRepo downloads a GitHub repository using the GitHub API
func CloneRepo(ctx context.Context, repoURL, targetDir string) error {
	owner, repo, err := ParseGitHubURL(repoURL)
	if err != nil {
		return fmt.Errorf("invalid GitHub repo URL: %w", err)
	}

	logger.LogDebug(ctx, "Cloning repository via GitHub API", "owner", owner, "repo", repo)

	// Create GitHub client (supports anonymous access or token-based)
	client := newGitHubClient(ctx)

	// Fetch repository contents
	repoContents, err := fetchRepoContents(ctx, client, owner, repo, "")
	if err != nil {
		return fmt.Errorf("failed to fetch repository contents: %w", err)
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Concurrent file downloading
	var wg sync.WaitGroup
	errChan := make(chan error, len(repoContents))

	// Start downloading files concurrently
	for _, content := range repoContents {
		wg.Add(1)
		go func(content *github.RepositoryContent) {
			defer wg.Done()
			if err := downloadGitHubFile(ctx, client, owner, repo, content, targetDir); err != nil {
				errChan <- err
			}
		}(content)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var downloadErrors []error
	for err := range errChan {
		downloadErrors = append(downloadErrors, err)
	}

	if len(downloadErrors) > 0 {
		return fmt.Errorf("some files failed to download: %v", downloadErrors)
	}

	logger.LogDebug(ctx, "Repository successfully cloned using GitHub API", "path", targetDir)
	return nil
}

// newGitHubClient initializes a GitHub API client (with optional authentication)
func newGitHubClient(ctx context.Context) *github.Client {
	token := os.Getenv("GITHUB_TOKEN") // Optional GitHub token for private repos
	if token == "" {
		return github.NewClient(nil) // Anonymous access
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

// fetchRepoContents retrieves the repository contents for a given path
func fetchRepoContents(ctx context.Context, client *github.Client, owner, repo, path string) ([]*github.RepositoryContent, error) {
	_, contents, _, err := client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contents for %s: %w", path, err)
	}
	return contents, nil
}

// downloadGitHubFile downloads a file or recursively processes a directory
func downloadGitHubFile(ctx context.Context, client *github.Client, owner, repo string, content *github.RepositoryContent, targetDir string) error {
	switch content.GetType() {
	case "dir":
		return downloadGitHubDirectory(ctx, client, owner, repo, content.GetPath(), targetDir)
	case "file":
		return downloadFile(ctx, content, targetDir)
	default:
		return fmt.Errorf("unsupported content type: %s", content.GetType())
	}
}

// downloadGitHubDirectory handles recursive downloading of a directory
func downloadGitHubDirectory(ctx context.Context, client *github.Client, owner, repo, dirPath, targetDir string) error {
	subContents, err := fetchRepoContents(ctx, client, owner, repo, dirPath)
	if err != nil {
		return fmt.Errorf("failed to fetch directory contents for %s: %w", dirPath, err)
	}

	// Concurrently process files inside the directory
	var wg sync.WaitGroup
	errChan := make(chan error, len(subContents))

	for _, subContent := range subContents {
		wg.Add(1)
		go func(subContent *github.RepositoryContent) {
			defer wg.Done()
			if err := downloadGitHubFile(ctx, client, owner, repo, subContent, targetDir); err != nil {
				errChan <- err
			}
		}(subContent)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var downloadErrors []error
	for err := range errChan {
		downloadErrors = append(downloadErrors, err)
	}

	if len(downloadErrors) > 0 {
		return fmt.Errorf("some files in directory %s failed to download: %v", dirPath, downloadErrors)
	}
	return nil
}

// downloadFile handles downloading and saving a single file
func downloadFile(ctx context.Context, content *github.RepositoryContent, targetDir string) error {
	// Ensure it's a file
	if content.GetType() != "file" {
		return fmt.Errorf("skipping non-file content: %s", content.GetPath())
	}

	// Download the file
	resp, err := http.Get(content.GetDownloadURL())
	if err != nil {
		return fmt.Errorf("failed to download file %s: %w", content.GetPath(), err)
	}
	defer resp.Body.Close()

	// Create local file path
	filePath := filepath.Join(targetDir, content.GetPath())

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("failed to create directories for %s: %w", filePath, err)
	}

	// Create and write to the file
	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file %s: %w", filePath, err)
	}

	logger.LogDebug(ctx, "Downloaded file", "path", filePath)
	return nil
}

func GenerateSBOM(ctx context.Context, repoDir, binaryPath string) (string, error) {
	logger.LogDebug(ctx, "Initializing SBOM generation with Syft")

	// Ensure Syft binary is executable
	if err := os.Chmod(binaryPath, 0o755); err != nil {
		return "", fmt.Errorf("failed to set executable permission for syft: %w", err)
	}

	// Generate SBOM using Syft
	sbomFile := "/tmp/sbom.spdx.json"
	dirFlags := fmt.Sprintf("dir:%s", repoDir)
	outputFlags := fmt.Sprintf("spdx-json=%s", sbomFile)

	args := []string{"scan", dirFlags, "-o", outputFlags}

	logger.LogDebug(ctx, "Executing SBOM command", "cmd", binaryPath, "args", args)

	// Run Syft
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = repoDir // Ensure it runs from the correct directory

	var outBuffer, errBuffer strings.Builder
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer

	if err := cmd.Run(); err != nil {
		logger.LogError(ctx, err, "Syft execution failed", "stderr", errBuffer.String(), "stdout", outBuffer.String())
		return "", fmt.Errorf("failed to run Syft: %w", err)
	}

	logger.LogDebug(ctx, "Syft Output", "stdout", outBuffer.String())

	// Wait for SBOM file to be created
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(sbomFile); err == nil {
			logger.LogDebug(ctx, "SBOM file created successfully", "path", sbomFile)
			return sbomFile, nil
		}
		time.Sleep(1 * time.Second) // Wait before retrying
	}

	return "", fmt.Errorf("SBOM file was not created: %s", sbomFile)
}
