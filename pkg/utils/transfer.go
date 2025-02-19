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

package utils

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func GetBinaryPath() (string, error) {
	ctx := context.Background()

	cacheDir := filepath.Join(os.Getenv("HOME"), ".sbommv/tools")
	syftBinary := filepath.Join(cacheDir, "bin/syft")

	// Check if Syft already exists and is executable
	if _, err := os.Stat(syftBinary); err == nil {
		return syftBinary, nil
	}

	// If not cached, clone and install Syft
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Clone Syft using Git
	syftRepo := "https://github.com/anchore/syft"
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", syftRepo, cacheDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to clone Syft: %w", err)
	}
	fmt.Println("cacheDir: ", cacheDir)
	fmt.Println("syftBinary: ", syftBinary)

	// Install Syft
	installScript := filepath.Join(cacheDir, "install.sh")
	cmd = exec.Command("/bin/sh", installScript)
	cmd.Dir = cacheDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to install Syft: %w", err)
	}

	// Verify Syft installation
	if _, err := os.Stat(syftBinary); err != nil {
		return "", fmt.Errorf("Syft binary not found after installation")
	}

	return syftBinary, nil
}

// ParseRepoVersion extracts the repository URL without version and version from a GitHub URL.
// For URLs like "https://github.com/owner/repo", returns ("https://github.com/owner/repo", "latest", nil).
// For URLs like "https://github.com/owner/repo@v1.0.0", returns ("https://github.com/owner/repo", "v1.0.0", nil).
func ParseRepoVersion(repoURL string) (string, string, error) {
	// Remove any trailing slashes
	repoURL = strings.TrimRight(repoURL, "/")

	// Check if URL is a GitHub URL
	if !strings.Contains(repoURL, "github.com") {
		return "", "", fmt.Errorf("not a GitHub URL: %s", repoURL)
	}

	// Split on @ to separate repo URL from version
	parts := strings.Split(repoURL, "@")
	if len(parts) > 2 {
		return "", "", fmt.Errorf("invalid GitHub URL format: %s", repoURL)
	}

	baseURL := parts[0]
	version := "latest"

	// Normalize the base URL format
	if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}

	// Validate repository path format (github.com/owner/repo)
	urlParts := strings.Split(baseURL, "/")
	if len(urlParts) < 4 || urlParts[len(urlParts)-2] == "" || urlParts[len(urlParts)-1] == "" {
		return "", "", fmt.Errorf("invalid repository path format: %s", baseURL)
	}

	// Get version if specified
	if len(parts) == 2 {
		version = parts[1]
		// Validate version format
		if !strings.HasPrefix(version, "v") {
			return "", "", fmt.Errorf("invalid version format (should start with 'v'): %s", version)
		}
	}

	return baseURL, version, nil
}

// ParseGithubURL extracts the repository owner, repo name, and it's version.
// For URLs like "https://github.com/interlynk-io/sbomqs@v1.0.0", returns "interlynk-io", "sbomqs", "v1.0.0", nil).
// For URLs like "https://github.com/interlynk-io/sbomqs", returns "interlynk-io", "sbomqs", "", nil).
// For URLs like "https://github.com/interlynk-io/", returns "interlynk-io", "", "", nil).
func ParseGithubURL(githubURL string) (owner, repo, version string, err error) {
	parsedURL, err := url.Parse(githubURL)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid GitHub URL: %w", err)
	}

	// Example: https://github.com/interlynk-io/sbomqs@v1.0.0
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 1 {
		return "", "", "", fmt.Errorf("invalid GitHub URL format")
	}

	owner = pathParts[0]
	if len(pathParts) > 1 {
		repoVersionParts := strings.Split(pathParts[1], "@")
		repo = repoVersionParts[0]
		if len(repoVersionParts) > 1 {
			version = repoVersionParts[1]
		}
	}
	return owner, repo, version, nil
}

// isValidURL checks if the given string is a valid URL
func IsValidURL(input string) bool {
	parsedURL, err := url.ParseRequestURI(input)
	if err != nil {
		return false
	}

	// Ensure it has a scheme (http or https) and a valid host
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}
	if parsedURL.Host == "" {
		return false
	}

	return true
}
