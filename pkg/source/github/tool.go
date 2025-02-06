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
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

// SupportedTools maps tool names to their GitHub repositories
var SupportedTools = map[string]string{
	"syft":    "https://github.com/anchore/syft.git",
	"spdxgen": "https://github.com/spdx/spdx-sbom-generator.git",
}

func GenerateSBOM(ctx *tcontext.TransferMetadata, repoDir, binaryPath string) (string, error) {
	logger.LogDebug(ctx.Context, "Initializing SBOM generation with Syft")

	// Ensure Syft binary is executable
	if err := os.Chmod(binaryPath, 0o755); err != nil {
		return "", fmt.Errorf("failed to set executable permission for syft: %w", err)
	}

	// Generate SBOM using Syft
	sbomFile := "/tmp/sbom.spdx.json"
	dirFlags := fmt.Sprintf("dir:%s", repoDir)
	outputFlags := fmt.Sprintf("spdx-json=%s", sbomFile)

	args := []string{"scan", dirFlags, "-o", outputFlags}

	logger.LogDebug(ctx.Context, "Executing SBOM command", "cmd", binaryPath, "args", args)

	// Run Syft
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = repoDir // Ensure it runs from the correct directory

	var outBuffer, errBuffer strings.Builder
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer

	if err := cmd.Run(); err != nil {
		logger.LogError(ctx.Context, err, "Syft execution failed", "stderr", errBuffer.String(), "stdout", outBuffer.String())
		return "", fmt.Errorf("failed to run Syft: %w", err)
	}

	logger.LogDebug(ctx.Context, "Syft Output", "stdout", outBuffer.String())

	// Wait for SBOM file to be created
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(sbomFile); err == nil {
			logger.LogDebug(ctx.Context, "SBOM file created successfully", "path", sbomFile)
			return sbomFile, nil
		}
		time.Sleep(1 * time.Second) // Wait before retrying
	}

	return "", fmt.Errorf("SBOM file was not created: %s", sbomFile)
}

// CloneRepoWithGit clones a GitHub repository using the Git command-line tool.
func CloneRepoWithGit(ctx *tcontext.TransferMetadata, repoURL, targetDir string) error {
	// Ensure Git is installed
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed, install Git or use --method=api")
	}

	fmt.Println("🚀 Cloning repository using Git:", repoURL)

	// Run `git clone --depth=1` for faster shallow cloning
	cmd := exec.CommandContext(ctx.Context, "git", "clone", "--depth=1", repoURL, targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	fmt.Println("✅ Repository successfully cloned using Git.")
	return nil
}
