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

package cmd

import (
	"context"
	"fmt"

	"github.com/interlynk-io/sbommv/pkg/engine"
	ifolder "github.com/interlynk-io/sbommv/pkg/source/folder"
	"github.com/interlynk-io/sbommv/pkg/target/dependencytrack"
	ofolder "github.com/interlynk-io/sbommv/pkg/target/folder"

	"github.com/interlynk-io/sbommv/pkg/source/github"
	"github.com/interlynk-io/sbommv/pkg/target/interlynk"
	"github.com/interlynk-io/sbommv/pkg/types"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var transferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Transfer SBOMs between systems",
	Long: `Transfer SBOMs from a source system (e.g., GitHub) to a target system (e.g., Interlynk).

Example usage:
# Fetch SBOM for sbomqs latest github release and upload to interlynk platform
sbommv transfer -D --input-adapter=github --in-github-url="https://github.com/interlynk-io/sbomqs" \
--output-adapter=interlynk --out-interlynk-url="http://localhost:3000/lynkapi"

# Fetch SBOMs using the GitHub adapter via the api method for the latest repository version
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" --in-github-method=api  \
--output-adapter=interlynk --out-interlynk-url="http://localhost:3000/lynkapi"

# Fetch SBOMs from github repo and save it to a folder "temp"
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/" --in-github-include-repos=cosign,fulcio,rekor \
--in-github-method="release" --output-adapter=folder --out-folder-path="temp"

# Fetch SBOMs from folder "temp" and upload/push it to a Interlynk
sbommv transfer --input-adapter=folder --in-folder-path="temp"  --in-folder-recursive=true  --output-adapter=interlynk \
--out-interlynk-url="http://localhost:3000/lynkapi"

	`,
	Args: cobra.NoArgs,
	RunE: transferSBOM,
}

func init() {
	rootCmd.AddCommand(transferCmd)

	// Input adapter flags
	transferCmd.Flags().String("input-adapter", "", "input adapter type (github, folder)")

	// Output adapter flags
	transferCmd.Flags().String("output-adapter", "", "output adapter type (dtrack, interlynk, folder)")

	transferCmd.Flags().BoolP("dry-run", "", false, "enable dry run mode")

	// processing mode: sequential or parallel
	transferCmd.Flags().String("processing-mode", "sequential", "processing strategy (parallel, sequential)")

	// enable daemon mode
	transferCmd.Flags().BoolP("daemon", "d", false, "enable daemon mode")

	transferCmd.Flags().BoolP("debug", "D", false, "enable debug logging")

	// Manually register adapter flags for each adapter
	registerAdapterFlags(transferCmd)
}

// registerAdapterFlags dynamically adds flags for the selected adapters after flag parsing
func registerAdapterFlags(cmd *cobra.Command) {
	// Register GitHub Adapter Flags
	githubAdapter := &github.GitHubAdapter{}
	githubAdapter.AddCommandParams(cmd)

	// Register Input Folder Adapter Flags
	folderInputAdapter := &ifolder.FolderAdapter{}
	folderInputAdapter.AddCommandParams(cmd)

	// Register Interlynk Adapter Flags
	interlynkAdapter := &interlynk.InterlynkAdapter{}
	interlynkAdapter.AddCommandParams(cmd)

	// Register Output Folder Adapter Flags
	folderOutputAdapter := &ofolder.FolderAdapter{}
	folderOutputAdapter.AddCommandParams(cmd)

	dtrackAdapter := &dependencytrack.DependencyTrackAdapter{}
	dtrackAdapter.AddCommandParams(cmd)
	// similarly for all other Adapters
}

func transferSBOM(cmd *cobra.Command, args []string) error {
	// Suppress automatic usage message for non-flag errors
	cmd.SilenceUsage = true

	// Initialize logger based on debug flag
	debug, _ := cmd.Flags().GetBool("debug")
	logger.InitLogger(debug, false)
	defer logger.Sync()

	ctx := logger.WithLogger(context.Background())
	viper.AutomaticEnv()
	logger.LogDebug(ctx, "Starting transferSBOM")

	// Parse config
	config, err := parseConfig(cmd)
	if err != nil {
		return err
	}

	logger.LogDebug(ctx, "configuration", "value", config)

	if err := engine.TransferRun(ctx, cmd, config); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func parseConfig(cmd *cobra.Command) (types.Config, error) {
	inputType, _ := cmd.Flags().GetString("input-adapter")
	outputType, _ := cmd.Flags().GetString("output-adapter")
	dr, _ := cmd.Flags().GetBool("dry-run")
	processingMode, _ := cmd.Flags().GetString("processing-mode")
	daemon, _ := cmd.Flags().GetBool("daemon")

	validInputAdapter := map[string]bool{"github": true, "folder": true}
	validOutputAdapter := map[string]bool{"interlynk": true, "folder": true, "dtrack": true}

	// Custom validation for required flags
	missingFlags := []string{}
	invalidFlags := []string{}

	if inputType == "" {
		missingFlags = append(missingFlags, "--input-adapter")
	}

	if outputType == "" {
		missingFlags = append(missingFlags, "--output-adapter")
	}

	validModes := map[string]bool{"sequential": true, "parallel": true}
	if !validModes[processingMode] {
		invalidFlags = append(invalidFlags, fmt.Sprintf("%s=%s (must be one of: sequential, parallel)", "--processing-mode", processingMode))
	}

	// Show error message if required flags are missing
	if len(invalidFlags) > 0 {
		return types.Config{}, fmt.Errorf("missing required flags: %v\n\nUse 'sbommv transfer --help' for usage details.", invalidFlags)
	}

	// Show error message if required flags are missing
	if len(missingFlags) > 0 {
		return types.Config{}, fmt.Errorf("missing required flags: %v\n\nUse 'sbommv transfer --help' for usage details.", missingFlags)
	}

	if !validInputAdapter[inputType] {
		return types.Config{}, fmt.Errorf("input adapter must be one of type: github, folder")
	}

	if !validOutputAdapter[outputType] {
		return types.Config{}, fmt.Errorf("output adapter must be one of type: dtrack, interlynk, folder")
	}
	config := types.Config{
		SourceAdapter:      inputType,
		DestinationAdapter: outputType,
		DryRun:             dr,
		ProcessingStrategy: processingMode,
		Daemon:             daemon,
	}

	return config, nil
}
