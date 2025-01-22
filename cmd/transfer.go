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
	"time"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/source/github"
	"github.com/interlynk-io/sbommv/pkg/target/interlynk"
	"github.com/interlynk-io/sbommv/pkg/utils"
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
	sbommv transfer --from-url=github.com/org/repo --to-url=https://api.interlynk.io --interlynk-project-id=1234 --gen-sbom-using=cdxgen
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

	outPutDir := "allSboms"
	jsonDir := "allSboms/json"

	// Download the SBOM
	allSBOMs, err := github.DownloadSBOM(ctx, cfg.FromURL, outPutDir)
	if err != nil {
		logger.LogError(ctx, err, "Failed to fetch SBOM")
		return err
	}

	// Convert all SBOMs to JSON format
	var jsonSBOMs []string
	for _, sbomPath := range allSBOMs {
		jsonPath, err := utils.ConvertToJSON(sbomPath, jsonDir)
		if err != nil {
			logger.LogError(ctx, err, "Failed to convert SBOM to JSON")
			continue
		}
		jsonSBOMs = append(jsonSBOMs, jsonPath)
		logger.LogInfo(ctx, "Converted SBOM to JSON", "original", sbomPath, "json", jsonPath)
	}

	// Initialize Interlynk client
	client := interlynk.NewClient(interlynk.Config{
		Token:     cfg.Token,
		ProjectID: cfg.ProjectID,
	})

	// Initialize upload service
	uploadService := interlynk.NewUploadService(client, interlynk.UploadOptions{
		MaxAttempts:   3,
		MaxConcurrent: 1,
		RetryDelay:    time.Second,
	})

	// Upload SBOMs
	results := uploadService.UploadSBOMs(ctx, jsonSBOMs)

	// Log results
	for _, result := range results {
		if result.Error != nil {
			logger.LogError(ctx, result.Error, "Failed to upload SBOM")
		} else {
			logger.LogInfo(ctx, "SBOM uploaded successfully", "file", result.Path)
		}
	}

	// Delete the "allSBOMs" folder
	if err := os.RemoveAll(outPutDir); err != nil {
		logger.LogError(ctx, err, "Failed to delete the directory")
		return fmt.Errorf("failed to delete directory %s: %w", outPutDir, err)
	}

	logger.LogInfo(ctx, "Successfully deleted the directory", "directory", outPutDir)

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
