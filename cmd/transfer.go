package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/source/github"
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
	// Download the SBOM
	err = github.DownloadSBOM(ctx, cfg.FromURL, outPutDir)
	if err != nil {
		logger.LogError(ctx, err, "Failed to fetch SBOM")
		return err
	}

	// Placeholder for actual transfer logic
	logger.LogInfo(ctx, "Starting SBOM transfer...")
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
	projectID, _ := cmd.Flags().GetString("project-id")

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
	targetURL, err := url.Parse(toURL)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	// Ensure target URL uses HTTPS
	if targetURL.Scheme != "https" {
		return fmt.Errorf("target URL must use HTTPS")
	}

	return nil
}
