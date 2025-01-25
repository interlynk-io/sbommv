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
	"time"

	adapter "github.com/interlynk-io/sbommv/pkg/adapters"
	source "github.com/interlynk-io/sbommv/pkg/adapters/source"

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
	Adapter   bool
}

var transferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Transfer SBOMs between systems",
	Long: `Transfer SBOMs from a source system (e.g., GitHub) to a target system (e.g., Interlynk).
	
Example usage:
	# transfer SBOMs from github to interlynk
	sbommv transfer --from-url=github.com/org/repo --to-url=https://api.interlynk.io --interlynk-project-id=1234 --gen-sbom-using=cdxgen
	
	# transfer SBOMs from local folder to interlynk
	sbommv transfer --from-url=/sboms-dir --to-url=https://api.interlynk.io --interlynk-project-id=1234

	# transfer single SBOM file from local to interlynk
	sbommv transfer --from-url=sboms.json --to-url=https://api.interlynk.io --interlynk-project-id=1234

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
	transferCmd.Flags().BoolP("adapter", "a", false, "adapter method")

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

	adpCfg, err := parseAdapterConfig(cfg)
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

	outPutDir := "allSboms"
	var allSBOMs []string
	if cfg.Adapter {

		logger.LogInfo(ctx, "adapter mode", "value", cfg.Adapter)

		sourceType, err := utils.DetectSourceType(adpCfg.URL)
		if err != nil {
			return fmt.Errorf("input URL is invalid source type")
		}

		logger.LogInfo(ctx, "input adapter", "source", sourceType)

		sourceAdapter, err := adapter.NewSourceAdapter(sourceType, adpCfg)
		if err != nil {
			return fmt.Errorf("Failed to get an Adapter")
		}

		allSBOMs, err = sourceAdapter.GetSBOMs(ctx)
		if err != nil {
			return fmt.Errorf("Failed to get SBOMs...")
		}

	} else {

		logger.LogInfo(ctx, "adapter mode", "value", cfg.Adapter)

		// Download the SBOM
		allSBOMs, err = github.GetSBOMs(ctx, cfg.FromURL, outPutDir)
		if err != nil {
			logger.LogError(ctx, err, "Failed to fetch SBOM")
			return err
		}
	}
	logger.LogInfo(ctx, "All retieved SBOMs from source", "sboms", allSBOMs)

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
	results := uploadService.UploadSBOMs(ctx, allSBOMs)

	noOfSuccessfullyUploadedFile := 0
	for _, result := range results {
		if result.Error != nil {
			logger.LogInfo(ctx, "Failed to upload SBOM", "path", result.Path)
			continue
		} else {
			noOfSuccessfullyUploadedFile++
			logger.LogInfo(ctx, "SBOM uploaded successfully", "file", result.Path)
		}
	}
	logger.LogInfo(ctx, "SBOM uploaded successfully", "total", noOfSuccessfullyUploadedFile)

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

func parseAdapterConfig(cfg *TransferCmd) (source.AdapterConfig, error) {
	adpCfg := source.AdapterConfig{}
	adpCfg.URL = cfg.FromURL

	// in case of file or folder, the source URL could be path
	adpCfg.Path = cfg.FromURL
	adpCfg.APIKey = cfg.Token

	// by default recurssive is false
	adpCfg.Recursive = false
	if cfg.SbomTool == "" {
		adpCfg.Method = source.MethodReleases
	} else {
		adpCfg.Method = source.MethodGenerate
	}
	return adpCfg, nil
}

func parseTransferConfig(ctx context.Context, cmd *cobra.Command) (*TransferCmd, error) {
	cfg := &TransferCmd{}

	// Parse required flags
	fromURL, _ := cmd.Flags().GetString("from-url")
	toURL, _ := cmd.Flags().GetString("to-url")
	projectID, _ := cmd.Flags().GetString("interlynk-project-id")

	adapter, _ := cmd.Flags().GetBool("adapter")

	// Validate URLs
	if err := utils.ValidateURLs(fromURL, toURL); err != nil {
		logger.LogError(ctx, err, "Error validating URLs")
		return nil, err
	}

	cfg.FromURL = fromURL
	cfg.ToURL = toURL
	cfg.ProjectID = projectID
	cfg.Adapter = adapter

	// Parse optional flags
	cfg.SbomTool, _ = cmd.Flags().GetString("gen-sbom-using")
	cfg.Debug, _ = cmd.Flags().GetBool("debug")

	// Get token from environment
	cfg.Token = viper.GetString("INTERLYNK_API_TOKEN")

	logger.LogDebug(ctx, "Parsed TransferCmd successfully", "from-url", cfg.FromURL, "to-url", cfg.ToURL, "project-id", cfg.ProjectID)

	return cfg, nil
}
