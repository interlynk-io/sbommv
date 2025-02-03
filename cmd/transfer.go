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

	"github.com/interlynk-io/sbommv/pkg/adapters/dest"
	source "github.com/interlynk-io/sbommv/pkg/adapters/source"
	"github.com/interlynk-io/sbommv/pkg/engine"
	"github.com/interlynk-io/sbommv/pkg/mvtypes"
	"github.com/interlynk-io/sbommv/pkg/utils"

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

// Define the flag metadata with support for multiple types
type FlagMeta struct {
	Usage   string
	Default interface{} // Use an empty interface to accommodate multiple types
	Type    string      // Type of the flag: "string", "bool", "int", etc.
}

func init() {
	rootCmd.AddCommand(transferCmd)

	// custom usage of command
	transferCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Print(customUsageFunc(cmd))
		return nil
	})
	setInputAdapterDynamicFlags(transferCmd)
	setOutputAdapterDynamicFlags(transferCmd)

	// Input adapter flags
	transferCmd.Flags().String("input-adapter", "", "input adapter type (github)")
	transferCmd.MarkFlagRequired("input-adapter")

	// Output adapter flags
	transferCmd.Flags().String("output-adapter", "", "output adapter type (interlynk)")
	transferCmd.MarkFlagRequired("output-adapter")

	transferCmd.Flags().BoolP("dry-run", "", false, "enable dry run mode")

	transferCmd.Flags().BoolP("debug", "D", false, "enable debug logging")
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

	// Ensure essential configs are not nil
	if config.SourceConfigs == nil || config.DestinationConfigs == nil {
		// TODO: validate config values
		logger.LogError(ctx, nil, "Missing required adapter configurations")
		return fmt.Errorf("failed to construct valid configuration: missing adapter settings")
	}

	logger.LogDebug(ctx, "configuration", "value", config)

	// Source-specific debug log
	if config.SourceType == string(source.SourceGithub) && config.SourceConfigs["version"] == "" {
		logger.LogDebug(ctx, "Fetching all SBOMs from all versions of the repository")
	}

	// validate output adapter i.e Interlynk is up and running
	if config.DestinationType == "interlynk" {
		url := config.DestinationConfigs["url"].(string)
		token := config.DestinationConfigs["token"].(string)

		if err := engine.ValidateInterlynkConnection(ctx, url, token); err != nil {
			return fmt.Errorf("Interlynk validation failed: %w", err)
		}
	}

	// Execute engine operation
	logger.LogDebug(ctx, "Executing SBOM transfer process")
	if err := engine.TransferRun(ctx, config); err != nil {
		logger.LogError(ctx, err, "Transfer operation failed")
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
	outputType, _ := cmd.Flags().GetString("output-adapter")
	dr, _ := cmd.Flags().GetBool("dry-run")

	config := mvtypes.Config{
		SourceType:         inputType,
		DestinationType:    outputType,
		SourceConfigs:      map[string]interface{}{},
		DestinationConfigs: map[string]interface{}{},
		DryRun:             dr,
	}

	// Parse input adapter configuration
	switch source.InputType(inputType) {

	case source.SourceGithub:
		url, err := cmd.Flags().GetString("in-github-url")
		if err != nil || url == "" {
			return config, fmt.Errorf("missing or invalid flag: : in-github-url")
		}

		repoURL, version, err := ParseRepoVersion(url)
		if err != nil {
			return config, fmt.Errorf("falied to parse github repo and version %v", err)
		}
		config.SourceConfigs["version"] = version

		allVersion, err := cmd.Flags().GetBool("in-github-all-versions")
		if err != nil {
			return config, fmt.Errorf("falied to parse github all version %v", err)
		}
		if allVersion {
			// remove specific version
			// this signifies to all versions
			version = ""
		}
		config.SourceConfigs["version"] = version
		config.SourceConfigs["url"] = repoURL

		// in-github-method
		method, err := cmd.Flags().GetString("in-github-method")
		if err != nil || url == "" {
			return config, fmt.Errorf("missing or invalid flag: in-github-method")
		}

		if method == "tool" {
			binaryPath, err := utils.GetBinaryPath()
			if err != nil {
				return config, fmt.Errorf("failed to get Syft binary: %w", err)
			}
			fmt.Println("Binary Path: ", binaryPath)
			config.SourceConfigs["binary"] = binaryPath
		}
		config.SourceConfigs["method"] = method
	default:
		return config, fmt.Errorf("unsupported input adapter: %s", inputType)
	}

	// Parse output adapter configuration
	switch dest.OutputType(outputType) {

	case dest.DestInterlynk:
		url, err := cmd.Flags().GetString("out-interlynk-url")
		if err != nil || url == "" {
			return config, fmt.Errorf("missing or invalid flag: out-interlynk-url")
		}

		// push all sbom version
		pushAllVersion, _ := cmd.Flags().GetBool("in-github-all-versions")
		projectID, err := cmd.Flags().GetString("out-interlynk-project-id")
		if err != nil {
			return config, fmt.Errorf("missing or invalid flag: out-interlynk-project-id")
		}

		// Get token from environment
		token := viper.GetString("INTERLYNK_SECURITY_TOKEN")

		config.DestinationConfigs["url"] = url
		config.DestinationConfigs["token"] = token
		config.DestinationConfigs["projectID"] = projectID
		config.DestinationConfigs["pushAllVersion"] = pushAllVersion

	default:
		return config, fmt.Errorf("unsupported output adapter: %s", outputType)
	}

	return config, nil
}

