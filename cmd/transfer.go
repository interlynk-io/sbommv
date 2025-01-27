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
	"net/url"
	"os"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/adapters/dest"
	source "github.com/interlynk-io/sbommv/pkg/adapters/source"
	"github.com/interlynk-io/sbommv/pkg/engine"

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
}

var transferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Transfer SBOMs between systems",
	Long: `Transfer SBOMs from a source system (e.g., GitHub) to a target system (e.g., Interlynk).
	
Example usage:
	sbommv transfer --from-url=<source-url> --to-url=<target-url> --interlynk-project-id=<project-id>

	# Fetch sbom from cosign and transfer to interlynk production to a specific project ID
	sbommv transfer --from-url="https://github.com/sigstore/cosign" --to-url="https://api.interlynk.io/lynkapi" --interlynk-project-id=85c9d898-00ac-44c2-b5df-de035b263104

	# Fetch sbom from cosign and transfer to interlynk localhost to a specific project ID in Debug Mode
	sbommv transfer  -D --from-url="https://github.com/sigstore/cosign" --to-url="http://localhost:3000/lynkapi" --interlynk-project-id=85c9d898-00ac-44c2-b5df-de035b26310
	
	`,
	Args: cobra.NoArgs,
	RunE: transferSBOM,
}

func init() {
	rootCmd.AddCommand(transferCmd)

	transferCmd.Flags().StringP("from-url", "f", "", "Source URL (e.g., GitHub repo or org)")
	transferCmd.MarkFlagRequired("from-url")

	transferCmd.Flags().StringP("to-url", "t", "", "Target URL (e.g., Interlynk API endpoint)")
	transferCmd.MarkFlagRequired("to-url")

	transferCmd.Flags().StringP("interlynk-project-id", "p", "", "Project ID in Interlynk")
	transferCmd.MarkFlagRequired("interlynk-project-id")

	// Optional
	transferCmd.Flags().StringP("gen-sbom-using", "s", "", "Tool for generating SBOM (e.g., cdxgen)")

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

	cfg, err := parseTransferConfig(ctx, cmd)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	sourceAdpCfg, destAdpCfg, err := parseAdapterConfig(cfg)
	if err != nil {
		logger.LogError(ctx, nil, "Failed to construct adapter config")
	}

	if cfg == nil {
		logger.LogError(ctx, nil, "Failed to construct TransferCmd")
		os.Exit(1)
	}

	// Validate authentication token
	if err := validateToken(cfg.Token); err != nil {
		logger.LogError(ctx, nil, "Missing or invalid token. Please set the INTERLYNK_API_TOKEN variable")
		return fmt.Errorf(" missing or invalid: %w", err)
	}
	logger.LogDebug(ctx, "Transfer command constructed successfully", "command", cfg)

	// execute core engine operation
	err = engine.TransferRun(ctx, sourceAdpCfg, destAdpCfg)
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

func validateToken(token string) error {
	if token == "" {
		return fmt.Errorf("INTERLYNK_API_TOKEN environment variable is not set")
	}
	if !strings.HasPrefix(token, "lynk_") {
		return fmt.Errorf("invalid token format - must start with 'lynk_'")
	}
	if len(token) < 32 {
		return fmt.Errorf("token is too short - must be at least 32 characters")
	}
	return nil
}

func parseAdapterConfig(cfg *TransferCmd) (source.AdapterConfig, dest.AdapterConfig, error) {
	sourceAdpCfg := source.AdapterConfig{}
	destAdpConfig := dest.AdapterConfig{}

	sourceAdpCfg.URL = cfg.FromURL

	destAdpConfig.BaseURL = cfg.ToURL
	destAdpConfig.APIKey = cfg.Token
	destAdpConfig.ProjectID = cfg.ProjectID

	sourceAdpCfg.APIKey = cfg.Token
	if cfg.SbomTool == "" {
		sourceAdpCfg.Method = source.MethodReleases
	} else {
		sourceAdpCfg.Method = source.MethodGenerate
	}
	return sourceAdpCfg, destAdpConfig, nil
}

func parseTransferConfig(ctx context.Context, cmd *cobra.Command) (*TransferCmd, error) {
	cfg := &TransferCmd{}

	// Parse required flags
	fromURL, _ := cmd.Flags().GetString("from-url")
	toURL, _ := cmd.Flags().GetString("to-url")
	projectID, _ := cmd.Flags().GetString("interlynk-project-id")

	// Validate URLs
	if err := validateURLs(fromURL, toURL); err != nil {
		logger.LogError(ctx, err, "Error validating URLs")
		return nil, err
	}

	cfg.FromURL = fromURL
	cfg.ToURL = toURL
	cfg.ProjectID = projectID

	// Parse optional flags
	cfg.SbomTool, _ = cmd.Flags().GetString("gen-sbom-using")
	cfg.Debug, _ = cmd.Flags().GetBool("debug")

	// Get token from environment
	cfg.Token = viper.GetString("INTERLYNK_API_TOKEN")

	logger.LogDebug(ctx, "Parsed TransferCmd successfully", "from-url", cfg.FromURL, "to-url", cfg.ToURL, "project-id", cfg.ProjectID)

	return cfg, nil
}

func validateURLs(fromURL, toURL string) error {
	// Validate source URL
	if _, err := url.Parse(fromURL); err != nil {
		return fmt.Errorf("invalid source URL: %w", err)
	}

	// Validate target URL
	_, err := url.Parse(toURL)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	return nil
}
