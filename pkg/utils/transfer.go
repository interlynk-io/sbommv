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
	"os"
	"os/exec"
	"path/filepath"
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