func setInputAdapterDynamicFlags(transferCmd *cobra.Command) {
	// Define input adapters and their flags with specific descriptions and default values
	inputAdapters := map[source.InputType]map[string]FlagMeta{
		source.SourceGithub: {
			"in-github-url":          {Usage: "URL for input adapter github", Default: "", Type: "string"},
			"in-github-method":       {Usage: "Method for input adapter github", Default: "release", Type: "string"},
			"in-github-all-versions": {Usage: "Fetch all SBOMs for all versions", Default: false, Type: "bool"},
		},
	}

	// Dynamically register input adapter flags with support for multiple types
	for _, flags := range inputAdapters {
		for flag, meta := range flags {
			switch meta.Type {
			case "string":
				if defaultValue, ok := meta.Default.(string); ok {
					transferCmd.Flags().String(flag, defaultValue, meta.Usage)
				} else {
					panic(fmt.Sprintf("Invalid default type for flag %s, expected string", flag))
				}
			case "bool":
				if defaultValue, ok := meta.Default.(bool); ok { // Updated type assertion for boolean
					transferCmd.Flags().Bool(flag, defaultValue, meta.Usage)
				} else {
					panic(fmt.Sprintf("Invalid default type for flag %s, expected bool", flag))
				}
			case "int":
				if defaultValue, ok := meta.Default.(int); ok {
					transferCmd.Flags().Int(flag, defaultValue, meta.Usage)
				} else {
					panic(fmt.Sprintf("Invalid default type for flag %s, expected int", flag))
				}
			default:
				panic(fmt.Sprintf("Unsupported flag type for %s: %s", flag, meta.Type))
			}
		}
	}
}

func setOutputAdapterDynamicFlags(transferCmd *cobra.Command) {
	// Define output adapters and their flags with specific descriptions and default values
	outputAdapters := map[dest.OutputType]map[string]FlagMeta{
		dest.DestInterlynk: {
			"out-interlynk-url":        {Usage: "URL for output adapter interlynk", Default: "https://api.interlynk.io/lynkapi", Type: "string"},
			"out-interlynk-project-id": {Usage: "Project ID for output adapter interlynk", Default: "", Type: "string"},
		},
	}

	// Dynamically register input adapter flags with support for multiple types
	for _, flags := range outputAdapters {
		for flag, meta := range flags {
			switch meta.Type {
			case "string":
				if defaultValue, ok := meta.Default.(string); ok {
					transferCmd.Flags().String(flag, defaultValue, meta.Usage)
				} else {
					panic(fmt.Sprintf("Invalid default type for flag %s, expected string", flag))
				}
			case "bool":
				if defaultValue, ok := meta.Default.(bool); ok { // Updated type assertion for boolean
					transferCmd.Flags().Bool(flag, defaultValue, meta.Usage)
				} else {
					panic(fmt.Sprintf("Invalid default type for flag %s, expected bool", flag))
				}
			case "int":
				if defaultValue, ok := meta.Default.(int); ok {
					transferCmd.Flags().Int(flag, defaultValue, meta.Usage)
				} else {
					panic(fmt.Sprintf("Invalid default type for flag %s, expected int", flag))
				}
			default:
				panic(fmt.Sprintf("Unsupported flag type for %s: %s", flag, meta.Type))
			}
		}
	}
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
			{"--in-github-all-versions", "Fetch all SBOMs for all versions (optional)"},
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
			{"--out-interlynk-url", "URL for output adapter interlynk (required)"},
			{"--out-interlynk-project-id", "Project ID for output adapter interlynk (required)"},
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
