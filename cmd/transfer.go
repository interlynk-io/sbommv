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

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var transferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Transfer SBOMs between systems",
	Long: `Transfer SBOMs from a source system (e.g., GitHub) to a target system (e.g., Interlynk).
	
Example usage:
	
	# Transfer all SBOMs from cosign release page to interlynk platform to a provided project ID
	sbommv transfer -D  --input-adapter=github  --in-github-url="https://github.com/sigstore/cosign" --output-adapter=interlynk  --out-interlynk-url="https://localhost:3000/lynkapi" --out-interlynk-project-id=014eda95-5ac6-4bd8-a24d-014217f0b873

	# Transfer all SBOMs from the Cosign release page for version 2.4.0 to the Interlynk platform under the specified project ID
	sbommv transfer -D  --input-adapter=github  --in-github-url="https://github.com/sigstore/cosign@v2.4.0" --output-adapter=interlynk  --out-interlynk-url="https://localhost:3000/lynkapi" --out-interlynk-project-id=07fb3477-1273-4996-bc14-fe0c2cc100d7

	# Transfer SBOMs for all versions of sbomqs from github and uploads it to dtrack, create new project in dtrack for each sbomqs version
	sbommv transfer --input-adapter=github --in-github-url=https://github.com/interlynk-io/sbomqs --in-github-all-versions=true --output-adapter=interlynk --out-interlynk-url=https://localhost:3000/lynkapi

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
	transferCmd.Flags().String("input-adapter", "", "Input adapter type (github, s3, file, folder, interlynk)")
	transferCmd.MarkFlagRequired("input-adapter")

	// Output adapter flags
	transferCmd.Flags().String("output-adapter", "", "Output adapter type (interlynk, dtrack)")
	transferCmd.MarkFlagRequired("output-adapter")

	transferCmd.Flags().BoolP("dry-run", "", false, "enable dry run mode")

	transferCmd.Flags().BoolP("debug", "D", false, "Enable debug logging")
}

func transferSBOM(cmd *cobra.Command, args []string) error {
	// Suppress automatic usage message for non-flag errors
	cmd.SilenceUsage = true

	// Initialize logger based on debug flag
	debug, _ := cmd.Flags().GetBool("debug")
	logger.InitLogger(debug, false) // Using console format as default
	defer logger.Sync()             // Ensure logs are flushed on exit

	ctx := logger.WithLogger(context.Background())

	viper.AutomaticEnv()

	config, err := parseAdaptersConfig(cmd)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	if config.SourceType == string(source.SourceGithub) {
		if config.SourceConfigs["version"] == "" {
			logger.LogDebug(ctx, "all SBOMs from all versions of the repository will be fetched")
		}
	}

	logger.LogDebug(ctx, "dry run mode", "value", config.DryRun)

	if config.DestinationConfigs == nil || config.SourceConfigs == nil {
		logger.LogError(ctx, nil, "Failed to construct config")
		os.Exit(1)
	}

	// execute core engine operation
	err = engine.TransferRun(ctx, config)
	if err != nil {
		return fmt.Errorf("Failed to process engine for transfer cmd %v", err)
	}

	// Delete the "sboms" folder
	if err := os.RemoveAll("sboms"); err != nil {
		logger.LogError(ctx, err, "Failed to delete the directory")
		return fmt.Errorf("failed to delete directory %s: %w", "sboms", err)
	}
	logger.LogDebug(ctx, "Successfully deleted the directory", "directory", "sboms")

	return nil
}

func parseAdaptersConfig(cmd *cobra.Command) (mvtypes.Config, error) {
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
		config.SourceConfigs["method"] = method

	case source.SourceDependencyTrack:
		url, err := cmd.Flags().GetString("in-dtrack-url")
		if err != nil || url == "" {
			return config, fmt.Errorf("missing or invalid flag: in-dtrack-url")
		}

		projectID, err := cmd.Flags().GetString("in-dtrack-project-id")
		if err != nil || projectID == "" {
			return config, fmt.Errorf("missing or invalid flag: in-dtrack-project-id")
		}

		// Get token from environment
		token := viper.GetString("DTRACK_API_TOKEN")

		config.SourceConfigs["url"] = url
		config.SourceConfigs["token"] = token
		config.SourceConfigs["projectID"] = projectID

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
		fmt.Println("DEST URL: ", url)

		// push all sbom version
		pushAllVersion, _ := cmd.Flags().GetBool("in-github-all-versions")
		projectID, err := cmd.Flags().GetString("out-interlynk-project-id")
		if err != nil || (projectID == "" && !pushAllVersion) {
			return config, fmt.Errorf("missing or invalid flag: out-interlynk-project-id")
		}

		// Get token from environment
		token := viper.GetString("INTERLYNK_API_TOKEN")

		config.DestinationConfigs["url"] = url
		config.DestinationConfigs["token"] = token
		config.DestinationConfigs["projectID"] = projectID
		config.DestinationConfigs["pushAllVersion"] = pushAllVersion

	case dest.DestDependencyTrack:
		url, err := cmd.Flags().GetString("out-dtrack-url")
		if err != nil || url == "" {
			return config, fmt.Errorf("missing or invalid flag: out-dtrack-url")
		}

		projectID, err := cmd.Flags().GetString("out-dtrack-project-id")
		if err != nil || projectID == "" {
			return config, fmt.Errorf("missing or invalid flag: out-dtrack-project-id")
		}

		// Get token from environment
		token := viper.GetString("DTRACK_API_TOKEN")

		config.DestinationConfigs["url"] = url
		config.DestinationConfigs["token"] = token
		config.DestinationConfigs["projectID"] = projectID

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
		source.SourceS3: {
			"in-s3-bucket": {Usage: "Bucket name for input adapter s3", Default: "", Type: "string"},
			"in-s3-region": {Usage: "Region for input adapter s3", Default: "", Type: "string"},
		},
		source.SourceDependencyTrack: {
			"in-dtrack-url":        {Usage: "URL for input adapter dtrack", Default: "", Type: "string"},
			"in-dtrack-project-id": {Usage: "Project ID for input adapter dtrack", Default: "", Type: "string"},
		},
		source.SourceInterlynk: {
			"in-interlynk-url":        {Usage: "URL for input adapter interlynk", Default: "", Type: "string"},
			"in-interlynk-project-id": {Usage: "Project ID for input adapter interlynk", Default: "", Type: "string"},
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
		dest.DestDependencyTrack: {
			"out-dtrack-url":        {Usage: "URL for output adapter dtrack", Default: "", Type: "string"},
			"out-dtrack-project-id": {Usage: "Project ID for output adapter dtrack", Default: "", Type: "string"},
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
		"dtrack": {
			{"--in-dtrack-url", "URL for input adapter dtrack (required)"},
			{"--in-dtrack-project-id", "Project ID for input adapter dtrack (required)"},
		},
		"interlynk": {
			{"--in-interlynk-url", "URL for input adapter interlynk (required)"},
			{"--in-interlynk-project-id", "Project ID for input adapter interlynk (required)"},
		},
		"s3": {
			{"--in-s3-bucket", "Bucket name for input adapter s3 (required)"},
			{"--in-s3-region", "Region for input adapter s3 (required)"},
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
		"dtrack": {
			{"--out-dtrack-url", "URL for output adapter dtrack (required)"},
			{"--out-dtrack-project-id", "Project ID for output adapter dtrack (required)"},
		},
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
	builder.WriteString("      --input-adapter Input adapter type (github, s3, file, folder, interlynk) (required)\n")
	builder.WriteString("      --output-adapter Output adapter type (interlynk, dtrack) (required)\n")

	return builder.String()
}
