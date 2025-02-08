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
	"os"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/engine"
	"github.com/interlynk-io/sbommv/pkg/mvtypes"
	"github.com/interlynk-io/sbommv/pkg/source/github"
	"github.com/interlynk-io/sbommv/pkg/target/interlynk"

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
	`,
	Args: cobra.NoArgs,
	RunE: transferSBOM,
}

func init() {
	rootCmd.AddCommand(transferCmd)

	// custom usage of command
	transferCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Print(customUsageFunc(cmd))
		return nil
	})

	// Input adapter flags
	transferCmd.Flags().String("input-adapter", "", "input adapter type (github)")
	transferCmd.MarkFlagRequired("input-adapter")

	// Output adapter flags
	transferCmd.Flags().String("output-adapter", "", "output adapter type (interlynk)")
	transferCmd.MarkFlagRequired("output-adapter")

	transferCmd.Flags().BoolP("dry-run", "", false, "enable dry run mode")

	transferCmd.Flags().BoolP("debug", "D", false, "enable debug logging")

	// Manually register adapter flags for each adapter
	registerAdapterFlags(transferCmd)
}

// registerAdapterFlags dynamically adds flags for the selected adapters after flag parsing
func registerAdapterFlags(cmd *cobra.Command) {
	// Register GitHub Adapter Flags
	githubAdapter := &github.GitHubAdapter{}
	githubAdapter.AddCommandParams(cmd)

	// Register Interlynk Adapter Flags
	interlynkAdapter := &interlynk.InterlynkAdapter{}
	interlynkAdapter.AddCommandParams(cmd)

	// similarly for all other Adapters
}

func transferSBOM(cmd *cobra.Command, args []string) error {
	// Suppress automatic usage message for non-flag errors
	cmd.SilenceUsage = true

	// Initialize logger based on debug flag
	debug, _ := cmd.Flags().GetBool("debug")
	logger.InitLogger(debug, false) // Using console format as default
	defer logger.Sync()             // Flush logs on exit

	ctx := logger.WithLogger(context.Background())
	viper.AutomaticEnv()

	// Parse config
	config, err := parseConfig(cmd)
	if err != nil {
		logger.LogError(ctx, err, "Invalid configuration")
		return fmt.Errorf("invalid configuration: %w", err)
	}

	logger.LogDebug(ctx, "configuration", "value", config)

	if err := engine.TransferRun(ctx, cmd, config); err != nil {
		return fmt.Errorf("failed to process engine for transfer cmd: %w", err)
	}

	// Clean up SBOMs folder if it exists
	if _, err := os.Stat("sboms"); err == nil {
		if err := os.RemoveAll("sboms"); err != nil {
			logger.LogError(ctx, err, "Failed to delete SBOM directory")
			return fmt.Errorf("failed to delete directory %s: %w", "sboms", err)
		}
		logger.LogDebug(ctx, "Successfully deleted SBOM directory", "directory", "sboms")
	}

	return nil
}

func parseConfig(cmd *cobra.Command) (mvtypes.Config, error) {
	inputType, _ := cmd.Flags().GetString("input-adapter")
	if inputType == "" {
		return mvtypes.Config{}, fmt.Errorf("missing flag: input-adapter")
	}
	outputType, _ := cmd.Flags().GetString("output-adapter")
	if inputType == "" {
		return mvtypes.Config{}, fmt.Errorf("missing flag: input-adapter")
	}

	dr, _ := cmd.Flags().GetBool("dry-run")

	config := mvtypes.Config{
		SourceType:      inputType,
		DestinationType: outputType,
		DryRun:          dr,
	}

	return config, nil
}

// Custom usage function for transferCmd
func customUsageFunc(_ *cobra.Command) string {
	builder := &strings.Builder{}

	builder.WriteString("Usage:\n")
	builder.WriteString("  transfer [flags]\n\n")
	builder.WriteString("Flags:\n")

	// Input Adapters
	builder.WriteString("Input Adapters:\n")
	inputAdapters := map[string][]struct {
		Name  string
		Usage string
	}{
		"github": {
			{"--in-github-url", "URL for input adapter github (required)"},
			{"--in-github-method", "Method for input adapter github (optional)"},
			{"--in-github-include-repos", "Comma-separated list of repositories to include (optional)"},
			{"--in-github-exclude-repos", "Comma-separated list of repositories to exclude (optional)"},
		},
	}

	for adapter, flags := range inputAdapters {
		builder.WriteString(fmt.Sprintf("  %s:\n", adapter))
		for _, flag := range flags {
			builder.WriteString(fmt.Sprintf("    %s %s\n", flag.Name, flag.Usage))
		}
		builder.WriteString("\n")
	}

	// Output Adapters
	builder.WriteString("Output Adapters:\n")
	outputAdapters := map[string][]struct {
		Name  string
		Usage string
	}{
		"interlynk": {
			{"--out-interlynk-url", "URL for output adapter interlynk (optional)"},
			{"--out-interlynk-project-id", "Project ID for output adapter interlynk (optional)"},
		},
	}

	for adapter, flags := range outputAdapters {
		builder.WriteString(fmt.Sprintf("  %s:\n", adapter))
		for _, flag := range flags {
			builder.WriteString(fmt.Sprintf("    %s %s\n", flag.Name, flag.Usage))
		}
		builder.WriteString("\n")
	}

	// Other Flags
	builder.WriteString("Other Flags:\n")
	builder.WriteString("  -D, --debug Enable debug logging\n")
	builder.WriteString("      --dry-run Enable dry run mode\n")
	builder.WriteString("  -h, --help help for transfer\n")
	builder.WriteString("      --input-adapter Input adapter type (github) (required)\n")
	builder.WriteString("      --output-adapter Output adapter type (interlynk) (required)\n")

	return builder.String()
}
