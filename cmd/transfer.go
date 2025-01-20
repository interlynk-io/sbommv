package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	// Initialize logger based on debug flag
	debug, _ := cmd.Flags().GetBool("debug")
	logger.InitLogger(debug, false) // Using console format as default
	defer logger.Sync()             // Ensure logs are flushed on exit

	ctx := logger.WithLogger(context.Background())

	viper.AutomaticEnv()

	// Retrieve API token and validate it
	token := viper.GetString("INTERLYNK_API_TOKEN")
	if token == "" || !strings.HasPrefix(token, "lynk_") {
		logger.LogError(ctx, nil, "Missing or invalid token. Please set the INTERLYNK_API_TOKEN environment variable")
		return fmt.Errorf("missing or invalid token")
	}

	tCmd := toTransferCmd(ctx, cmd)

	if tCmd == nil {
		logger.LogError(ctx, nil, "Failed to construct TransferCmd")
		os.Exit(1)
	}

	logger.LogDebug(ctx, "Transfer command constructed successfully", "command", tCmd)

	// Placeholder for actual transfer logic
	logger.LogInfo(ctx, "Starting SBOM transfer...")
	return nil
}

func toTransferCmd(ctx context.Context, cmd *cobra.Command) *TransferCmd {
	tCmd := &TransferCmd{}

	tCmd.FromURL, _ = cmd.Flags().GetString("from-url")
	tCmd.ToURL, _ = cmd.Flags().GetString("to-url")
	tCmd.ProjectID, _ = cmd.Flags().GetString("interlynk-project-id")
	tCmd.SbomTool, _ = cmd.Flags().GetString("gen-sbom-using")
	tCmd.Debug, _ = cmd.Flags().GetBool("debug")
	tCmd.Token = viper.GetString("INTERLYNK_API_TOKEN")

	// Validate critical inputs
	if tCmd.FromURL == "" || tCmd.ToURL == "" || tCmd.ProjectID == "" {
		logger.LogError(ctx, nil, "Missing required fields: from-url, to-url, or interlynk-project-id")
		return nil
	}

	logger.LogDebug(ctx, "Parsed TransferCmd successfully", "from-url", tCmd.FromURL, "to-url", tCmd.ToURL, "project-id", tCmd.ProjectID)

	return tCmd
}
