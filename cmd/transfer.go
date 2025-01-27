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

type TransferCmd struct {
	FromURL   string
	ToURL     string
	ProjectID string
	SbomTool  string
	Debug     bool
	Token     string

	InputAdapter   string
	OutputAdataper string
}

var transferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Transfer SBOMs between systems",
	Long: `Transfer SBOMs from a source system (e.g., GitHub) to a target system (e.g., Interlynk).
	
Example usage:
	
	# transfer all SBOMs from cosign release page to interlynk platform to a provided project ID
	sbommv transfer -D  --input-adapter=github  --in-github-url="https://github.com/sigstore/cosign" --output-adapter=interlynk  --out-dtrack-url="https://localhost:3000/lynkapi" --out-interlynk-project-id=014eda95-5ac6-4bd8-a24d-014217f0b873
	`,
	Args: cobra.NoArgs,
	RunE: transferSBOM,
}

func init() {
	rootCmd.AddCommand(transferCmd)
	setInputAdapterDynamicFlags(transferCmd)
	setOutputAdapterDynamicFlags(transferCmd)

	// Input adapter flags
	transferCmd.Flags().String("input-adapter", "", "Input adapter type (github, s3, file, folder, interlynk)")
	transferCmd.MarkFlagRequired("input-adapter")

	// Output adapter flags
	transferCmd.Flags().String("output-adapter", "", "Output adapter type (interlynk, dtrack)")
	transferCmd.MarkFlagRequired("output-adapter")

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
	logger.LogInfo(ctx, "Successfully deleted the directory", "directory", "sboms")

	return nil
}

func parseAdaptersConfig(cmd *cobra.Command) (mvtypes.Config, error) {
	inputType, _ := cmd.Flags().GetString("input-adapter")
	outputType, _ := cmd.Flags().GetString("output-adapter")

	config := mvtypes.Config{
		SourceType:         inputType,
		DestinationType:    outputType,
		SourceConfigs:      map[string]interface{}{},
		DestinationConfigs: map[string]interface{}{},
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
		config.SourceConfigs["url"] = repoURL
		config.SourceConfigs["version"] = version

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

		projectID, err := cmd.Flags().GetString("out-interlynk-project-id")
		if err != nil || projectID == "" {
			return config, fmt.Errorf("missing or invalid flag: out-interlynk-project-id")
		}
		// Get token from environment
		token := viper.GetString("INTERLYNK_API_TOKEN")

		config.DestinationConfigs["url"] = url
		config.DestinationConfigs["token"] = token
		config.DestinationConfigs["projectID"] = projectID

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
	inputAdapters := map[source.InputType]map[string]struct {
		Usage   string
		Default string
	}{
		source.SourceGithub: {
			"in-github-url":    {"URL for input adapter github", ""},
			"in-github-method": {"Method for input adapter github", "release"},
		},
		source.SourceS3: {
			"in-s3-bucket": {"Bucket name for input adapter s3", ""},
			"in-s3-region": {"Region for input adapter s3", ""},
		},
		source.SourceDependencyTrack: {
			"in-dtrack-url":        {"URL for input adapter dtrack", ""},
			"in-dtrack-project-id": {"Project ID for input adapter dtrack", ""},
		},
		source.SourceInterlynk: {
			"in-interlynk-url":        {"URL for input adapter interlynk", ""},
			"in-interlynk-project-id": {"Project ID for input adapter interlynk", ""},
		},
	}

	// Dynamically register input adapter flags with default values
	for _, flags := range inputAdapters {
		for flag, meta := range flags {
			transferCmd.Flags().String(flag, meta.Default, meta.Usage)
		}
	}
}

func setOutputAdapterDynamicFlags(transferCmd *cobra.Command) {
	// Define output adapters and their flags with specific descriptions and default values
	outputAdapters := map[dest.OutputType]map[string]struct {
		Usage   string
		Default string
	}{
		dest.DestInterlynk: {
			"out-interlynk-url":        {"URL for output adapter interlynk", "https://api.interlynk.io/lynkapi"},
			"out-interlynk-project-id": {"Project ID for output adapter interlynk", ""},
		},
		dest.DestDependencyTrack: {
			"out-dtrack-url":        {"URL for output adapter dtrack", ""},
			"out-dtrack-project-id": {"Project ID for output adapter dtrack", ""},
		},
	}

	// Dynamically register output adapter flags with default values
	for _, flags := range outputAdapters {
		for flag, meta := range flags {
			transferCmd.Flags().String(flag, meta.Default, meta.Usage)
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
